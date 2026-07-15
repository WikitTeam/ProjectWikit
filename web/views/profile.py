from django.contrib.auth.mixins import LoginRequiredMixin
from django.views.generic import DetailView, UpdateView
from django.shortcuts import resolve_url, redirect
from django.conf import settings
from django.urls import reverse
from django.db.models import Q
from django.http import Http404

from renderer import single_pass_render
from renderer.parser import RenderContext
from web.forms import UserProfileForm
from web.models.messages import DirectMessageBlock
from web.models.users import User, canonicalize_username


class ProfileView(DetailView):
    model = User
    slug_field = "username"

    def get_context_data(self, **kwargs):
        ctx = super().get_context_data(**kwargs)
        user = self.get_object(self.get_queryset())

        if user.type == User.UserType.Wikidot:
            ctx['avatar'] = settings.WIKIDOT_AVATAR
            ctx['displayname'] = 'wd:'+(user.display_name or user.wikidot_username)
        else:
            ctx['avatar'] = user.get_avatar(default=settings.DEFAULT_AVATAR)
            ctx['displayname'] = user.display_name or user.username
        
        ctx['subtitle'] = ', '.join(user.showcase['titles'])
        ctx['bio_rendered'] = single_pass_render(user.bio, RenderContext(article=None, source_article=None, path_params=None, user=self.request.user), 'inline')

        viewer = self.request.user
        if viewer.is_authenticated and viewer.id != user.id:
            ctx['can_direct_message'] = True
            ctx['is_blocked'] = DirectMessageBlock.objects.filter(blocker=viewer, blocked=user).exists()
        else:
            ctx['can_direct_message'] = False
            ctx['is_blocked'] = False
        return ctx

    def get(self, request, *args, **kwargs):
        self.object = self.get_object()
        context = self.get_context_data(object=self.object)
        if request.user.is_staff and request.user.has_perm('roles.manage_users'):
            context['can_manage_users'] = True
            context['manage_url'] = '%s?_popup=1' % reverse(
                'admin:%s_%s_change' % (self.object._meta.app_label, self.object._meta.model_name),
                args=[self.object.pk],
            )
        return self.render_to_response(context)

    def get_object(self, queryset=None):
        # 按用户名访问：把输入(可能带空格/下划线/大小写)归一后，匹配 username 或 wikidot_username
        if 'name' in self.kwargs:
            canon = canonicalize_username(self.kwargs['name'])
            qs = queryset if queryset is not None else self.get_queryset()
            user = qs.filter(Q(username=canon) | Q(wikidot_username=canon)).first()
            if user is None:
                raise Http404('用户不存在')
            return user
        return super().get_object(queryset=queryset)


class ChangeProfileView(LoginRequiredMixin, UpdateView):
    form_class = UserProfileForm
    redirect_field_name = 'to'

    def get_success_url(self):
        return "/-/profile/edit"

    def get_object(self, queryset=None):
        return self.request.user

    def get_form(self, form_class=None):
        form = super().get_form(form_class=form_class)
        form.fields['bio'].label += ' （支持Wiki语法）'
        return form