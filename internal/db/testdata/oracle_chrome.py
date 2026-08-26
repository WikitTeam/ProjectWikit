"""Print what Django answers for the page shell's own queries.

    docker compose exec -T web python manage.py shell < internal/db/testdata/oracle_chrome.py

It reads the pages oracle_seed.py creates, so run that one first. The answers
here are what internal/db/chrome_test.go asserts.
"""

from web.controllers import articles
from web.models.articles import Category


def show_breadcrumbs(full_name):
    crumbs = articles.get_breadcrumbs(full_name)
    print('breadcrumbs %s = %s' % (full_name, [articles.get_full_name(a) for a in crumbs]))


def show_tags(full_name):
    block = articles.get_tags_categories(full_name)
    print('tags %s = %s' % (full_name, [
        (category.name, category.priority, [tag.full_name for tag in tags])
        for category, tags in block.items()
    ]))


def show_rev(full_name):
    article = articles.get_article(full_name)
    entry = articles.get_latest_log_entry(article)
    print('rev %s = %s' % (full_name, entry.rev_number if entry else None))


def show_indexed(name):
    print('indexed %s = %s' % (name, Category.get_or_default_category(name).is_indexed))


for name in ('probe:full', 'probe:parent', 'probe:bare'):
    show_breadcrumbs(name)
    show_tags(name)
    show_rev(name)

for name in ('probe', 'no-such-category', '_default'):
    show_indexed(name)
