"""Records what the SiteChanges module renders for a reader browsing the log.

Run oracle_seed.py first, then, with the corpus the Go test exports under -update:

    docker compose exec -T -e SITECHANGES_CORPUS="$(cat internal/repo/testdata/sitechanges_corpus.json)" \
        web python manage.py shell < internal/repo/testdata/oracle_sitechanges.py

Compare the output with internal/repo/testdata/sitechanges.golden.
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
from web.models.users import User
from web.util.json import replace_json_dumps_default

replace_json_dumps_default()

CORPUS = json.loads(os.environ['SITECHANGES_CORPUS'])


def path_params(case):
    params = dict(sorted((case.get('path') or {}).items()))
    for key in case.get('bare') or []:
        params[key] = None
    return params


def main():
    article = articles.get_article('probe:changes')
    if article is None:
        raise SystemExit('probe:changes is missing; run oracle_seed.py first')

    out = []
    for case in CORPUS:
        with threadvars.context():
            threadvars.put('current_site', Site.objects.first())
            viewer = AnonymousUser()
            if case['viewer']:
                viewer = User.objects.get(username=case['viewer'])
            threadvars.put('current_user', viewer)
            context = RenderContext(article, article, path_params(case), viewer)
            params = dict(sorted((case.get('params') or {}).items()))
            try:
                body = modules.render_module('SiteChanges', context, params)
            except ModuleError as error:
                body = '!error: %s' % error.message
            out.append('=== %s\ntitle: %s\nstatus: %d\n%s\n' % (
                case['name'], context.title, context.status, body))
    sys.stdout.write(''.join(out))


main()
