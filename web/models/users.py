__all__ = [
    'User',
    'UsedToken'
]

import auto_prefetch
import re
import unicodedata

from datetime import datetime
from zoneinfo import ZoneInfo

from django.core.exceptions import ValidationError
from django.core.validators import RegexValidator
from django.contrib.auth.hashers import make_password
from django.contrib.auth.models import AbstractUser, AnonymousUser
from django.db.models.functions import Lower
from django.db import models
from django.conf import settings

import web.fields
from web.models.roles import RolesMixin


DISPLAY_NAME_MAX_LENGTH = 50

FALLBACK_USERNAME_PREFIX = 'wkt-uid'

RESERVED_USERNAME_RE = re.compile(r'^%s-\d+(-\d+)*\Z' % FALLBACK_USERNAME_PREFIX)

# Cc 控制符 Cf 零宽与双向覆盖 Cs 代理对 Co 私用区 Cn 未分配 Zl/Zp 行段分隔
_BANNED_CHAR_CATEGORIES = frozenset({'Cc', 'Cf', 'Cs', 'Co', 'Cn', 'Zl', 'Zp'})


class StrictUsernameValidator(RegexValidator):
    # 不加 re.ASCII，中日韩身份名要能过
    regex = r'^[^\W_]+(-[^\W_]+)*\Z'
    message = '用户名只能由字母、数字以及连接它们的单个连字符组成。'


def canonicalize_username(name: str) -> str:
    # 与 Wikidot unix name 同构；一个字符都折不出来时返回空串，由调用方兜底
    normalized = unicodedata.normalize('NFKC', name or '').lower()
    return re.sub(r'[\W_]+', '-', normalized).strip('-')


def normalize_display_name(name: str) -> str:
    # NFKC 把全角和 NBSP 收敛掉，否则肉眼同名的输入会算成两个人
    normalized = unicodedata.normalize('NFKC', name or '')
    return re.sub(r'\s+', ' ', normalized).strip()


def validate_display_name(name: str):
    # . @ # 之类最终都会折成 -，没必要在这里列举允许的标点
    if not name:
        raise ValidationError('显示名不能为空。')
    if len(name) > DISPLAY_NAME_MAX_LENGTH:
        raise ValidationError('显示名不能超过 %d 个字符。' % DISPLAY_NAME_MAX_LENGTH)
    for char in name:
        category = unicodedata.category(char)
        if category in _BANNED_CHAR_CATEGORIES:
            raise ValidationError('显示名不能包含控制符、零宽字符等不可见字符。')
        if category == 'Zs' and char != ' ':
            raise ValidationError('显示名中的空格只能是普通空格。')
    if unicodedata.category(name[0]) in ('Mn', 'Mc'):
        raise ValidationError('显示名不能以组合记号开头。')


class StrictDisplayNameValidator:
    def __call__(self, value):
        validate_display_name(value)

    def __eq__(self, other):
        return isinstance(other, StrictDisplayNameValidator)

    def deconstruct(self):
        return ('web.models.users.StrictDisplayNameValidator', [], {})

class CSSValueValidator(RegexValidator):
    regex = r'^[^;\n\r]+\Z'
    message = 'CSS 值不能包含 ";" 或换行符。'
    flags = re.ASCII


class ExtendedAnonymousUser(AnonymousUser):
    def get_avatar(self, default=None):
        return default or settings.DEFAULT_AVATAR

class User(AbstractUser, RolesMixin):
    class Meta(RolesMixin.Meta):
        verbose_name = '用户'
        verbose_name_plural = '用户列表'

        constraints = [
            models.UniqueConstraint(Lower('email'), name='user_email_ci_uniqueness',
            condition=models.Q(email__isnull=False) & ~models.Q(email=''))
        ]

        abstract = False

    class UserType(models.TextChoices):
        Normal = ('normal', '普通用户')
        Wikidot = ('wikidot', 'Wikidot 用户')
        System = ('system', '系统用户')
        Bot = ('bot', '机器人')

    username = web.fields.CITextField(
        max_length=150, validators=[StrictUsernameValidator()], unique=True,
        verbose_name='用户名',
        error_messages={
            'unique': '用户名已存在',
        },
    )

    wikidot_username = web.fields.CITextField('Wikidot用户名', unique=True, max_length=150, validators=[StrictUsernameValidator()], null=True, blank=False)

    display_name = models.CharField('显示名', max_length=150, null=True, blank=True, validators=[StrictDisplayNameValidator()])

    type = models.TextField('用户类型', choices=UserType.choices, default=UserType.Normal)

    avatar = models.ImageField('头像', null=True, blank=True, upload_to='-/users')
    bio = models.TextField('个人简介', blank=True)

    api_key = models.CharField('API密钥', unique=True, blank=True, null=True, max_length=255)

    is_forum_active = models.BooleanField('论坛权限已启用', default=True)
    forum_inactive_until = models.DateTimeField('论坛权限禁用至', null=True)

    can_send_direct_messages = models.BooleanField('允许发送私信', default=True)

    is_active = models.BooleanField('已启用', default=True)
    inactive_until = models.DateTimeField('禁用至', null=True)

    is_staff = RolesMixin.is_staff # type: ignore

    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)
        if self.inactive_until:
            self.is_active = False
        if self.inactive_until and not self.is_active and datetime.now(ZoneInfo('UTC')) > self.inactive_until:
            self.inactive_until = None
            self.is_active = True
        if self.forum_inactive_until:
            self.is_forum_active = False
        if self.forum_inactive_until and not self.is_forum_active and datetime.now(ZoneInfo('UTC')) > self.forum_inactive_until:
            self.forum_inactive_until = None
            self.is_forum_active = True

    def get_avatar(self, default=None):
        if self.avatar:
            return '/local--files/%s' % self.avatar
        return default

    @property
    def url_name(self):
        # 用于个人主页 URL 的规范身份名
        if self.type == User.UserType.Wikidot:
            return self.wikidot_username or self.username
        return self.username

    def __str__(self):
        if self.type == User.UserType.Wikidot:
            return 'wd:%s' % (self.display_name or self.wikidot_username or self.username)
        return self.display_name or self.username

    def _generate_apikey(self, commit=True):
        self.password = ''
        self.api_key = make_password(self.username)[21:]
        if commit:
            self.save()

    def clean(self):
        super().clean()
        if self.username is None and self.wikidot_username is None:
            raise ValidationError('用户名或Wikidot用户名必须填写。')

    def save(self, *args, **kwargs):
        if not self.wikidot_username:
            self.wikidot_username = None
        if self.type == 'bot':
            if not self.api_key:
                self._generate_apikey(commit=False)
        else:
            self.api_key = None
        return super().save(*args, **kwargs)


def allocate_fallback_username(user_pk: int) -> str:
    # 认领时不能拦，所以 Wikidot 上真名就叫 wkt-uid-34 的账号可能已经占住了它
    base = '%s-%d' % (FALLBACK_USERNAME_PREFIX, user_pk)
    candidate = base
    for suffix in range(2, 1000):
        if not User.objects.filter(username=candidate).exists():
            return candidate
        candidate = '%s-%d' % (base, suffix)
    raise ValidationError('无法为该显示名分配身份用户名，请换一个显示名。')


class UsedToken(auto_prefetch.Model):
    class Meta(auto_prefetch.Model.Meta):
        verbose_name = '已使用的令牌'
        verbose_name_plural = '已使用的令牌列表'

    token = models.TextField('令牌', null=False)
    is_case_sensitive = models.BooleanField('区分大小写', null=False, default=True)

    @classmethod
    def is_used(cls, token):
        if cls.objects.filter(token=token, is_case_sensitive=True).exists():
            return True
        return cls.objects.filter(token__iexact=token, is_case_sensitive=False).exists()

    @classmethod
    def mark_used(cls, token, is_case_sensitive):
        cls(token=token, is_case_sensitive=is_case_sensitive).save()
