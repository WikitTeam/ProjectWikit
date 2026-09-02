"""Compares pwikit's /local--html/ against Django's for a block that is there.

Run oracle_seed.py first, then start both servers and:

    python internal/localitem/testdata/oracle_localitem.py

The hash in the URL is the md5 of the block, so it moves with the seed. This
reads it off the rendered page instead of hardcoding it. The other cases of the
three routes are in internal/difftest/corpus/media.txt.
"""
import http.client
import re
import sys

PWIKIT_PORT = 8080
DJANGO_PORT = 8000
MAIN_HOST = 'localhost'
MEDIA_HOST = 'media.localhost'
PAGE = '/probeoff:unratable'

COMPARED = ['content-type', 'content-length', 'vary']

SRC = re.compile(r'/local--html/[^"\'\s]+')


def fetch(port, host, path):
    conn = http.client.HTTPConnection('127.0.0.1', port, timeout=30)
    conn.putrequest('GET', path, skip_host=True, skip_accept_encoding=True)
    conn.putheader('Host', host)
    conn.endheaders()
    resp = conn.getresponse()
    body = resp.read()
    headers = {k: resp.getheader(k) for k in COMPARED}
    conn.close()
    return resp.status, headers, body


def block_urls():
    _, _, page = fetch(DJANGO_PORT, MAIN_HOST, PAGE)
    found = SRC.findall(page.decode('utf-8'))
    if not found:
        raise SystemExit('%s carries no external html block; re-run oracle_seed.py' % PAGE)
    return found


def main():
    unexplained = 0
    for path in block_urls():
        a = fetch(PWIKIT_PORT, MEDIA_HOST, path)
        b = fetch(DJANGO_PORT, MEDIA_HOST, path)
        if a == b:
            print('same      %s' % path)
            continue
        unexplained += 1
        print('DIFF      %s' % path)
        print('          pwikit  status=%s len=%d' % (a[0], len(a[2])))
        print('          django  status=%s len=%d' % (b[0], len(b[2])))
        for k in COMPARED:
            if a[1][k] != b[1][k]:
                print('          %-16s pwikit=%r django=%r' % (k, a[1][k], b[1][k]))
        if a[2] != b[2]:
            print('          body pwikit=%r' % a[2][:200])
            print('          body django=%r' % b[2][:200])
    return 1 if unexplained else 0


if __name__ == '__main__':
    sys.exit(main())
