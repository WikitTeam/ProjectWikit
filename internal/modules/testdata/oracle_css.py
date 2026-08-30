"""Records what the CSS module returns and what it leaves on the context.

Run with the corpus the Go test exports under -update:

    docker compose exec -T -e CSS_CORPUS="$(cat internal/modules/testdata/css_corpus.json)" \
        web python manage.py shell < internal/modules/testdata/oracle_css.py

The minified text will not match Go, which uses a different minifier; the point
of this oracle is everything around it. Compare with
internal/modules/testdata/css.golden.
"""
import json
import os
import sys

from django.contrib.auth.models import AnonymousUser

import modules.css
from renderer.parser import RenderContext
from web import threadvars
from web.controllers import articles
from web.models.site import Site

CORPUS = json.loads(os.environ['CSS_CORPUS'])


def main():
    article = articles.get_article('main')
    if article is None:
        raise SystemExit('main is missing; run manage.py seed first')

    out = []
    for case in CORPUS:
        with threadvars.context():
            threadvars.put('current_site', Site.objects.first())
            threadvars.put('current_user', AnonymousUser())
            context = RenderContext(article, None, dict(case.get('path') or {}), AnonymousUser())
            body = modules.css.render(context, case.get('params') or {}, case.get('body') or '')
            out.append('=== %s\nreturned: %s\ncomputed: %s\naddcss: %s\n' % (
                case['name'],
                json.dumps(str(body)),
                json.dumps(context.computed_style),
                json.dumps(context.add_css)))
    sys.stdout.write(''.join(out))


main()
