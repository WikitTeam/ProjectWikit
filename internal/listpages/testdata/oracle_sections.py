"""Record what Python's regex pulls out of a module body.

    docker compose exec -T -e SECTIONS_CORPUS="$(cat sections_corpus.json)" \
        web python manage.py shell < oracle_sections.py > sections.django

The pattern is lifted out of the module source rather than copied here, so a
transcription slip in this script cannot make the two sides agree by accident.
Compare the output with testdata/sections.golden.
"""

import ast
import json
import os
import re
import sys

try:
    CORPUS = json.loads(os.environ['SECTIONS_CORPUS'])
except KeyError:
    raise SystemExit(__doc__)

SOURCE = 'modules/listpages/__init__.py'


def find_pattern():
    tree = ast.parse(open(SOURCE, encoding='utf-8').read())
    for node in ast.walk(tree):
        if not isinstance(node, ast.Call):
            continue
        func = node.func
        if not isinstance(func, ast.Attribute) or func.attr != 'match':
            continue
        if not isinstance(func.value, ast.Name) or func.value.id != 're':
            continue
        first = node.args[0]
        if isinstance(first, ast.Constant) and 'head]]' in first.value:
            return first.value
    raise SystemExit('no head/body/foot pattern found in %s' % SOURCE)


def main():
    pattern = re.compile(find_pattern(), re.S | re.I | re.M)
    out = []
    for case in CORPUS:
        selection = pattern.match(case['content'])
        head = body = foot = ''
        if selection:
            head = selection.group('head') or ''
            body = selection.group('body') or ''
            foot = selection.group('foot') or ''
        out.append('=== %s\nhead=%s\nbody=%s\nfoot=%s\n' % (
            case['name'], json.dumps(head), json.dumps(body), json.dumps(foot)))
    sys.stdout.write(''.join(out))


main()
