"""Creates the articles oracle_vars.py and the Go side both read.

Run it inside the Django container, then run oracle_vars.py:

    docker compose exec -T web python manage.py shell < internal/page/testdata/oracle_seed.py

It commits. Timestamps are forced to fixed values afterwards so the golden does
not change every time the seed is re-run.

probe:host and probe:included are not in the golden: they are what the include
pass is checked on, by loading /probe:host from Django and rendering the same
source with pwikit render -page host -category probe.
"""
import datetime

from django.utils import timezone

from web.controllers import articles
from web.models.articles import Article, Category, Tag, TagsCategory, Vote
from web.models.settings import Settings
from web.models.site import Site
from web.models.users import User

CREATED_AT = datetime.datetime(2021, 3, 4, 5, 6, 7, tzinfo=datetime.timezone.utc)
UPDATED_AT = datetime.datetime(2022, 7, 8, 9, 10, 11, tzinfo=datetime.timezone.utc)


def make_user(username, **kwargs):
    user, created = User.objects.get_or_create(username=username, defaults=kwargs)
    if not created:
        for key, value in kwargs.items():
            setattr(user, key, value)
        user.save()
    return user


def make_article(full_name, title, source, user):
    article = articles.get_article(full_name)
    if article is None:
        article = articles.create_article(full_name, user)
        articles.create_article_version(article, source, user)
    article.title = title
    article.save()
    return article


def freeze(article):
    Article.objects.filter(pk=article.pk).update(created_at=CREATED_AT, updated_at=UPDATED_AT)


def make_tag(slug, name):
    category, _ = TagsCategory.objects.get_or_create(slug=slug, defaults=dict(name=slug))
    tag, _ = Tag.objects.get_or_create(category=category, name=name.lower())
    return tag


def vote(article, user, rate):
    Vote.objects.update_or_create(
        article=article, user=user,
        defaults=dict(rate=rate, date=timezone.now()),
    )


author = make_user(
    'probeauthor',
    display_name='Probe Author',
    type=User.UserType.Normal,
    api_key='probe-key-author',
)
coauthor = make_user(
    'probewd',
    display_name='Probe WD',
    wikidot_username='Probe WD Original',
    type=User.UserType.Wikidot,
    api_key='probe-key-wd',
)
voter = make_user('probevoter', type=User.UserType.Normal, api_key='probe-key-voter')
crowd = [
    make_user('probecrowd%d' % i, type=User.UserType.Normal, api_key='probe-key-crowd%d' % i)
    for i in range(8)
]

parent = make_article('probe:parent', 'Probe Parent', 'parent source', author)
full = make_article('probe:full', 'Probe Full', 'full source\nsecond line', author)
bare = make_article('probe:bare', 'Probe Bare', 'bare source', None)
rated = make_article('probestars:rated', 'Probe Rated', 'rated source', author)
half = make_article('probe:half', 'Probe Half', 'half source', author)
included = make_article(
    'probe:included', 'Included Page',
    'host=%%this|title%% full=%%this|fullname%% own=%%title%% miss=%%this|nosuchvar%%', None)
host = make_article('probe:host', 'Probe Host', '[[include probe:included]]', None)

full.parent = parent
full.save()
full.authors.set([author, coauthor])
bare.authors.clear()
full.tags.set([make_tag('_default', 'Zeta'), make_tag('_default', 'alpha'), make_tag('lang', 'en')])
bare.tags.clear()

vote(full, author, 1)
vote(full, coauthor, 1)
vote(full, voter, -1)
vote(rated, author, 4)
vote(rated, voter, 5)
for index, member in enumerate(crowd):
    vote(half, member, 1 if index == 0 else -1)

stars_category, _ = Category.objects.get_or_create(name='probestars')
Settings.objects.update_or_create(
    category=stars_category,
    defaults=dict(rating_mode=Settings.RatingMode.Stars),
)
site_settings, _ = Settings.objects.get_or_create(site=Site.objects.first())

for article in (parent, full, bare, rated, half, included, host):
    freeze(article)

print('site rating_mode =', site_settings.rating_mode)
print('seeded', ', '.join(a.full_name for a in (parent, full, bare, rated, half, included, host)))
