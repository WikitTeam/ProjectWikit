__all__ = [
    'Site',
    'Theme',
    'SystemUpdate',
    'get_current_site',
    'get_site_theme_url',
    'get_active_theme_meta',
]

from functools import cached_property
from typing import Literal, Optional, overload

from solo.models import SingletonModel
from django.conf import settings as django_settings
from django.core.cache import cache
from django.db import models
from django.db.models.signals import post_save
from django.dispatch import receiver

from web import threadvars
from .settings import Settings


_THEME_META_CACHE_KEY = 'active_theme_meta_v1'
_THEME_META_TTL = 60


class Theme(models.Model):
    class Meta:
        verbose_name = '主题'
        verbose_name_plural = '主题'

    class Mode(models.TextChoices):
        Inline = ('inline', '内联CSS')
        External = ('external', '外部链接')

    name = models.TextField('主题名称', null=False)
    mode = models.TextField('类型', choices=Mode.choices, default=Mode.Inline, null=False)
    css = models.TextField('CSS 内容', blank=True, default='')
    external_url = models.TextField('外部 CSS 链接', blank=True, default='')
    updated_at = models.DateTimeField('更新时间', auto_now=True)

    def __str__(self) -> str:
        return self.name


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

    home_page = models.TextField('主页名称', null=False, default='main')
    active_theme = models.ForeignKey(
        'Theme', verbose_name='站点主题', null=True, blank=True,
        on_delete=models.SET_NULL, related_name='+',
    )

    @cached_property
    def settings(self):
        return Settings.objects.filter(site=self).first() or Settings.get_default_settings()

    def __str__(self) -> str:
        return f'{self.title} ({self.domain})'


class SystemUpdate(Site):
    class Meta:
        proxy = True
        verbose_name = '系统更新'
        verbose_name_plural = '系统更新'


@overload
def get_current_site(required: Literal[True]=True) -> Site: ...

@overload
def get_current_site(required: Literal[False]) -> Optional[Site]: ...

def get_current_site(required: bool=True) -> Optional[Site]:
    site = threadvars.get('current_site')
    if site is None and required:
        raise ValueError('当前站点不存在但被要求必须存在')
    return site


def get_active_theme_meta() -> dict:
    meta = cache.get(_THEME_META_CACHE_KEY)
    if meta is not None:
        return meta

    site = get_current_site(required=False)
    theme = None
    if site is not None and site.active_theme_id:
        theme = (Theme.objects
                 .filter(pk=site.active_theme_id)
                 .only('id', 'updated_at', 'mode', 'external_url')
                 .first())

    if theme is None:
        meta = {'none': True}
    else:
        meta = {
            'id': theme.id,
            'v': int(theme.updated_at.timestamp()),
            'mode': theme.mode,
            'external_url': (theme.external_url or '').strip(),
        }

    cache.set(_THEME_META_CACHE_KEY, meta, _THEME_META_TTL)
    return meta


def get_site_theme_url() -> str:
    meta = get_active_theme_meta()
    if meta.get('none'):
        return django_settings.STATIC_URL + 'theme.css'
    return '/-/theme.css?v=%s-%s' % (meta['id'], meta['v'])


@receiver(post_save, sender=Theme)
@receiver(post_save, sender=Site)
def _invalidate_active_theme_meta(sender, **kwargs):
    cache.delete(_THEME_META_CACHE_KEY)