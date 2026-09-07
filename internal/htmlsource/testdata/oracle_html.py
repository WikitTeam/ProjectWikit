"""Record what the archive importer turns forum HTML into.

    docker compose cp internal/htmlsource/testdata/cases.html web:/tmp/cases.html
    docker compose exec -T -e CASES=/tmp/cases.html web python manage.py shell --no-imports \n        < internal/htmlsource/testdata/oracle_html.py \n        > internal/htmlsource/testdata/html.golden

Writes the case headers back with each case replaced by its wikidot source.
"""

import os
import sys

from web import threadvars
from web.seeds.wikit_archive import html_to_source

CASES = os.environ.get('CASES', '/tmp/cases.html')


def split(text):
    name = None
    body = []
    for line in text.split('\n'):
        if line.startswith('=== ') and line.endswith(' ==='):
            if name is not None:
                yield name, '\n'.join(body)
            name = line[4:-4]
            body = []
            continue
        if name is not None:
            body.append(line)
    if name is not None:
        yield name, '\n'.join(body)


def main():
    try:
        with open(CASES, encoding='utf-8') as f:
            raw = f.read()
    except FileNotFoundError:
        raise SystemExit('cases not found at %s; copy cases.html in and set CASES' % CASES)

    out = []
    with threadvars.context():
        threadvars.put('threadid', 0)
        for name, body in split(raw):
            out.append('=== %s ===' % name)
            out.append(html_to_source(body.strip('\n')))
    sys.stdout.write('\n'.join(out) + '\n')


main()
