"""Record what Django's ListPagesParams makes of each corpus case.

    docker compose exec -T -e PARAMS_CORPUS="$(cat params_corpus.json)" \
        web python manage.py shell < oracle_params.py > params.django

The corpus is exported by `go test ./internal/listpages -update`, which needs
PWIKIT_TEST_DSN pointing at the same database this runs against. Row ids are
printed raw, so the two sides only line up when both read that one database.
Compare the output with testdata/params.golden.
"""

import json
import os
import sys

from web import threadvars
from web.controllers import articles
from web.models.site import Site
from web.models.users import User
from modules.listpages import param
from modules.listpages.params import ListPagesParams

try:
    CORPUS = json.loads(os.environ['PARAMS_CORPUS'])
except KeyError:
    raise SystemExit(__doc__)


def one(parsed, kind):
    found = parsed.get_type(kind)
    return found[0] if found else None


def ids(tags):
    return ','.join(str(t.pk) for t in sorted(tags, key=lambda t: t.pk))


def optional(value):
    return '-' if value is None else str(value)


def pointer_id(holder, attr):
    if holder is None:
        return '-'
    value = getattr(holder, attr)
    return 'null' if value is None else str(value.pk)


def stamp(value):
    return value.strftime('%Y-%m-%dT%H:%M:%S')


def dump_time(created):
    if created is None:
        return '-'
    return '%s %s %s' % (created.type, stamp(created.start), stamp(created.end))


def dump_num(holder, attr):
    if holder is None:
        return '-'
    return '%s %.6f' % (holder.type, getattr(holder, attr))


def dump(parsed):
    article_param = one(parsed, param.Article)
    full_name = one(parsed, param.FullName)
    page_type = one(parsed, param.Type)
    name = one(parsed, param.Name)
    prefix = one(parsed, param.NamePrefix)
    tags = one(parsed, param.Tags)
    exact = one(parsed, param.ExactTags)
    category = one(parsed, param.Category)
    parent = one(parsed, param.Parent)
    not_parent = one(parsed, param.NotParent)
    created_by = one(parsed, param.CreatedBy)
    sort = one(parsed, param.Sort)
    offset = one(parsed, param.Offset)
    limit = one(parsed, param.Limit)
    pagination = one(parsed, param.Pagination)

    lines = [
        ('invalid', 'true' if parsed.has_type(param.Invalid) else 'false'),
        ('only', '-' if article_param is None else str(article_param.article.pk)),
        ('fullname', '-' if full_name is None else full_name.full_name),
        ('pagetype', '-' if page_type is None else page_type.type),
        ('name', '-' if name is None else name.name),
        ('nameprefix', '-' if prefix is None else prefix.prefix),
        ('notags', 'true' if parsed.has_type(param.NoTags) else 'false'),
        ('required', '' if tags is None else ids(tags.required)),
        ('present', '' if tags is None else ids(tags.present)),
        ('absent', '' if tags is None else ids(tags.absent)),
        ('exact', '' if exact is None else ids(exact.tags)),
        ('categories', '' if category is None else ','.join(str(c) for c in category.allowed)),
        ('notcategories', '' if category is None else ','.join(str(c) for c in category.not_allowed)),
        ('parent', pointer_id(parent, 'parent')),
        ('notparent', pointer_id(not_parent, 'parent')),
        ('author', '-' if created_by is None else str(created_by.user.pk)),
        ('created_at', dump_time(one(parsed, param.CreatedAt))),
        ('rating', dump_num(one(parsed, param.Rating), 'rating')),
        ('votes', dump_num(one(parsed, param.Votes), 'votes')),
        ('popularity', dump_num(one(parsed, param.Popularity), 'popularity')),
        ('sort', '-' if sort is None else '%s %s' % (sort.column, sort.direction)),
        ('offset', '0' if offset is None else str(offset.offset)),
        ('limit', '-' if limit is None else str(limit.limit)),
        ('page', '1' if pagination is None else str(pagination.page)),
        ('perpage', '20' if pagination is None else str(pagination.per_page)),
    ]
    return ''.join('%s=%s\n' % pair for pair in lines)


def main():
    out = []
    with threadvars.context():
        threadvars.put('current_site', Site.objects.first())
        for case in CORPUS:
            article = articles.get_article(case['page']) if case.get('page') else None
            viewer = None
            if case.get('viewer'):
                viewer = User.objects.filter(username=case['viewer']).first()
            parsed = ListPagesParams(article, viewer, dict(case.get('params') or {}),
                                     dict(case.get('path') or {}))
            out.append('=== %s\n%s' % (case['name'], dump(parsed)))
    sys.stdout.write(''.join(out))


main()
