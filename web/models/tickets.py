__all__ = [
    'UserTicket',
    'SupportTicket',
    'MembershipApplication',
]

import auto_prefetch

from django.contrib.auth import get_user_model
from django.db import models

from .roles import Role


User = get_user_model()


class UserTicket(auto_prefetch.Model):
    class Meta(auto_prefetch.Model.Meta):
        verbose_name = '用户提交'
        verbose_name_plural = '用户提交'
        indexes = [
            models.Index(fields=['kind', 'status', 'created_at']),
            models.Index(fields=['author', 'kind']),
        ]

    class Kind(models.TextChoices):
        Ticket = ('ticket', '工单')
        MembershipApply = ('membershipapply', '入组申请')

    class Status(models.TextChoices):
        Pending = ('pending', '待处理')
        Approved = ('approved', '已通过')
        Rejected = ('rejected', '已驳回')
        Closed = ('closed', '已关闭')

    kind = models.TextField('类型', choices=Kind.choices, default=Kind.Ticket, null=False, blank=False)
    author = auto_prefetch.ForeignKey(
        User, on_delete=models.SET_NULL, null=True, blank=True,
        verbose_name='提交人', related_name='tickets_submitted',
    )
    subject = models.TextField('标题', blank=True)
    body = models.TextField('正文', blank=False, null=False)
    source_page = models.TextField('提交页面', blank=True)

    status = models.TextField(
        '状态', choices=Status.choices, default=Status.Pending, null=False, blank=False,
    )
    granted_role = auto_prefetch.ForeignKey(
        Role, on_delete=models.SET_NULL, null=True, blank=True,
        verbose_name='授予的角色', related_name='+',
    )
    admin_notes = models.TextField('处理备注', blank=True)

    created_at = models.DateTimeField('提交时间', auto_now_add=True, null=False, blank=False)
    reviewed_at = models.DateTimeField('处理时间', null=True, blank=True)
    reviewed_by = auto_prefetch.ForeignKey(
        User, on_delete=models.SET_NULL, null=True, blank=True,
        verbose_name='处理人', related_name='tickets_reviewed',
    )

    def __str__(self):
        return f'#{self.pk} {self.author or "(已删除)"} {self.subject or self.get_kind_display()}'


class SupportTicketManager(auto_prefetch.Manager):
    def get_queryset(self):
        return super().get_queryset().filter(kind=UserTicket.Kind.Ticket)


class SupportTicket(UserTicket):
    class Meta(auto_prefetch.Model.Meta):
        proxy = True
        verbose_name = '用户工单'
        verbose_name_plural = '用户工单'

    objects = SupportTicketManager()


class MembershipApplicationManager(auto_prefetch.Manager):
    def get_queryset(self):
        return super().get_queryset().filter(kind=UserTicket.Kind.MembershipApply)


class MembershipApplication(UserTicket):
    class Meta(auto_prefetch.Model.Meta):
        proxy = True
        verbose_name = '申请书'
        verbose_name_plural = '申请书'

    objects = MembershipApplicationManager()
