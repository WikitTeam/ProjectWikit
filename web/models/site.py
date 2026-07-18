__all__ = [
    'Site',
    'Theme',
    'SystemUpdate',
    'get_current_site',
    'get_site_theme_url',
    'get_theme_dir',
]

import os

from functools import cached_property
from pathlib import Path
from typing import Literal, Optional, overload

from solo.models import SingletonModel
from django.conf import settings as django_settings
from django.core.validators import RegexValidator
from django.db import models
from django.db.models.signals import post_delete
from django.dispatch import receiver


slug_validator = RegexValidator(r'^[A-Za-z0-9_-]+$', '标识名只能包含英文字母、数字、- 和 _')

from web import threadvars
from .settings import Settings


def get_theme_dir() -> Path:
    return Path(django_settings.MEDIA_ROOT) / 'theme'


class Theme(models.Model):
    class Meta:
        verbose_name = '主题'
        verbose_name_plural = '主题'

    class Mode(models.TextChoices):
        Inline = ('inline', '内联CSS')
        External = ('external', '外部链接')

    name = models.TextField('主题名称', null=False)
    slug = models.TextField('标识名', unique=True, validators=[slug_validator])
    mode = models.TextField('类型', choices=Mode.choices, default=Mode.Inline, null=False)
    css = models.TextField('CSS 内容', blank=True, default='')
    external_url = models.TextField('外部 CSS 链接', blank=True, default='')
    updated_at = models.DateTimeField('更新时间', auto_now=True)

    def __str__(self) -> str:
        return self.name

    @property
    def css_path(self) -> Path:
        return get_theme_dir() / (self.slug + '.css')

    def write_css_file(self):
        directory = get_theme_dir()
        directory.mkdir(parents=True, exist_ok=True)
        tmp = directory / (self.slug + '.css.tmp')
        with open(tmp, 'w', encoding='utf-8') as f:
            f.write(self.css or '')
        os.replace(tmp, self.css_path)

    def delete_css_file(self):
        try:
            self.css_path.unlink()
        except OSError:
            pass

    def save(self, *args, **kwargs):
        super().save(*args, **kwargs)
        if self.mode == self.Mode.Inline and self.slug:
            try:
                self.write_css_file()
            except OSError:
                pass


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


def get_site_theme_url() -> str:
    default_url = django_settings.STATIC_URL + 'theme.css'

    site = get_current_site(required=False)
    if site is None or not site.active_theme_id:
        return default_url

    theme = (Theme.objects
             .filter(pk=site.active_theme_id)
             .only('slug', 'mode', 'external_url', 'updated_at')
             .first())
    if theme is None:
        return default_url

    if theme.mode == Theme.Mode.External:
        return (theme.external_url or '').strip() or default_url

    return '/-/theme/%s.css?v=%s' % (theme.slug, int(theme.updated_at.timestamp()))


@receiver(post_delete, sender=Theme)
def _delete_theme_css_file(sender, instance, **kwargs):
    instance.delete_css_file()
