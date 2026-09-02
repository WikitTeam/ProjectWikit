"""Compares pwikit's module API against Django's, call by call.

Run oracle_seed.py first, then start both servers and:

    python internal/moduleapi/testdata/oracle_moduleapi.py

Only the render method is compared. Everything else still goes to Django, so
there is nothing to compare it against.
"""
import http.client
import json
import sys

PWIKIT_PORT = 8080
DJANGO_PORT = 8000
HOST = 'localhost'
PATH = '/pw-api/modules'

COMPARED = ['content-type']

# Differences pwikit produces on purpose.
KNOWN = {
    'nosuchmodule': 'the error text comes from the catalog, not from a literal',
    'nosuchpage': 'the error text comes from the catalog, not from a literal',
    'nomodule': 'the error text comes from the catalog, not from a literal',
}


def cases():
    yield 'listpages', {'module': 'listpages', 'pageId': 'probe:listed', 'method': 'render',
                        'params': {'category': 'probe', 'order': 'name', 'perPage': 3}}
    yield 'listpages-page-2', {'module': 'listpages', 'pageId': 'probe:listed', 'method': 'render',
                               'params': {'category': 'probe', 'order': 'name', 'perPage': 3},
                               'pathParams': {'p': '2'}}
    yield 'countpages', {'module': 'countpages', 'pageId': 'probe:full', 'method': 'render',
                         'params': {'category': 'probe'}, 'content': '%%total%%'}
    yield 'sitechanges', {'module': 'sitechanges', 'pageId': 'probe:changes', 'method': 'render',
                          'params': {'perpage': 5}}
    yield 'wantedpages', {'module': 'wantedpages', 'pageId': 'probe:full', 'method': 'render',
                          'params': {}}
    yield 'recentposts', {'module': 'recentposts', 'pageId': 'probe:full', 'method': 'render',
                          'params': {}}
    yield 'forumthread', {'module': 'forumthread', 'pageId': 'probe:full', 'method': 'render',
                          'params': {'t': 1}}
    yield 'forumstart', {'module': 'forumstart', 'pageId': 'probe:full', 'method': 'render',
                         'params': {}}
    yield 'tagcloud', {'module': 'tagcloud', 'pageId': 'probe:full', 'method': 'render',
                       'params': {}}
    yield 'rate', {'module': 'rate', 'pageId': 'probestars:rated', 'method': 'render',
                   'params': {}}
    yield 'no-page-id', {'module': 'tagcloud', 'method': 'render', 'params': {}}
    yield 'nosuchpage', {'module': 'tagcloud', 'pageId': 'no-such-page-at-all',
                         'method': 'render', 'params': {}}
    yield 'nosuchmodule', {'module': 'NoSuchModule', 'pageId': 'probe:full', 'method': 'render',
                           'params': {}}
    yield 'nomodule', {'module': 'tagcloud', 'pageId': 'probe:full', 'method': 'render',
                       'params': {}, 'pathParams': {'nomodule': 'true'}}


def fetch(port, call):
    body = json.dumps(call).encode('utf-8')
    conn = http.client.HTTPConnection('127.0.0.1', port, timeout=30)
    conn.putrequest('POST', PATH, skip_host=True, skip_accept_encoding=True)
    conn.putheader('Host', HOST)
    conn.putheader('Content-Type', 'application/json')
    conn.putheader('Content-Length', str(len(body)))
    conn.endheaders()
    conn.send(body)
    resp = conn.getresponse()
    payload = resp.read()
    headers = {k: resp.getheader(k) for k in COMPARED}
    conn.close()
    return resp.status, headers, payload


def main():
    same = expected = unexplained = 0

    for name, call in cases():
        a = fetch(PWIKIT_PORT, call)
        b = fetch(DJANGO_PORT, call)
        if a == b:
            same += 1
            print('same      %s' % name)
            continue
        if name in KNOWN:
            expected += 1
            print('expected  %s  (%s)' % (name, KNOWN[name]))
            continue

        unexplained += 1
        print('DIFF      %s' % name)
        print('          pwikit  status=%s len=%d' % (a[0], len(a[2])))
        print('          django  status=%s len=%d' % (b[0], len(b[2])))
        for k in COMPARED:
            if a[1][k] != b[1][k]:
                print('          %-14s pwikit=%r django=%r' % (k, a[1][k], b[1][k]))
        if a[2] != b[2]:
            for i in range(min(len(a[2]), len(b[2]))):
                if a[2][i] != b[2][i]:
                    print('          first difference at byte %d' % i)
                    print('          pwikit %r' % a[2][max(0, i - 60):i + 60])
                    print('          django %r' % b[2][max(0, i - 60):i + 60])
                    break

    print('\n%d same, %d different on purpose, %d unexplained' % (same, expected, unexplained))
    return 1 if unexplained else 0


if __name__ == '__main__':
    sys.exit(main())
