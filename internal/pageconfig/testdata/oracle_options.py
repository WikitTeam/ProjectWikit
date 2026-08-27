"""Record what Django puts in options_config, one case at a time.

    OPTIONS_CORPUS="$(cat internal/pageconfig/testdata/options_corpus.json)" \
    docker compose exec -T -e OPTIONS_CORPUS \
        web python manage.py shell < internal/pageconfig/testdata/oracle_options.py \
        > options.django.golden

Then compare it with internal/pageconfig/testdata/options.golden.

The dict below is copied from ArticleView.get_context_data, since the view
builds it inline and there is no function to call. Everything it reads is real:
the roles, the article, the votes and the settings are created in a transaction
that is rolled back afterwards. Every case gets its own pk because the
preferences manager caches by pk and the cache outlives the rollback.
Only commentCount, canCreateTags and isWatching
come from the corpus, because each of those has its own check in
internal/db/chrome_test.go.
"""

import io
import json
import os
import sys

from django.contrib.auth.models import Permission
from django.db import transaction

from web import threadvars
from web.controllers import articles
from web.models.articles import Article, Category, Vote
from web.models.roles import Role
from web.models.settings import Settings
from web.models.site import Site
from web.models.users import ExtendedAnonymousUser, User
from web.permissions import get_role_permissions_content_type
from web.util.json import replace_json_dumps_default

replace_json_dumps_default()

CORPUS = json.loads(os.environ['OPTIONS_CORPUS'])
CONTENT_TYPE = get_role_permissions_content_type()

sys.stdout = io.TextIOWrapper(sys.stdout.buffer, encoding='utf-8', newline='\n')


class Rollback(Exception):
    pass


def permissions(codenames):
    return [Permission.objects.get(codename=code, content_type=CONTENT_TYPE) for code in codenames or []]


def build_roles(specs):
    built = []
    for spec in specs or []:
        role, _ = Role.objects.get_or_create(slug=spec['slug'])
        role.permissions.set(permissions(spec.get('permissions')))
        role.restrictions.set(permissions(spec.get('restrictions')))
        built.append(role)
    return built


def build_user(case, roles, index):
    if case['anonymous']:
        return ExtendedAnonymousUser()
    user = User.objects.create(
        id=424242 + index * 100,
        username='optionsprobe%d' % index,
        type='normal',
        is_active=not case['inactive'],
        is_superuser=case['superuser'],
        api_key='options-probe-key-%d' % index,
    )
    user.roles.set(roles)
    if case['advanced_source_editor']:
        user.preferences['qol__advanced_source_editor_enabled'] = True
    return User.objects.get(pk=user.pk)


def build_article(case, user, index):
    if not case['has_article']:
        return None
    category = Category.objects.create(name='optionsprobe%d' % index)
    Settings.objects.create(category=category, rating_mode=case['rating_mode'])
    article = Article.objects.create(
        category=category.name,
        name='probe',
        title='probe',
        locked=case['locked'],
    )
    if case['author'] and not user.is_anonymous:
        article.authors.add(user)
    for i, rate in enumerate(case['votes'].get('rates') or []):
        voter = User.objects.create(
            id=424243 + index * 100 + i, username='optionsvoter%d-%d' % (index, i), type='normal',
            api_key='options-voter-key-%d-%d' % (index, i),
        )
        Vote.objects.create(article=article, user=voter, rate=rate)
    return Article.objects.get(pk=article.pk)


def path_params(path):
    from web.views.article import ArticleView
    return ArticleView.get_path_params(path)[1] if path else {}


def config(case, index):
    out = None
    try:
        with transaction.atomic():
            roles = build_roles(case['roles'])
            user = build_user(case, roles, index)
            article = build_article(case, user, index)
            rating, votes, popularity, mode = articles.get_rating(article)
            params = path_params(case['path'])
            out = json.dumps({
                'optionsEnabled': article is not None,
                'editable': user.has_perm('roles.edit_articles', article),
                'lockable': user.has_perm('roles.lock_articles', article),
                'tagable': user.has_perm('roles.tag_articles', article),
                'pageId': case['page_id'],
                'rating': rating,
                'ratingMode': mode,
                'ratingVotes': votes,
                'ratingPopularity': popularity,
                'pathParams': params,
                'canRate': user.has_perm('roles.rate_articles', article),
                'canComment': user.has_perm('roles.comment_articles', article) if article else False,
                'canViewComments': user.has_perm('roles.view_article_comments', article) if article else False,
                'commentThread': ('/%s/comments/show' % case['page_id']) if article else None,
                'commentCount': case['comment_count'],
                'canDelete': user.has_perm('roles.delete_articles', article),
                'canCreateTags': case['can_create_tags'],
                'canManageFiles': user.has_perm('roles.manage_article_files', article),
                'canRename': user.has_perm('roles.move_articles', article),
                'canCreateHere': user.has_perm('roles.create_articles', article),
                'canManageAuthors': user.has_perm('roles.manage_article_authors', article),
                'canResetVotes': user.has_perm('roles.reset_article_votes', article),
                'canWatch': not user.is_anonymous,
                'preferences': {} if user.is_anonymous else user.preferences.all(),
                'isWatching': case['is_watching'],
            })
            raise Rollback
    except Rollback:
        pass
    return out


with threadvars.context():
    threadvars.put('current_site', Site.objects.first())
    for index, case in enumerate(CORPUS):
        sys.stdout.write('=== %s\n%s\n' % (case['name'], config(case, index)))
