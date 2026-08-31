from solo.admin import SingletonModelAdmin
from adminsortable2.admin import SortableAdminMixin

from django.db.models.query import QuerySet
from django.db.models import ExpressionWrapper, F, Case, When, BooleanField
from django.contrib.admin import SimpleListFilter
from django.contrib.auth.models import Permission
from django.contrib.auth.admin import UserAdmin
from django.contrib.auth.forms import UserChangeForm
from django.contrib import admin
from django.urls import path, resolve
from django.utils import timezone
from django.utils.html import format_html, format_html_join
from django import forms

import web.fields
from web.views.sus_users import AdminSusActivityView

from .models import *
from .fields import CITextField
from .views.invite import InviteView
from .views.invite import GenerateClaimLinkView
from .views.invite import GenerateInviteLinkView
from .views.invite import invite_url
from .views.bot import CreateBotView
from .views.reset_votes import ResetUserVotesView
from .controllers import logging
from .permissions import get_role_permissions_content_type


class TagsCategoryForm(forms.ModelForm):
    class Meta:
        model = TagsCategory
        widgets = {
            'name': forms.TextInput,
            'slug': forms.TextInput,
        }
        fields = ('name', 'slug', 'description', 'priority')


@admin.register(TagsCategory)
class TagsCategoryAdmin(admin.ModelAdmin):
    form = TagsCategoryForm
    search_fields = ['name', 'slug', 'description']
    list_display = ['name', 'description', 'priority', 'slug']


class TagForm(forms.ModelForm):
    class Meta:
        model = Tag
        widgets = {
            'name': forms.TextInput
        }
        fields = '__all__'


@admin.register(Tag)
class TagAdmin(admin.ModelAdmin):
    form = TagForm
    search_fields = ['name', 'category__name']
    list_filter = ['category']
    list_display = ['name', 'category']


class SettingsForm(forms.ModelForm):
    class Meta:
        model = Settings
        widgets = {
            'rating_mode': forms.Select
        }
        fields = '__all__'
        exclude = ['site', 'category']


class SettingsAdmin(admin.StackedInline):
    form = SettingsForm
    model = Settings
    can_delete = False
    max_num = 1


class CategoryForm(forms.ModelForm):
    class Meta:
        model = Category
        widgets = {
            'name': forms.TextInput,
        }
        exclude = ['permissions_override']

    _add_override_roles_ = forms.ModelMultipleChoiceField(label='添加要覆盖的角色', queryset=QuerySet(Role), required=False)
    _remove_override_roles_ = forms.ModelMultipleChoiceField(label='移除要覆盖的角色', queryset=QuerySet(Role), required=False)
    _perms_override_ = web.fields.PermissionsOverrideField(exclude_admin=True)

    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)

        instance = kwargs.get('instance')
        if instance:
            self.fields['_perms_override_'].widget.instance = instance

            overrided_roles = Role.objects.filter(rolepermissionsoverride__in=instance.permissions_override.all())
            self.fields['_add_override_roles_'].queryset = Role.objects.exclude(id__in=overrided_roles)
            self.fields['_remove_override_roles_'].queryset = overrided_roles
        else:
            self.fields['_perms_override_'].widget.instance = None
            self.fields['_add_override_roles_'].queryset = Role.objects.all()
            self.fields['_remove_override_roles_'].disabled = True


    def save(self, commit=True):
        instance = super().save(commit=False)

        instance.save()

        roles_data = self.cleaned_data.get('_perms_override_', {})
        if roles_data:
            content_type = get_role_permissions_content_type()
            overrides = []

            instance.permissions_override.all().delete()

            for role_id, perms_data in roles_data.items():
                perms_override = RolePermissionsOverride.objects.create(role_id=role_id)

                if perms_data['allow']:
                    perms = Permission.objects.filter(codename__in=perms_data['allow'], content_type=content_type)
                    perms_override.permissions.set(perms)

                if perms_data['deny']:
                    restrictions = Permission.objects.filter(codename__in=perms_data['deny'], content_type=content_type)
                    perms_override.restrictions.set(restrictions)

                overrides.append(perms_override)

            instance.permissions_override.add(*overrides)

        roles_to_override = self.cleaned_data.get('_add_override_roles_', {})
        if roles_to_override:
            overrides = []
            for role in roles_to_override:
                overrides.append(RolePermissionsOverride.objects.create(role=role))

            instance.permissions_override.add(*overrides)

        roles_to_cancel_override = self.cleaned_data.get('_remove_override_roles_', {})
        if roles_to_cancel_override:
            instance.permissions_override.all().filter(role__in=roles_to_cancel_override).delete()

        if commit:
            instance.save()

        return instance


@admin.register(Category)
class CategoryAdmin(admin.ModelAdmin):
    form = CategoryForm
    fieldsets = (
        (None, {
            'fields': ('name', 'is_indexed')
        }),
        ('权限覆盖', {
            'fields': ('_add_override_roles_', '_remove_override_roles_', '_perms_override_')
        })
    )
    inlines = [SettingsAdmin]


class ThemeForm(forms.ModelForm):
    class Meta:
        model = Theme
        widgets = {
            'name': forms.TextInput,
            'slug': forms.TextInput,
            'external_url': forms.TextInput,
            'css': forms.Textarea(attrs={'rows': 24, 'style': 'font-family:monospace;width:100%'}),
        }
        fields = '__all__'


@admin.register(Theme)
class ThemeAdmin(admin.ModelAdmin):
    form = ThemeForm
    list_display = ['name', 'slug', 'mode', 'updated_at']
    fields = ['name', 'slug', 'mode', 'css', 'external_url']


class SiteForm(forms.ModelForm):
    class Meta:
        model = Site
        widgets = {
            'slug': forms.TextInput,
            'title': forms.TextInput,
            'headline': forms.TextInput,
            'domain': forms.TextInput,
            'media_domain': forms.TextInput,
            'home_page': forms.TextInput,
        }

        fields = '__all__'


@admin.register(Site)
class SiteAdmin(SingletonModelAdmin):
    form = SiteForm
    inlines = [SettingsAdmin]
    fieldsets = (
        (None, {
            'fields': ('slug', 'title', 'headline', 'domain', 'media_domain', 'home_page', 'active_theme')
        }),
        ('外观', {
            'fields': ('icon', 'auth_icon', 'footer_license')
        }),
        ('注册与入组', {
            'fields': ('signup_notice', 'default_role', 'verified_role',
                       'membership_password_enabled', 'membership_password', 'membership_password_role')
        }),
    )

    def get_readonly_fields(self, request, obj=None):
        readonly_fields = super().get_readonly_fields(request, obj)
        # 这几项决定谁能自动拿到哪个角色，等同于发权限，所以跟着权限表那道门走。
        if not request.user.has_perm('roles.manage_permissions'):
            readonly_fields = list(readonly_fields) + [
                'default_role', 'verified_role',
                'membership_password_enabled', 'membership_password', 'membership_password_role',
            ]
        return readonly_fields

    def save_model(self, request, obj, form, change):
        super().save_model(request, obj, form, change)


@admin.register(SystemUpdate)
class SystemUpdateAdmin(admin.ModelAdmin):
    def has_add_permission(self, request):
        return False

    def has_delete_permission(self, request, obj=None):
        return False

    def has_change_permission(self, request, obj=None):
        return False

    def has_view_permission(self, request, obj=None):
        return request.user.has_perm('roles.manage_updates')

    def has_module_permission(self, request):
        return request.user.has_perm('roles.manage_updates')

    def changelist_view(self, request, extra_context=None):
        from django.shortcuts import redirect
        return redirect('wu_update')


class ForumSectionForm(forms.ModelForm):
    class Meta:
        model = ForumSection
        widgets = {
            'name': forms.TextInput,
        }
        fields = '__all__'


@admin.register(ForumSection)
class ForumSectionAdmin(admin.ModelAdmin):
    form = ForumSectionForm
    search_fields = ['name', 'description']


class ForumCategoryForm(forms.ModelForm):
    class Meta:
        model = ForumCategory
        widgets = {
            'name': forms.TextInput,
        }
        fields = '__all__'


@admin.register(ForumCategory)
class ForumCategoryAdmin(admin.ModelAdmin):
    form = ForumCategoryForm
    search_fields = ['name', 'description']
    list_filter = ['section']
    list_display = ['name', 'section']


class AdvancedUserChangeForm(UserChangeForm):
    class Meta:
        # 修复用户名和wikidot用户名字段类型
        widgets = {
            'username': forms.TextInput(attrs={'class': 'vTextField'}),
            'wikidot_username': forms.TextInput(attrs={'class': 'vTextField'})
        }

    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)
        if 'roles' in self.fields:
            self.fields['roles'].queryset = Role.objects.exclude(slug__in=['everyone', 'registered'])


@admin.register(User)
class AdvancedUserAdmin(ProtectSensitiveAdmin, UserAdmin):
    form = AdvancedUserChangeForm

    list_filter = ['is_superuser', 'is_active', 'roles']
    list_display = ['username_or_wd', 'email', 'is_active']
    search_fields = ['username', 'wikidot_username', 'display_name', 'email']
    readonly_fields = ['api_key', '_op_index']
    sensitive_fields = ['email']

    fieldsets = UserAdmin.fieldsets
    fieldsets[0][1]['fields'] = ('username', 'wikidot_username', 'display_name', 'type', 'password', 'api_key', '_op_index')
    fieldsets[1][1]['fields'] += ('bio', 'avatar')
    fieldsets[2][1]['fields'] = ('is_active', 'inactive_until', 'is_forum_active', 'forum_inactive_until', 'can_send_direct_messages', 'roles', 'is_superuser')

    @admin.display(ordering='username_or_wd')
    def username_or_wd(self, obj):
        return obj.__str__()
    
    @admin.display(description='权限级别')
    def _op_index(self, obj):
        return obj.operation_index

    def get_urls(self):
        urls = super().get_urls()
        new_urls = [
            path('invite/', InviteView.as_view()),
            path('generate-link/', GenerateInviteLinkView.as_view()),
            path('claim-link/', GenerateClaimLinkView.as_view()),
            path('newbot/', CreateBotView.as_view()),
            path('<id>/activate/', InviteView.as_view()),
            path('<id>/reset_votes/', ResetUserVotesView.as_view()),
        ]
        return new_urls + urls

    def get_form(self, request, *args, **kwargs):
        form = super().get_form(request, *args, **kwargs)
        not_required = ['inactive_until', 'forum_inactive_until', 'wikidot_username']
        for not_required_field in not_required:
            if not_required_field in form.base_fields:
                form.base_fields[not_required_field].required = False
        return form

    def get_readonly_fields(self, request, obj=None):
        readonly_fields = super().get_readonly_fields(request, obj)
        if not request.user.is_superuser:
            readonly_fields += ['is_superuser']
        return readonly_fields
    
    def get_queryset(self, request):
        qs = super(AdvancedUserAdmin, self).get_queryset(request)
        return qs.annotate(
                username_or_wd=ExpressionWrapper(
                    Case(
                        When(type=User.UserType.Wikidot, then=F('wikidot_username')),
                        default=F('username'),
                        output_field=CITextField()
                    ),
                    output_field=CITextField()
                )
            ).order_by('username_or_wd')

    def formfield_for_manytomany(self, db_field, request, **kwargs):
        if db_field.name == 'roles' and not self._may_set_roles(request, self._editing(request)):
            kwargs['disabled'] = True
        return super().formfield_for_manytomany(db_field, request, **kwargs)

    # 发角色就是发权限，所以这道门是「访问权限表」而不是「管理用户」——
    # 否则一个只被勾了「管理用户」的人可以给自己加任意角色。
    def _may_set_roles(self, request, obj=None):
        if request.user.is_superuser:
            return True
        if obj is not None and obj.is_superuser:
            return False
        return request.user.has_perm('roles.manage_permissions')

    # formfield_for_manytomany 拿不到正在编辑的对象，只能从 URL 里认。
    def _editing(self, request):
        object_id = resolve(request.path_info).kwargs.get('object_id')
        if not object_id:
            return None
        return self.get_object(request, object_id)
    
    def has_change_permission(self, request, obj=None):
        if obj and not request.user.is_superuser and obj.operation_index <= request.user.operation_index:
            return False
        return super().has_change_permission(request, obj)
    
    def save_model(self, request, obj, form, change):
        if obj.pk:
            target = User.objects.get(id=obj.id)
            if change:
                if not request.user.is_superuser:
                    obj.is_superuser = target.is_superuser
                if not self._may_set_roles(request, target):
                    obj.roles.set(target.roles.all())
        super().save_model(request, obj, form, change)


class ActionsLogForm(forms.ModelForm):
    class Meta:
        model = ActionLogEntry
        exclude = ['meta']


@admin.register(ActionLogEntry)
class ActionsLogAdmin(ProtectSensitiveAdmin, admin.ModelAdmin):
    form = ActionsLogForm
    list_filter = ['user', 'type', 'created_at', 'origin_ip']
    list_display = ['user_or_name', 'type', 'info', 'created_at', 'origin_ip']
    search_fields = ['meta']
    sensitive_fields = ['origin_ip']

    @admin.display(description=User.Meta.verbose_name)
    def user_or_name(self, obj):
        if obj.user == None:
            return f'{obj.stale_username} (已删除)'
        return obj.user
    
    @admin.display(description='详细信息')
    def info(self, obj):
        return logging.get_action_log_entry_description(obj)

    def has_add_permission(self, request):
        return False

    def has_delete_permission(self, request, obj=None):
        return False
    
    def has_change_permission(self, request, obj=None):
        return False
    
    def get_urls(self):
        urls = super().get_urls()
        new_urls = [
            path('sus', AdminSusActivityView.as_view())
        ]
        return new_urls + urls


class RoleCategoryForm(forms.ModelForm):
    class Meta:
        model = RoleCategory
        fields = '__all__'


@admin.register(RoleCategory)
class RoleCategoryAdmin(admin.ModelAdmin):
    form = RoleCategoryForm


class RoleForm(forms.ModelForm):
    class Meta:
        model = Role
        exclude = ['permissions', 'restrictions']

    _perms_ = web.fields.PermissionsField()

    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)

        instance = kwargs.get('instance')
        self.fields['_perms_'].widget.instance = instance

    def save(self, commit=True):
        instance = super().save(commit=False)

        instance.save()

        perms_data = self.cleaned_data.get('_perms_', {})
        if perms_data:
            content_type = get_role_permissions_content_type()

            instance.permissions.clear()
            instance.restrictions.clear()

            if perms_data['allow']:
                perms = Permission.objects.filter(codename__in=perms_data['allow'], content_type=content_type)
                instance.permissions.set(perms)

            if perms_data['deny']:
                restrictions = Permission.objects.filter(codename__in=perms_data['deny'], content_type=content_type)
                instance.restrictions.set(restrictions)

        if commit:
            instance.save()

        return instance


class IsVisualRoleFilter(SimpleListFilter):
    title = '视觉角色'
    parameter_name = 'is_visual_role'

    def lookups(self, request, model_admin):
        return [
            (True, '是'),
            (False, '否')
        ]

    def queryset(self, request, queryset):
        if self.value():
            return queryset.annotate(is_visual_role=ExpressionWrapper(
                F('group_votes') or \
                F('inline_visual_mode') != Role.InlineVisualMode.Hidden or \
                F('profile_visual_mode') != Role.ProfileVisualMode.Hidden,
                output_field=BooleanField()
            )).filter(is_visual_role=self.value())
        else:
            return queryset

@admin.register(Role)
class RoleAdmin(SortableAdminMixin, admin.ModelAdmin):
    form = RoleForm
    list_filter = ['category', 'is_staff', IsVisualRoleFilter]
    list_display = ['__str__', '_users_number', '_idx']
    fieldsets = (
        (None, {
            'fields': ('slug', 'name', 'short_name', 'category', 'is_staff')
        }),
        ('视觉', {
            'fields': ('group_votes', 'votes_title', 'inline_visual_mode', 'profile_visual_mode', 'color', 'icon', 'badge_text', 'badge_bg', 'badge_text_color', 'badge_show_border')
        }),
        ('访问权限', {
            'fields': ('_perms_',)
        })
    )

    @admin.display(description='索引')
    def _idx(self, obj):
        return obj.index

    @admin.display(description='用户数')
    def _users_number(self, obj):
        if obj.slug in ['everyone', 'registered']:
            return User.objects.all().count()
        return obj.users.all().count()
    
    @property
    def change_list_template(self):
        return 'admin/%s/%s/change_list.html' % (self.opts.app_label, self.opts.model_name)
    
    @property
    def change_list_results_template(self):
        return 'admin/%s/%s/change_list_results.html' % (self.opts.app_label, self.opts.model_name)
    
    def has_delete_permission(self, request, obj=None):
        if obj and obj.slug  in ['everyone', 'registered']:
            return False
        return super().has_delete_permission(request, obj)
    
    def has_change_permission(self, request, obj=None):
        if obj and not request.user.is_superuser and obj.index < request.user.operation_index:
            return False
        return super().has_change_permission(request, obj)
        
    def get_readonly_fields(self, request, obj=None):
        readonly_fields = self.readonly_fields
        if obj and obj.slug in ['everyone', 'registered']:
            readonly_fields = readonly_fields + ("slug",)
        if not request.user.is_superuser and not request.user.has_perm('roles.manage_permissions'):
            readonly_fields = readonly_fields + ("_perms_",)
        return readonly_fields


class UserReportForm(forms.ModelForm):
    class Meta:
        model = UserReport
        fields = ['status', 'admin_notes']


@admin.register(UserReport)
class UserReportAdmin(admin.ModelAdmin):
    form = UserReportForm
    list_display = ['id', 'reporter_display', 'reported_display', 'status', 'message_count', 'created_at']
    list_filter = ['status', 'created_at']
    search_fields = ['reason', 'reporter__username', 'reported__username', 'admin_notes']
    readonly_fields = [
        'reporter_display', 'reported_display', 'reason', 'reported_messages_preview',
        'created_at', 'reviewed_at', 'reviewed_by', 'full_conversation_link',
    ]
    fieldsets = (
        ('检举信息', {
            'fields': ('reporter_display', 'reported_display', 'created_at', 'reason', 'reported_messages_preview'),
        }),
        ('管理员操作', {
            'fields': ('status', 'admin_notes', 'reviewed_at', 'reviewed_by', 'full_conversation_link'),
        }),
    )
    @admin.display(description='举报人')
    def reporter_display(self, obj):
        if obj.reporter is None:
            return '(已删除)'
        return obj.reporter

    @admin.display(description='被举报人')
    def reported_display(self, obj):
        if obj.reported is None:
            return '(已删除)'
        return obj.reported

    @admin.display(description='涉及消息数', ordering=None)
    def message_count(self, obj):
        return len(obj.reported_messages or [])

    @admin.display(description='被举报的消息')
    def reported_messages_preview(self, obj):
        rows = []
        for msg in obj.reported_messages or []:
            rows.append((msg.get('sender_name', ''), msg.get('created_at', ''), msg.get('body', '')))
        if not rows:
            return '(无)'
        return format_html(
            '<div style="max-width: 720px;">{}</div>',
            format_html_join(
                '',
                '<div style="border:1px solid #ddd;border-radius:4px;padding:8px;margin-bottom:6px;background:#fafafa;">'
                '<div style="font-size:12px;color:#666;margin-bottom:4px;"><strong>{}</strong> · {}</div>'
                '<div style="white-space:pre-wrap;word-break:break-word;">{}</div>'
                '</div>',
                rows,
            ),
        )

    @admin.display(description='查看完整会话')
    def full_conversation_link(self, obj):
        if not obj or not obj.pk:
            return ''
        return format_html(
            '<a href="/api/admin/reports/{}/full-conversation" target="_blank">'
            '打开完整会话 JSON（需要"查看被检举会话全部记录"权限）</a>',
            obj.pk,
        )

    def has_add_permission(self, request):
        return False

    def has_delete_permission(self, request, obj=None):
        return False

    def has_view_permission(self, request, obj=None):
        if request.user.is_superuser:
            return True
        return request.user.has_perm('roles.view_user_reports')

    def has_change_permission(self, request, obj=None):
        if request.user.is_superuser:
            return True
        return request.user.has_perm('roles.view_user_reports')

    def save_model(self, request, obj, form, change):
        if change and 'status' in form.changed_data:
            obj.reviewed_at = timezone.now()
            obj.reviewed_by = request.user
        super().save_model(request, obj, form, change)


class UserTicketAdmin(admin.ModelAdmin):
    kind = None
    review_permission = None

    list_display = ['id', 'author_display', 'subject_display', 'status', 'source_page', 'created_at']
    list_filter = ['status', 'created_at']
    search_fields = ['subject', 'body', 'author__username', 'admin_notes']
    readonly_fields = ['author_display', 'subject_display', 'body_preview', 'source_page', 'created_at', 'reviewed_at', 'reviewed_by']

    @admin.display(description='提交人')
    def author_display(self, obj):
        if obj.author is None:
            return '(已删除)'
        return obj.author

    # 表单可以关掉标题栏，所以列表要有个不空的东西可显示。
    @admin.display(description='标题', ordering='subject')
    def subject_display(self, obj):
        if obj.subject:
            return obj.subject
        body = (obj.body or '').strip().splitlines()
        first = body[0] if body else ''
        return (first[:40] + '…') if len(first) > 40 else (first or '(无标题)')

    @admin.display(description='正文')
    def body_preview(self, obj):
        return format_html(
            '<div style="max-width:720px;white-space:pre-wrap;word-break:break-word;'
            'border:1px solid #ddd;border-radius:4px;padding:8px;background:#fafafa;">{}</div>',
            obj.body,
        )

    def get_queryset(self, request):
        return super().get_queryset(request).filter(kind=self.kind)

    def has_add_permission(self, request):
        return False

    def has_delete_permission(self, request, obj=None):
        return False

    def has_view_permission(self, request, obj=None):
        if request.user.is_superuser:
            return True
        return request.user.has_perm('roles.view_user_tickets')

    def has_change_permission(self, request, obj=None):
        if request.user.is_superuser:
            return True
        return request.user.has_perm(self.review_permission)

    def save_model(self, request, obj, form, change):
        if change and 'status' in form.changed_data:
            obj.reviewed_at = timezone.now()
            obj.reviewed_by = request.user
        super().save_model(request, obj, form, change)


class SupportTicketForm(forms.ModelForm):
    class Meta:
        model = SupportTicket
        fields = ['status', 'admin_notes']


@admin.register(SupportTicket)
class SupportTicketAdmin(UserTicketAdmin):
    form = SupportTicketForm
    kind = UserTicket.Kind.Ticket
    review_permission = 'roles.view_user_tickets'
    fieldsets = (
        ('工单内容', {
            'fields': ('author_display', 'subject_display', 'created_at', 'source_page', 'body_preview'),
        }),
        ('管理员操作', {
            'fields': ('status', 'admin_notes', 'reviewed_at', 'reviewed_by'),
        }),
    )


class MembershipApplicationForm(forms.ModelForm):
    class Meta:
        model = MembershipApplication
        fields = ['status', 'granted_role', 'admin_notes']


@admin.register(MembershipApplication)
class MembershipApplicationAdmin(UserTicketAdmin):
    form = MembershipApplicationForm
    kind = UserTicket.Kind.MembershipApply
    review_permission = 'roles.review_membership_applications'
    list_display = ['id', 'author_display', 'subject_display', 'status', 'granted_role', 'created_at']
    list_filter = ['status', 'granted_role', 'created_at']
    fieldsets = (
        ('申请内容', {
            'fields': ('author_display', 'subject_display', 'created_at', 'source_page', 'body_preview'),
        }),
        ('管理员操作', {
            'fields': ('status', 'granted_role', 'admin_notes', 'reviewed_at', 'reviewed_by'),
        }),
    )

    def save_model(self, request, obj, form, change):
        super().save_model(request, obj, form, change)
        # 通过就当场把角色发下去，否则管理员还得再去用户页点一次，两处状态就会对不上。
        if obj.status == UserTicket.Status.Approved and obj.granted_role and obj.author:
            obj.author.roles.add(obj.granted_role)


@admin.register(InviteLink)
class InviteLinkAdmin(admin.ModelAdmin):
    list_display = ['id', 'kind', 'delivery', 'recipient', 'state', 'activated_username', 'created_by', 'created_at', 'activated_at']
    list_filter = ['kind', 'delivery', 'created_at']
    search_fields = ['email', 'wikidot_username', 'activated_username', 'token']
    readonly_fields = [
        'kind', 'delivery', 'recipient', 'state', 'link', 'target',
        'created_by', 'created_at', 'activated_username', 'activated_at',
    ]
    fieldsets = (
        ('链接', {
            'fields': ('kind', 'delivery', 'recipient', 'link'),
        }),
        ('状态', {
            'fields': ('state', 'activated_username', 'activated_at'),
        }),
        ('来源', {
            'fields': ('target', 'created_by', 'created_at'),
        }),
    )

    @admin.display(description='发给谁')
    def recipient(self, obj):
        return obj.email or obj.wikidot_username or obj.target or '(已删除)'

    @admin.display(description='是否激活', boolean=True, ordering='activated_at')
    def state(self, obj):
        return obj.is_activated

    # 链接只有完整可复制才有用，所以这里给出带域名的那一份而不是路径。
    @admin.display(description='链接')
    def link(self, obj):
        url = invite_url(self.request, obj) if getattr(self, 'request', None) else obj.path
        return format_html('<input class="vTextField" style="width: 100%;" readonly value="{}">', url)

    def get_queryset(self, request):
        self.request = request
        return super().get_queryset(request)

    def has_add_permission(self, request):
        return False

    def has_view_permission(self, request, obj=None):
        if request.user.is_superuser:
            return True
        return request.user.has_perm('roles.manage_users')

    def has_change_permission(self, request, obj=None):
        return False

    def has_delete_permission(self, request, obj=None):
        if request.user.is_superuser:
            return True
        return request.user.has_perm('roles.manage_users')
