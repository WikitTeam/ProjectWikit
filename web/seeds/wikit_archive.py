# 注意：此迁移脚本读取 wikitCLI 生成的备份归档（与 DBotThePony 的 Wikidot 备份格式兼容，wikitCLI 已做兼容处理）。
# 依赖 py7zr（解压页面/帖子内容）与 beautifulsoup4（论坛 HTML 转 Wikidot 源码），两者均已列入 requirements.txt。
#
# 通过 `python manage.py seed -a <归档路径>` 调用，可用 flag 控制迁移范围与各项设置，详见 web/management/commands/seed.py。

import py7zr
import shutil
import os, os.path
import sys
import json
import re
import time
import codecs
import datetime
import math
import threading
import logging
from pathlib import Path
from uuid import uuid4

from bs4 import BeautifulSoup

from django.conf import settings
from django.db import transaction, IntegrityError, connection

from web.controllers import articles
from web import threadvars
from wikitgo import urls
from web.models.articles import ArticleVersion, ArticleLogEntry, Article, Vote
from web.models.files import File
from web.models.forum import ForumSection, ForumCategory, ForumThread, ForumPost, ForumPostVersion
from web.models.site import get_current_site
from web.models.users import User


####################################################################################################
# 共享辅助函数
####################################################################################################


def maybe_load_pages_meta(base_path_or_list):
    if type(base_path_or_list) == str:
        allfiles = os.listdir('%s/meta/pages' % base_path_or_list)
        pages = []
        for f in allfiles:
            with codecs.open('%s/meta/pages/%s' % (base_path_or_list, f), 'r', encoding='utf-8') as fp:
                meta = json.load(fp)
                meta['revisions'].sort(key=lambda x: x['revision'], reverse=True)
                meta['filename'] = f
                pages.append(meta)
        return pages
    return base_path_or_list


def run_in_threads(fn, pages):
    thread_count = 4
    per_thread_pages = []
    for i in range(thread_count):
        single_thread_cnt = int(math.ceil(len(pages) / thread_count))
        per_thread_pages.append(pages[i*single_thread_cnt:(i+1)*single_thread_cnt])
    threads = []
    site = get_current_site()
    for thread_work in per_thread_pages:
        def fn_wrapper(thread_work):
            try:
                with threadvars.context():
                    threadvars.put('current_site', site)
                    fn(thread_work)
            except:
                logging.error('线程失败：', exc_info=True)
                sys.exit(1)
        t = threading.Thread(target=fn_wrapper, args=(thread_work,))
        t.daemon = True
        t.start()
        threads.append(t)
    while True:
        any_alive = [t for t in threads if t.is_alive()]
        if any_alive:
            time.sleep(1)
        else:
            break


get_or_create_user_lock = threading.Lock()


def normalize_username(name: str) -> str:
    return re.sub(r'[^A-Za-z0-9]+', '-', name).strip('-')


@transaction.atomic
def get_or_create_user(user_name_or_id, user_data, user_data_by_un):
    if user_name_or_id is None:
        return None
    with get_or_create_user_lock:
        if type(user_name_or_id) == str:
            user_attrs = user_data_by_un.get(user_name_or_id, None)
            if user_attrs:
                user_name = normalize_username(user_attrs['full_name']) or user_name_or_id
            else:
                user_name = user_name_or_id
        elif type(user_name_or_id) == int:
            user_attrs = user_data.get(str(user_name_or_id), None)
            if user_attrs:
                user_name = normalize_username(user_attrs['full_name']) or user_attrs['username']
            else:
                user_name = '已删除-%d' % user_name_or_id
        else:
            raise TypeError('Wikidot用户的参数无效：%s' % repr(user_name_or_id))
        # 由于某些原因，这在线程中会导致问题。
        existing = list(User.objects.filter(wikidot_username=user_name))
        try:
            if not existing:
                with transaction.atomic():
                    new_user = User(type=User.UserType.Wikidot, username=uuid4(), wikidot_username=user_name, is_active=False)
                    new_user.save()
                return new_user
            return existing[0]
        except IntegrityError:
            existing = list(User.objects.filter(wikidot_username=user_name))
            return existing[0]


def init_users(base_path):
    allfiles = os.listdir('%s/_users' % base_path)
    g_users = {}
    g_users_by_username = {}
    for f in allfiles:
        if f == 'pending.json':
            continue
        with codecs.open('%s/_users/%s' % (base_path, f), 'r', encoding='utf-8') as fp:
            user_bucket = json.load(fp)
            for k in user_bucket:
                g_users[k] = user_bucket[k]
                g_users_by_username[user_bucket[k]['username']] = user_bucket[k]

    return g_users, g_users_by_username


####################################################################################################
# 迁移入口
####################################################################################################


def run(base_path, *, scope='all', force_tags=False, import_votes=True, update_existing=False):
    # 解压wikitCLI备份格式的wikidot存档
    # scope: 'all' 全部, 'pages' 仅页面（含文件/标签/评分/父页面）, 'forum' 仅论坛
    # force_tags: 强制迁移标签（绕过站点“禁止用户创建标签”设置，按需新建标签）
    # import_votes: 是否迁移页面评分（votings）
    # update_existing: 对已存在的文章也重新同步标签与评分（默认已存在文章整体跳过）
    base_path = base_path.rstrip('/')

    if scope in ('all', 'pages'):
        run_pages(base_path, force_tags=force_tags, import_votes=import_votes, update_existing=update_existing)

    if scope in ('all', 'forum'):
        # 论坛的评论串需要引用已存在的文章，因此必须在页面迁移之后运行
        run_forum(base_path)


####################################################################################################
# 页面 / 文件 / 标签 / 评分 / 父页面 迁移
####################################################################################################


def run_pages(base_path, force_tags=False, import_votes=True, update_existing=False):
    site = get_current_site()

    t = time.time()
    t_lock = threading.RLock()
    pages = maybe_load_pages_meta(base_path)

    g_users, g_users_by_username = init_users(base_path)

    total_pages = len(pages)

    total_revisions = 0
    total_files = 0
    for meta in pages:
        total_files += len(meta.get('files', []))
        total_revisions += len(meta.get('revisions', []))

    from_files = '%s/files' % base_path
    to_files = str(Path(settings.MEDIA_ROOT) / 'media')

    if os.path.exists(to_files):
        logging.info('正在移除旧文件...')
        shutil.rmtree(to_files, ignore_errors=False)

    def file_worker_thread(pages):
        nonlocal total_cnt
        nonlocal t

        users = {}
        for meta in pages:
            article = articles.get_article(meta['name'])
            if article is None:
                logging.warning('缺少用于文件导入的文章 \'%s\'', meta['name'])
                continue
            files = meta.get('files', [])
            for file in files:
                total_cnt += 1
                if file['author'] in users:
                    file_user = users[file['author']]
                else:
                    file_user = users[file['author']] = get_or_create_user(file['author'], g_users, g_users_by_username)
                from_path = '%s/%s/%d' % (from_files, urls.partial_quote(meta['name']), file['file_id'])
                _, ext = os.path.splitext(file['name'])
                media_name = str(uuid4()) + ext
                if File.objects.filter(name=file['name'], article=article):
                    logging.warning('警告：文件已存在：%s/%s', meta['name'], file['name'])
                    continue
                new_file = File(
                    name=file['name'],
                    media_name=media_name,
                    author=file_user,
                    article=article,
                    mime_type=file['mime'],
                    size=file['size_bytes']
                )
                local_media_dir = os.path.dirname(new_file.local_media_path)
                if not os.path.exists(local_media_dir):
                    os.makedirs(local_media_dir, exist_ok=True)
                to_path = new_file.local_media_path
                if not os.path.exists(from_path):
                    logging.warning('警告：文件未找到：%s/%s', meta['name'], file['name'])
                    continue
                shutil.copyfile(from_path, to_path)
                new_file.save()
                new_file.created_at = datetime.datetime.fromtimestamp(file['stamp'], tz=datetime.timezone.utc)
                new_file.save()
                with t_lock:
                    if time.time() - t > 1:
                        logging.info('已添加：%d/%d' % (total_cnt, total_files))
                        t = time.time()

    def page_worker_thread(pages):
        nonlocal total_cnt
        nonlocal total_cnt_rev
        nonlocal t

        users = {}
        for meta in pages:
            total_cnt += 1
            f = meta['filename']
            pagename = meta['name']
            title = meta['title'] if 'title' in meta else None
            tags = meta['tags'] if 'tags' in meta else []
            updated_at = datetime.datetime.fromtimestamp(meta['revisions'][0]['stamp'], tz=datetime.timezone.utc)
            created_at = datetime.datetime.fromtimestamp(meta['revisions'][-1]['stamp'], tz=datetime.timezone.utc)
            fn_7z = '.'.join(f.split('.')[:-1]) + '.7z'
            fn_7z = '%s/pages/%s' % (base_path, fn_7z)
            if not os.path.exists(fn_7z):
                continue

            # 获取创建者和更新者的用户
            article_author = meta['revisions'][-1]['author']
            if article_author in users:
                user = users[article_author]
            else:
                user = users[article_author] = get_or_create_user(article_author, g_users, g_users_by_username)

            # 创建文章并设置标签
            article = articles.get_article(pagename)
            if article:
                total_cnt_rev += len(meta.get('revisions', []))
                if update_existing:
                    # 文章已存在：不重建内容，仅重新同步标签与评分
                    logging.info('文章已存在，同步标签/评分：%s', pagename)
                    if tags:
                        set_article_tags(article, tags, force_tags)
                    if import_votes:
                        import_article_votes(article, meta, users, g_users, g_users_by_username)
                else:
                    logging.warning('警告：文章已存在：%s', pagename)
                continue
            article = articles.create_article(pagename, user=user)
            article.created_at = created_at
            if title is not None:
                article.title = title
            else:
                article.title = ''
            article.save()
            # 强制将updated_at字段设置为旧值
            Article.objects.filter(pk=article.pk).update(updated_at=updated_at)
            if tags:
                set_article_tags(article, tags, force_tags)

            # 迁移页面评分（votings）
            if import_votes:
                import_article_votes(article, meta, users, g_users, g_users_by_username)

            # 添加所有修订版本
            revisions = list(reversed(meta['revisions']))

            last_source_version = None

            with py7zr.SevenZipFile(fn_7z) as z:
                all_file_names = ['%d.txt' % x['revision'] for x in revisions if 'S' in x['flags'] or 'N' in x['flags']]
                text_revisions = z.read(all_file_names)
                for revision in revisions:
                    total_cnt_rev += 1
                    # 如果是源代码修订版本，则添加修订内容
                    if revision['author'] in users:
                        user = users[revision['author']]
                    else:
                        user = users[revision['author']] = get_or_create_user(revision['author'], g_users, g_users_by_username)
                    log = ArticleLogEntry(
                        rev_number=revision['revision'],
                        article=article,
                        user=user,
                        type=ArticleLogEntry.LogEntryType.Wikidot,
                        comment=revision['commentary']
                    )
                    if 'S' in revision['flags'] or 'N' in revision['flags']:
                        content = text_revisions['%d.txt' % revision['revision']].read().decode('utf-8')

                        for k, v in settings.ARTICLE_IMPORT_REPLACE_CONFIG.items():
                            content = content.replace(k, v)

                        version = ArticleVersion(
                            article=article,
                            source=content,
                            rendered=None,
                        )
                        version.save()
                        last_source_version = version
                        log.meta = {'version_id': version.id}
                        if 'N' in revision['flags']:
                            log.type = ArticleLogEntry.LogEntryType.New
                            log.meta['title'] = article.title
                        else:
                            log.type = ArticleLogEntry.LogEntryType.Source
                    log.save()
                    log.created_at = datetime.datetime.fromtimestamp(revision['stamp'], tz=datetime.timezone.utc)
                    log.save()
                    with t_lock:
                        if time.time() - t > 1:
                            logging.info('已添加：%d/%d（修订版本：%d/%d）' % (total_cnt, total_pages, total_cnt_rev, total_revisions))
                            t = time.time()

            if last_source_version:
                # 待办：待此问题不再挂起时重新启用
                articles.refresh_article_links(last_source_version)

            with t_lock:
                if time.time() - t > 1:
                    logging.info('已添加：%d/%d（修订版本：%d/%d）' % (total_cnt, total_pages, total_cnt_rev, total_revisions))
                    t = time.time()

    total_cnt = 0
    total_cnt_rev = 0
    logging.info('正在添加文章...')
    run_in_threads(page_worker_thread, pages)
    total_cnt = 0
    logging.info('正在添加文件...')
    run_in_threads(file_worker_thread, pages)
    logging.info('正在设置父页面...')
    run_in_threads(set_parents, pages)


def set_article_tags(article, tags, force_tags):
    # 默认走普通的 set_tags；当站点禁止用户创建标签时，库中尚不存在的标签会被静默丢弃。
    # force_tags=True 时先按需新建标签（create=True），再直接写入，绕过该站点级限制。
    if force_tags:
        tag_objs = [
            t for t in (articles.get_tag(name, create=True) for name in tags if articles.is_tag_name_allowed(name))
            if t is not None
        ]
        articles.set_tags_internal(article, tag_objs, log=False)
    else:
        articles.set_tags(article, tags, log=False)


def import_article_votes(article, meta, users, g_users, g_users_by_username):
    # 备份中的 votings 为 [[wikidot用户id, 投票值], ...]。
    # 投票值：布尔 true=+1 / false=-1（点赞点踩制）；数字则按原值写入（星级制）。
    # 备份不含单票时间戳，因此 date 留空（null）——前端会显示为“wikit迁移”。
    # 预置为该文章已有投票的用户，避免覆盖站点上已存在的真实投票、也避免触发唯一约束。
    seen_users = set(Vote.objects.filter(article=article).values_list('user_id', flat=True))
    for voting in meta.get('votings', []):
        if not voting:
            continue
        vote_user_id = voting[0]
        vote_value = voting[1] if len(voting) > 1 else None
        if vote_value is None:
            continue

        if vote_user_id in users:
            vote_user = users[vote_user_id]
        else:
            vote_user = users[vote_user_id] = get_or_create_user(vote_user_id, g_users, g_users_by_username)
        if vote_user is None or vote_user.id in seen_users:
            continue
        seen_users.add(vote_user.id)

        if isinstance(vote_value, bool):
            rate = 1.0 if vote_value else -1.0
        else:
            rate = float(vote_value)

        vote = Vote(article=article, user=vote_user, rate=rate, role=None)
        vote.save()
        # date 字段为 auto_now_add，首次 save 会写入当前时间；再置空并保存以清除，标记为“无时间轴（wikit迁移）”
        vote.date = None
        vote.save()


def set_parents(base_path_or_list):
    pages = maybe_load_pages_meta(base_path_or_list)
    # 收集页面重命名记录
    # （最大日期，名称，最新）
    page_renames = []
    for meta in pages:
        for rev in meta['revisions']:
            if rev['commentary'].startswith('You successfully renamed the page: "') or\
                    rev['commentary'].startswith('Вы переименовали страницу: "'):
                renamed_from = rev['commentary'].split('"')[1]
                renamed_at = rev['stamp']
                page_renames.append((renamed_at, renamed_from, meta['name']))
    page_renames.sort(key=lambda x: x[0])

    for meta in pages:
        pagename = meta['name']
        parent = None
        for rev in meta['revisions']:
            if rev['commentary'].startswith('Parent page set to: "') or\
                    rev['commentary'].startswith('Родительской страницей установлена: "'):
                parent = rev['commentary'].split('"')[1]
                # 尝试查找是否被重命名
                for rename in page_renames:
                    if rename[0] >= rev['stamp'] and rename[1] == parent:
                        logging.info('父页面已被重命名：%s -> %s' % (rename[1], parent))
                        parent = rename[2]
                break

        article = articles.get_article(pagename)
        if article:
            if parent:
                logging.info('父页面：%s -> %s' % (pagename, parent))
            parent_article = articles.get_article(parent)
            if parent_article:
                article.parent = parent_article
                article.save()


####################################################################################################
# 论坛迁移
####################################################################################################


def count_posts(posts):
    p = len(posts)
    for post in posts:
        if 'children' in post:
            p += count_posts(post['children'])
    return p


def post_filenames(posts):
    filenames = []
    for post in posts:
        filenames.append('%d/latest.html' % post['id'])
        if 'revisions' in post:
            for rev in post['revisions']:
                filenames.append('%d/%d.html' % (post['id'], rev['id']))
        if 'children' in post:
            filenames += post_filenames(post['children'])
    return filenames


def post_highest_timestamp(posts):
    ts = 0
    for post in posts:
        ts = max(ts, post['stamp'])
        if 'children' in post:
            ts = max(post_highest_timestamp(post['children']) or ts, ts)
    return ts


def run_forum(base_path):
    g_users, g_users_by_username = init_users(base_path)

    logging.info('Loading pages...')
    pages = maybe_load_pages_meta(base_path)
    comment_thread_map = {}
    for page in pages:
        if 'forum_thread' in page:
            comment_thread_map[page['forum_thread']] = page['name']

    logging.info('Loading existing pages...')
    all_articles = Article.objects.all()
    renames = ArticleLogEntry.objects.filter(type=ArticleLogEntry.LogEntryType.Name).order_by('-created_at')
    page_name_map = {}
    for article in all_articles:
        page_renames = [x for x in renames if x.article_id == article.id]
        name = article.full_name
        if page_renames:
            name = page_renames[-1].meta['prev_name']
        page_name_map[name] = article

    logging.info('Loading categories...')
    categories = {}
    for category_file in os.listdir('%s/meta/forum/category' % base_path):
        with codecs.open('%s/meta/forum/category/%s' % (base_path, category_file), encoding='utf-8') as f:
            category = json.load(f)
            category['threads'] = []
            category['isForComments'] = False
            category['local'] = None
            categories[category['id']] = category

    t = time.time()
    t_lock = threading.RLock()

    total_threads = 0
    total_posts = 0
    done_threads = 0
    done_posts = 0
    threads = []

    for category_id, category in categories.items():
        logging.info('Loading threads for "%s"...', category['title'])
        category_dir = '%s/meta/forum/%d' % (base_path, category['id'])
        # 空分类在备份中可能没有帖子目录，跳过即可（分类本身仍会被创建，只是没有线程）
        if not os.path.isdir(category_dir):
            logging.warning('警告：分类 “%s”（id=%d）没有帖子目录，跳过', category['title'], category['id'])
            continue
        for thread_file in os.listdir(category_dir):
            with codecs.open('%s/%s' % (category_dir, thread_file), encoding='utf-8') as f:
                thread = json.load(f)
                category['threads'].append(thread)
                thread['categoryId'] = category_id
                if thread['id'] in comment_thread_map:
                    category['isForComments'] = True
                threads.append(thread)
                total_threads += 1
                total_posts += count_posts(thread['posts'])

    # clear the forum. this is required because we need to set IDs
    with connection.cursor() as cursor:
        cursor.execute('TRUNCATE TABLE web_forumsection CASCADE')

    # generate section for imported content
    section = ForumSection(name='Imported', description='Imported from archive')
    section.save()

    # generate categories
    for _, category in categories.items():
        c = ForumCategory(id=category['id'], name=category['title'], description=category['description'], section=section, is_for_comments=category['isForComments'])
        c.save()
        category['local'] = c

    orphan_category = None
    orphan_category_lock = threading.RLock()

    def convert_threads(work):
        nonlocal done_threads
        nonlocal done_posts
        nonlocal t
        nonlocal orphan_category

        for thread in work:
            user = get_or_create_user(thread['startedUser'], g_users, g_users_by_username)

            category = categories[thread['categoryId']]['local']
            article = None

            is_orphan = False

            if thread['id'] in comment_thread_map:
                category = None
                article = page_name_map.get(comment_thread_map[thread['id']])
                if article is None:
                    logging.warning('Warn: comment thread %d for nonexistent article %s', thread['id'], comment_thread_map[thread['id']])
                    is_orphan = True

            if category and category.is_for_comments:
                logging.warning('Warn: attempt to put thread %d in a comment category, but it\'s not a comment thread', thread['id'])
                is_orphan = True

            if is_orphan:
                with orphan_category_lock:
                    if not orphan_category:
                        orphan_category = ForumCategory(name='Orphan threads', description='Comment threads that could not be reliably assigned to an article', section=section)
                        orphan_category.save()
                category = orphan_category
                article = None

            th = ForumThread(id=thread['id'], category=category, article=article, name=thread['title'], description=thread['description'], author=user, is_pinned=thread['sticky'])
            th.save()
            created_at = datetime.datetime.fromtimestamp(thread['started'], tz=datetime.timezone.utc)
            # updated_at is the highest post creation timestamp
            updated_at = datetime.datetime.fromtimestamp(post_highest_timestamp(thread['posts']) or thread['started'], tz=datetime.timezone.utc)
            ForumThread.objects.filter(id=th.id).update(created_at=created_at, updated_at=updated_at)

            post_data_7z = '%s/forum/%d/%d.7z' % (base_path, thread['categoryId'], thread['id'])

            if thread['posts']:
                with py7zr.SevenZipFile(post_data_7z) as z:
                    all_file_names = post_filenames(thread['posts'])
                    post_contents = z.read(all_file_names)
            else:
                post_contents = dict()

            def add_post(post):
                nonlocal done_posts
                nonlocal t

                # create post
                user = get_or_create_user(post['poster'], g_users, g_users_by_username)
                created_at = datetime.datetime.fromtimestamp(post['stamp'], tz=datetime.timezone.utc)
                updated_at = datetime.datetime.fromtimestamp(post.get('lastEdit') or post['stamp'], tz=datetime.timezone.utc)
                p = ForumPost(id=post['id'], thread=th, name=post.get('title') or '', author=user, reply_to=post.get('replyTo', None))
                p.save()
                ForumPost.objects.filter(id=p.id).update(created_at=created_at, updated_at=updated_at)

                threadvars.put('threadid', th.id)

                if post.get('revisions', []):
                    for rev in post['revisions']:
                        rev_user = get_or_create_user(rev['author'], g_users, g_users_by_username)
                        rev_created_at = datetime.datetime.fromtimestamp(rev['stamp'], tz=datetime.timezone.utc)
                        source_in_file = '%d/%d.html' % (post['id'], rev['id'])
                        source = html_to_source(post_contents[source_in_file].read().decode('utf-8'))

                        for k, v in settings.ARTICLE_IMPORT_REPLACE_CONFIG.items():
                            source = source.replace(k, v)

                        r = ForumPostVersion(post=p, author=rev_user, source=source)
                        r.save()
                        ForumPostVersion.objects.filter(id=r.id).update(created_at=rev_created_at)
                else:
                    source_in_file = '%d/latest.html' % post['id']
                    source = html_to_source(post_contents[source_in_file].read().decode('utf-8'))

                    for k, v in settings.ARTICLE_IMPORT_REPLACE_CONFIG.items():
                        source = source.replace(k, v)

                    r = ForumPostVersion(post=p, author=user, source=source)
                    r.save()
                    ForumPostVersion.objects.filter(id=r.id).update(created_at=created_at)

                with t_lock:
                    done_posts += 1
                    if time.time() - t > 1:
                        logging.info(
                            'Added: %d/%d (posts: %d/%d)' % (done_threads, total_threads, done_posts, total_posts))
                        t = time.time()

                if 'children' in post:
                    for post in post['children']:
                        post['replyTo'] = p
                        add_post(post)

            for post in thread['posts']:
                add_post(post)

            with t_lock:
                done_threads += 1
                if time.time() - t > 1:
                    logging.info('Added: %d/%d (posts: %d/%d)' % (done_threads, total_threads, done_posts, total_posts))
                    t = time.time()

    run_in_threads(convert_threads, threads)
    logging.info('Done; Added: %d/%d (posts: %d/%d)' % (done_threads, total_threads, done_posts, total_posts))


####################################################################################################
# The entire following section is dedicated to conversion of HTML to Wikidot markup
####################################################################################################


def elements_to_source(iterable):
    output = ''
    for el in iterable:
        output += element_to_source(el)
    return output


def attr_value_to_source(value):
    value = value.replace('\\', '\\\\')
    value = value.replace('"', '\\"')
    return value


def attrs_to_source(el, not_attrs=[]):
    output = ''
    for attr, value in el.attrs.items():
        if attr in not_attrs:
            continue
        if attr == 'class':
            value = ' '.join(value)
        value = attr_value_to_source(value)
        output += ' %s="%s"' % (attr, value)
    return output


def element_has_allowed_class(el, cls):
    if not el.attrs.get('class'):
        return True
    for c in cls:
        if c in el['class']:
            return c
    return False


def element_to_source(el):
    if isinstance(el, str):
        text = el.text
        # newlines have special meaning; newlines from here should be dropped
        text = text.replace('\n', '')
        return text
    elif el.name == 'p':
        return elements_to_source(el) + '\n\n'
    elif el.name == 'em':
        return '//' + elements_to_source(el) + '//'
    elif el.name == 'strong' or el.name == 'b':
        return '**' + elements_to_source(el) + '**'
    elif el.name == 'u':
        return '__' + elements_to_source(el) + '__'
    elif el.name == 'strike' or el.name == 's':
        return '--' + elements_to_source(el) + '--'
    elif el.name == 'sup':
        if 'footnoteref' in el.attrs.get('class', []):
            # this is not a sup, this is a footnote. find corresponding footnote in the footnote block (it must exist somewhere at root level)
            number = el.text.strip()
            root = el
            while root.parent:
                root = root.parent
            footnoteblock = root.find('div', class_='footnotes-footer')
            footnote_nodes = footnoteblock.find_all('div', class_='footnote-footer')
            footnotes = dict()
            for node in footnote_nodes:
                if '_p_text' in node.attrs and '_p_number' in node.attrs:
                    footnotes[node.attrs['_p_number']] = node.attrs['_p_text']
                else:
                    node_number = node.a.text.strip()
                    node.a.decompose()
                    children = [x for x in node]
                    children[0].replace_with(children[0][2:])
                    footnotes[node_number] = elements_to_source(node)
                    node.attrs['_p_number'] = node_number
                    node.attrs['_p_text'] = footnotes[node_number]
            gen_text = '[[footnote]]%s[[/footnote]]' % footnotes[number]
            return gen_text
        return '^^' + elements_to_source(el) + '^^'
    elif el.name == 'sub':
        return ',,' + elements_to_source(el) + ',,'
    elif el.name == 'br':
        return '\n'
    elif el.name == 'iframe':
        src = el["src"]
        return '[[iframe %s%s]]' % (src, attrs_to_source(el, ['src']))
    elif el.name == 'span':
        if 'class' in el.attrs and 'printuser' in el['class']:
            with_avatar = 'avatarhover' in el['class']
            # <a href="http://wikidot.com/user:info/<user_name>">
            user_name = el.find('a')['href'].split('/')[-1]
            return '[[%suser %s]]' % ('*' if with_avatar else '', user_name)
        if 'class' in el.attrs and 'math-inline' in el['class']:
            content = el.text.strip('$').strip()
            return '[[$ %s $]]' % content
        if 'class' in el.attrs and 'equation-number' in el['class']:
            return ''
        contents = elements_to_source(el)
        attrs = attrs_to_source(el)
        return '[[span%s]]%s[[/span]]' % (attrs, contents)
    elif el.name == 'blockquote':
        contents = '> ' + '\n> '.join(elements_to_source(el).strip().split('\n')) + '\n'
        return contents
    elif el.name == 'div':
        if element_has_allowed_class(el, ['rimg', 'limg', 'cimg', 'blockquote', 'сimg', 'scpnet-progress-bar', 'scpnet-progress-bar__tick', 'block-error', 'collapsible-block-unfolded-link']):
            return '[[div%s]]\n%s[[/div]]\n' % (attrs_to_source(el), elements_to_source(el))
        elif 'collapsible-block' in el['class']:
            # detect collapsibles
            # hidelocation is not preserved
            show = el.find('div', class_='collapsible-block-folded').find('a', class_='collapsible-block-link').text.replace('\n', ' ').strip()
            hide = el.find('div', class_='collapsible-block-unfolded').find('div', class_='collapsible-block-unfolded-link').find('a', class_='collapsible-block-link').text.replace('\n', ' ').strip()
            contents = elements_to_source(el.find('div', class_='collapsible-block-content'))
            src = '[[collapsible show="%s" hide="%s"]]\n' % (attr_value_to_source(show), attr_value_to_source(hide))
            src += contents
            src += '[[/collapsible]]\n'
            return src
        elif 'yui-navset' in el['class']:
            # detect tabview
            nav = el.find('ul', class_='yui-nav').find_all('li')
            tabnames = []
            for li in nav:
                tabnames.append(li.text.replace('\n', ' ').strip())
            src = '[[tabview]]\n'
            tabs = el.find('div', class_='yui-content')
            num = 0
            for tab in tabs:
                if tab.name != 'div':
                    continue
                src += '[[tab title="%s"]]\n' % (attr_value_to_source(tabnames[num]))
                src += elements_to_source(tab)
                src += '[[/tab]]\n'
                num += 1
            src += '[[/tabview]]\n'
            return src
        elif 'code' in el['class']:
            # detect [[code]]
            if el.pre is not None and el.pre.code is not None:
                code = el.pre.code.text
                return '[[code]]\n%s\n[[/code]]\n' % code
            sub_el = el.find('div', class_='hl-main')
            if sub_el and sub_el.pre:
                code = sub_el.pre.text
                return '[[code]]\n%s\n[[/code]]\n' % code
            return '[[div%s]]\n%s[[/div]]\n' % (attrs_to_source(el), elements_to_source(el))
        elif 'footnotes-footer' in el['class']:
            title = el.find('div', class_='title').text.replace('\n', ' ').strip()
            return '[[footnoteblock title="%s"]]\n' % attr_value_to_source(title)
        elif 'bibitems' in el['class']:
            title = el.find('div', class_='title').text.replace('\n', ' ').strip()
            src = '[[bibliography title="%s"]]\n' % attr_value_to_source(title)
            iv = 0
            for item in el.find_all('div', class_='bibitem'):
                iv += 1
                children = [x for x in item.children]
                children[0].replace_with(children[0][2:])
                src += ': cite%d : %s\n' % (iv, elements_to_source(item).strip())
            src += '[[/bibliography]]\n'
            return src
        elif 'image-container' in el['class']:
            # this is f>image or f<image
            prefix = ''
            if 'floatleft' in el['class']:
                prefix = 'f<'
            if 'floatright' in el['class']:
                prefix = 'f>'
            if 'alignleft' in el['class']:
                prefix = '<'
            if 'alignright' in el['class']:
                prefix = '>'
            if 'aligncenter' in el['class']:
                prefix = '='
            src = el.img["src"]
            return '[[%simage %s%s]]' % (prefix, src, attrs_to_source(el.img, ['src', 'alt']))
        elif 'content-separator' in el['class']:
            return '====\n'
        elif 'math-equation' in el['class']:
            content = el.text.strip()
            return '[[math]]\n%s\n[[/math]]\n' % content
        elif 'wiki-note' in el['class']:
            return '[[note]]\n%s[[/note]]\n' % elements_to_source(el)
        else:
            logging.warning('Warn: unknown div class %s in thread %s, falling back to [[div]]', el.get('class'), threadvars.get('threadid'))
            return '[[div%s]]\n%s[[/div]]\n' % (attrs_to_source(el), elements_to_source(el))
    elif el.name == 'a':
        # just generate [[a]]
        return '[[a%s]]%s[[/a]]' % (attrs_to_source(el), elements_to_source(el))
    elif el.name == 'img':
        url = el["src"]
        return '[[image %s%s]]' % (url, attrs_to_source(el, ['src', 'alt']))
    elif el.name == 'hr':
        return '----\n'
    elif el.name == 'ul' or el.name == 'li' or el.name == 'ol':
        return '[[%s]]\n%s[[/%s]]\n' % (el.name, elements_to_source(el), el.name)
    elif el.name in ['h1', 'h2', 'h3', 'h4', 'h5', 'h6', 'h7']:
        content = el.span.text.replace('\n', ' ').strip()
        return ('+' * int(el.name[1:])) + ' ' + content + '\n'
    elif el.name == 'tt':
        return '{{' + elements_to_source(el) + '}}'
    elif el.name == 'table':
        return '[[table%s]]\n%s[[/table]]\n' % (attrs_to_source(el), elements_to_source(el))
    elif el.name == 'tbody':
        return elements_to_source(el)
    elif el.name == 'tr':
        return '[[row%s]]\n%s[[/row]]\n' % (attrs_to_source(el), elements_to_source(el))
    elif el.name == 'td':
        return '[[cell%s]]\n%s[[/cell]]\n' % (attrs_to_source(el), elements_to_source(el))
    elif el.name == 'th':
        return '[[hcell%s]]\n%s[[/hcell]]\n' % (attrs_to_source(el), elements_to_source(el))
    elif el.name == 'script':
        return ''
    elif el.name == 'dl':
        src = ''
        for node in el:
            if node.name == 'dt':
                dd = node.find_next('dd')
                src += ': %s : %s\n' % (node.text.replace('\n', ' '), dd.text.replace('\n', ' '))
        return src
    else:
        print('thread = %d' % threadvars.get('threadid'))
        raise ValueError(repr(el))
    return ''


def html_to_source(html):
    soup = BeautifulSoup(html, features='html.parser')
    return elements_to_source(soup)
