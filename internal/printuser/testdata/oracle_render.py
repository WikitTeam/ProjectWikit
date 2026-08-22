import json
import os
import sys

from django.conf import settings
from django.contrib.auth.models import AnonymousUser
from django.db import transaction

from renderer.utils import render_user_to_html, render_external_user_to_html
from web.models.roles import Role, RoleCategory
from web.models.users import User

CORPUS = json.loads(os.environ['RENDER_CORPUS'])


class Rollback(Exception):
    pass


def write_icons():
    for name, body in CORPUS['icons'].items():
        path = os.path.join(settings.MEDIA_ROOT, *name.split('/'))
        os.makedirs(os.path.dirname(path), exist_ok=True)
        with open(path, 'w', encoding='utf-8') as f:
            f.write(body)


def make_roles():
    categories = {}
    made = {}
    for spec in CORPUS['roles']:
        category = None
        if spec['category']:
            if spec['category'] not in categories:
                categories[spec['category']] = RoleCategory.objects.create(name=spec['category'])
            category = categories[spec['category']]
        made[spec['slug']] = Role.objects.create(
            slug=spec['slug'],
            name=spec['name'],
            short_name=spec['short_name'],
            category=category,
            inline_visual_mode=spec['inline_visual_mode'],
            profile_visual_mode=spec['profile_visual_mode'],
            color=spec['color'],
            icon=spec['icon'],
            badge_text=spec['badge_text'],
            badge_bg=spec['badge_bg'],
            badge_text_color=spec['badge_text_color'],
            badge_show_border=spec['badge_show_border'],
        )
    # Role.save renumbers every row, and ties between equal indexes resolve in
    # whatever order Postgres returns them, so the corpus index is forced back
    # on afterwards rather than passed to create.
    for spec in CORPUS['roles']:
        Role.objects.filter(pk=made[spec['slug']].pk).update(index=spec['index'])
    return made


def make_user(spec, api_key):
    user = User.objects.create(
        username=spec['username'],
        wikidot_username=spec['wikidot_username'] or None,
        display_name=spec['display_name'] or None,
        type=spec['type'],
        avatar=spec['avatar'],
        is_active=spec['is_active'],
        api_key=api_key,
    )
    return user


def render_case(case, made, index):
    avatar = case['avatar']
    hover = case['hover']
    if case['kind'] == 'system':
        return render_user_to_html(None, avatar=avatar, hover=hover)
    if case['kind'] == 'anonymous':
        return render_user_to_html(AnonymousUser(), avatar=avatar, hover=hover)
    if case['kind'] == 'external':
        return render_external_user_to_html(case['external'], avatar=avatar, hover=hover)

    spec = case['user']
    user = make_user(spec, 'oracle-key-%d' % index)
    if case['roles']:
        user.roles.add(*[made[slug] for slug in case['roles']])
    html = render_user_to_html(user, avatar=avatar, hover=hover)
    # Ids come from a sequence this script cannot control; the corpus id is the
    # one both sides agree on.
    html = html.replace('data-user-id="%d"' % user.pk, 'data-user-id="%d"' % spec['id'])
    user.delete()
    return html


def main():
    write_icons()
    out = []
    try:
        with transaction.atomic():
            made = make_roles()
            for index, case in enumerate(CORPUS['cases']):
                out.append('=== %s\n%s\n' % (case['name'], render_case(case, made, index)))
            raise Rollback()
    except Rollback:
        pass
    sys.stdout.write(''.join(out))


main()
