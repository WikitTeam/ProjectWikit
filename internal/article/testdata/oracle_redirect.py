"""Record where Django sends a request whose page name is not normalized.

    python internal/article/testdata/oracle_redirect.py

Reads redirect_corpus.json next to this file and prints the redirect golden.
The encoding of the parameters is inline in ArticleView.get_context_data, so
only the response can answer for it.
"""

import http.client
import json
import os
import sys

PORT = int(os.environ.get('DJANGO_PORT', '8000'))
HOST = os.environ.get('DJANGO_HOST', 'localhost')


def target(path):
    conn = http.client.HTTPConnection('127.0.0.1', PORT, timeout=10)
    conn.putrequest('GET', '/' + path, skip_host=True, skip_accept_encoding=True)
    conn.putheader('Host', HOST)
    conn.endheaders()
    resp = conn.getresponse()
    resp.read()
    conn.close()
    if resp.status != 302:
        return '<none>'
    return resp.getheader('Location')


def main():
    here = os.path.dirname(os.path.abspath(__file__))
    with open(os.path.join(here, 'redirect_corpus.json'), encoding='utf-8') as f:
        corpus = json.load(f)
    for path in corpus:
        print('%s -> %s' % (path, target(path)))


if __name__ == '__main__':
    sys.exit(main())
