__all__ = [
    'DirectMessage',
    'DirectMessageBlock',
]

import auto_prefetch

from django.contrib.auth import get_user_model
from django.db import models


User = get_user_model()


class DirectMessage(auto_prefetch.Model):
    class Meta(auto_prefetch.Model.Meta):
        verbose_name = '私信'
        verbose_name_plural = '私信'
        indexes = [
            models.Index(fields=['recipient', 'is_read']),
            models.Index(fields=['sender', 'recipient', 'created_at']),
            models.Index(fields=['recipient', 'sender', 'created_at']),
        ]

    PREVIEW_MAX_SIZE = 150

    sender = auto_prefetch.ForeignKey(User, on_delete=models.CASCADE, verbose_name='发送者', related_name='sent_direct_messages')
    recipient = auto_prefetch.ForeignKey(User, on_delete=models.CASCADE, verbose_name='接收者', related_name='received_direct_messages')
    body = models.TextField('内容', blank=False, null=False)
    created_at = models.DateTimeField('发送时间', auto_now_add=True, blank=False, null=False)
    is_read = models.BooleanField('已读', default=False, blank=False, null=False)

    def preview(self) -> str:
        if len(self.body) <= self.PREVIEW_MAX_SIZE:
            return self.body
        return self.body[:self.PREVIEW_MAX_SIZE] + '…'


class DirectMessageBlock(auto_prefetch.Model):
    class Meta(auto_prefetch.Model.Meta):
        verbose_name = '私信拉黑'
        verbose_name_plural = '私信拉黑'
        constraints = [
            models.UniqueConstraint(fields=['blocker', 'blocked'], name='dm_block_uniqueness'),
        ]

    blocker = auto_prefetch.ForeignKey(User, on_delete=models.CASCADE, verbose_name='拉黑者', related_name='dm_blocks_out')
    blocked = auto_prefetch.ForeignKey(User, on_delete=models.CASCADE, verbose_name='被拉黑者', related_name='dm_blocks_in')
    created_at = models.DateTimeField('创建时间', auto_now_add=True, blank=False, null=False)
