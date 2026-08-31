"""Dump the same corpus with Python's own json.dumps.

    WIKIJSON_CORPUS="$(cat internal/wikijson/testdata/wikijson_corpus.json)" \
        python internal/wikijson/testdata/oracle_wikijson.py > wikijson.python.golden

It needs no Django, so a host Python does. To check the version the server runs:

    WIKIJSON_CORPUS="$(cat internal/wikijson/testdata/wikijson_corpus.json)" \
        docker compose exec -T -e WIKIJSON_CORPUS web python \
        < internal/wikijson/testdata/oracle_wikijson.py

Then compare with internal/wikijson/testdata/wikijson.golden. Each corpus entry is a
JSON document; json.loads keeps the key order and tells an int from a float,
which is what the comparison is about.
"""

import io
import json
import os
import sys

corpus = os.environ.get('WIKIJSON_CORPUS')
if not corpus:
    raise SystemExit(__doc__)

out = io.TextIOWrapper(sys.stdout.buffer, encoding='utf-8', newline='\n')
for doc in json.loads(corpus):
    out.write('=== %s\n%s\n' % (doc, json.dumps(json.loads(doc))))
out.flush()
