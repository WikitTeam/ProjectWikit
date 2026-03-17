__all__ = [
    'File'
]

from django.conf import settings
import auto_prefetch
from django.db import models
from pathlib import Path
from django.contrib.auth import get_user_model

from .articles import Article
from .site import get_current_site

import urllib.parse


User = get_user_model()


class File(auto_prefetch.Model):
    class Meta(auto_prefetch.Model.Meta):
        verbose_name = "文件"
        verbose_name_plural = "文件"

        indexes = [models.Index(fields=['article', 'name'])]
        constraints = [models.UniqueConstraint(fields=['article', 'name', 'deleted_at'], name='%(app_label)s_%(class)s_unique')]  # 逻辑：未删除的文件应保持唯一

    article = auto_prefetch.ForeignKey(Article, on_delete=models.CASCADE, verbose_name="文章")

    name = models.TextField(verbose_name="文件名")
    media_name = models.TextField(verbose_name="文件系统中的文件名")

    mime_type = models.TextField(verbose_name="MIME类型")
    size = models.PositiveBigIntegerField(verbose_name="文件大小")

    author = auto_prefetch.ForeignKey(User, on_delete=models.SET_NULL, verbose_name="文件作者", null=True, related_name='created_files')
    created_at = models.DateTimeField(auto_now_add=True, verbose_name="创建时间")

    deleted_by = auto_prefetch.ForeignKey(User, on_delete=models.SET_NULL, verbose_name="删除文件的用户", blank=True, null=True, related_name='deleted_files')
    deleted_at = models.DateTimeField(blank=True, null=True)

    def __str__(self) -> str:
        return f"{self.name} ({self.media_name})"

    @staticmethod
    def escape_media_name(name) -> str:
        return name.replace(':', '%3A').replace('/', '%2F')

    @property
    def media_url(self) -> str:
        site = get_current_site()
        return '//%s/%s/%s' % (site.media_domain, urllib.parse.quote(self.article.full_name), urllib.parse.quote(self.name))

    @property
    def local_media_path(self) -> str:
        return Path(settings.MEDIA_ROOT) / 'media' / self.escape_media_name(self.article.media_name) / self.escape_media_name(self.media_name)
    
    @property
    def local_media_destination(self):
        return Path(self.escape_media_name(self.article.media_name)) / self.escape_media_name(self.media_name)