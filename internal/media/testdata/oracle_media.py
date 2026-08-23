# Compares pwikit's /local--files/ against Django's, response by response.
# Run oracle_seed.py first, then start both servers and:
#
#   python internal/media/testdata/oracle_media.py <article-media-name>
#
# The argument is what oracle_seed.py printed. Only the headers the media view
# itself sets are compared; the rest belong to the entry layer.
import hashlib
import http.client
import sys

PWIKIT_PORT = 8080
DJANGO_PORT = 8000
HOST = 'media.localhost'

COMPARED = [
    'content-type',
    'content-length',
    'content-range',
    'content-disposition',
    'accept-ranges',
    'access-control-expose-headers',
    'content-encoding',
]

# Differences pwikit produces on purpose. Each one is a live Django defect.
KNOWN = {
    'bytes=299-299': 'Django sends Content-Length: 300 with an empty body',
    'bytes=300-400': 'Django sends Content-Length: 300 with an empty body',
    '/local--files/-/roles': 'a directory makes Django raise IsADirectoryError',
    'manage.py': 'Django serves anything reachable through ..',
}


def cases(article_media_name):
    probe = '/local--files/-/probe/'
    yield from ((probe + name, None) for name in ('a.pdf', 'a.txt', 'a.txt.gz', 'a.bin', 'empty'))
    yield from ((probe + 'a.pdf', r) for r in (
        'bytes=0-99', 'bytes=100-199', 'bytes=250-', 'bytes=0-0', 'bytes=-50',
        'bytes=0-99999', 'bytes=299-299', 'bytes=300-400', 'items=0-10',
    ))
    yield probe + 'nope.pdf', None
    yield '/local--files/nope/nope', None
    yield '/local--files/main/probe%20attach.pdf', None
    yield '/local--files/main/probe%20attach.pdf', 'bytes=0-99'
    yield '/local--files/%s/11111111-2222-3333-4444-555555555555' % article_media_name, None
    yield '/local--files/-/roles', None
    yield probe + '../../../manage.py', None


def fetch(port, path, rng):
    conn = http.client.HTTPConnection('127.0.0.1', port, timeout=10)
    conn.putrequest('GET', path, skip_host=True, skip_accept_encoding=True)
    conn.putheader('Host', HOST)
    if rng:
        conn.putheader('Range', rng)
    conn.endheaders()
    resp = conn.getresponse()
    try:
        body = resp.read()
    except http.client.IncompleteRead as e:
        body = e.partial
    headers = {k: resp.getheader(k) for k in COMPARED}
    conn.close()
    return resp.status, headers, hashlib.sha256(body).hexdigest()[:12], len(body)


def excuse(label):
    for marker, reason in KNOWN.items():
        if marker in label:
            return reason
    return None


def main(argv):
    if len(argv) != 2:
        print(__doc__ or 'usage: oracle_media.py <article-media-name>')
        return 2
    same = expected = unexpected = 0

    for path, rng in cases(argv[1]):
        label = path + (' Range=' + rng if rng else '')
        a = fetch(PWIKIT_PORT, path, rng)
        b = fetch(DJANGO_PORT, path, rng)
        if a == b:
            same += 1
            print('same      %s' % label)
            continue

        reason = excuse(label)
        if reason:
            expected += 1
            print('expected  %s  (%s)' % (label, reason))
            continue

        unexpected += 1
        print('DIFF      %s' % label)
        print('          pwikit  status=%s body=%s(%d)' % (a[0], a[2], a[3]))
        print('          django  status=%s body=%s(%d)' % (b[0], b[2], b[3]))
        for k in COMPARED:
            if a[1][k] != b[1][k]:
                print('          %-30s pwikit=%r django=%r' % (k, a[1][k], b[1][k]))

    print('\n%d same, %d different on purpose, %d unexplained' % (same, expected, unexpected))
    return 1 if unexpected else 0


if __name__ == '__main__':
    sys.exit(main(sys.argv))
