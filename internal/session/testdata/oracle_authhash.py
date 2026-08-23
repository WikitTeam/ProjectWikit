"""docker compose exec -T web python - < internal/session/testdata/oracle_authhash.py"""
from django.utils.crypto import salted_hmac

KEY_SALT = 'django.contrib.auth.models.AbstractBaseUser.get_session_auth_hash'

VECTORS = [
    ('test-secret-key', 'pbkdf2_sha256$1000000$abc$def='),
    ('test-secret-key', ''),
    ('中文密钥', 'pbkdf2_sha256$1000000$abc$def='),
    ('another', 'x'),
]

for secret, password in VECTORS:
    digest = salted_hmac(KEY_SALT, password, secret=secret, algorithm='sha256').hexdigest()
    print('{%r, %r, %r},' % (secret, password, digest))
