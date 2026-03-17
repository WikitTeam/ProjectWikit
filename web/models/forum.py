__all__ = [
    'ForumSection',
    'ForumCategory',
    'ForumThread',
    'ForumPost',
    'ForumPostVersion'
]

import auto_prefetch
from django.db import models
from django.db.models import Func, Value
from django.db.models.lookups import Exact
from django.contrib.auth import get_user_model

from web.models.roles import PermissionsOverrideMixin
from .articles import Article


User = get_user_model()


class ForumSection(auto_prefetch.Model, PermissionsOverrideMixin):
    class Meta(auto_prefetch.Model.Meta):
        verbose_name = '论坛分类'
        verbose_name_plural = '论坛分类'

    name = models.TextField('名称')
    description = models.TextField('描述', blank=True)
    order = models.IntegerField('排序顺序', default=0, blank=True)
    # 此设置对所有人隐藏，除非他们点击"显示隐藏"
    is_hidden = models.BooleanField('隐藏分类', default=False)
    # 此设置对版主和管理员显示，但对普通用户完全隐藏
    is_hidden_for_users = models.BooleanField('仅管理员可见', default=False)

    def __str__(self):
        return self.name
    
    def override_perms(self, user_obj, perms: set, roles=[]):
        if self.is_hidden_for_users and 'roles.view_forum_sections' in perms and 'roles.view_hidden_forum_sections' not in perms:
            perms.remove('roles.view_forum_sections')
        return super().override_perms(user_obj, perms, roles)


class ForumCategory(auto_prefetch.Model):
    class Meta(auto_prefetch.Model.Meta):
        verbose_name = '论坛版块'
        verbose_name_plural = '论坛版块'

    name = models.TextField('名称')
    description = models.TextField('描述', blank=True)
    order = models.IntegerField('排序顺序', default=0, blank=True)
    section = auto_prefetch.ForeignKey(ForumSection, on_delete=models.DO_NOTHING, verbose_name='分类')  # 待办：后续审查
    is_for_comments = models.BooleanField('在此版块显示文章评论', default=False)

    def __str__(self):
        return self.name


class ForumThread(auto_prefetch.Model, PermissionsOverrideMixin):
    class Meta(auto_prefetch.Model.Meta):
        verbose_name = '论坛主题'
        verbose_name_plural = '论坛主题'

        constraints = [
            # 逻辑：主题必须分配了'article'或'category'
            # 需要postgres >9.2
            models.CheckConstraint(
                name='%(app_label)s_%(class)s_category_or_article',
                check=Exact(
                    lhs=Func('article_id', 'category_id', function='num_nonnulls', output_field=models.IntegerField()),
                    rhs=Value(1),
                ),
            )
        ]

    roles_override_pipeline = ['article']

    name = models.TextField('标题')
    description = models.TextField('描述', blank=True)
    created_at = models.DateTimeField('创建时间', auto_now_add=True)
    updated_at = models.DateTimeField('修改时间', auto_now_add=True)
    category = auto_prefetch.ForeignKey(ForumCategory, on_delete=models.DO_NOTHING, null=True, verbose_name='版块（如果是主题）')  # 待办：后续审查
    article = auto_prefetch.ForeignKey(Article, on_delete=models.CASCADE, null=True, verbose_name='文章（如果是评论）')
    author = auto_prefetch.ForeignKey(User, on_delete=models.SET_NULL, null=True, verbose_name='作者')
    is_pinned = models.BooleanField('置顶', default=False)
    is_locked = models.BooleanField('已锁定', default=False)

    def override_perms(self, user_obj, perms: set, roles=[]):
        if user_obj == self.author:
            perms.add('roles.edit_forum_threads')
        if not user_obj.is_anonymous and not user_obj.is_forum_active or self.is_locked and 'roles.lock_forum_threads' not in perms:
            perms_to_revoke = {'roles.comment_articles', 'roles.create_forum_posts', 'roles.edit_forum_posts', 'roles.delete_forum_posts', 'roles.edit_forum_threads', 'roles.pin_forum_threads', 'roles.move_forum_threads'}
            perms = perms.difference(perms_to_revoke)
        return super().override_perms(user_obj, perms, roles)


class ForumPost(auto_prefetch.Model, PermissionsOverrideMixin):
    class Meta(auto_prefetch.Model.Meta):
        verbose_name = '论坛帖子'
        verbose_name_plural = '论坛帖子'

    perms_override_pipeline = ['thread']

    name = models.TextField('标题', blank=True)
    created_at = models.DateTimeField('创建时间', auto_now_add=True, )
    updated_at = models.DateTimeField('修改时间', auto_now_add=True)
    author = auto_prefetch.ForeignKey(User, on_delete=models.SET_NULL, null=True, verbose_name='作者')
    reply_to = auto_prefetch.ForeignKey('ForumPost', on_delete=models.SET_NULL, null=True, verbose_name='回复的评论')
    thread = auto_prefetch.ForeignKey(ForumThread, on_delete=models.CASCADE, verbose_name='主题')

    def override_perms(self, user_obj, perms: set, roles=[]):
        if user_obj == self.author and user_obj.has_perm('roles.create_forum_posts'):
            perms.add('roles.edit_forum_posts')
        return super().override_perms(user_obj, perms, roles)


class ForumPostVersion(auto_prefetch.Model):
    class Meta(auto_prefetch.Model.Meta):
        verbose_name = '论坛帖子版本'
        verbose_name_plural = '论坛帖子版本'

    post = auto_prefetch.ForeignKey(ForumPost, on_delete=models.CASCADE, verbose_name='帖子')
    source = models.TextField('帖子内容')
    created_at = models.DateTimeField('创建时间', auto_now_add=True)
    author = auto_prefetch.ForeignKey(User, on_delete=models.SET_NULL, null=True, verbose_name='编辑作者')