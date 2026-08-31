"""Records what the CountPages module renders for each set of parameters.

Run oracle_seed.py first, then, with the corpus the Go test exports under -update:

    docker compose exec -T -e COUNTPAGES_CORPUS="$(cat internal/repo/testdata/countpages_corpus.json)" \
        web python manage.py shell < internal/repo/testdata/oracle_countpages.py

Compare the output with internal/repo/testdata/countpages.golden.
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

CORPUS = json.loads(os.environ['COUNTPAGES_CORPUS'])


def main():
    out = []
    for case in CORPUS:
        article = articles.get_article(case.get('article') or 'main')
        if article is None:
            raise SystemExit('%s is missing; run oracle_seed.py first' % (case.get('article') or 'main'))
        with threadvars.context():
            threadvars.put('current_site', Site.objects.first())
            viewer = AnonymousUser()
            threadvars.put('current_user', viewer)
            context = RenderContext(article, article, dict(case.get('path') or {}), viewer)
            try:
                body = modules.render_module('CountPages', context, dict(case.get('params') or {}), case.get('body') or '')
            except ModuleError as error:
                body = '!error: %s' % error.message
            out.append('=== %s\n%s\n' % (case['name'], body))
    sys.stdout.write(''.join(out))


main()
