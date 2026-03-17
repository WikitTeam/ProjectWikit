import auto_prefetch

from functools import cached_property
from urllib.parse import quote
from typing import Optional

from django.core.exceptions import ValidationError
from django.contrib.auth.models import Permission
from django.db import models
from django.contrib import admin

import web.permissions.articles
import web.permissions.forum

from web import fields
from web.util.pydantic import JSONInterface


__all__ = [
    'RoleCategory',
    'Role',
    'RolePermissionsOverride',
    'RolesMixin',
    'PermissionsOverrideMixin',
    'RolePermissionsOverrideMixin',
    'ProtectSensitiveAdmin'
]


def svg_file_validator(file):
    if not file.name.lower().endswith('.svg'):
        raise ValidationError('只允许SVG格式的文件。')
    try:
        header = file.read(1024).decode('utf-8', errors='ignore')
        file.seek(0)
    except Exception:
        raise ValidationError('无法读取文件。')

    if '<svg' not in header:
        raise ValidationError('文件不包含SVG标记。')
    

class RoleBadgeJSON(JSONInterface):
    text: str
    bg: Optional[str]=None
    text_color: Optional[str]=None
    show_border: bool=False
    tooltip: Optional[str]=None


class RoleIconJSON(JSONInterface):
    icon: str
    color: Optional[str]=None
    tooltip: Optional[str]=None


class RoleCategory(models.Model):
    class Meta:
        verbose_name = '角色分类'
        verbose_name_plural = '角色分类'

    name = models.CharField('名称')

    def __str__(self):
        return self.name


class Role(auto_prefetch.Model):
    class Meta(auto_prefetch.Model.Meta):
        verbose_name = '角色'
        verbose_name_plural = '角色'

        ordering = ['index']
        indexes = [models.Index(fields=['slug']), models.Index(fields=['category'])]

    class InlineVisualMode(models.TextChoices):
        Hidden = ('hidden', '隐藏')
        Badge = ('badge', '徽章')
        Icon = ('icon', '图标')

    class ProfileVisualMode(models.TextChoices):
        Hidden = ('hidden', '隐藏')
        Badge = ('badge', '徽章')
        Status = ('status', '状态')
    
    slug = models.CharField('标识符', unique=True, blank=False, null=False)
    name = models.CharField('完整名称', blank=True)
    short_name = models.CharField('简称', blank=True)
    category = auto_prefetch.ForeignKey(RoleCategory, verbose_name='分类', on_delete=models.SET_NULL, blank=True, null=True)
    index = models.PositiveIntegerField('优先级', default=0, editable=False, db_index=True, blank=False, null=False)

    is_staff = models.BooleanField('可访问管理后台', default=False, blank=False, null=False)

    votes_title = models.CharField('投票组标签', blank=True)
    group_votes = models.BooleanField('分组显示投票', default=False, blank=False, null=False)

    inline_visual_mode = models.CharField('用户名旁显示模式', choices=InlineVisualMode.choices, default=InlineVisualMode.Hidden)
    profile_visual_mode = models.CharField('个人资料显示模式', choices=ProfileVisualMode.choices, default=ProfileVisualMode.Hidden)

    color = fields.CSSColorField('颜色', default='#000000', blank=False, null=False)
    icon = models.FileField('图标', upload_to='-/roles', validators=[svg_file_validator], blank=True)

    badge_text = models.CharField('徽章文本',  blank=True)
    badge_bg = fields.CSSColorField('徽章背景色', default='#808080', blank=False, null=False)
    badge_text_color = fields.CSSColorField('文本颜色', default='#ffffff', blank=False, null=False)
    badge_show_border = models.BooleanField('显示边框', default=False, blank=False, null=False)

    permissions = models.ManyToManyField(Permission, verbose_name='权限', related_name='role_permissions_set', blank=True)
    restrictions = models.ManyToManyField(Permission, verbose_name='限制', related_name='role_restrictions_set', blank=True)

    @property
    def is_visual(self):
        return self.group_votes or self.inline_visual_mode != Role.InlineVisualMode.Hidden or self.profile_visual_mode != Role.ProfileVisualMode.Hidden

    def __str__(self):
        return self.short_name or self.name or self.slug

    def save(self, *args, **kwargs):
        super().save(*args, **kwargs)

        last_index = Role.objects.aggregate(max_index=models.Max('index'))['max_index']

        Role.objects.filter(slug='registered').update(index=last_index+1)
        Role.objects.filter(slug='everyone').update(index=last_index+2)

        roles_to_update = []
        for n, role in enumerate(Role.objects.order_by('index')):
            role.index = n
            roles_to_update.append(role)

        Role.objects.bulk_update(roles_to_update, ['index'])

    def delete(self, *args, **kwargs):
        if self.slug == 'everyone':
            raise ValidationError('角色"everyone"无法删除。')
        if self.slug == 'registered':
            raise ValidationError('角色"registered"无法删除。')
        return super().delete(*args, **kwargs)
    
    def get_name_tail(self):
        if self.inline_visual_mode == Role.InlineVisualMode.Badge:
            return RoleBadgeJSON(
                text=self.badge_text or self.slug,
                bg=self.badge_bg,
                text_color=self.badge_text_color,
                show_border=self.badge_show_border,
                tooltip=self.name
            )
        elif self.inline_visual_mode == Role.InlineVisualMode.Icon:
            if self.icon:
                with self.icon.open('r') as f:
                    icon = f.read()

                icon_parts:list = icon[icon.index('<svg'):].split('>')
                icon_parts.insert(1, f'<style>svg{{color:{self.color}}}</style')
                colored_icon = quote('>'.join(icon_parts))

                return RoleIconJSON(
                    icon=colored_icon,
                    color=self.color,
                    tooltip=self.name
                )

        return None
    
    @staticmethod
    def get_or_create_default_role():
        everyone, created = Role.objects.get_or_create(
            slug='everyone',
        )
        if created:
            everyone.permissions.add(
                web.permissions.articles.ViewArticlesPermission.as_permission(),
                web.permissions.forum.ViewForumSectionsPermission.as_permission(),
                web.permissions.forum.ViewForumCategoriesPermission.as_permission(),
                web.permissions.forum.ViewForumThreadsPermission.as_permission(),
                web.permissions.forum.ViewForumPostsPermission.as_permission(),
                )
            everyone.index = 0
            everyone.save()
        return everyone
    
    @staticmethod
    def get_or_create_registered_role():
        registred, created = Role.objects.get_or_create(
            slug='registered',
        )
        if created:
            registred.index = 1
            registred.save()
        return registred


# Category.objects.get(name='草稿').permissions_override.add(RolePermissionsOverride.objects.create(role=Role.objects.get(slug='test_role2')))
class RolePermissionsOverride(auto_prefetch.Model):
    role = auto_prefetch.ForeignKey(Role, on_delete=models.CASCADE)
    permissions = models.ManyToManyField(Permission, related_name='override_role_permissions_set', blank=True)
    restrictions = models.ManyToManyField(Permission, related_name='override_role_restrictions_set', blank=True)


class RolesMixin(models.Model):
    class Meta:
        abstract = True

    roles = models.ManyToManyField(Role, verbose_name='角色', blank=True, related_name='users', related_query_name='user')

    @property
    def is_staff(self):
        if self.is_superuser: # type: ignore
            return True
        for role in self.roles.all():
            if role.is_staff:
                return True
        return False
    
    @is_staff.setter
    def is_staff(self, new_value):
        pass

    @cached_property
    def operation_index(self):
        op_index = self.roles.aggregate(min_index=models.Min('index'))['min_index']
        return op_index if op_index is not None else float('inf')
    
    @cached_property
    def vote_role(self):
        everyone_role = Role.get_or_create_default_role()
        if self.is_anonymous and everyone_role.group_votes: # type: ignore
            return everyone_role
        
        registered_role = Role.get_or_create_registered_role()
        if registered_role.group_votes:
            return registered_role
        elif everyone_role.group_votes:
            return everyone_role

        return self.roles.all().filter(group_votes=True).order_by('index').first()
    
    @cached_property
    def name_tails(self):
        if not self.is_active and not self.type == self.UserType.Wikidot: # type: ignore
            return {
                'badges': [RoleBadgeJSON(
                    text='封禁',
                    bg='#000000',
                    text_color="#FFFFFF",
                    show_border=False,
                    tooltip='用户已被封禁'
                )],
                'icons': []
            }
        elif self.type == self.UserType.Bot: # type: ignore
            return {
                'badges': [RoleBadgeJSON(
                    text='机器人',
                    bg='#77A',    #a1abca    #737d9b    #4463bf
                    text_color='#FFFFFF',
                    show_border=False,
                    tooltip='机器账户'
                )],
                'icons': []
            }
        visual_roles = self.roles.all() \
        .exclude(inline_visual_mode=Role.InlineVisualMode.Hidden) \
        .annotate(typed_category=models.functions.ConcatPair(models.F('inline_visual_mode'), models.F('category_id'), output_field=models.CharField())).order_by('typed_category', 'index')
        catigorized_candidates = visual_roles.exclude(category__isnull=True).distinct('typed_category')
        uncatigorized_candidates = visual_roles.filter(category__isnull=True)
        candidates = catigorized_candidates.union(uncatigorized_candidates).order_by('index')
        badges = []
        icons = []

        for role in candidates:
            tail = role.get_name_tail()
            if tail:
                if isinstance(tail, RoleBadgeJSON):
                    badges.append(tail)
                else:
                    icons.append(tail)
        
        return {
            'badges': badges,
            'icons': icons
        }
    
    
    @cached_property
    def showcase(self):
        if not self.is_active: # type: ignore
            if self.type == self.UserType.Wikidot: # type: ignore
                return {
                    'badges': [],
                    'titles': ['未激活']
                }
            else:
                return {
                    'badges': [],
                    'titles': ['已封禁']
                }
        elif self.type == self.UserType.Bot: # type: ignore
            return {
                'badges': [],
                'titles': ['机器人']
            }
        
        visual_roles = self.roles.all().exclude(profile_visual_mode=Role.ProfileVisualMode.Hidden)
        catigorized_candidates = visual_roles.exclude(category__isnull=True).order_by('category', 'index').distinct('category')
        uncatigorized_candidates = visual_roles.filter(category__isnull=True)
        candidates = catigorized_candidates.union(uncatigorized_candidates).order_by('index')

        badges = []
        titles = []

        for role in candidates:
            if role.profile_visual_mode == Role.ProfileVisualMode.Badge:
                badges.append(RoleBadgeJSON(
                    text=role.badge_text or role.name or role.slug,
                    bg=role.badge_bg,
                    text_color=role.badge_text_color,
                    show_border=role.badge_show_border,
                    tooltip=role.name
                ))
            elif role.profile_visual_mode == Role.ProfileVisualMode.Status:
                titles.append(role.name or role.slug)
        
        return {
            'badges': badges,
            'titles': titles
        }


class PermissionsOverrideMixin:
    roles_override_pipeline: Optional[list[str]] = None
    perms_override_pipeline: Optional[list[str]] = None

    def override_role(self, user_obj, perms, role=None):
        pipeline = self.__class__.roles_override_pipeline
        if pipeline:
            for unit in pipeline:
                obj = getattr(self, unit)
                if obj:
                    perms = obj.override_role(user_obj, perms, role)
        return perms
    
    def override_perms(self, user_obj, perms, roles=[]):
        pipeline = self.__class__.perms_override_pipeline
        if pipeline:
            for unit in pipeline:
                obj = getattr(self, unit)
                if obj:
                    perms = obj.override_perms(user_obj, perms, roles)
        return perms


class RolePermissionsOverrideMixin(models.Model, PermissionsOverrideMixin):
    class Meta:
        abstract = True

    permissions_override = models.ManyToManyField(RolePermissionsOverride)

    def delete(self, *args, **kwargs):
        self.permissions_override.all().delete()
        return super().delete(*args, **kwargs)

    def override_role(self, user_obj, perms: set, role=None):
        if not role or not self.pk:
            return perms
        for perm_override in self.permissions_override.all().filter(role=role):
            for perm in perm_override.permissions.all():
                codename = f'roles.{perm.codename}'
                perms.add(codename)
            for perm in perm_override.restrictions.all():
                codename = f'roles.{perm.codename}'
                if codename in perms:
                    perms.remove(codename)
            break
        return perms
    

class ProtectSensitiveAdmin(admin.ModelAdmin):
    sensitive_fields: list[str] = []

    def __init__(self, *args, **kwargs):
        self.sensitive_fields = self.__class__.sensitive_fields
        super().__init__(*args, **kwargs)

    def get_fieldsets(self, request, obj=None):
        fieldsets = super().get_fieldsets(request, obj)
        if not request.user.has_perm('roles.view_sensitive_info'):
            for _, fieldset in fieldsets:
                fieldset['fields'] = [field for field in fieldset['fields'] if field not in self.sensitive_fields]
        return fieldsets
    
    def get_list_filter(self, request) -> list[str]:
        if not request.user.has_perm('roles.view_sensitive_info'):
            return [field for field in self.list_filter if field not in self.sensitive_fields]
        return self.list_filter
    
    def get_list_display(self, request) -> list[str]:
        if not request.user.has_perm('roles.view_sensitive_info'):
            return [field for field in self.list_display if field not in self.sensitive_fields]
        return self.list_display
    
    def get_search_fields(self, request) -> list[str]:
        if not request.user.has_perm('roles.view_sensitive_info'):
            return [field for field in self.search_fields if field not in self.sensitive_fields]
        return list(self.search_fields)