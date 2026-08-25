"""Record the path layer's answers from the running Django.

    PATH_CORPUS="$(cat internal/article/testdata/path_corpus.json)" \
    THISPAGE_CORPUS="$(cat internal/article/testdata/thispage_corpus.json)" \
    docker compose exec -T -e PATH_CORPUS -e THISPAGE_CORPUS \
        web python manage.py shell < internal/article/testdata/oracle_path.py

Prints the two golden files with a separator between them. The forum rewrite is
inline in ArticleView.get_context_data rather than a method, so those four
patterns are the one thing here copied instead of called.
"""

import json
import os
import re

from renderer.templates import apply_template
from web.views.article import ArticleView

PATH_CORPUS = json.loads(os.environ['PATH_CORPUS'])
THISPAGE_CORPUS = json.loads(os.environ['THISPAGE_CORPUS'])

PRINTABLE_ESCAPES = {'"': '\\"', '\\': '\\\\', '\n': '\\n', '\r': '\\r', '\t': '\\t'}


def goquote(s):
    out = ['"']
    for ch in s:
        if ch in PRINTABLE_ESCAPES:
            out.append(PRINTABLE_ESCAPES[ch])
        elif ord(ch) < 0x20 or ord(ch) == 0x7f:
            out.append('\\x%02x' % ord(ch))
        else:
            out.append(ch)
    out.append('"')
    return ''.join(out)


def rewrite_forum(path):
    match = re.match(r'^forum/start(.*)$', path)
    if match:
        return 'forum:start' + match[1]
    match = re.match(r'^forum/c-(\d+)(.*)$', path)
    if match:
        return 'forum:category/c/' + match[1] + match[2]
    match = re.match(r'^forum/t-(\d+)(.*)$', path)
    if match:
        return 'forum:thread/t/' + match[1] + match[2]
    match = re.match(r'^forum/s-(\d+)(.*)$', path)
    if match:
        return 'forum:start/s/' + match[1] + match[2]
    return path


def path_block():
    lines = []
    for raw in PATH_CORPUS:
        name, params = ArticleView.get_path_params(rewrite_forum(raw))
        lines.append('=== %s' % raw)
        lines.append('name %s' % goquote(name))
        for key, value in params.items():
            if value is None:
                lines.append('param %s = <none>' % goquote(key))
            else:
                lines.append('param %s = %s' % (goquote(key), goquote(value)))
    return lines


def thispage_block():
    params = {}
    for entry in THISPAGE_CORPUS['params']:
        params[entry['key']] = entry['value']
    more = {'canonical_url': THISPAGE_CORPUS['canonical_url']}

    lines = []
    for name in THISPAGE_CORPUS['names']:
        source = '%%' + name + '%%'
        try:
            value = apply_template(source, lambda p: ArticleView.get_this_page_params(params, p, more))
        except Exception as exc:
            lines.append('%s -> <%s>' % (name, type(exc).__name__))
            continue
        lines.append('%s -> %s' % (name, goquote(value)))
    return lines


print('== path ==')
print('\n'.join(path_block()))
print('== thispage ==')
print('\n'.join(thispage_block()))
