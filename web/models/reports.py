__all__ = [
    'UserReport',
]

import auto_prefetch

from django.contrib.auth import get_user_model
from django.db import models


User = get_user_model()


class UserReport(auto_prefetch.Model):
    class Meta(auto_prefetch.Model.Meta):
        verbose_name = '用户检举'
        verbose_name_plural = '用户检举'
        indexes = [
            models.Index(fields=['status', 'created_at']),
            models.Index(fields=['reported', 'status']),
        ]

    class ReportStatus(models.TextChoices):
        Pending = ('pending', '待处理')
        Reviewed = ('reviewed', '已处理')
        Dismissed = ('dismissed', '已驳回')

    reporter = auto_prefetch.ForeignKey(
        User, on_delete=models.SET_NULL, null=True, blank=True,
        verbose_name='举报人', related_name='reports_submitted',
    )
    reported = auto_prefetch.ForeignKey(
        User, on_delete=models.SET_NULL, null=True, blank=True,
        verbose_name='被举报人', related_name='reports_received',
    )
    reason = models.TextField('举报理由', blank=False, null=False)
    reported_messages = models.JSONField('举报的消息（快照）', default=list, blank=True)
    status = models.TextField(
        '状态', choices=ReportStatus.choices, default=ReportStatus.Pending,
        blank=False, null=False,
    )
    admin_notes = models.TextField('处理备注', blank=True)
    created_at = models.DateTimeField('提交时间', auto_now_add=True, blank=False, null=False)
    reviewed_at = models.DateTimeField('处理时间', null=True, blank=True)
    reviewed_by = auto_prefetch.ForeignKey(
        User, on_delete=models.SET_NULL, null=True, blank=True,
        verbose_name='处理人', related_name='reports_reviewed',
    )

    def __str__(self):
        return f'#{self.pk} {self.reporter or "(已删除)"}→{self.reported or "(已删除)"}'
