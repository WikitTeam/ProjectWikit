"""Records what the WantedPages module renders for each set of parameters.

Run oracle_seed.py first, then, with the corpus the Go test exports under -update:

    docker compose exec -T -e WANTEDPAGES_CORPUS="$(cat internal/repo/testdata/wantedpages_corpus.json)" \
        web python manage.py shell < internal/repo/testdata/oracle_wantedpages.py

The parameters are handed over already sorted, and category is pre-set when
category_from is given, so the JSON this writes into the widget comes out in key
order. The module would otherwise print them in the order it received them,
which is the order ftml handed them over in, and that is not stable between two
renders of the same page.

Compare the output with internal/repo/testdata/wantedpages.golden.
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

CORPUS = json.loads(os.environ['WANTEDPAGES_CORPUS'])


def sorted_params(case):
    params = dict(case.get('params') or {})
    if 'category_from' in params and 'category' not in params:
        params['category'] = params['category_from']
    return dict(sorted(params.items()))


def main():
    article = articles.get_article('main')
    if article is None:
        raise SystemExit('main is missing; run oracle_seed.py first')

    out = []
    for case in CORPUS:
        with threadvars.context():
            threadvars.put('current_site', Site.objects.first())
            viewer = AnonymousUser()
            threadvars.put('current_user', viewer)
            context = RenderContext(article, article, dict(case.get('path') or {}), viewer)
            try:
                body = modules.render_module('WantedPages', context, sorted_params(case))
            except ModuleError as error:
                body = '!error: %s' % error.message
            except Exception as error:
                body = '!crash: %s: %s' % (type(error).__name__, error)
            out.append('=== %s\n%s\n' % (case['name'], body))
    sys.stdout.write(''.join(out))


main()
