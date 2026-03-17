__all__ = [
    'Settings'
]

import auto_prefetch

from typing import Optional

from django.db import models


class Settings(auto_prefetch.Model):
    class Meta(auto_prefetch.Model.Meta):
        verbose_name = "设置"
        verbose_name_plural = "设置"

    class RatingMode(models.TextChoices):
        Default = ('default', '默认')
        Disabled = ('disabled', '禁用')
        UpDown = ('updown', '点赞/点踩')
        Stars = ('stars', '星级评分')

    class UserCreateTagsMode(models.TextChoices):
        Default = ('default', '默认')
        Disabled = ('disabled', '禁止')
        Enabled = ('enabled', '允许')

    site = auto_prefetch.OneToOneField('Site', on_delete=models.CASCADE, null=True, related_name='_settings')
    category = auto_prefetch.OneToOneField('Category', on_delete=models.CASCADE, null=True, related_name='_settings')

    rating_mode = models.TextField(choices=RatingMode.choices, default=RatingMode.Default, verbose_name="评分系统", null=False)
    can_user_create_tags = models.TextField(choices=UserCreateTagsMode.choices, default=UserCreateTagsMode.Default, verbose_name="用户是否可以创建标签", null=False)

    # 层级关系：
    # 默认设置 -> 站点设置 -> 分类设置
    @classmethod
    def get_default_settings(cls):
        return cls(rating_mode=Settings.RatingMode.Stars, can_user_create_tags=Settings.UserCreateTagsMode.Disabled)

    # 使用另一个对象中非空的值覆盖当前对象的字段。
    # 返回一个副本。
    def merge(self, other: Optional['Settings']) -> 'Settings':
        if other is None:
            return self
        new_settings = Settings()
        new_settings.rating_mode = other.rating_mode if other.rating_mode != Settings.RatingMode.Default else self.rating_mode
        new_settings.can_user_create_tags = other.can_user_create_tags if other.can_user_create_tags != Settings.UserCreateTagsMode.Default else self.can_user_create_tags
        return new_settings

    @property
    def creating_tags_allowed(self) -> bool:
        return self.can_user_create_tags == Settings.UserCreateTagsMode.Enabled