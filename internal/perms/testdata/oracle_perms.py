"""Record has_perm answers from the running Django, one scenario at a time.

    PERMS_CORPUS="$(cat internal/perms/testdata/perms_corpus.json)" \
    docker compose exec -T -e PERMS_CORPUS \
        web python manage.py shell < internal/perms/testdata/oracle_perms.py

Every scenario builds its roles, user and page inside a transaction that is
rolled back afterwards, so the development database keeps its own rows.
"""

import json
import os

from django.contrib.auth.models import Permission
from django.db import transaction

from web.models.articles import Article, Category
from web.models.forum import ForumCategory, ForumPost, ForumSection, ForumThread
from web.models.roles import Role, RolePermissionsOverride
from web.models.users import ExtendedAnonymousUser, User
from web.permissions import get_role_permissions_content_type

CORPUS = json.loads(os.environ['PERMS_CORPUS'])
CONTENT_TYPE = get_role_permissions_content_type()


class Rollback(Exception):
    pass


def permissions(codenames):
    return [Permission.objects.get(codename=code, content_type=CONTENT_TYPE) for code in codenames]


def build_roles(specs):
    roles = {}
    for spec in specs:
        role, _ = Role.objects.get_or_create(slug=spec['slug'])
        role.permissions.set(permissions(spec.get('permissions') or []))
        role.restrictions.set(permissions(spec.get('restrictions') or []))
        roles[spec['slug']] = role
    return roles


def build_user(spec, roles, index):
    if spec['kind'] == 'anonymous':
        return ExtendedAnonymousUser()
    user = User.objects.create(
        username='permprobe%d' % index,
        type='normal',
        is_active=spec['kind'] not in ('inactive', 'inactive_superuser'),
        is_superuser=spec['kind'] in ('superuser', 'inactive_superuser'),
        is_forum_active=not spec.get('forum_barred'),
    )
    user.roles.set([roles[slug] for slug in (spec.get('roles') or [])])
    return user


def build_forum_object(spec, user, index):
    section = ForumSection.objects.create(
        name='permprobe%d' % index,
        is_hidden_for_users=bool(spec.get('hidden_for_users')),
    )
    if spec['kind'] == 'forum_section':
        return ForumSection.objects.get(pk=section.pk)
    category = ForumCategory.objects.create(name='permprobe%d' % index, section=section)
    if spec['kind'] == 'forum_category':
        return ForumCategory.objects.get(pk=category.pk)
    author = None if user.is_anonymous else user
    thread_author = spec.get('author') if spec['kind'] == 'forum_thread' else spec.get('thread_author')
    thread = ForumThread.objects.create(
        name='permprobe%d' % index,
        category=category,
        author=author if thread_author else None,
        is_locked=bool(spec.get('locked')),
    )
    if spec['kind'] == 'forum_thread':
        return ForumThread.objects.get(pk=thread.pk)
    post = ForumPost.objects.create(
        name='permprobe%d' % index,
        thread=thread,
        author=author if spec.get('author') else None,
    )
    return ForumPost.objects.get(pk=post.pk)


def build_object(spec, roles, user, index):
    if spec is None:
        return None
    if spec['kind'].startswith('forum_'):
        return build_forum_object(spec, user, index)
    category = Category.objects.create(name='permprobe%d' % index)
    for override_spec in spec.get('overrides') or []:
        override = RolePermissionsOverride.objects.create(role=roles[override_spec['role']])
        override.permissions.set(permissions(override_spec.get('permissions') or []))
        override.restrictions.set(permissions(override_spec.get('restrictions') or []))
        category.permissions_override.add(override)
    if spec['kind'] == 'category':
        return Category.objects.get(pk=category.pk)
    article = Article.objects.create(
        category=category.name,
        name='probe',
        title='probe',
        locked=bool(spec.get('locked')),
    )
    if spec.get('author') and not user.is_anonymous:
        article.authors.add(user)
    return Article.objects.get(pk=article.pk)


def answers(scenario, index):
    out = []
    try:
        with transaction.atomic():
            roles = build_roles(scenario['roles'])
            user = build_user(scenario['user'], roles, index)
            obj = build_object(scenario.get('object'), roles, user, index)
            if not user.is_anonymous:
                user = User.objects.get(pk=user.pk)
            for code in CORPUS['queried']:
                out.append('%s = %s' % (code, 'true' if user.has_perm('roles.' + code, obj) else 'false'))
            raise Rollback
    except Rollback:
        pass
    return out


lines = []
for i, scenario in enumerate(CORPUS['scenarios']):
    lines.append('=== %s' % scenario['name'])
    lines.extend(answers(scenario, i))
print('== perms ==')
print('\n'.join(lines))
