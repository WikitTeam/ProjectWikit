"""Records what the rate module renders for a page a reader is looking at.

Run oracle_seed.py first, then, with the corpus the Go test exports under -update:

    docker compose exec -T -e RATE_CORPUS="$(cat internal/modules/testdata/rate_corpus.json)" \
        web python manage.py shell < internal/modules/testdata/oracle_rate.py

The module's own render is called, so the templates and the number formatting
come from the source and not from a copy here. Compare the output with
internal/modules/testdata/rate.golden.
"""
import json
import os
import sys

from django.contrib.auth.models import AnonymousUser

import modules.rate
from renderer.parser import RenderContext
from web import threadvars
from web.controllers import articles
from web.models.site import Site
from web.models.users import User

CORPUS = json.loads(os.environ['RATE_CORPUS'])


def main():
    out = []
    for case in CORPUS:
        article = articles.get_article(case['page'])
        if article is None:
            raise SystemExit('%s is missing; run oracle_seed.py first' % case['page'])
        with threadvars.context():
            threadvars.put('current_site', Site.objects.first())
            viewer = AnonymousUser()
            if case['viewer']:
                viewer = User.objects.get(username=case['viewer'])
            threadvars.put('current_user', viewer)
            context = RenderContext(article, article, {}, viewer)
            out.append('=== %s\n%s\n' % (case['name'], modules.rate.render(context, {})))
    sys.stdout.write(''.join(out))


main()
