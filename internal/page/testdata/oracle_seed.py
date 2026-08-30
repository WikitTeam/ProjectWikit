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
import itertools

from django.utils import timezone

from web.controllers import articles
from web.models.articles import Article, Category, Tag, TagsCategory, Vote
from web.models.forum import ForumCategory, ForumPost, ForumPostVersion, ForumSection, ForumThread
from web.models.notifications import UserNotification, UserNotificationMapping, UserNotificationSubscription
from web.models.settings import Settings
from web.models.site import Site
from web.models.users import User

NL = chr(10)

CREATED_AT = datetime.datetime(2021, 3, 4, 5, 6, 7, tzinfo=datetime.timezone.utc)
UPDATED_AT = datetime.datetime(2022, 7, 8, 9, 10, 11, tzinfo=datetime.timezone.utc)
POSTED_AT = datetime.datetime(2023, 9, 10, 11, 12, 13, tzinfo=datetime.timezone.utc)
POSTED_MINUTES = itertools.count()


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


def notify(user, kind, viewed):
    notification, _ = UserNotification.objects.get_or_create(type=kind, meta={'probe': kind})
    UserNotificationMapping.objects.update_or_create(
        recipient=user, notification=notification, defaults=dict(is_viewed=viewed),
    )


def comment_thread(article, user, posts):
    thread, _ = ForumThread.objects.get_or_create(article=article, defaults=dict(name=article.title))
    for i in range(posts):
        ForumPost.objects.get_or_create(
            thread=thread, name='probe post %d' % i, defaults=dict(author=user),
        )
    return thread


def forum_section(name, order, **kwargs):
    section, _ = ForumSection.objects.get_or_create(name=name)
    section.order = order
    for key, value in kwargs.items():
        setattr(section, key, value)
    section.save()
    return section


def forum_category(section, name, order, **kwargs):
    category, _ = ForumCategory.objects.get_or_create(section=section, name=name)
    category.order = order
    for key, value in kwargs.items():
        setattr(category, key, value)
    category.save()
    return category


def forum_thread(category, name, user, posts):
    thread, _ = ForumThread.objects.get_or_create(
        category=category, name=name, defaults=dict(author=user, description='%s description' % name))
    made = []
    for i in range(posts):
        post, created = ForumPost.objects.get_or_create(
            thread=thread, name='%s post %d' % (name, i), defaults=dict(author=user))
        if created:
            ForumPostVersion.objects.create(post=post, source='%s body %d' % (name, i), author=user)
        made.append(post)
    ForumThread.objects.filter(pk=thread.pk).update(created_at=CREATED_AT, updated_at=UPDATED_AT)
    for post in made:
        at = POSTED_AT + datetime.timedelta(minutes=next(POSTED_MINUTES))
        ForumPost.objects.filter(pk=post.pk).update(created_at=at, updated_at=at)
    return thread


def subscribe(user, article=None, forum_thread=None):
    UserNotificationSubscription.objects.get_or_create(
        subscriber=user, article=article, forum_thread=forum_thread,
    )


def vote(article, user, rate):
    Vote.objects.update_or_create(
        article=article, user=user,
        defaults=dict(rate=rate, date=timezone.now()),
    )


author = make_user(
    'probe-author',
    display_name='Probe Author',
    type=User.UserType.Normal,
    api_key='probe-key-author',
)
# An imported Wikidot account nobody has claimed yet. The uuid is fixed rather
# than generated so the golden stays put; the importer generates one per user.
coauthor = make_user(
    '576c0df3-8a28-4468-9770-ede851d88c67',
    display_name='Probe WD',
    wikidot_username='probe-wd-original',
    type=User.UserType.Wikidot,
    is_active=False,
    api_key='probe-key-wd',
)
voter = make_user('probevoter', type=User.UserType.Normal, api_key='probe-key-voter')
make_user(
    'probe-staff',
    display_name='Probe Staff',
    type=User.UserType.Normal,
    api_key='probe-key-staff',
    is_superuser=True,
)
crowd = [
    make_user('probecrowd%d' % i, type=User.UserType.Normal, api_key='probe-key-crowd%d' % i)
    for i in range(8)
]

parent = make_article('probe:parent', 'Probe Parent', 'parent source', author)
full = make_article('probe:full', 'Probe Full', 'full source\nsecond line', author)
bare = make_article('probe:bare', 'Probe Bare', 'bare source', None)
rated = make_article('probestars:rated', 'Probe Rated', 'rated source', author)
unrated = make_article('probestars:unrated', 'Probe Unrated', 'unrated source', author)
third = make_article('probestars:third', 'Probe Third', '[[module Rate]]', author)
quarter = make_article('probestars:quarter', 'Probe Quarter', 'quarter source', author)
unratable = make_article('probeoff:unratable', 'Probe Unratable', 'unratable source', author)
half = make_article('probe:half', 'Probe Half', 'half source', author)
included = make_article(
    'probe:included', 'Included Page',
    'host=%%this|title%% full=%%this|fullname%% own=%%title%% miss=%%this|nosuchvar%%', None)
host = make_article('probe:host', 'Probe Host', '[[include probe:included]]', None)
redirect = make_article('probe:redirect', 'Probe Redirect',
                        NL.join(['before', '[[module Redirect destination="/probe:full"]]', 'after']), None)
described = make_article('probe:described', 'Probe Described',
                         NL.join(['visible text', '[[module PageDescription]]custom description[[/module]]']), None)
imaged = make_article('probe:imaged', 'Probe Imaged',
                      '[[module PageImage src="probe:full/cover.png"]]body text', None)
tagged = make_article('probe:tagged', 'Probe Tagged', '[[module PagesByTag tag="lang:en"]]', None)
taggedplain = make_article('probe:taggedplain', 'Probe Tagged Plain',
                           '[[module PagesByTag tag="zeta"]]', None)
unknownmodule = make_article('probe:unknownmodule', 'Probe Unknown Module',
                             '[[module NoSuchModule]]', None)
# pwikit resolves a display name here and Django does not, so this page is the
# one the corpus carries the exemption for.
bydisplay = make_article('probe:bydisplay', 'Probe By Display Name',
                         '[[*user Probe WD]]', None)

listrow = NL.join([
    '[[module ListPages category="probe" order="name" perPage="3" separate="yes"]]',
    '%%index%%/%%total%% [[[%%fullname%%|%%title%%]]] %%rating%%',
    '[[/module]]',
])
listed = make_article('probe:listed', 'Probe Listed', listrow, None)
listjoined = make_article(
    'probe:listjoined', 'Probe List Joined',
    NL.join([
        '[[module ListPages category="probe" order="name" separate="no" prependline="||~ page||" appendline="end"]]',
        '||%%name%%||',
        '[[/module]]',
    ]), None)
listnowrap = make_article(
    'probe:listnowrap', 'Probe List No Wrapper',
    NL.join([
        '[[module ListPages category="probe" order="name" wrapper="no" limit="2"]]',
        '%%name%%',
        '[[/module]]',
    ]), None)
listsections = make_article(
    'probe:listsections', 'Probe List Sections',
    NL.join([
        '[[module ListPages category="probe" order="name" perPage="2"]]',
        '[[head]]',
        'top',
        '[[/head]]',
        '[[body]]',
        '%%name%%',
        '[[/body]]',
        '[[foot]]',
        'bottom',
        '[[/foot]]',
        '[[/module]]',
    ]), None)
listtags = make_article(
    'probe:listtags', 'Probe List Tags',
    NL.join([
        '[[module ListPages category="*" tags="+lang:en -zeta" order="fullname"]]',
        '%%fullname%%',
        '[[/module]]',
    ]), None)
listempty = make_article(
    'probe:listempty', 'Probe List Empty',
    NL.join([
        '[[module ListPages category="probe" name="no-such-name-at-all"]]',
        '%%name%%',
        '[[/module]]',
    ]), None)
listurl = make_article(
    'probe:listurl', 'Probe List Url',
    NL.join([
        '[[module ListPages category="probe" order="name" name="@url|probe*"]]',
        '%%name%%',
        '[[/module]]',
    ]), None)
listbyvotes = make_article(
    'probe:listbyvotes', 'Probe List By Votes',
    NL.join([
        '[[module ListPages category="*" order="votes desc" perPage="5" rating=">-10"]]',
        '%%fullname%% %%rating%% %%rating_votes%% %%popularity%%',
        '[[/module]]',
    ]), None)

full.parent = parent
full.save()
full.authors.set([author, coauthor])
bare.authors.clear()
full.tags.set([make_tag('_default', 'Zeta'), make_tag('_default', 'alpha'), make_tag('lang', 'en')])
TagsCategory.objects.filter(slug='lang').update(priority=1)
bare.tags.clear()

open_section = forum_section('Probe Open', 0, description='an open section')
hidden_section = forum_section('Probe Hidden', 1, description='hidden from the listing', is_hidden=True)
staff_section = forum_section('Probe Staff', 2, description='only staff may browse', is_hidden_for_users=True)

chat = forum_category(open_section, 'Probe Chat', 0, description='ordinary threads')
comments = forum_category(open_section, 'Probe Comments', 1, description='article comments', is_for_comments=True)
quiet = forum_category(open_section, 'Probe Quiet', 2, description='nobody has posted here')
forum_category(hidden_section, 'Probe Hidden Chat', 0, description='inside the hidden section')
forum_category(staff_section, 'Probe Staff Chat', 0, description='inside the staff section')

forum_thread(chat, 'Probe Thread', author, 3)
forum_thread(chat, 'Probe Locked Thread', voter, 1).is_locked = True
ForumThread.objects.filter(category=chat, name='Probe Locked Thread').update(is_locked=True)

thread = comment_thread(full, author, 2)
subscribe(author, article=full)
subscribe(author, forum_thread=thread)
author.preferences['qol__advanced_source_editor_enabled'] = True

notify(author, UserNotification.NotificationType.Welcome, False)
notify(author, UserNotification.NotificationType.DirectMessage, True)

vote(full, author, 1)
vote(full, coauthor, 1)
vote(full, voter, -1)
vote(rated, author, 4)
vote(rated, voter, 5)
vote(third, author, 3)
vote(third, voter, 3)
vote(third, coauthor, 4)
vote(quarter, author, 0.5)
vote(quarter, voter, 1)
for index, member in enumerate(crowd):
    vote(half, member, 1 if index == 0 else -1)

stars_category, _ = Category.objects.get_or_create(name='probestars')
Settings.objects.update_or_create(
    category=stars_category,
    defaults=dict(rating_mode=Settings.RatingMode.Stars),
)
off_category, _ = Category.objects.get_or_create(name='probeoff')
Settings.objects.update_or_create(
    category=off_category,
    defaults=dict(rating_mode=Settings.RatingMode.Disabled),
)
site_settings, _ = Settings.objects.get_or_create(site=Site.objects.first())

for article in (parent, full, bare, rated, unrated, third, quarter, unratable, half,
                included, host, redirect, described,
                imaged, tagged, taggedplain, unknownmodule, bydisplay, listed, listjoined,
                listnowrap, listsections, listtags, listempty, listurl, listbyvotes):
    freeze(article)

print('site rating_mode =', site_settings.rating_mode)
print('forum sections =', ForumSection.objects.count(), 'categories =', ForumCategory.objects.count())
print('seeded', ', '.join(a.full_name for a in (parent, full, bare, rated, unrated, third,
                                                 quarter, unratable, half, included, host,
                                                 redirect, described, imaged, tagged, taggedplain,
                                                 unknownmodule, bydisplay, listed, listjoined,
                                                 listnowrap, listsections, listtags, listempty,
                                                 listurl, listbyvotes)))
