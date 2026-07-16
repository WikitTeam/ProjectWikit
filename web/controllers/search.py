import json
import base64
import re

from datetime import datetime, timezone
from typing import Literal
from uuid import uuid4

from django.db import models
from django.db.models import Q, Count
from django.contrib.postgres.search import SearchQuery, SearchRank, SearchVector, SearchHeadline

from renderer import RenderContext, single_pass_render_text

from web.controllers import articles
from web.models import ArticleSearchIndex, Article
from web.models.articles import Tag
from web.models.forum import ForumPost
from web.models.users import User, canonicalize_username


def search_articles(text, user: User=None, is_source=False, cursor=None, limit=25, explain=False):
    hidden_categories = articles.get_hidden_categories_for(user)
    if is_source:
        cursor_parameters = decode_cursor(cursor, 'source', ['id__lt', 'id'])

        results = ArticleSearchIndex.objects.filter(
            content_source__icontains=text,
        ).exclude(
            article__category__in=hidden_categories
        ).order_by('-id')
        if cursor_parameters:
            results = results.filter(cursor_parameters)
        results = results[:limit]
        if explain:
            print(results.explain(analyze=True))

        output = []

        results = list(results)
        for article in results:
            output.append({
                'article': article,
                'words': [text]
            })

        if results:
            next_cursor = encode_cursor('source', [
                dict(id__lt=results[-1].id)
            ])
        else:
            next_cursor = encode_cursor('source', [
                dict(id=-1)
            ])

        return output, next_cursor
    else:
        cursor_parameters = decode_cursor(cursor, 'plain', ['rank_str__lt', 'rank_str', 'id__lt', 'id'])
        search_query_en = SearchQuery(text, config='english', search_type="websearch")
        search_query_ru = SearchQuery(text, config='russian', search_type="websearch")
        search_query = search_query_en | search_query_ru
        mark_name = str(uuid4())
        mark_open = f'<{mark_name}>'
        mark_close = f'</{mark_name}>'
        results = ArticleSearchIndex.objects.annotate(
            rank=SearchRank(
                models.F('vector_plaintext'),
                search_query,
                cover_density=True,
                normalization=32
            ),
            rank_str=models.Func(
                'rank',
                template="TO_CHAR(%(expressions)s, '000.999999')",
                output_field=models.CharField()
            ),
            headline_en=SearchHeadline(
                models.F('content_plaintext'),
                search_query,
                config='english',
                start_sel=mark_open,
                stop_sel=mark_close,
                max_words=35,
                min_words=20,
                max_fragments=3,
                fragment_delimiter=' ... '
            ),
            headline_ru=SearchHeadline(
                models.F('content_plaintext'),
                search_query,
                config='russian',
                start_sel=mark_open,
                stop_sel=mark_close,
                max_words=35,
                min_words=20,
                max_fragments=3,
                fragment_delimiter=' ... '
            )
        ).filter(
            models.Q(vector_plaintext=search_query)
        ).exclude(
            article__category__in=hidden_categories
        ).order_by('-rank_str', '-id')
        if cursor_parameters:
            results = results.filter(cursor_parameters)
        results = results[:limit]

        if explain:
            print(results.explain(analyze=True))

        output = []

        results = list(results)
        for article in results:
            highlighted_snippet = article.headline_en + article.headline_ru

            import re
            matched = re.findall(r'<%s>(.*?)</%s>' % (mark_name, mark_name), highlighted_snippet)
            article.matched_words = list(set(matched))  # Dedupe

            output.append({
                'article': article,
                'words': article.matched_words
            })

        if results:
            next_cursor = encode_cursor('plain', [
                dict(rank_str__lt=results[-1].rank_str),
                dict(rank_str=results[-1].rank_str, id__lt=results[-1].id)
            ])
        else:
            next_cursor = encode_cursor('plain', [
                dict(id=-1)
            ])

        return output, next_cursor


def _parse_date(value, end_of_day=False):
    if not value:
        return None
    try:
        d = datetime.strptime(value.strip(), '%Y-%m-%d')
    except (ValueError, AttributeError):
        return None
    if end_of_day:
        d = d.replace(hour=23, minute=59, second=59)
    return d.replace(tzinfo=timezone.utc)


def _find_tags(name):
    if ':' in name:
        category, tag_name = articles.get_name(name)
        return list(Tag.objects.filter(category__slug=category, name=tag_name))
    return list(Tag.objects.filter(name=name))


def _make_excerpt(plaintext, words, length=240):
    body = plaintext.split('\n\n', 1)[-1] if plaintext else ''
    body = re.sub(r'\s+', ' ', body).strip()
    start = 0
    if words:
        positions = [p for p in (body.lower().find(w.lower()) for w in words if w) if p >= 0]
        if positions:
            start = max(0, min(positions) - 60)
    snippet = body[start:start + length]
    if start > 0:
        snippet = '…' + snippet
    if start + length < len(body):
        snippet = snippet + '…'
    return snippet


def _format_rating(tup):
    rating, votes, _popularity, mode = tup
    if mode == 'updown':
        return '%+d' % rating
    if mode == 'stars':
        return '—' if not votes else '%.1f' % rating
    if mode == 'disabled':
        return None
    return '%d' % rating


def search_articles_filtered(text, *, author=None, tags=None, date_from=None, date_to=None,
                             user: User = None, offset=0, limit=20):
    text = (text or '').strip()
    hidden_categories = articles.get_hidden_categories_for(user)

    art_qs = Article.objects.exclude(category__in=hidden_categories)

    if author:
        canon = canonicalize_username(author)
        author_user = User.objects.filter(
            Q(username=canon) | Q(wikidot_username=canon) | Q(display_name__iexact=author)
        ).first()
        if author_user is None:
            return [], False, 0
        art_qs = art_qs.filter(authors=author_user)

    include_tag_ids = set()
    if tags:
        for name in tags:
            if name.startswith('-'):
                excluded = [t for t in _find_tags(name[1:]) if t is not None]
                if excluded:
                    art_qs = art_qs.exclude(tags__in=excluded)
            else:
                found = [t for t in _find_tags(name) if t is not None]
                if not found:
                    return [], False, 0
                art_qs = art_qs.filter(tags__in=found)
                include_tag_ids.update(t.id for t in found)

    df = _parse_date(date_from)
    dt = _parse_date(date_to, end_of_day=True)
    if df:
        art_qs = art_qs.filter(created_at__gte=df)
    if dt:
        art_qs = art_qs.filter(created_at__lte=dt)

    allowed_ids = art_qs.values('id')

    qs = ArticleSearchIndex.objects.filter(article__isnull=False, article_id__in=allowed_ids)

    words = [w for w in re.split(r'\s+', text) if w] if text else []
    for w in words:
        qs = qs.filter(content_plaintext__icontains=w)
    qs = qs.order_by('-article__created_at')

    qs = qs.select_related('article').prefetch_related('article__authors', 'article__tags__category')

    total = qs.count()
    page = list(qs[offset:offset + limit + 1])
    has_more = len(page) > limit
    page = page[:limit]

    page_ids = [si.article_id for si in page]
    ratings = articles.get_all_ratings(Article.objects.filter(id__in=page_ids))
    comment_counts = {
        row['thread__article_id']: row['c']
        for row in ForumPost.objects.filter(thread__article_id__in=page_ids)
        .values('thread__article_id').annotate(c=Count('id'))
    }

    items = []
    for si in page:
        article = si.article
        author_obj = article.authors.all()[0] if article.authors.all() else None
        rating = ratings.get(article.id)
        shown_tags = [t.full_name for t in article.tags.all() if t.id in include_tag_ids] if include_tag_ids else []
        items.append({
            'title': article.title or article.display_name,
            'url': '/%s' % article.full_name,
            'excerpt': _make_excerpt(si.content_plaintext, words),
            'words': words,
            'author': ({'name': str(author_obj), 'url': '/-/users/%s' % author_obj.url_name} if author_obj else None),
            'tags': shown_tags,
            'comments': comment_counts.get(article.id, 0),
            'createdAt': article.created_at.isoformat() if article.created_at else None,
            'updatedAt': article.updated_at.isoformat() if article.updated_at else None,
            'rating': _format_rating(rating) if rating else None,
        })

    return items, has_more, total


def decode_cursor(cursor: str | None, expected_type: Literal['source', 'plain'], whitelist=None) -> models.Q | None:
    if cursor is None:
        return None
    try:
        data = base64.b64decode(cursor).decode('utf-8')
        data = json.loads(data)
        if (expected_type == 'source' and data.get('t') == 'source') or \
                (expected_type == 'plain' and data.get('t') == 'plain'):
            options = None
            for option in data.get('o'):
                for k in option:
                    if whitelist is None or k not in whitelist:
                        del option[k]
                if options is None:
                    options = models.Q(**option)
                else:
                    options |= models.Q(**option)
            return options
        return None
    except:
        return None


def encode_cursor(cursor_type: Literal['source', 'plain'], parameters: list[dict[str, any]]) -> str:
    data = {'t': cursor_type, 'o': parameters}
    data = json.dumps(data)
    data = base64.b64encode(data.encode('utf-8')).decode('ascii')
    return data


def update_search_index(article: Article):
    version = articles.get_latest_version(article)

    if version is None:
        return

    search_obj, created = ArticleSearchIndex.objects.get_or_create(article=article)
    context = RenderContext(article=version.article, source_article=article)
    search_obj.content_source = article.title + '\n\n' + version.source
    try:
        search_obj.content_plaintext = article.title + '\n\n' + single_pass_render_text(version.source, context, 'system')
    except:
        search_obj.content_plaintext = search_obj.content_source
    search_obj.save()

    ArticleSearchIndex.objects.filter(pk=search_obj.pk).update(
        vector_plaintext=SearchVector('content_plaintext', config='english') + SearchVector('content_plaintext', config='russian')
    )
