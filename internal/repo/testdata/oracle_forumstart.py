"""Records what the ForumStart module renders for a reader browsing the forum.

Run oracle_seed.py first, then, with the corpus the Go test exports under -update:

    docker compose exec -T -e FORUMSTART_CORPUS="$(cat internal/repo/testdata/forumstart_corpus.json)" \
        web python manage.py shell < internal/repo/testdata/oracle_forumstart.py

Section ids are looked up by name so the corpus does not depend on what the
sequence handed out. Compare the output with internal/repo/testdata/forumstart.golden.
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
from web.models.forum import ForumSection
from web.models.site import Site
from web.models.users import User

CORPUS = json.loads(os.environ['FORUMSTART_CORPUS'])


def path_params(case):
    params = dict(case.get('path') or {})
    if case.get('section'):
        params['s'] = str(ForumSection.objects.get(name=case['section']).id)
    return params


def main():
    article = articles.get_article('forum:start')
    if article is None:
        raise SystemExit('forum:start is missing; run manage.py seed first')

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
                body = modules.render_module('ForumStart', context, {})
            except ModuleError as error:
                body = '!error: %s' % error.message
            out.append('=== %s\ntitle: %s\nstatus: %d\n%s\n' % (
                case['name'], context.title, context.status, body))
    sys.stdout.write(''.join(out))


main()
