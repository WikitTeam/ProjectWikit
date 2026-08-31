"""Records what the ListUsers module renders for a reader and for page authors.

Run oracle_seed.py first, then, with the corpus the Go test exports under -update:

    docker compose exec -T -e LISTUSERS_CORPUS="$(cat internal/repo/testdata/listusers_corpus.json)" \
        web python manage.py shell < internal/repo/testdata/oracle_listusers.py

Two variables deliberately differ from what Go answers. %%title%% is the display
name there and the account name here, and %%author_linked%% is a working user
chip there where the renderer escapes it into text here. Every other case has to
match internal/repo/testdata/listusers.golden.
"""
import json
import os
import sys

from django.contrib.auth.models import AnonymousUser

import modules
from modules import ModuleError
from renderer.parser import RenderContext
from web import threadvars
from web.controllers import articles
from web.models.site import Site
from web.models.users import ExtendedAnonymousUser, User

CORPUS = json.loads(os.environ['LISTUSERS_CORPUS'])


def main():
    out = []
    for case in CORPUS:
        name = case.get('article') or 'main'
        article = articles.get_article(name)
        if article is None:
            raise SystemExit('%s is missing; run oracle_seed.py first' % name)
        with threadvars.context():
            threadvars.put('current_site', Site.objects.first())
            viewer = ExtendedAnonymousUser()
            if case['viewer']:
                viewer = User.objects.get(username=case['viewer'])
            threadvars.put('current_user', viewer)
            context = RenderContext(article, article, {}, viewer)
            try:
                body = modules.render_module('ListUsers', context, dict(case.get('params') or {}), case.get('body') or '')
            except ModuleError as error:
                body = '!error: %s' % error.message
            out.append('=== %s\n%s\n' % (case['name'], body))
    sys.stdout.write(''.join(out))


main()
