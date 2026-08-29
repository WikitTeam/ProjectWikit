"""Print what Django answers for the forum listing's own queries.

    docker compose exec -T web python manage.py shell < internal/db/testdata/oracle_forum.py

It reads the rows oracle_seed.py creates, so run that one first. The expressions
are the ones modules/forumstart.py uses. The answers here are what
internal/db/forum_test.go asserts.
"""

from web.models.forum import ForumCategory, ForumPost, ForumSection, ForumThread


def show_sections():
    for section in ForumSection.objects.all().order_by('order', 'id'):
        print('section %s hidden=%s hidden_for_users=%s' % (
            section.name, section.is_hidden, section.is_hidden_for_users))


def show_categories():
    sections = {s.id: s.name for s in ForumSection.objects.all()}
    for category in ForumCategory.objects.all().order_by('order', 'id'):
        print('category %s section=%s for_comments=%s' % (
            category.name, sections[category.section_id], category.is_for_comments))


def show_counts(category):
    if category.is_for_comments:
        threads = ForumThread.objects.filter(article_id__isnull=False).count()
        posts = ForumPost.objects.filter(thread__article_id__isnull=False).count()
        last = ForumPost.objects.filter(thread__article_id__isnull=False).order_by('-created_at')[:1]
    else:
        threads = ForumThread.objects.filter(category=category).count()
        posts = ForumPost.objects.filter(thread__category=category).count()
        last = ForumPost.objects.filter(thread__category=category).order_by('-created_at')[:1]
    if last:
        post = last[0]
        tail = 'last=%s thread=%s article=%s' % (
            post.name, post.thread.name, post.thread.article_id is not None)
    else:
        tail = 'last=None'
    print('counts %s threads=%d posts=%d %s' % (category.name, threads, posts, tail))


show_sections()
show_categories()
for category in ForumCategory.objects.all().order_by('order', 'id'):
    show_counts(category)
