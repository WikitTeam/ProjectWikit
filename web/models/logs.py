__all__ = [
    'ActionLogEntry'
]

import auto_prefetch
from django.db import models
from django.contrib.auth import get_user_model


User = get_user_model()


class ActionLogEntry(auto_prefetch.Model):
    class Meta(auto_prefetch.Model.Meta):
        verbose_name = '操作记录'
        verbose_name_plural = '操作记录'

        indexes = [
            models.Index(
                fields=['user', 'origin_ip'],
                name='user_origin_ip_idx'
            ),
        ]

    class ActionType(models.TextChoices):
        Vote = ('vote', '评分')
        NewArticle = ('create_article', '页面已创建')
        EditArticle = ('edit_article', '页面已编辑')
        RemoveArticle = ('remove_article', '页面已删除')
        NewForumPost = ('add_forum_post', '新论坛帖子')
        EditForumPost = ('edit_forum_post', '论坛帖子已修改')
        RemoveForumPost = ('remove_forum_post', '论坛帖子已删除')
        ChangeProfileInfo = ('change_profile_info', '个人资料信息已修改')

    user = auto_prefetch.ForeignKey(User, verbose_name='用户', on_delete=models.DO_NOTHING)
    stale_username = models.TextField(verbose_name='操作时的用户名')
    type = models.TextField(choices=ActionType.choices, verbose_name='类型')
    meta = models.JSONField(default=dict, blank=True)
    created_at = models.DateTimeField(auto_now_add=True, verbose_name='时间')
    origin_ip = models.GenericIPAddressField(verbose_name='IP地址', blank=True, null=True)