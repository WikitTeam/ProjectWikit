__all__ = [
    'Site',
    'get_current_site'
]

from functools import cached_property
from typing import Literal, Optional, overload

from solo.models import SingletonModel
from django.db import models

from web import threadvars
from .settings import Settings


class Site(SingletonModel):
    class Meta(SingletonModel.Meta):
        verbose_name = '站点'
        verbose_name_plural = '站点'

        constraints = [
            models.UniqueConstraint(fields=['domain'], name='%(app_label)s_%(class)s_domain_unique'),
            models.UniqueConstraint(fields=['slug'], name='%(app_label)s_%(class)s_slug_unique'),
        ]

    slug = models.TextField('缩写', null=False)

    title = models.TextField('标题', null=False)
    headline = models.TextField('副标题', null=False)
    icon = models.ImageField('图标', null=True, blank=True, upload_to='-/sites')

    domain = models.TextField('文章域名', null=False)
    media_domain = models.TextField('文件域名', null=False)

    @cached_property
    def settings(self):
        return Settings.objects.filter(site=self).first() or Settings.get_default_settings()

    def __str__(self) -> str:
        return f'{self.title} ({self.domain})'


@overload
def get_current_site(required: Literal[True]=True) -> Site: ...

@overload
def get_current_site(required: Literal[False]) -> Optional[Site]: ...

def get_current_site(required: bool=True) -> Optional[Site]:
    site = threadvars.get('current_site')
    if site is None and required:
        raise ValueError('当前站点不存在但被要求必须存在')
    return site