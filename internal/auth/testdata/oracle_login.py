"""usage: python internal/auth/testdata/oracle_login.py <username> <password>"""
import os
import re
import sys
import urllib.error
import urllib.parse
import urllib.request

BASE = os.environ.get('PWIKIT_ORACLE_BASE', 'http://127.0.0.1:8000')
HOST = os.environ.get('PWIKIT_ORACLE_HOST', 'localhost')
LOGIN = '/-/login'


def secret_key():
    with open('.env', encoding='utf-8') as f:
        for line in f:
            key, _, value = line.partition('=')
            if key.strip() == 'SECRET_KEY':
                return value.strip()
    raise SystemExit('.env has no SECRET_KEY')


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None


def cookies_from(response):
    jar = {}
    for header in response.headers.get_all('Set-Cookie') or []:
        name, _, rest = header.partition('=')
        jar[name.strip()] = rest.split(';', 1)[0]
    return jar


def request(path, jar, data=None, referer=None):
    body = urllib.parse.urlencode(data).encode() if data else None
    req = urllib.request.Request(BASE + path, data=body)
    req.add_header('Host', HOST)
    if jar:
        req.add_header('Cookie', '; '.join('%s=%s' % kv for kv in jar.items()))
    if referer:
        req.add_header('Referer', referer)
    opener = urllib.request.build_opener(NoRedirect)
    try:
        return opener.open(req)
    except urllib.error.HTTPError as e:
        return e


def main(argv):
    if len(argv) != 3:
        raise SystemExit(__doc__)
    username, password = argv[1], argv[2]

    form = request(LOGIN, {})
    jar = cookies_from(form)
    token = re.search(r'name="csrfmiddlewaretoken" value="([^"]+)"', form.read().decode())
    if not token:
        raise SystemExit('login form carries no csrf token')

    response = request(LOGIN, jar, data={
        'csrfmiddlewaretoken': token.group(1),
        'username': username,
        'password': password,
    }, referer='http://%s%s' % (HOST, LOGIN))
    jar.update(cookies_from(response))

    session_key = jar.get('pwikit_sessionid')
    if not session_key:
        raise SystemExit('login did not set a session cookie; check the credentials')

    print('PWIKIT_TEST_SECRET_KEY=%s' % secret_key())
    print('PWIKIT_TEST_SESSION_KEY=%s' % session_key)
    print('PWIKIT_TEST_SESSION_USER=%s' % username)


if __name__ == '__main__':
    main(sys.argv)
