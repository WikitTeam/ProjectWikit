"""Records what the ForumCategory module renders for a reader browsing a category.

Run oracle_seed.py first, then, with the corpus the Go test exports under -update:

    docker compose exec -T -e FORUMCATEGORY_CORPUS="$(cat internal/repo/testdata/forumcategory_corpus.json)" \
        web python manage.py shell < internal/repo/testdata/oracle_forumcategory.py

Category ids are looked up by name so the corpus does not depend on what the
sequence handed out. Compare the output with internal/repo/testdata/forumcategory.golden.
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
from web.models.forum import ForumCategory
from web.models.site import Site
from web.models.users import User

CORPUS = json.loads(os.environ['FORUMCATEGORY_CORPUS'])


def path_params(case):
    params = dict(case.get('path') or {})
    if case.get('category'):
        params['c'] = str(ForumCategory.objects.get(name=case['category']).id)
    return params


def main():
    article = articles.get_article('forum:category')
    if article is None:
        raise SystemExit('forum:category is missing; run manage.py seed first')

    out = []
    for case in CORPUS:
        with threadvars.context():
            threadvars.put('current_site', Site.objects.first())
            viewer = AnonymousUser()
            if case['viewer']:
                viewer = User.objects.get(username=case['viewer'])
            threadvars.put('current_user', viewer)
            context = RenderContext(article, article, path_params(case), viewer)
            try:
                body = modules.render_module('ForumCategory', context, {})
            except ModuleError as error:
                body = '!error: %s' % error.message
            out.append('=== %s\ntitle: %s\nstatus: %d\n%s\n' % (
                case['name'], context.title, context.status, body))
    sys.stdout.write(''.join(out))


main()
