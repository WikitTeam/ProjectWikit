from datetime import datetime, timezone

from modules import ModuleError
from renderer import RenderContext, render_template_from_string
import json

from renderer.utils import render_user_to_json
from web.controllers import articles, notifications
from web.models.forum import ForumCategory, ForumThread, ForumPost, ForumPostVersion


def has_content():
    return False


def allow_api():
    return True


def render(context: RenderContext, params):
    context.title = '创建主题'

    c = context.path_params.get('c')
    try:
        c = int(c)
        category = ForumCategory.objects.filter(id=c)
        category = category[0] if category else None
    except:
        category = None

    if category is None:
        context.status = 404
        raise ModuleError('未找到版块 "%s"' % c)

    if not context.user.has_perm('roles.create_forum_threads', category):
        raise ModuleError('权限不足，无法创建主题')

    num_threads = ForumThread.objects.filter(category=category).count()
    num_posts = ForumPost.objects.filter(thread__category=category).count()

    canonical_url = '/forum/c-%d/%s' % (category.id, articles.normalize_article_name(category.name))

    editor_config = {
        'categoryId': category.id,
        'user': render_user_to_json(context.user),
        'cancelUrl': canonical_url
    }

    return render_template_from_string(
        """
        <div class="forum-new-thread-box">
            <div class="forum-breadcrumbs">
                <a href="/forum/start">论坛</a>
                &raquo;
                <a href="{{ canonical_url }}">{{ breadcrumb }}</a>
                &raquo;
                创建主题
            </div>
            <div class="description well">
                <div class="statistics">
                    主题数：{{ num_threads }}
                    <br>
                    帖子数：{{ num_posts }}
                </div>
                {{ description }}
            </div>
        </div>
        <!-- 希望没人试图给论坛页面加样式 -->
        <div class="w-forum-new-thread" data-config="{{ editor_config }}"></div>
        """,
        breadcrumb='%s / %s' % (category.section.name, category.name),
        description=category.description,
        num_threads=num_threads,
        num_posts=num_posts,
        canonical_url=canonical_url,
        editor_config=json.dumps(editor_config)
    )


def api_submit(context, params):
    title = (params.get('name') or '').strip()
    description = (params.get('description') or '').strip()[:1000]
    source = (params.get('source') or '').strip()

    if not title:
        raise ModuleError('未指定主题标题')

    if not source:
        raise ModuleError('未提供首帖内容')

    c = params.get('categoryid')
    try:
        c = int(c)
        category = ForumCategory.objects.filter(id=c)
        category = category[0] if category else None
    except:
        category = None

    if category is None:
        context.status = 404
        raise ModuleError('版块未找到或未指定')

    if not context.user.has_perm('roles.create_forum_threads', category):
        raise ModuleError('权限不足，无法创建主题')

    thread = ForumThread(category=category, name=title, description=description, author=context.user)
    thread.save()

    first_post = ForumPost(thread=thread, author=context.user, name=title)
    first_post.save()
    first_post.updated_at = datetime.now(timezone.utc)
    first_post.save()

    first_post_content = ForumPostVersion(post=first_post, source=source, author=context.user)
    first_post_content.save()

    notifications.subscribe_to_notifications(subscriber=context.user, forum_thread=thread)

    return {'url': '/forum/t-%d/%s' % (thread.id, articles.normalize_article_name(title))}