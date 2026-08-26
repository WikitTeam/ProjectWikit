"""Render the page shell templates in the running Django, one case at a time.

    SHELL_CORPUS="$(cat internal/shell/testdata/shell_corpus.json)" \
    docker compose exec -T -e SHELL_CORPUS \
        web python manage.py shell < internal/shell/testdata/oracle_shell.py \
        > shell.django.golden

Then compare it with internal/shell/testdata/shell.golden.

The asset URLs and the theme URL are fed in from the corpus instead of being
computed here, so the comparison stays about the templates. Line endings are
normalised because a Windows checkout carries CRLF into the container.
"""

import io
import json
import os
import sys
from datetime import datetime
from django.template.loader import render_to_string
from django.utils.safestring import mark_safe

import web.models.site as site_models
import web.templatetags.md5url as md5url_tags

CORPUS = json.loads(os.environ['SHELL_CORPUS'])

sys.stdout = io.TextIOWrapper(sys.stdout.buffer, encoding='utf-8', newline='\n')

md5url_tags.UrlCache._md5_sum = dict(CORPUS['assets'])

THEME_URL = ''
site_models.get_site_theme_url = lambda: THEME_URL


class Named:
    def __init__(self, **fields):
        self.__dict__.update(fields)


def tags_categories(specs):
    return {
        Named(name=spec['name']): [
            Named(name=tag['name'], full_name=tag['full_name'])
            for tag in spec['tags']
        ]
        for spec in specs
    }


def page_context(spec):
    return {
        'site_name': spec['site_name'],
        'site_headline': spec['site_headline'],
        'site_title': spec['site_title'],
        'site_icon': spec['site_icon'],
        'og_title': spec['og_title'],
        'og_description': spec['og_description'],
        'og_image': spec['og_image'],
        'og_url': spec['og_url'],
        'noindex': spec['noindex'],
        'google_tag_id': spec['google_tag_id'],
        'computed_style': spec['computed_style'],
        'nav_top': mark_safe(spec['nav_top']),
        'nav_side': mark_safe(spec['nav_side']),
        'title': spec['title'],
        'content': mark_safe(spec['content']),
        'breadcrumbs': list(spec['breadcrumbs'] or []),
        'tags_categories': tags_categories(spec['tags_categories'] or []),
        'rev_number': spec['rev_number'],
        'updated_at': datetime.fromisoformat(spec['updated_at']) if spec['updated_at'] else None,
        'login_status_config': spec['login_status_config'],
        'options_config': spec['options_config'],
    }


def render(case):
    global THEME_URL
    if case['kind'] == 'page':
        THEME_URL = case['page']['theme_url']
        return render_to_string('page.html', page_context(case['page']))
    if case['kind'] == 'not_found':
        return render_to_string('page_404.html', {
            'page_id': case.get('page_id', ''),
            'allow_create': case.get('allow_create', False),
            'options': case.get('options', ''),
        })
    if case['kind'] == 'forbidden':
        return render_to_string('page_403.html', {'page_id': case.get('page_id', '')})
    raise SystemExit('case %s has kind %r, want one of page, not_found, forbidden'
                     % (case['name'], case['kind']))


for case in CORPUS['cases']:
    sys.stdout.write('=== %s\n%s\n' % (case['name'], render(case).replace('\r\n', '\n')))
