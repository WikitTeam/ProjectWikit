"""Records what Django resolves each page variable to, for vars.golden.

Run oracle_seed.py first, then, with the corpus the Go test exports under -update:

    docker compose exec -T -e VARS_CORPUS="$(cat internal/page/testdata/vars_corpus.json)" \\
        web python manage.py shell < internal/page/testdata/oracle_vars.py
"""
import json
import os
import sys

from modules.listpages import get_page_vars, render_var
from renderer import get_this_page_params
from web import threadvars
from web.controllers import articles
from web.models.site import Site
from web.models.users import User

CORPUS = json.loads(os.environ['VARS_CORPUS'])


def resolve(page_vars, article, name):
    if name.startswith('this|'):
        return get_this_page_params(page_vars, name)
    value = render_var(name, page_vars, article)
    if value is None:
        return '%%' + name + '%%'
    return value


def main():
    out = []
    for case in CORPUS['articles']:
        article = articles.get_article(case['name'])
        if article is None:
            raise SystemExit('%s is missing; run oracle_seed.py first' % case['name'])
        with threadvars.context():
            threadvars.put('current_site', Site.objects.first())
            viewer = None
            if case['viewer']:
                viewer = User.objects.get(username=case['viewer'])
            threadvars.put('current_user', viewer)
            page_vars = get_page_vars(article)
            for name in CORPUS['names']:
                out.append('=== %s %s\n%s\n' % (case['name'], name, resolve(page_vars, article, name)))
    sys.stdout.write(''.join(out))


main()
