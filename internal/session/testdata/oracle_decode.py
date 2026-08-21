import json
import os

from django.core import signing
from django.contrib.sessions.serializers import JSONSerializer

KEY = "test-secret-key-do-not-use"
SALT = "django.contrib.sessions.SessionStore"

VECTORS = json.loads(os.environ["GO_SESSIONS"])

print("---ORACLE-START---")
failures = 0
for v in VECTORS:
    try:
        got = signing.loads(
            v["encoded"], key=KEY, salt=SALT, serializer=JSONSerializer
        )
    except Exception as e:
        print("FAIL %s: %s" % (json.dumps(v["data"], ensure_ascii=False), e))
        failures += 1
        continue
    if got != v["data"]:
        print("FAIL %s: decoded %s" % (
            json.dumps(v["data"], ensure_ascii=False),
            json.dumps(got, ensure_ascii=False),
        ))
        failures += 1
    else:
        print("ok   %s" % json.dumps(v["data"], ensure_ascii=False))
print("%d/%d accepted by Django" % (len(VECTORS) - failures, len(VECTORS)))
