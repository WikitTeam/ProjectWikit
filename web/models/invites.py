__all__ = [
    'InviteLink',
]

import auto_prefetch

from django.contrib.auth import get_user_model
from django.db import models


User = get_user_model()


class InviteLink(auto_prefetch.Model):
    class Meta(auto_prefetch.Model.Meta):
        verbose_name = '邀请链接'
        verbose_name_plural = '邀请链接'
        ordering = ['-created_at']
        indexes = [
            models.Index(fields=['kind', 'activated_at']),
            models.Index(fields=['token']),
        ]

    class Kind(models.TextChoices):
        Register = ('register', '邀请注册')
        Claim = ('claim', '认领')

    class Delivery(models.TextChoices):
        Link = ('link', '手动发链接')
        Email = ('email', '邮件发送')

    kind = models.TextField('类型', choices=Kind.choices, default=Kind.Register, null=False, blank=False)
    delivery = models.TextField('发放方式', choices=Delivery.choices, default=Delivery.Link, null=False, blank=False)

    target = auto_prefetch.ForeignKey(
        User, on_delete=models.SET_NULL, null=True, blank=True,
        verbose_name='目标账号', related_name='invite_links',
    )
    # 目标账号可能被删掉，而这一行还要说得清它当初是发给谁的。
    email = models.TextField('邮箱', blank=True)
    wikidot_username = models.TextField('Wikidot 用户名', blank=True)

    token = models.TextField('令牌', null=False, blank=False)
    uidb64 = models.TextField('账号编码', null=False, blank=False)

    created_at = models.DateTimeField('创建时间', auto_now_add=True, null=False, blank=False)
    created_by = auto_prefetch.ForeignKey(
        User, on_delete=models.SET_NULL, null=True, blank=True,
        verbose_name='创建人', related_name='invite_links_created',
    )

    activated_at = models.DateTimeField('激活时间', null=True, blank=True)
    activated_username = models.TextField('激活后用户名', blank=True)

    @property
    def path(self):
        return '/-/accept/%s/%s' % (self.uidb64, self.token)

    @property
    def is_activated(self):
        return self.activated_at is not None

    # 记的是身份用户名而不是 __str__ 的显示名，因为这一列的用处是回头去找那个账号。
    @classmethod
    def mark_activated(cls, token, user, when):
        cls.objects.filter(token=token, activated_at__isnull=True).update(
            activated_at=when,
            activated_username=user.username,
        )

    def __str__(self):
        return '#%s %s → %s' % (self.pk, self.get_kind_display(), self.email or self.wikidot_username or self.target)
