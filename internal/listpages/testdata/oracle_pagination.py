"""Record Django's own pager for the cases the Go golden covers.

    docker compose exec -T -e PAGINATION_CORPUS="$(cat pagination_corpus.json)" \
        web python manage.py shell < oracle_pagination.py > pagination.django

The corpus is exported by `go test ./internal/listpages -update`. Compare the
output with testdata/pagination.golden; they have to be byte for byte the same.
"""

import json
import os
import sys

from modules.listpages import render_pagination

try:
    CORPUS = json.loads(os.environ['PAGINATION_CORPUS'])
except KeyError:
    raise SystemExit(__doc__)


def main():
    out = []
    for case in CORPUS:
        html = render_pagination(case['base_path'], case['page'], case['total_pages'])
        out.append('=== %s\n%s\n' % (case['name'], html))
    sys.stdout.write(''.join(out))


main()
