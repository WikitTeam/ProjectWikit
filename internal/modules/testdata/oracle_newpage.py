"""Records what the NewPage module returns for each set of parameters.

Run with the corpus the Go test exports under -update:

    docker compose exec -T -e NEWPAGE_CORPUS="$(cat internal/modules/testdata/newpage_corpus.json)" \
        web python manage.py shell < internal/modules/testdata/oracle_newpage.py

Two things here are deliberately not what Go emits -- the wrapper markup, and
the placeholder, which Go fills with the bare example= and leaves empty when
there is none. What has to keep matching is data-config and the submit value,
against internal/modules/testdata/newpage.golden.
"""
import json
import os
import sys

from django.contrib.auth.models import AnonymousUser

import modules.newpage
from renderer.parser import RenderContext
from web import threadvars
from web.controllers import articles
from web.models.site import Site

CORPUS = json.loads(os.environ['NEWPAGE_CORPUS'])


def main():
    article = articles.get_article('main')
    if article is None:
        raise SystemExit('main is missing; run manage.py seed first')

    out = []
    for case in CORPUS:
        with threadvars.context():
            threadvars.put('current_site', Site.objects.first())
            threadvars.put('current_user', AnonymousUser())
            context = RenderContext(article, None, {}, AnonymousUser())
            body = modules.newpage.render(context, case.get('params') or {})
            out.append('=== %s\n%s\n' % (case['name'], str(body)))
    sys.stdout.write(''.join(out))


main()
