__all__ = [
    'UserNotification',
    'UserNotificationMapping',
    'UserNotificationSubscription'
]

import auto_prefetch

from django.db import models
from django.contrib.auth import get_user_model

from web.models.articles import Article
from web.models.forum import ForumThread


User = get_user_model()


class UserNotification(auto_prefetch.Model):
    class Meta(auto_prefetch.Model.Meta):
        verbose_name = '通知'
        verbose_name_plural = '通知'

    POST_REPLY_TTL = 100
    POST_PREVIEW_MAX_SIZE = 150

    class NotificationType(models.TextChoices):
        Welcome = ('welcome', '欢迎消息')
        NewPostReply = ('new_post_reply', '帖子回复')
        NewThreadPost = ('new_thread_post', '新帖子')
        NewArticleRevision = ('new_article_revision', '文章编辑')
        ForumMention = ('forum_mention', '论坛提及')

    type = models.TextField('通知类型', choices=NotificationType.choices, blank=False, null=False)
    meta = models.JSONField('元数据', default=dict, blank=True, null=False)
    created_at = models.DateTimeField('发送日期', auto_now_add=True, blank=False, null=False)


class UserNotificationMapping(auto_prefetch.Model):
    recipient = auto_prefetch.ForeignKey(User, on_delete=models.CASCADE, blank=False, null=False)
    notification = auto_prefetch.ForeignKey(UserNotification, on_delete=models.CASCADE)
    is_viewed = models.BooleanField(blank=False, null=False, default=False)


class UserNotificationSubscription(auto_prefetch.Model):
    class Meta(auto_prefetch.Model.Meta):
        verbose_name = '通知订阅'
        verbose_name_plural = '通知订阅'

    subscriber = auto_prefetch.ForeignKey(User, on_delete=models.CASCADE, verbose_name='订阅者', blank=False, null=False)
    article = auto_prefetch.ForeignKey(Article, on_delete=models.CASCADE, verbose_name='文章', blank=True, null=True)
    forum_thread = auto_prefetch.ForeignKey(ForumThread, on_delete=models.CASCADE, verbose_name='论坛主题', blank=True, null=True)