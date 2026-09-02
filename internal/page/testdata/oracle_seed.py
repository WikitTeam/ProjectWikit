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

from django.contrib.auth.models import AnonymousUser
from django.utils import timezone

from web import threadvars
from web.controllers import articles
from web.models.articles import (Article, ArticleLogEntry, ArticleVersion, Category, ExternalLink,
                                 Tag, TagsCategory, Vote)
from web.models.forum import ForumCategory, ForumPost, ForumPostVersion, ForumSection, ForumThread
from web.models.notifications import UserNotification, UserNotificationMapping, UserNotificationSubscription
from web.models.settings import Settings
from web.models.site import Site
from web.models.users import User

NL = chr(10)

CREATED_AT = datetime.datetime(2021, 3, 4, 5, 6, 7, tzinfo=datetime.timezone.utc)
UPDATED_AT = datetime.datetime(2022, 7, 8, 9, 10, 11, tzinfo=datetime.timezone.utc)
POSTED_AT = datetime.datetime(2023, 9, 10, 11, 12, 13, tzinfo=datetime.timezone.utc)
LOGGED_AT = datetime.datetime(2024, 11, 12, 13, 14, 15, tzinfo=datetime.timezone.utc)
POSTED_MINUTES = itertools.count()
THREAD_MINUTES = itertools.count()


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


def comment_thread(article, authors):
    thread, _ = ForumThread.objects.get_or_create(article=article, defaults=dict(name=article.title))
    for i, user in enumerate(authors):
        post, _ = ForumPost.objects.get_or_create(
            thread=thread, name='probe post %d' % i, defaults=dict(author=user),
        )
        ForumPostVersion.objects.get_or_create(
            post=post, defaults=dict(source='probe comment %d' % i, author=user))
        at = POSTED_AT + datetime.timedelta(minutes=next(POSTED_MINUTES))
        ForumPost.objects.filter(pk=post.pk).update(created_at=at, updated_at=at)
    return thread


def forum_reply(parent, name, user, source):
    post, _ = ForumPost.objects.get_or_create(
        thread=parent.thread, reply_to=parent, name=name, defaults=dict(author=user))
    ForumPostVersion.objects.get_or_create(post=post, defaults=dict(source=source, author=user))
    at = POSTED_AT + datetime.timedelta(minutes=next(POSTED_MINUTES))
    ForumPost.objects.filter(pk=post.pk).update(created_at=at, updated_at=at)
    return post


def root_posts(thread):
    return list(ForumPost.objects.filter(thread=thread, reply_to__isnull=True).order_by('created_at'))


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


def forum_thread(category, name, user, posts, pinned=False):
    thread, _ = ForumThread.objects.get_or_create(
        category=category, name=name, defaults=dict(author=user, description='%s description' % name))
    made = []
    for i in range(posts):
        post, created = ForumPost.objects.get_or_create(
            thread=thread, name='%s post %d' % (name, i), defaults=dict(author=user))
        if created:
            ForumPostVersion.objects.create(post=post, source='%s body %d' % (name, i), author=user)
        made.append(post)
    minutes = next(THREAD_MINUTES)
    ForumThread.objects.filter(pk=thread.pk).update(
        created_at=CREATED_AT + datetime.timedelta(minutes=minutes),
        updated_at=UPDATED_AT - datetime.timedelta(minutes=minutes),
        is_pinned=pinned)
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
styled = make_article('probecss:styled', 'Probe Styled',
                      NL.join(['[[module CSS]]',
                               '#page-content { color : red ; }',
                               '@media (max-width: 767px) { #main { padding : 0 ; } }',
                               '[[/module]]',
                               'styled body']), None)
styledhead = make_article('probecss:styledhead', 'Probe Styled Head',
                          NL.join(['[[module CSS head="true"]]',
                                   '.a { color: #ff0000 }',
                                   '[[/module]]',
                                   'head styled body']), None)
changes = make_article('probe:changes', 'Probe Changes',
                       '[[module SiteChanges]]', author)
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

# The tag cloud sizes every tag against the busiest one, so the counts here have
# to differ. A tag whose name starts with _ never reaches the cloud, and the
# topic category is the one whose name is not its slug.
rated.tags.set([make_tag('_default', 'alpha'), make_tag('topic', 'scp')])
unrated.tags.set([make_tag('_default', 'alpha'), make_tag('_default', '_staff')])
quarter.tags.set([make_tag('_default', 'alpha'), make_tag('topic', 'scp')])
third.tags.set([make_tag('_default', 'Zeta'), make_tag('_default', '_staff')])
# aaa sorts before alpha by bare name and after it by full name, which is what
# tells apart the two orders the cloud could be using.
unratable.tags.set([make_tag('topic', 'scp'), make_tag('lang', 'aaa')])
TagsCategory.objects.filter(slug='topic').update(
    priority=2, name='Probe Topic', description='what a page is about')

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
forum_thread(chat, 'Probe Pinned Thread', voter, 2, pinned=True)

busy = forum_category(open_section, 'Probe Busy', 3, description='enough threads for a second page')
for index in range(21):
    forum_thread(busy, 'Probe Busy %02d' % index, author, 1)

comment_thread(rated, [author, voter])
thread = comment_thread(full, [author, author])
subscribe(author, article=full)
subscribe(author, forum_thread=thread)

talk = forum_category(open_section, 'Probe Talk', 4, description='threads that carry replies')

deep = forum_thread(talk, 'Probe Deep Thread', author, 2)
deep_roots = root_posts(deep)
reply = forum_reply(deep_roots[0], 'Probe Deep reply 0', voter,
                    'reply naming @probe-author and @nobody-at-all')
forum_reply(reply, 'Probe Deep reply 0 0', author, 'a reply to a reply')
forum_reply(deep_roots[0], 'Probe Deep reply 1', None, 'a reply the site itself made')
forum_reply(deep_roots[0], '   ', voter, 'a reply whose title is only spaces')

edited = deep_roots[1]
ForumPost.objects.filter(pk=edited.pk).update(
    updated_at=edited.created_at + datetime.timedelta(minutes=5))
ForumPostVersion.objects.get_or_create(
    post=edited, source='Probe Deep Thread body 1 edited', defaults=dict(author=voter))

forum_thread(talk, 'Probe Long Thread', voter, 12)

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

changes_comments, _ = ForumThread.objects.get_or_create(article=changes)
ForumThread.objects.filter(pk=changes_comments.pk).update(
    created_at=CREATED_AT, updated_at=UPDATED_AT)

LOG_TYPE = ArticleLogEntry.LogEntryType
LOG_REVS = itertools.count(1)

ArticleLogEntry.objects.filter(article=changes).exclude(type=LOG_TYPE.New).delete()


def log_entry(kind, meta, user=author, comment=''):
    ArticleLogEntry.objects.create(
        article=changes, user=user, type=kind, meta=meta,
        comment=comment, rev_number=next(LOG_REVS))


log_entry(LOG_TYPE.Source, {'version_id': 0}, comment='a source edit')
log_entry(LOG_TYPE.Source, {'version_id': 0}, comment='   ')
log_entry(LOG_TYPE.Source, {'version_id': 0}, user=None)
log_entry(LOG_TYPE.Source, {'version_id': 0}, user=coauthor)
log_entry(LOG_TYPE.Source, {'version_id': 0}, user=voter)
log_entry(LOG_TYPE.Title, {'prev_title': 'Probe "Old" <b>', 'title': 'Probe Changes'})
log_entry(LOG_TYPE.Name, {'prev_name': 'probe:was-here', 'name': 'probe:changes'})
log_entry(LOG_TYPE.Tags, {'added_tags': [{'id': 1, 'name': 'alpha'}], 'removed_tags': []})
log_entry(LOG_TYPE.Tags, {'added_tags': [], 'removed_tags': [{'id': 2, 'name': 'lang:en'}]})
log_entry(LOG_TYPE.Tags, {'added_tags': [{'id': 1, 'name': 'alpha'}, {'id': 3, 'name': 'Zeta'}],
                          'removed_tags': [{'id': 2, 'name': 'lang:en'}]})
log_entry(LOG_TYPE.Tags, {})
log_entry(LOG_TYPE.Parent, {'prev_parent': None, 'parent': 'probe:parent'})
log_entry(LOG_TYPE.Parent, {'prev_parent': 'probe:parent', 'parent': None})
log_entry(LOG_TYPE.Parent, {'prev_parent': 'probe:parent', 'parent': 'probe:full'})
log_entry(LOG_TYPE.Parent, {'prev_parent': None, 'parent': None})
log_entry(LOG_TYPE.FileAdded, {'name': 'cover.png', 'id': 1})
log_entry(LOG_TYPE.FileDeleted, {'name': 'cover.png', 'id': 1})
log_entry(LOG_TYPE.FileRenamed, {'prev_name': 'cover.png', 'name': 'banner.png'})
log_entry(LOG_TYPE.VotesDeleted,
          {'rating_mode': 'updown', 'rating': 3, 'votes_count': 5, 'popularity': 60})
log_entry(LOG_TYPE.VotesDeleted,
          {'rating_mode': 'updown', 'rating': -3.7, 'votes_count': 9, 'popularity': 11})
log_entry(LOG_TYPE.VotesDeleted,
          {'rating_mode': 'stars', 'rating': 4.25, 'votes_count': 4, 'popularity': 75})
log_entry(LOG_TYPE.VotesDeleted,
          {'rating_mode': 'disabled', 'rating': 0, 'votes_count': 0, 'popularity': 0})
log_entry(LOG_TYPE.Authorship, {'added_authors': [author.id], 'removed_authors': []})
log_entry(LOG_TYPE.Authorship, {'added_authors': [author.id, voter.id], 'removed_authors': []})
log_entry(LOG_TYPE.Authorship, {'added_authors': [], 'removed_authors': [coauthor.id]})
log_entry(LOG_TYPE.Authorship, {'added_authors': [voter.id], 'removed_authors': [coauthor.id]})
log_entry(LOG_TYPE.Authorship, {})
log_entry(LOG_TYPE.Wikidot, {}, comment='imported from wikidot')
log_entry(LOG_TYPE.Revert, {'rev_number': 2, 'subtypes': ['source', 'title']})
log_entry(LOG_TYPE.Revert, {'rev_number': 0, 'subtypes': []})
log_entry(LOG_TYPE.Revert, {'rev_number': 1})

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
                imaged, tagged, taggedplain, unknownmodule, bydisplay, styled, styledhead, changes, listed, listjoined,
                listnowrap, listsections, listtags, listempty, listurl, listbyvotes):
    freeze(article)

for offset, entry_id in enumerate(ArticleLogEntry.objects.order_by('id').values_list('id', flat=True)):
    ArticleLogEntry.objects.filter(pk=entry_id).update(
        created_at=LOGGED_AT + datetime.timedelta(minutes=offset))

def relink(article, source):
    version = articles.get_latest_version(article)
    ArticleVersion.objects.filter(pk=version.pk).update(source=source)


# The wanted-pages list reads the link table, and nothing rebuilds that table on
# save, so the seed points a few pages at names that do not exist and then
# rebuilds it by hand.
relink(unrated, NL.join([
    'unrated source',
    '[[[probe:no-such-one|first]]] [[[probe:no-such-two]]] [[[wanted:alpha|alpha]]]',
]))
relink(quarter, NL.join([
    'quarter source',
    '[[[wanted:beta]]] [[[probe:no-such-one]]]',
]))
relink(unratable, NL.join([
    'unratable source',
    '[[[wanted:gamma]]] [[[probe:full|a page that exists]]]',
    '[[code type="css"]]',
    '.probe-code { color : red ; }',
    '[[/code]]',
    '[[code]]',
    'plain probe code',
    '[[/code]]',
    '[[html external="true"]]',
    '<b>probe html</b>',
    '[[/html]]',
    '[[module CSS]]',
    '.probe-theme { color : {$probecolor} ; }',
    '[[/module]]',
    '[[noinclude]]',
    '[[module CSS]]',
    '.probe-noinclude { color : green ; }',
    '[[/module]]',
    '[[/noinclude]]',
]))

with threadvars.context():
    threadvars.put('current_site', Site.objects.first())
    threadvars.put('current_user', AnonymousUser())
    for one in Article.objects.all():
        latest = articles.get_latest_version(one)
        if latest:
            articles.refresh_article_links(latest)

print('external links =', ExternalLink.objects.count())
print('site rating_mode =', site_settings.rating_mode)
print('forum sections =', ForumSection.objects.count(), 'categories =', ForumCategory.objects.count())
print('log entries =', ArticleLogEntry.objects.count())
print('seeded', ', '.join(a.full_name for a in (parent, full, bare, rated, unrated, third,
                                                 quarter, unratable, half, included, host,
                                                 redirect, described, imaged, tagged, taggedplain,
                                                 unknownmodule, bydisplay, styled, styledhead, changes,
                                                 listed, listjoined,
                                                 listnowrap, listsections, listtags, listempty,
                                                 listurl, listbyvotes)))
