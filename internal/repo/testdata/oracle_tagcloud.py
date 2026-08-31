"""Records what the TagCloud module renders for each set of parameters.

Run oracle_seed.py first, then, with the corpus the Go test exports under -update:

    docker compose exec -T -e TAGCLOUD_CORPUS="$(cat internal/repo/testdata/tagcloud_corpus.json)" \
        web python manage.py shell < internal/repo/testdata/oracle_tagcloud.py

Compare the output with internal/repo/testdata/tagcloud.golden.
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

CORPUS = json.loads(os.environ['TAGCLOUD_CORPUS'])


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
            context = RenderContext(article, article, {}, viewer)
            try:
                body = modules.render_module('TagCloud', context, dict(case.get('params') or {}))
            except ModuleError as error:
                body = '!error: %s' % error.message
            except Exception as error:
                body = '!crash: %s: %s' % (type(error).__name__, error)
            out.append('=== %s\n%s\n' % (case['name'], body))
    sys.stdout.write(''.join(out))


main()
