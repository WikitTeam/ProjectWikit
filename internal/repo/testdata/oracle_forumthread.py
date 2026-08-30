"""Records what the ForumThread module renders for a reader browsing a thread.

Run oracle_seed.py first, then, with the corpus the Go test exports under -update:

    docker compose exec -T -e FORUMTHREAD_CORPUS="$(cat internal/repo/testdata/forumthread_corpus.json)" \
        web python manage.py shell < internal/repo/testdata/oracle_forumthread.py

Threads and posts are looked up by name so the corpus does not depend on what
the sequence handed out. Compare the output with
internal/repo/testdata/forumthread.golden.
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
from web.models.forum import ForumPost, ForumThread
from web.models.site import Site
from web.models.users import User
from web.util.json import replace_json_dumps_default

replace_json_dumps_default()

CORPUS = json.loads(os.environ['FORUMTHREAD_CORPUS'])


def thread_id(case):
    if case.get('thread'):
        return ForumThread.objects.get(name=case['thread'], category__isnull=False).id
    article = articles.get_article(case['article'])
    if article is None:
        raise SystemExit('%s is missing; run oracle_seed.py first' % case['article'])
    return ForumThread.objects.get(article=article).id


def path_params(case):
    params = dict(case.get('path') or {})
    if case.get('thread') or case.get('article'):
        params['t'] = str(thread_id(case))
    if case.get('post'):
        params['post'] = str(ForumPost.objects.get(name=case['post']).id)
    return params


def main():
    article = articles.get_article('forum:thread')
    if article is None:
        raise SystemExit('forum:thread is missing; run manage.py seed first')

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
                body = modules.render_module('ForumThread', context, case.get('params') or {})
            except ModuleError as error:
                body = '!error: %s' % error.message
            out.append('=== %s\ntitle: %s\nstatus: %d\n%s\n' % (
                case['name'], context.title, context.status, body))
    sys.stdout.write(''.join(out))


main()
