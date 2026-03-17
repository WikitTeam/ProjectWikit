from multiprocessing import Value
import unicodedata
import shutil
import datetime
import re
import os.path

from pathlib import Path
from typing import Optional, Union, Sequence, Tuple, Dict

from django.conf import settings
from django.contrib.auth.models import AnonymousUser
from django.db import transaction
from django.db.models import QuerySet, Sum, Avg, Count, Max, IntegerField, Q, F
from django.db.models.functions import Coalesce

import renderer
from web.events import EventBase
from web.controllers import notifications, media
from web.models.articles import Article, ArticleLogEntry, ArticleVersion, Category, ExternalLink, Tag, TagsCategory, Vote
from web.models.files import File
from web.models.settings import Settings
from web.models.site import get_current_site
from web.models.users import User
from web.models.forum import ForumThread, ForumPost
from web.models.roles import Role
from web.util import lock_table
from web.types import _UserType, _FullNameOrArticle, _FullNameOrCategory, _FullNameOrTag, _UserIdOrUser


class AbstractArticleEvent(EventBase, is_abstract=True):
    user: _UserType
    full_name_or_article: _FullNameOrArticle

    @property
    def fullname(self):
        if isinstance(self.full_name_or_article, Article):
            return self.full_name_or_article.full_name
        return self.full_name_or_article
    
    @property
    def article(self):
        if isinstance(self.full_name_or_article, str):
            self.full_name_or_article = get_article(self.full_name_or_article)
        return self.full_name_or_article


class OnVote(AbstractArticleEvent):
    old_vote: Optional[Vote]
    new_vote: Optional[Vote]

    @property
    def is_new(self):
        return self.new_vote != None and self.old_vote == None
    
    @property
    def is_change(self):
        return self.new_vote != None and self.old_vote != None
    
    @property
    def is_remove(self):
        return self.new_vote == None and self.old_vote != None
    

class OnCreateArticle(AbstractArticleEvent):
    pass


class OnDeleteArticle(AbstractArticleEvent):
    pass


class OnEditArticle(AbstractArticleEvent):
    log_entry: ArticleLogEntry

    @property
    def is_new(self):
        return self.log_entry.type == ArticleLogEntry.LogEntryType.New


# 从完整名称获取（分类，名称）
def get_name(full_name: str) -> Tuple[str, str]:
    split = full_name.split(':', 1)
    if len(split) == 2:
        return split[0], split[1]
    return '_default', split[0]


def strip_accents(s):
   return ''.join(c for c in unicodedata.normalize('NFD', s) if unicodedata.category(c) != 'Mn')


def normalize_article_name(full_name: str) -> str:
    full_name = strip_accents(full_name.lower())
    translit_map = {
        'а': 'a', 'б': 'b', 'в': 'v', 'г': 'g', 'д': 'd', 'е': 'e', 'ж': 'z',
        'з': 'z', 'и': 'i', 'й': 'i', 'к': 'k', 'л': 'l', 'м': 'm', 'н': 'n', 'о': 'o',
        'п': 'p', 'р': 'r', 'с': 's', 'т': 't', 'у': 'u', 'ф': 'f', 'х': 'h', 'ц': 'c',
        'ч': 'c', 'ы': 'i', 'э': 'e', 'ю': 'u', 'я': 'a', 'і': 'i', 'ї': 'i', 'є': 'e',
        'ь': '', 'ъ': ''
    }
    full_name = ''.join(translit_map.get(c, c) for c in full_name)
    n = re.sub(r'[^A-Za-z0-9\-_:]+', '-', full_name).strip('-')
    n = re.sub(r':+', ':', n).lower().strip(':')
    category, name = get_name(n)
    if category == '_default':
        return name
    return '%s:%s' % (category, name)


def denormalize_article_name(full_name: str):
    if ':' not in full_name:
        return f'_default:{full_name}'
    return full_name


def get_article(full_name_or_article: _FullNameOrArticle) -> Optional[Article]:
    if full_name_or_article is None:
        return None
    if type(full_name_or_article) == str:
        full_name_or_article = full_name_or_article.lower()
        category, name = get_name(full_name_or_article)
        return Article.objects.filter(category=category, name=name).first()
    if not isinstance(full_name_or_article, Article):
        raise ValueError('预期为字符串或Article对象')
    return full_name_or_article


def get_full_name(full_name_or_article: _FullNameOrArticle) -> str:
    if full_name_or_article is None:
        return ''
    if isinstance(full_name_or_article, str):
        return full_name_or_article
    return full_name_or_article.full_name


def deduplicate_name(full_name: str, allowed_article: Optional[Article] = None) -> str:
    i = 0
    while True:
        i += 1
        name_to_try = '%s-%d' % (full_name, i) if i > 1 else full_name
        article2 = get_article(name_to_try)
        if not article2 or (allowed_article and article2.pk == allowed_article.pk):
            return name_to_try


# 使用指定ID创建文章。不添加版本
def create_article(full_name: str, user: _UserType=None) -> Article:
    category, name = get_name(full_name)
    article = Article(
        category=category,
        name=name,
        created_at=datetime.datetime.now(),
        title=name,
    )
    article.save()
    if user:
        article.authors.add(user)
    OnCreateArticle(user, article).emit()
    return article


# 向文章添加日志条目
def add_log_entry(full_name_or_article: _FullNameOrArticle, log_entry: ArticleLogEntry):
    article = get_article(full_name_or_article)
    if not article:
        raise ValueError(f'未找到文章 {full_name_or_article}')

    with transaction.atomic():
        with lock_table(ArticleLogEntry):
            # 此黑魔法强制锁定此文章ID上的ArticleLogEntry表，
            # 使两个并发的日志条目相互等待，从而不会违反rev_number的唯一约束。
            current_log = ArticleLogEntry.objects.select_related('article')\
                                                 .select_for_update()\
                                                 .filter(article=article)
            max_rev_number = current_log.aggregate(max=Max('rev_number')).get('max')
            if max_rev_number is None:
                max_rev_number = -1
            log_entry.rev_number = max_rev_number + 1
            log_entry.save()

            OnEditArticle(log_entry.user, article, log_entry).emit()

            article.updated_at = log_entry.created_at
            article.save()


# 获取文章的所有日志条目（已排序）
def get_log_entries(full_name_or_article: _FullNameOrArticle) -> QuerySet[ArticleLogEntry]:
    article = get_article(full_name_or_article)
    return ArticleLogEntry.objects.filter(article=article).order_by('-rev_number')


# 获取文章的最新日志条目
def get_latest_log_entry(full_name_or_article: _FullNameOrArticle) -> Optional[ArticleLogEntry]:
    return get_log_entries(full_name_or_article).first()


# 获取文章的日志条目列表（已排序），指定范围
def get_log_entries_paged(full_name_or_article: _FullNameOrArticle, c_from: int, c_to: int, get_all: bool = False) -> Tuple[QuerySet[ArticleLogEntry], int]:
    log_entries = get_log_entries(full_name_or_article)
    total_count = len(log_entries)
    if not get_all:
        log_entries = log_entries[c_from:c_to]
    return log_entries, total_count


# 回退所有修订版本至特定版本
def revert_article_version(full_name_or_article: _FullNameOrArticle, rev_number: int, user: _UserType=None):
    article = get_article(full_name_or_article)
    if not article:
        raise ValueError(f'未找到文章 {full_name_or_article}')
    
    pref_full_name = get_full_name(full_name_or_article)

    new_props = {}

    for entry in get_log_entries(article):
        if entry.rev_number <= rev_number:
            break

        if entry.type == ArticleLogEntry.LogEntryType.Source:
            new_props['source'] = get_previous_version(entry.meta['version_id']).source
        elif entry.type == ArticleLogEntry.LogEntryType.Title:
            new_props['title'] = entry.meta['prev_title']
        elif entry.type == ArticleLogEntry.LogEntryType.Name:
            new_props['name'] = entry.meta['prev_name']
        elif entry.type == ArticleLogEntry.LogEntryType.Tags:
            if 'added_tags' not in new_props:
                new_props['added_tags'] = []
            if 'removed_tags' not in new_props:
                new_props['removed_tags'] = []
            # 逻辑：之前移除的标签现在应添加
            #        之前添加的标签现在应移除
            for tag in entry.meta['added_tags']:
                try:
                    new_props['added_tags'].remove(tag['id'])
                except ValueError:
                    pass
                new_props['removed_tags'].append(tag['id'])
            for tag in entry.meta['removed_tags']:
                try:
                    new_props['removed_tags'].remove(tag['id'])
                except ValueError:
                    pass
                new_props['added_tags'].append(tag['id'])
        elif entry.type == ArticleLogEntry.LogEntryType.New:
            # 'new'无法回退。此外，我们根本不应达到此点。
            # 但如果达到了，就停止
            break
        elif entry.type == ArticleLogEntry.LogEntryType.Parent:
            new_props['parent'] = entry.meta['prev_parent_id']
        elif entry.type == ArticleLogEntry.LogEntryType.FileAdded and 'id' in entry.meta:
            if 'files_deleted' not in new_props:
                new_props['files_deleted'] = {}
            if 'files_restored' not in new_props:
                new_props['files_restored'] = {}
            new_props['files_deleted'][entry.meta['id']] = True
            new_props['files_restored'][entry.meta['id']] = False
        elif entry.type == ArticleLogEntry.LogEntryType.FileDeleted and 'id' in entry.meta:
            if 'files_restored' not in new_props:
                new_props['files_restored'] = {}
            if 'files_deleted' not in new_props:
                new_props['files_deleted'] = {}
            new_props['files_restored'][entry.meta['id']] = True
            new_props['files_deleted'][entry.meta['id']] = False
        elif entry.type == ArticleLogEntry.LogEntryType.FileRenamed and 'id' in entry.meta:
            if 'files_renamed' not in new_props:
                new_props['files_renamed'] = {}
            new_props['files_renamed'][entry.meta['id']] = entry.meta['prev_name']
        elif entry.type == ArticleLogEntry.LogEntryType.Wikidot:
            # 这是一个伪修订类型。
            pass
        elif entry.type == ArticleLogEntry.LogEntryType.VotesDeleted:
            new_props['votes'] = entry.meta
        elif entry.type == ArticleLogEntry.LogEntryType.Authorship:
            if 'added_authors' not in new_props:
                new_props['added_authors'] = []
            if 'removed_authors' not in new_props:
                new_props['removed_authors'] = []
            # 逻辑：之前移除的作者现在应添加
            #        之前添加的作者现在应移除
            for author in entry.meta['added_authors']:
                try:
                    new_props['added_authors'].remove(author)
                except ValueError:
                    pass
                new_props['removed_authors'].append(author)
            for author in entry.meta['removed_authors']:
                try:
                    new_props['removed_authors'].remove(author)
                except ValueError:
                    pass
                new_props['added_authors'].append(author)
        elif entry.type == ArticleLogEntry.LogEntryType.Revert:
            if 'source' in entry.meta:
                new_props['source'] = get_previous_version(entry.meta['source']['version_id']).source
            if 'title' in entry.meta:
                new_props['title'] = entry.meta['title']['prev_title']
            if 'name' in entry.meta:
                new_props['name'] = entry.meta['name']['prev_name']
            if 'parent' in entry.meta:
                new_props['parent'] = entry.meta['parent']['prev_parent_id']
            if 'tags' in entry.meta:
                if 'added_tags' not in new_props:
                    new_props['added_tags'] = []
                if 'removed_tags' not in new_props:
                    new_props['removed_tags'] = []
                # 逻辑：之前移除的标签现在应添加
                #        之前添加的标签现在应移除
                for tag in entry.meta['tags']['added']:
                    try:
                        new_props['added_tags'].remove(tag)
                    except ValueError:
                        pass
                    new_props['removed_tags'].append(tag)
                for tag in entry.meta['tags']['removed']:
                    try:
                        new_props['removed_tags'].remove(tag)
                    except ValueError:
                        pass
                    new_props['added_tags'].append(tag)
            if 'files' in entry.meta:
                if 'files_restored' not in new_props:
                    new_props['files_restored'] = {}
                if 'files_deleted' not in new_props:
                    new_props['files_deleted'] = {}
                if 'files_renamed' not in new_props:
                    new_props['files_renamed'] = {}
                for f in entry.meta['files']['added']:
                    new_props['files_deleted'][f['id']] = True
                    new_props['files_restored'][f['id']] = False
                for f in entry.meta['files']['deleted']:
                    new_props['files_restored'][f['id']] = True
                    new_props['files_deleted'][f['id']] = False
                for f in entry.meta['files']['renamed']:
                    new_props['files_renamed'][f['id']] = f['prev_name']
            if 'votes' in entry.meta:
                new_props['votes'] = entry.meta['votes']
            if 'authorship' in entry.meta:
                if 'added_authors' not in new_props:
                    new_props['added_authors'] = []
                if 'removed_authors' not in new_props:
                    new_props['removed_authors'] = []
                # 逻辑：之前移除的作者现在应添加
                #        之前添加的作者现在应移除
                for author in entry.meta['authorship']['added']:
                    try:
                        new_props['added_authors'].remove(author)
                    except ValueError:
                        pass
                    new_props['removed_authors'].append(author)
                for author in entry.meta['authorship']['removed']:
                    try:
                        new_props['removed_authors'].remove(author)
                    except ValueError:
                        pass
                    new_props['added_authors'].append(author)

    subtypes = []

    meta = {}

    files_added_meta = []
    files_deleted_meta = []
    files_renamed_meta = []

    for f_id, new_name in new_props.get('files_renamed', {}).items():
        try:
            file = File.objects.get(id=f_id)
            files_renamed_meta.append({'id': f_id, 'name': new_name, 'prev_name': file.name})
            file.name = new_name
            file.save()
        except File.DoesNotExist:
            continue

    for f_id, deleted in new_props.get('files_deleted', {}).items():
        if not deleted:
            continue
        try:
            file = File.objects.get(id=f_id)
            if not file.deleted_at:
                files_deleted_meta.append({'id': f_id, 'name': file.name})
                file.deleted_at = datetime.datetime.now()
                file.deleted_by = user
                file.save()
        except File.DoesNotExist:
            continue

    for f_id, restored in new_props.get('files_restored', {}).items():
        if not restored:
            continue
        try:
            file = File.objects.get(id=f_id)
            if file.deleted_at:
                files_added_meta.append({'id': f_id, 'name': file.name})
                file.deleted_at = None
                file.deleted_by = None
                file.save()
        except File.DoesNotExist:
            continue

    if files_added_meta or files_deleted_meta or files_renamed_meta:
        if files_added_meta:
            subtypes.append(ArticleLogEntry.LogEntryType.FileAdded)
        if files_deleted_meta:
            subtypes.append(ArticleLogEntry.LogEntryType.FileDeleted)
        if files_renamed_meta:
            subtypes.append(ArticleLogEntry.LogEntryType.FileRenamed)
        meta['files'] = {
            'added': files_added_meta,
            'deleted': files_deleted_meta,
            'renamed': files_renamed_meta
        }

    tags_added_meta = []
    tags_removed_meta = []

    tags = [x.pk for x in get_tags_internal(article)]
    for tag in new_props.get('removed_tags', []):
        # 安全措施：某些旧版本的修订中此处可能是字符串标签
        if not isinstance(tag, int):
            continue
        try:
            tags.remove(tag)
            tags_removed_meta.append(tag)
        except ValueError:
            pass
    for tag in new_props.get('added_tags', []):
        # 安全措施：某些旧版本的修订中此处可能是字符串标签
        if not isinstance(tag, int):
            continue
        tags.append(tag)
        tags_added_meta.append(tag)
    new_tags = list(Tag.objects.filter(id__in=tags))
    set_tags_internal(article, new_tags, user, False)

    if tags_added_meta or tags_removed_meta:
        subtypes.append(ArticleLogEntry.LogEntryType.Tags)
        meta['tags'] = {
            'added': tags_added_meta,
            'removed': tags_removed_meta
        }

    if 'source' in new_props:
        subtypes.append(ArticleLogEntry.LogEntryType.Source)
        version = ArticleVersion(
            article=article,
            source=new_props['source'],
            rendered=None
        )
        version.save()
        meta['source'] = {'version_id': version.pk}

    if 'title' in new_props:
        subtypes.append(ArticleLogEntry.LogEntryType.Title)
        meta['title'] = {'prev_title': article.title, 'title': new_props['title']}
        article.title = new_props['title']
        article.save()

    if 'name' in new_props:
        subtypes.append(ArticleLogEntry.LogEntryType.Name)
        meta['name'] = {'prev_name': article.name, 'name': new_props['name']}
        update_full_name(article, new_props['name'], user, False)

    if 'parent' in new_props:
        subtypes.append(ArticleLogEntry.LogEntryType.Parent)
        try:
            parent = Article.objects.get(id=new_props['parent'])
        except Article.DoesNotExist:
            parent = None
        meta['parent'] = {
            'parent': get_full_name(parent),
            'parent_id': new_props['parent'],
            'prev_parent': get_full_name(article.parent),
            'prev_parent_id': article.parent.id if article.parent else None
        }
        article.parent = parent
        article.save()

    if 'votes' in new_props:
        subtypes.append(ArticleLogEntry.LogEntryType.VotesDeleted)
        votes_meta = _get_article_votes_meta(article)
        meta['votes'] = votes_meta
        with transaction.atomic():
            Vote.objects.filter(article=article).delete()
            for vote in new_props['votes']['votes']:
                try:
                    vote_role = Role.objects.get(id=vote['role_id'])
                except Role.DoesNotExist:
                    vote_role = None
                try:
                    vote_user = User.objects.get(id=vote['user_id'])
                except User.DoesNotExist:
                    # 缺少用户ID意味着跳过此投票且无法恢复。
                    continue
                vote_date = datetime.datetime.fromisoformat(vote['date']) if vote['date'] else None
                new_vote = Vote(article=article, user=vote_user, date=vote_date, rate=vote['vote'], role=vote_role)
                new_vote.save()
                new_vote.date = vote_date
                new_vote.save()

    authors_added_meta = []
    authors_removed_meta = []

    authors = [x.id for x in get_authors(article)]
    for author in new_props.get('removed_authors', []):
        try:
            authors.remove(author)
            authors_removed_meta.append(author)
        except ValueError:
            pass
    for author in new_props.get('added_authors', []):
        authors.append(author)
        authors_added_meta.append(author)
    new_authors: list[_UserIdOrUser] = list(User.objects.filter(id__in=authors))
    set_authors(article, new_authors, user)

    if authors_added_meta or authors_removed_meta:
        subtypes.append(ArticleLogEntry.LogEntryType.Authorship)
        meta['authorship'] = {
            'added': tags_added_meta,
            'removed': tags_removed_meta
        }

    meta['rev_number'] = rev_number
    meta['subtypes'] = subtypes

    log = ArticleLogEntry(
        article=article,
        user=user,
        type=ArticleLogEntry.LogEntryType.Revert,
        meta=meta,
        comment=''
    )

    add_log_entry(article, log)
    media.symlinks_article_update(article, pref_full_name)


# 为指定文章创建新版本
def create_article_version(full_name_or_article: _FullNameOrArticle, source: str, user: _UserType = None, comment: str='') -> ArticleVersion:
    article = get_article(full_name_or_article)
    if not article:
        raise ValueError(f'未找到文章 {full_name_or_article}')
    is_new = get_latest_version(article) is None
    version = ArticleVersion(
        article=article,
        source=source,
        rendered=None
    )
    version.save()
    # 要么是NEW，要么是SOURCE
    if is_new:
        log = ArticleLogEntry(
            article=article,
            user=user,
            type=ArticleLogEntry.LogEntryType.New,
            meta={'version_id': version.pk, 'title': article.title},
            comment=comment
        )
    else:
        log = ArticleLogEntry(
            article=article,
            user=user,
            type=ArticleLogEntry.LogEntryType.Source,
            meta={'version_id': version.pk},
            comment=comment
        )
    add_log_entry(article, log)
    return version


# 基于文章版本刷新链接。
def refresh_article_links(article_version: ArticleVersion):
    article = article_version.article
    article_name = get_full_name(article)
    # 删除之前已知的所有链接
    ExternalLink.objects.filter(link_from=article_name).delete()
    # 解析当前源代码
    already_added = []
    rc = renderer.RenderContext(article=article_version.article, source_article=article_version.article, path_params={}, user=None)
    linked_pages, included_pages = renderer.single_pass_fetch_backlinks(article_version.source, rc)
    for linked_page in linked_pages:
        kt = '%s:include:%s' % (article_name.lower(), linked_page.lower())
        if kt in already_added:
            continue
        already_added.append(kt)
        new_link = ExternalLink(link_from=article_name.lower(), link_type=ExternalLink.Type.Include, link_to=linked_page.lower())
        new_link.save()
    # 查找链接
    for included_page in included_pages:
        kt = '%s:link:%s' % (article_name.lower(), included_page.lower())
        if kt in already_added:
            continue
        already_added.append(kt)
        new_link = ExternalLink(link_from=article_name.lower(), link_type=ExternalLink.Type.Link, link_to=included_page.lower())
        new_link.save()


# 更新文章名称
def update_full_name(full_name_or_article: _FullNameOrArticle, new_full_name: str, user: _UserType = None, log: bool = True):
    article = get_article(full_name_or_article)
    if not article:
        raise ValueError(f'未找到文章 {full_name_or_article}')

    prev_full_name = get_full_name(full_name_or_article)

    category, name = get_name(new_full_name)
    article.category = category
    article.name = name
    article.save()

    # 更新链接
    ExternalLink.objects.filter(link_from=new_full_name).delete()  # 这不应发生，但为确保起见
    ExternalLink.objects.filter(link_from=prev_full_name).update(link_from=new_full_name)

    if log:
        log_entry = ArticleLogEntry(
            article=article,
            user=user,
            type=ArticleLogEntry.LogEntryType.Name,
            meta={'name': new_full_name, 'prev_name': prev_full_name}
        )
        add_log_entry(article, log_entry)

    media.symlinks_article_update(article, prev_full_name)


def _get_article_votes_meta(full_name_or_article: _FullNameOrArticle):
    article = get_article(full_name_or_article)

    # 对于修订日志，我们存储：
    # - 评分模式
    # - 评分
    # - 投票数
    # - 人气值
    # - 每个用户的单独投票，表示为内部数据库值
    #   （用户ID + 用户名 + 投票值）

    rating, rating_votes, popularity, rating_mode = get_rating(article)
    votes = list(Vote.objects.filter(article=article))
    votes_meta = {
        'rating_mode': str(rating_mode),
        'rating': rating,
        'votes_count': rating_votes,
        'popularity': popularity,
        'votes': []
    }
    for vote in votes:
        votes_meta['votes'].append({
            'user_id': vote.user_id,  # type: ignore
            'vote': vote.rate,
            'role_id': vote.role_id, # type: ignore
            'date': vote.date.isoformat() if vote.date else None
        })
    return votes_meta

def delete_article_votes(full_name_or_article: _FullNameOrArticle, user: _UserType = None, log: bool = True):
    article = get_article(full_name_or_article)

    # 获取现有投票
    votes_meta = _get_article_votes_meta(article)
    Vote.objects.filter(article=article).delete()

    if log:
        log_entry = ArticleLogEntry(
            article=article,
            user=user,
            type=ArticleLogEntry.LogEntryType.VotesDeleted,
            meta=votes_meta
        )
        add_log_entry(article, log_entry)


# 更新文章标题
def update_title(full_name_or_article: _FullNameOrArticle, new_title: str, user: _UserType = None):
    article = get_article(full_name_or_article)
    if not article:
        raise ValueError(f'未找到文章 {full_name_or_article}')
    prev_title = article.title
    article.title = new_title
    article.save()
    log = ArticleLogEntry(
        article=article,
        user=user,
        type=ArticleLogEntry.LogEntryType.Title,
        meta={'title': new_title, 'prev_title': prev_title}
    )
    add_log_entry(article, log)


def delete_article(full_name_or_article: _FullNameOrArticle):
    article = get_article(full_name_or_article)
    if not article:
        raise ValueError(f'未找到文章 {full_name_or_article}')
    ExternalLink.objects.filter(link_from=get_full_name(full_name_or_article)).delete()
    media.symlinks_article_delete(article)
    article.delete()
    file_storage = Path(settings.MEDIA_ROOT) / 'media' / article.media_name
    # 这可能与文件上传存在竞态条件，因为文件系统不了解数据库事务
    for _ in range(3):
        try:
            if os.path.exists(file_storage):
                shutil.rmtree(file_storage)
                break
        except IOError:  # 预期：在我们删除时有用户上传，导致“目录不为空”
            pass


# 获取文章的特定条目
def get_log_entry(full_name_or_article: _FullNameOrArticle, rev_number: int) -> Optional[ArticleLogEntry]:
    try:
        article = get_article(full_name_or_article)
        return ArticleLogEntry.objects.get(article=article, rev_number=rev_number)
    except ArticleLogEntry.DoesNotExist:
        pass


# 获取文章的特定版本
def get_version(version_id: int) -> Optional[ArticleVersion]:
    try:
        return ArticleVersion.objects.get(id=version_id)
    except ArticleVersion.DoesNotExist:
        pass


# 获取相对于指定版本的上一版本
def get_previous_version(version_id: int) -> Optional[ArticleVersion]:
    try:
        version = ArticleVersion.objects.get(id=version_id)
        prev_version = ArticleVersion.objects.filter(article_id=version.article_id, created_at__lt=version.created_at).order_by('-created_at')[:1] # type: ignore
        if not prev_version:
            return None
        return prev_version[0]
    except ArticleVersion.DoesNotExist:
        pass


# 获取文章的最新版本
def get_latest_version(full_name_or_article: _FullNameOrArticle) -> Optional[ArticleVersion]:
    article = get_article(full_name_or_article)
    if article is None:
        return None
    latest_version = ArticleVersion.objects.filter(article=article).order_by('-created_at')[:1]
    if latest_version:
        return latest_version[0]


# 获取文章的最新源代码
def get_latest_source(full_name_or_article: _FullNameOrArticle) -> Optional[str]:
    ver = get_latest_version(full_name_or_article)
    if ver is not None:
        return ver.source
    return None


# 获取特定修订号处的文章源代码
def get_source_at_rev_num(full_name_or_article: _FullNameOrArticle, rev_num: int) -> Optional[str]:
    article = get_article(full_name_or_article)
    entry = get_log_entry(article, rev_num)
    if not entry:
        return None

    def get_version_from_meta(meta):
        if 'source' in meta:
            return get_version(meta['source']['version_id'])
        elif "version_id" in meta:
            return get_version(meta["version_id"])
        else:
            return None

    version = get_version_from_meta(entry.meta)
    if not version:
        log_entries = list(get_log_entries(article))
        for old_entry in log_entries[log_entries.index(entry):]:
            version = get_version_from_meta(old_entry.meta)
            if version:
                break

    if not version:
        return None

    return version.source


# 获取文章的父页面
def get_parent(full_name_or_article: _FullNameOrArticle) -> Optional[str]:
    article = get_article(full_name_or_article)
    if article is not None and article.parent:
        return article.parent.full_name


# 设置文章的父页面
def set_parent(full_name_or_article: _FullNameOrArticle, full_name_of_parent: _FullNameOrArticle, user: _UserType = None):
    article = get_article(full_name_or_article)
    if not article:
        raise ValueError(f'未找到文章 {full_name_or_article}')
    parent = get_article(full_name_of_parent) if full_name_of_parent else None
    prev_parent = get_full_name(article.parent) if article.parent else None
    if article.parent == parent:
        return
    parent_id = parent.pk if parent else None
    prev_parent_id = article.parent.id if article.parent else None
    article.parent = parent
    article.save()
    log = ArticleLogEntry(
        article=article,
        user=user,
        type=ArticleLogEntry.LogEntryType.Parent,
        meta={'parent': full_name_of_parent, 'prev_parent': prev_parent, 'parent_id': parent_id, 'prev_parent_id': prev_parent_id}
    )
    add_log_entry(article, log)


# 获取所有父页面（面包屑导航）
def get_breadcrumbs(full_name_or_article: _FullNameOrArticle) -> Sequence[Article]:
    article = get_article(full_name_or_article)
    output = []
    breadcrumb_ids = []
    while article and article.pk not in breadcrumb_ids:
        output.append(article)
        breadcrumb_ids.append(article.pk)
        article = article.parent
    return list(reversed(output))


# 获取页面分类
def get_category(full_name_or_category: _FullNameOrCategory) -> Optional[Category]:
    if isinstance(full_name_or_category, str):
        try:
            return Category.objects.get(name=full_name_or_category)
        except Category.DoesNotExist:
            return
    return full_name_or_category


def get_article_category(full_name_or_article: _FullNameOrArticle) -> Optional[Category]:
    if isinstance(full_name_or_article, str):
        category, _ = get_name(full_name_or_article)
    else:
        article = get_article(full_name_or_article)
        if not article:
            return None
        category = article.category
    return get_category(category)


# 标签名称验证
def is_tag_name_allowed(name: str) -> bool:
    return ' ' not in name


def get_tag(full_name_or_tag: _FullNameOrTag, create: bool = False) -> Optional[Tag]:
    if full_name_or_tag is None:
        return None
    if type(full_name_or_tag) == str:
        full_name_or_tag = full_name_or_tag.lower()
        category_name, name = get_name(full_name_or_tag)
        if create:
            category, _ = TagsCategory.objects.get_or_create(slug=category_name)
            tag, _ = Tag.objects.get_or_create(category=category, name=name)
            return tag
        try:
            category = TagsCategory.objects.get(slug=category_name)
            return Tag.objects.get(category=category, name=name)
        except (Tag.DoesNotExist, TagsCategory.DoesNotExist):
            return None
    if not isinstance(full_name_or_tag, Tag):
        raise ValueError('预期为字符串或Tag对象')
    return full_name_or_tag


# 从文章获取标签
def get_tags(full_name_or_article: _FullNameOrArticle) -> Sequence[str]:
    return list(sorted([x.full_name.lower() for x in get_tags_internal(full_name_or_article)]))


def get_tags_internal(full_name_or_article: _FullNameOrArticle) -> QuerySet[Tag]:
    article = get_article(full_name_or_article)
    if article:
        return article.tags.select_related('category')
    return Tag.objects.none()


def get_tags_categories(full_name_or_article: _FullNameOrArticle) -> Dict[TagsCategory, Sequence[Tag]]:
    article = get_article(full_name_or_article)
    if article:
        tags =