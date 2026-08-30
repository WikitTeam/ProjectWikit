"""Records what the ForumNewThread module renders for a reader starting a thread.

Run oracle_seed.py first, then, with the corpus the Go test exports under -update:

    docker compose exec -T -e FORUMNEWTHREAD_CORPUS="$(cat internal/repo/testdata/forumnewthread_corpus.json)" \
        web python manage.py shell < internal/repo/testdata/oracle_forumnewthread.py

Category ids are looked up by name so the corpus does not depend on what the
sequence handed out. Compare the output with
internal/repo/testdata/forumnewthread.golden.
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
from web.util.json import replace_json_dumps_default

replace_json_dumps_default()

CORPUS = json.loads(os.environ['FORUMNEWTHREAD_CORPUS'])


def path_params(case):
    params = dict(case.get('path') or {})
    if case.get('category'):
        params['c'] = str(ForumCategory.objects.get(name=case['category']).id)
    return params


def main():
    article = articles.get_article('forum:new-thread')
    if article is None:
        raise SystemExit('forum:new-thread is missing; run manage.py seed first')

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
                body = modules.render_module('ForumNewThread', context, {})
            except ModuleError as error:
                body = '!error: %s' % error.message
            out.append('=== %s\ntitle: %s\nstatus: %d\n%s\n' % (
                case['name'], context.title, context.status, body))
    sys.stdout.write(''.join(out))


main()
