"""Compares the search module's api_search against Django's, call by call.

Run oracle_seed.py first, then start both servers and:

    python internal/modules/testdata/oracle_search.py

The excerpt is left out of the comparison on purpose: pwikit cuts a window
around the word that was searched for, Django takes the head of the page.
The category filter is pwikit's own, so those cases only check pwikit answers.
"""
import http.client
import json
import sys

PWIKIT_PORT = 8080
DJANGO_PORT = 8000
HOST = 'localhost'
PATH = '/pw-api/modules'

PWIKIT_ONLY = {'category', 'category-and-word', 'category-unknown'}


def cases():
    yield 'word', {'q': 'probe'}
    yield 'two-words', {'q': 'probe source'}
    yield 'no-match', {'q': 'nothing matches this at all'}
    yield 'nothing-asked', {}
    yield 'author', {'author': 'probe-author'}
    yield 'author-unknown', {'author': 'nobody-at-all'}
    yield 'author-and-word', {'q': 'source', 'author': 'probe-author'}
    yield 'tag', {'tags': 'alpha'}
    yield 'tag-excluded', {'tags': 'alpha -zeta'}
    yield 'tag-unknown', {'tags': 'no-such-tag'}
    yield 'tag-with-category', {'tags': 'lang:en'}
    yield 'dates', {'datefrom': '2021-01-01', 'dateto': '2021-12-31'}
    yield 'dates-empty', {'datefrom': '1999-01-01', 'dateto': '1999-12-31'}
    yield 'offset', {'q': 'source', 'offset': 1}
    yield 'category', {'category': 'probe'}
    yield 'category-and-word', {'q': 'source', 'category': 'probestars'}
    yield 'category-unknown', {'category': 'nosuchcategory'}


def strip(payload):
    try:
        body = json.loads(payload)
    except ValueError:
        return payload
    for item in body.get('results', []):
        item.pop('excerpt', None)
    return body


def fetch(port, params):
    body = json.dumps({'module': 'search', 'method': 'search', 'params': params}).encode('utf-8')
    conn = http.client.HTTPConnection('127.0.0.1', port, timeout=30)
    conn.putrequest('POST', PATH, skip_host=True, skip_accept_encoding=True)
    conn.putheader('Host', HOST)
    conn.putheader('Content-Type', 'application/json')
    conn.putheader('Content-Length', str(len(body)))
    conn.endheaders()
    conn.send(body)
    resp = conn.getresponse()
    payload = resp.read()
    conn.close()
    return resp.status, strip(payload)


def main():
    same = only = unexplained = 0

    for name, params in cases():
        a = fetch(PWIKIT_PORT, params)
        if name in PWIKIT_ONLY:
            only += 1
            print('pwikit    %-18s total=%s' % (name, a[1].get('total') if isinstance(a[1], dict) else '?'))
            continue
        b = fetch(DJANGO_PORT, params)
        if a == b:
            same += 1
            print('same      %s' % name)
            continue
        unexplained += 1
        print('DIFF      %s' % name)
        print('          pwikit %s' % json.dumps(a[1], ensure_ascii=False)[:400])
        print('          django %s' % json.dumps(b[1], ensure_ascii=False)[:400])

    print('\n%d same, %d pwikit-only, %d unexplained' % (same, only, unexplained))
    return 1 if unexplained else 0


if __name__ == '__main__':
    sys.exit(main())
