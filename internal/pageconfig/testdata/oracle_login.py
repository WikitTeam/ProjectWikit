"""Record what Django puts in login_status_config, one case at a time.

    LOGIN_CORPUS="$(cat internal/pageconfig/testdata/login_corpus.json)" \
    docker compose exec -T -e LOGIN_CORPUS \
        web python manage.py shell < internal/pageconfig/testdata/oracle_login.py \
        > pageconfig.django.golden

Then compare it with internal/pageconfig/testdata/pageconfig.golden.

Every case builds its roles and user inside a transaction that is rolled back
afterwards, so the development database keeps its own rows and every case can
reuse the same username and pk. json.dumps is patched the way wsgi.py patches
it, which is what teaches it these dataclasses. notificationCount comes from the
corpus rather than from rows, because the count itself is what
internal/db/chrome_test.go checks.
"""

import io
import json
import os
import sys

from django.contrib.auth.models import Permission
from django.db import transaction

from web.models.roles import Role
from web.models.users import ExtendedAnonymousUser, User
from web.permissions import get_role_permissions_content_type
from renderer.utils import render_user_to_json
from web.util.json import replace_json_dumps_default

replace_json_dumps_default()

CORPUS = json.loads(os.environ['LOGIN_CORPUS'])
CONTENT_TYPE = get_role_permissions_content_type()

sys.stdout = io.TextIOWrapper(sys.stdout.buffer, encoding='utf-8', newline='\n')


class Rollback(Exception):
    pass


def build_roles(specs):
    built = []
    for spec in specs or []:
        role, _ = Role.objects.get_or_create(slug=spec['slug'])
        role.index = spec['index']
        role.is_staff = spec['is_staff']
        role.group_votes = spec['group_votes']
        role.inline_visual_mode = spec['inline_visual_mode']
        role.profile_visual_mode = spec['profile_visual_mode']
        role.save()
        if spec['can_edit_articles']:
            role.permissions.set([Permission.objects.get(
                codename='edit_articles', content_type=CONTENT_TYPE)])
        else:
            role.permissions.set([])
        built.append(role)
    return built


def build_user(spec):
    user = User.objects.create(
        id=424242,
        username=spec['username'],
        type=spec['type'],
        display_name=spec['display_name'],
        wikidot_username=spec['wikidot_username'] or None,
        avatar=spec['avatar'],
        is_active=spec['is_active'],
        is_superuser=spec['is_superuser'],
        api_key='login-probe-key',
    )
    return user


def config(case):
    if case['kind'] == 'system':
        return json.dumps(render_user_to_json(None))
    out = None
    try:
        with transaction.atomic():
            roles = build_roles(case.get('roles'))
            if case['kind'] == 'anonymous':
                user = ExtendedAnonymousUser()
            else:
                user = build_user(case['user'])
                user.roles.set(roles)
                user = User.objects.get(pk=user.pk)
            out = json.dumps({
                'user': render_user_to_json(user),
                'notificationCount': case['notification_count'],
            })
            raise Rollback
    except Rollback:
        pass
    return out


for case in CORPUS:
    sys.stdout.write('=== %s\n%s\n' % (case['name'], config(case)))
