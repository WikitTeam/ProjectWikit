__all__ = [
    'TagsCategory',
    'Tag',
    'Category',
    'Article',
    'ArticleVersion',
    'ArticleLogEntry',
    'Vote',
    'ExternalLink'
]

import re
import auto_prefetch

from uuid import uuid4
from typing import Optional
from functools import cached_property

from django.core.validators import RegexValidator
from django.db import models
from django.contrib.auth import get_user_model

from web.fields import CITextField
from web.util import uuid4_str
from .roles import Role, PermissionsOverrideMixin, RolePermissionsOverrideMixin
from .settings import Settings
from .site import get_current_site


User = get_user_model()


class TagsCategorySlugValidator(RegexValidator):
    regex = r'^[a-zа-я0-9.-_]+\Z'
    message = '标签分类标识符只能包含小写字母、数字和符号[.-_]（不含括号）。'
    flags = re.ASCII


class TagsCategory(auto_prefetch.Model):
    class Meta(auto_prefetch.Model.Meta):
        verbose_name = '标签分类'
        verbose_name_plural = '标签分类'

        constraints = [models.UniqueConstraint(fields=['slug'], name='%(app_label)s_%(class)s_unique')]
        indexes = [models.Index(fields=['name'])]

    name = models.TextField('完整名称')
    description = models.TextField('描述', blank=True)
    priority = models.PositiveIntegerField(null=True, blank=True, unique=True, verbose_name='排序编号')
    slug = models.TextField('标识符', unique=True, validators=[TagsCategorySlugValidator()])

    def __str__(self):
        return f'{self.name} ({self.slug})'

    @staticmethod
    def get_or_create_default_tags_category():
        category, _ = TagsCategory.objects.get_or_create(slug='_default', defaults=dict(name='默认'))
        return category.pk

    @property
    def is_default(self) -> bool:
        return self.slug == '_default'

    def save(self, *args, **kwargs):
        if not self.pk and not self.name:
            self.name = self.slug
        return super(TagsCategory, self).save(*args, **kwargs)


class Tag(auto_prefetch.Model):
    class Meta(auto_prefetch.Model.Meta):
        verbose_name = '标签'
        verbose_name_plural = '标签'

        constraints = [models.UniqueConstraint(fields=['category', 'name'], name='%(app_label)s_%(class)s_unique')]
        indexes = [models.Index(fields=['category', 'name'])]

    category = auto_prefetch.ForeignKey(TagsCategory, null=False, blank=False, on_delete=models.CASCADE, verbose_name='分类', default=TagsCategory.get_or_create_default_tags_category)
    name = models.TextField('名称')

    def __str__(self):
        return self.full_name

    @property
    def full_name(self) -> str:
        if self.category and not self.category.is_default:
            return f'{self.category.slug}:{self.name}'
        return self.name

    def save(self, *args, **kwargs):
        self.name = self.name.lower()
        return super(Tag, self).save(*args, **kwargs)


class Category(auto_prefetch.Model, RolePermissionsOverrideMixin):
    class Meta(auto_prefetch.Model.Meta):
        verbose_name = '分类设置'
        verbose_name_plural = '分类设置'

        constraints = [models.UniqueConstraint(fields=['name'], name='%(app_label)s_%(class)s_unique')]
        indexes = [models.Index(fields=['name'])]

    name = CITextField('名称')

    is_indexed = models.BooleanField('是否被搜索引擎索引', null=False, default=True)

    def __str__(self) -> str:
        return self.name
    
    def __eq__(self, value):
        if isinstance(value, str) and self.name == value:
            return True
        return super().__eq__(value)
    
    def __hash__(self):
        return super().__hash__()

    # 此函数返回由分类设置覆盖的站点设置。
    # 如果两者均未设置，则回退到Settings类中定义的默认值。
    @cached_property
    def settings(self):
        category_settings = Settings.objects.filter(category=self).first()
        site_settings = get_current_site().settings
        return Settings.get_default_settings().merge(site_settings).merge(category_settings)
    
    @staticmethod
    def get_or_default_category(category):
        cat = Category.objects.filter(name=category)
        if not cat:
            return Category(name=category)
        else:
            return cat[0]


class Article(auto_prefetch.Model, PermissionsOverrideMixin):
    class Meta(auto_prefetch.Model.Meta):
        verbose_name = '文章'
        verbose_name_plural = '文章'

        constraints = [models.UniqueConstraint(fields=['category', 'name'], name='%(app_label)s_%(class)s_unique')]
        indexes = [models.Index(fields=['category']), models.Index(fields=['name']), models.Index(fields=['complete_full_name']), models.Index(fields=['created_at']), models.Index(fields=['updated_at'])]

    roles_override_pipeline = ['category_as_object']

    category = CITextField('分类', default='_default')
    name = CITextField('名称')
    complete_full_name = models.GeneratedField(
        expression=models.functions.Concat(
            'category', models.Value(':', output_field=CITextField()), 'name',
        ),
        output_field=CITextField(),
        db_persist=True
    )
    title = models.TextField('标题')

    parent = auto_prefetch.ForeignKey('self', on_delete=models.SET_NULL, null=True, blank=True, verbose_name='父页面')
    tags = models.ManyToManyField(Tag, blank=True, related_name='articles', verbose_name='标签')
    authors = models.ManyToManyField(User, blank=False, related_name='authored_by', verbose_name='作者')

    locked = models.BooleanField('页面已保护', default=False)

    created_at = models.DateTimeField('创建时间', auto_now_add=True)
    updated_at = models.DateTimeField('修改时间', auto_now_add=True)

    media_name = models.TextField('文件系统中文件文件夹的名称', unique=True, default=uuid4_str)

    @cached_property
    def settings(self):
        if self.category_as_object:
            return self.category_as_object.settings
        else:
            site_settings = get_current_site().settings
            return Settings.get_default_settings().merge(site_settings)

    @property
    def full_name(self) -> str:
        if self.category != '_default':
            return f'{self.category}:{self.name}'
        return self.name

    @property
    def display_name(self) -> str:
        return self.title.strip() or self.full_name
    
    @cached_property
    def category_as_object(self) -> Optional[Category]:
        return Category.objects.filter(name=self.category).first()

    def __str__(self) -> str:
        return f'{self.title} ({self.full_name})'
    
    def override_perms(self, user_obj, perms: set, roles=[]):
        if self.locked:
            if 'roles.lock_articles' not in perms:
                lockable_perms = {'roles.edit_articles', 'roles.manage_article_authors', 'roles.manage_article_files', 'roles.tag_articles', 'roles.move_articles', 'roles.delete_articles'}
                perms = perms.difference(lockable_perms)
        elif user_obj and user_obj in self.authors.all():
            perms.add('roles.manage_article_authors')
        return super().override_perms(user_obj, perms, roles)


class ArticleVersion(auto_prefetch.Model):
    class Meta(auto_prefetch.Model.Meta):
        verbose_name = '文章版本'
        verbose_name_plural = '文章版本'

        indexes = [models.Index(fields=['article', 'created_at'])]

    article = auto_prefetch.ForeignKey(Article, on_delete=models.CASCADE, verbose_name='文章', related_name='versions')
    source = models.TextField('源代码')
    ast = models.JSONField('文章的AST树', blank=True, null=True)
    rendered = models.TextField('文章渲染结果', blank=True, null=True)
    created_at = models.DateTimeField('创建时间', auto_now_add=True)

    def __str__(self) -> str:
        return f'{self.created_at.strftime('%Y-%m-%d, %H:%M:%S')} - {self.article}'


class ArticleLogEntry(auto_prefetch.Model):
    class Meta(auto_prefetch.Model.Meta):
        verbose_name = '日志条目'
        verbose_name_plural = '日志条目'

        constraints = [models.UniqueConstraint(fields=['article', 'rev_number'], name='%(app_label)s_%(class)s_unique')]

    class LogEntryType(models.TextChoices):
        Source = ('source', '内容更改')
        Title = ('title', '标题更改')
        Name = ('name', '页面地址更改')
        Tags = ('tags', '标签更改')
        New = ('new', '页面创建')
        Parent = ('parent', '父页面更改')
        FileAdded = ('file_added', '文件添加')
        FileDeleted = ('file_deleted', '文件删除')
        FileRenamed = ('file_renamed', '文件重命名')
        VotesDeleted = ('votes_deleted', '评分重置')
        Authorship = ('authorship', '作者变更')
        Wikidot = ('wikidot', '来自Wikidot的编辑')
        Revert = ('revert', '版本回退')

    article = auto_prefetch.ForeignKey(Article, on_delete=models.CASCADE, verbose_name='文章')
    user = auto_prefetch.ForeignKey(User, null=True, blank=True, on_delete=models.SET_NULL, verbose_name='用户')
    type = models.TextField('类型', choices=LogEntryType.choices)
    meta = models.JSONField('元数据', default=dict, blank=True)
    created_at = models.DateTimeField('创建时间', auto_now_add=True)
    comment = models.TextField('注释', blank=True)
    rev_number = models.PositiveIntegerField('版本号')

    def __str__(self) -> str:
        return f'#{self.rev_number} - {self.article}'


class Vote(auto_prefetch.Model):
    class Meta(auto_prefetch.Model.Meta):
        verbose_name = '评分'
        verbose_name_plural = '评分'

        constraints = [models.UniqueConstraint(fields=['article', 'user'], name='%(app_label)s_%(class)s_unique')]
        indexes = [models.Index(fields=['article'])]

    article = auto_prefetch.ForeignKey(Article, on_delete=models.CASCADE, verbose_name='文章', related_name='votes')
    user = auto_prefetch.ForeignKey(User, null=True, on_delete=models.SET_NULL, verbose_name='用户')
    rate = models.FloatField('评分')
    date = models.DateTimeField('投票日期', auto_now_add=True, null=True)
    role = auto_prefetch.ForeignKey(Role, on_delete=models.SET_NULL, verbose_name='角色', null=True)

    def __str__(self) -> str:
        return f'{self.article}: {self.user} - {self.rate}'


class ExternalLink(auto_prefetch.Model):
    class Meta(auto_prefetch.Model.Meta):
        verbose_name = '链接关系'
        verbose_name_plural = '链接关系'

        indexes = [
            models.Index(fields=['link_from', 'link_to']),
            models.Index(fields=['link_type'])
        ]

        constraints = [models.UniqueConstraint(fields=['link_from', 'link_to', 'link_type'], name='%(app_label)s_%(class)s_unique')]

    class Type(models.TextChoices):
        Include = 'include'
        Link = 'link'

    link_from = CITextField('源文章', null=False)
    link_to = CITextField('目标文章', null=False)
    link_type = models.TextField('链接类型', choices=Type.choices, null=False)