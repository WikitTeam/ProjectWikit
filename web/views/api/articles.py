import logging
from django.conf import settings
from django.http import HttpRequest, HttpResponse

from modules.sitechanges import log_entry_default_comment
from shared_data import shared_articles
from . import APIView, APIError, takes_json

from web.controllers import articles, notifications

from renderer.utils import render_user_to_json
from renderer import single_pass_render
from renderer.parser import RenderContext

import json

from web.controllers.search import update_search_index
from web.models.articles import Category, ExternalLink, Article

from modules import rate, ModuleError


class AllArticlesView(APIView):
    def get(self, request: HttpRequest):
        result = []
        hidden_categories = articles.get_hidden_categories_for(request.user)
        for category, entries in shared_articles.get_all_articles().items():
            if category in hidden_categories:
                continue
            result.extend(entries)
        return self.render_json(200, result)


class ArticleView(APIView):
    def _validate_article_data(self, data, allow_partial=False):
        if not data:
            raise APIError('无效请求', 400)
        if 'pageId' not in data or not data['pageId'] or not articles.is_full_name_allowed(data['pageId']):
            raise APIError('无效的页面ID', 400)
        if ('source' not in data or not (data['source'] or '').strip()) and not (
                allow_partial and 'source' not in data):
            raise APIError('缺少页面源代码', 400)
        if ('title' not in data or data['title'] is None) and not (allow_partial and 'title' not in data):
            raise APIError('缺少页面标题', 400)
        if 'source' in data and len(data['source']) > settings.ARTICLE_SOURCE_LIMIT:
            raise APIError('超过页面大小限制')

class CreateView(ArticleView):
    @takes_json
    def post(self, request: HttpRequest, *args, **kwargs) -> HttpResponse:
        data = self.json_input

        self._validate_article_data(data)

        category, name = articles.get_name(data['pageId'])
        if not request.user.has_perm('roles.create_articles', Category.get_or_default_category(category)):
            raise APIError('权限不足', 403)

        article = articles.get_article(data['pageId'])
        if article is not None:
            raise APIError('此ID的页面已存在', 409)

        # 创建页面
        article = articles.create_article(articles.normalize_article_name(data['pageId']), request.user)
        article.title = data['title']
        article.save()
        version = articles.create_article_version(article, data['source'], request.user)
        articles.refresh_article_links(version)
        
        if data.get('parent') is not None:
            articles.set_parent(article, articles.normalize_article_name(data['parent']), request.user)

        notifications.subscribe_to_notifications(subscriber=request.user, article=article)

        return self.render_json(201, {'status': 'ok'})


class FetchOrUpdateView(ArticleView):
    def get(self, request: HttpRequest, full_name: str) -> HttpResponse:
        category = articles.get_article_category(full_name)
        if not request.user.has_perm('roles.view_articles', category):
            raise APIError('权限不足', 403)
        # 查找页面
        article = articles.get_article(full_name)
        if article is None:
            raise APIError('页面未找到', 404)

        return self.render_article(article)

    def render_article(self, article: Article):
        source = articles.get_latest_source(article)
        authors = list(article.authors.all())
        authors = [render_user_to_json(author) for author in authors] if authors else [render_user_to_json(None)]

        return self.render_json(200, {
            'uid': article.id,
            'pageId': articles.get_full_name(article),
            'title': article.title,
            'source': source,
            'tags': articles.get_tags(article),
            'author': authors[0],
            'authors': authors,
            'parent': articles.get_parent(article),
            'locked': article.locked
        })

    @takes_json
    def put(self, request: HttpRequest, full_name: str) -> HttpResponse:
        # 查找页面
        article = articles.get_article(full_name)
        if article is None:
            category = articles.get_article_category(full_name)
            if not request.user.has_perm('roles.view_articles', category):
                raise APIError('权限不足', 403)
            raise APIError('页面未找到', 404)

        can_edit_articles = request.user.has_perm('roles.edit_articles', article)

        data = self.json_input
        self._validate_article_data(data, allow_partial=True)

        # 检查是否重命名
        if data['pageId'] != full_name:
            new_name = articles.normalize_article_name(data['pageId'])
            new_category = articles.get_article_category(new_name)
            if not request.user.has_perm('roles.move_articles', article) or \
               not request.user.has_perm('roles.move_articles', new_category) if new_category else False:
                raise APIError('权限不足', 403)
            article2 = articles.get_article(new_name)
            if article2 is not None and article2.id != article.id and not data.get('forcePageId'):
                raise APIError('此ID的页面已存在', 409)
            new_name = articles.deduplicate_name(new_name, article)
            articles.update_full_name(article, new_name, request.user)

        # 检查是否修改标题
        if 'title' in data and data['title'] != article.title:
            if not can_edit_articles:
                raise APIError('权限不足', 403)
            articles.update_title(article, data['title'], request.user)

        # 检查是否修改源代码
        if 'source' in data and data['source'] != articles.get_latest_source(article):
            if not can_edit_articles:
                raise APIError('权限不足', 403)
            version = articles.create_article_version(article, data['source'], request.user, data.get('comment', ''))
            articles.refresh_article_links(version)

        # 检查是否修改标签
        if 'tags' in data:
            if not request.user.has_perm('roles.tag_articles', article):
                raise APIError('权限不足', 403)
            articles.set_tags(article, data['tags'], request.user)

        # 检查是否修改父页面
        if 'parent' in data:
            if not can_edit_articles:
                raise APIError('权限不足', 403)
            articles.set_parent(article, data['parent'], request.user)

        # 检查是否锁定页面
        if 'locked' in data:
            if data['locked'] != article.locked:
                if request.user.has_perm('roles.lock_articles', article):
                    articles.set_lock(article, data['locked'], request.user)
                else:
                    raise APIError('权限不足', 403)
                
        # 检查是否修改作者
        if 'authorsIds' in data:
            if isinstance(data['authorsIds'], list) and all(map(lambda a: isinstance(a, str), data)):
                if can_edit_articles and request.user.has_perm('roles.manage_article_authors', article):
                    articles.set_authors(article, data['authorsIds'], request.user)
                else:
                    raise APIError('权限不足', 403)


        article.refresh_from_db()
        update_search_index(article)
        return self.render_article(article)

    def delete(self, request: HttpRequest, full_name: str) -> HttpResponse:
        # 查找页面
        article = articles.get_article(full_name)
        if article is None:
            category = articles.get_article_category(full_name)
            if not request.user.has_perm('roles.view_articles', category):
                raise APIError('权限不足', 403)
            raise APIError('页面未找到', 404)

        if not request.user.has_perm('roles.delete_articles', article):
            raise APIError('权限不足', 403)

        articles.OnDeleteArticle(request.user, article).emit()
        articles.delete_article(article)

        return self.render_json(200, {'status': 'ok'})


class FetchOrRevertLogView(APIView):
    def get(self, request: HttpRequest, full_name: str) -> HttpResponse:
        category = articles.get_article_category(full_name)
        if not request.user.has_perm('roles.view_articles', category):
            raise APIError('权限不足', 403)
        try:
            c_from = int(request.GET.get('from', '0'))
            c_to = int(request.GET.get('to', '25'))
            get_all = bool(request.GET.get('all'))
        except ValueError:
            raise APIError('无效的列表范围参数', 400)

        log_entries, total_count = articles.get_log_entries_paged(full_name, c_from, c_to, get_all)

        output = []
        for entry in log_entries:
            output.append({
                'revNumber': entry.rev_number,
                'user': render_user_to_json(entry.user),
                'comment': entry.comment,
                'defaultComment': log_entry_default_comment(entry),
                'createdAt': entry.created_at.isoformat(),
                'type': entry.type,
                'meta': entry.meta
            })

        return self.render_json(200, {'count': total_count, 'entries': output})

    @takes_json
    def put(self, request: HttpRequest, full_name: str) -> HttpResponse:
        article = articles.get_article(full_name)
        if article is None:
            category = articles.get_article_category(full_name)
            if not request.user.has_perm('roles.view_articles', category):
                raise APIError('权限不足', 403)
            raise APIError('页面未找到', 404)

        if not request.user.has_perm('roles.edit_articles', article):
            raise APIError('权限不足', 403)

        data = self.json_input

        if not ("revNumber" in data and isinstance(data["revNumber"], int)):
            raise APIError('无效的版本号', 400)

        articles.revert_article_version(article, data["revNumber"], request.user)
        version = articles.get_latest_version(article)
        articles.refresh_article_links(version)

        article.refresh_from_db()
        update_search_index(article)
        return self.render_json(200, {"pageId": article.full_name})


class FetchVersionView(APIView):
    def get(self, request: HttpRequest, full_name: str) -> HttpResponse:
        category = articles.get_article_category(full_name)
        if not request.user.has_perm('roles.view_articles', category):
            raise APIError('权限不足', 403)
        
        article = articles.get_article(full_name)
        source = articles.get_source_at_rev_num(article, int(request.GET.get('revNum')))

        if source:
            context = RenderContext(article, article,
                                    json.loads(request.GET.get('pathParams', "{}")), self.request.user)
            rendered = single_pass_render(source, context)

            return self.render_json(200, {'source': source, "rendered": rendered})
        raise APIError('指定版本不存在', 404)


class FetchExternalLinks(APIView):
    def get(self, request: HttpRequest, full_name: str) -> HttpResponse:
        category = articles.get_article_category(full_name)
        if not request.user.has_perm('roles.view_articles', category):
            raise APIError('权限不足', 403)
        
        article = articles.get_article(full_name)
        if not article:
            raise APIError('页面未找到', 404)

        links_children = [{'id': x.full_name, 'title': x.title, 'exists': True} for x in
                          Article.objects.filter(parent=article)]

        links_all = ExternalLink.objects.filter(link_to=full_name)

        links_include = []
        links_links = []

        articles_dict = articles.fetch_articles_by_names([link.link_from.lower() for link in links_all])

        for link in links_all:
            article = articles_dict.get(link.link_from.lower())
            article_record = {'id': article.full_name, 'title': article.title, 'exists': True} if article else {
                'id': link.link_from.lower(), 'title': link.link_from.lower(), 'exists': False}
            if link.link_type == ExternalLink.Type.Include:
                links_include.append(article_record)
            elif link.link_type == ExternalLink.Type.Link:
                links_links.append(article_record)

        return self.render_json(200, {'children': links_children, 'includes': links_include, 'links': links_links})

class FetchOrUpdateVotesView(APIView):
    def get(self, request: HttpRequest, full_name: str) -> HttpResponse:
        category = articles.get_article_category(full_name)
        if not request.user.has_perm('roles.view_articles', category):
            raise APIError('权限不足', 403)
        
        article = articles.get_article(full_name)
        if not article:
            raise APIError('页面未找到', 404)

        try:
            return self.render_json(200, rate.api_get_votes(RenderContext(article=article, source_article=article, user=request.user), {}))
        except ModuleError as e:
            raise APIError(e.message, 500)

    def delete(self, request: HttpRequest, full_name: str) -> HttpResponse:
        article = articles.get_article(full_name)
        if article is None:
            category = articles.get_article_category(full_name)
            if not request.user.has_perm('roles.view_articles', category):
                raise APIError('权限不足', 403)
            raise APIError('页面未找到', 404)

        if not request.user.has_perm('roles.reset_article_votes', article):
            raise APIError('权限不足', 403)

        articles.delete_article_votes(article, user=request.user)

        return self.get(request, full_name)