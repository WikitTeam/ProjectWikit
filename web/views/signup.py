from django.conf import settings
from django.contrib.auth.models import AnonymousUser
from django.contrib.auth.models import AbstractUser as _UserType
from django.utils.http import urlsafe_base64_decode
from django.utils.encoding import force_str
from django.utils import timezone
from django.contrib.auth import get_user_model
from django.http import HttpRequest, HttpResponseRedirect, JsonResponse
from django.contrib.auth import login
from django.views.generic.base import TemplateResponseMixin, ContextMixin, View
from django.views import View as BaseView

import requests

from uuid import uuid4

from django.contrib.auth.password_validation import validate_password
from django.core.exceptions import ValidationError
from django.db import transaction

from web.models.users import (
    RESERVED_USERNAME_RE,
    UsedToken,
    allocate_fallback_username,
    canonicalize_username,
    normalize_display_name,
    validate_display_name,
)
from .invite import account_activation_token
from web.events import EventBase
from django.shortcuts import redirect
from web.models.invites import InviteLink
from web.models.roles import Role
from web.models.site import get_current_site


User = get_user_model()
WIKIT_VERIFY_API = "https://wikit.unitreaty.org/projwikit"

WD_VERIFY_NAME_KEY = 'wd_verify_name'
WD_VERIFY_FOR_KEY = 'wd_verify_for'


class OnUserSignUp(EventBase, name='on_user_signup'):
    request: HttpRequest
    user: _UserType


class AcceptInvitationView(TemplateResponseMixin, ContextMixin, View):
    template_name = "signup/accept.html"

    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)

    def get_user(self):
        try:
            uid = force_str(urlsafe_base64_decode(self.kwargs["uidb64"]))
            user = User.objects.get(pk=uid)
        except (TypeError, ValueError, OverflowError, User.DoesNotExist):
            user = None
        return user

    def get(self, request, *args, **kwargs):
        if not isinstance(request.user, AnonymousUser):
            return HttpResponseRedirect(redirect_to=settings.LOGIN_REDIRECT_URL)
        path = request.META['RAW_PATH'][1:]
        context = self.get_context_data(path=path)
        user = self.get_user()
        if UsedToken.is_used(self.kwargs['token']) or not account_activation_token.check_token(user, self.kwargs["token"]):
            context.update({'error': '无效邀请。', 'error_fatal': True})
            return self.render_to_response(context)
        if user.type == User.UserType.Wikidot:
            context.update({'is_wikidot': True, 'username': user.wikidot_username})
        return self.render_to_response(context)

    def post(self, request, *args, **kwargs):
        path = request.META['RAW_PATH'][1:]
        context = self.get_context_data(path=path)
        user = self.get_user()
        if UsedToken.is_used(self.kwargs['token']) or not account_activation_token.check_token(user, self.kwargs['token']):
            context.update({'error': '无效邀请。', 'error_fatal': True})
            return self.render_to_response(context)
        display = None
        if user.type == User.UserType.Wikidot:
            username = user.wikidot_username  # 已是规范身份，显示名沿用迁移时的 full_name
            context.update({'is_wikidot': True})
        else:
            raw = normalize_display_name(request.POST.get('username', ''))
            try:
                validate_display_name(raw)
            except ValidationError as e:
                context.update({'username': raw, 'error': e.messages[0]})
                return self.render_to_response(context)
            username = canonicalize_username(raw)  # 归一为身份用户名
            if RESERVED_USERNAME_RE.match(username):
                context.update({'username': raw, 'error': '该用户名是保留形式，请换一个。'})
                return self.render_to_response(context)
            if not username:
                # 纯符号/表情折不出身份名
                username = allocate_fallback_username(user.pk)
            display = raw if raw != username else None
        context.update({'username': username})
        password1 = request.POST.get('password', '')
        password2 = request.POST.get('password2', '')
        user_exists = User.objects.filter(username=username)
        wd_user_exists = User.objects.filter(wikidot_username=username)
        if (user_exists and user_exists[0] != user) or (wd_user_exists and wd_user_exists[0] != user):
            context.update({'error': '所选用户名已被使用。'})
            return self.render_to_response(context)
        if not password1:
            context.update({'error': '必须填写密码。'})
            return self.render_to_response(context)
        if password1 != password2:
            context.update({'error': '两次输入的密码不一致。'})
            return self.render_to_response(context)
        try:
            validate_password(password1, user)
        except ValidationError as e:
            context.update({'error': ' '.join(e.messages)})
            return self.render_to_response(context)
        if user.type != User.UserType.Wikidot:
            user.username = username
            if display:
                user.display_name = display
        else:
            user.username = user.wikidot_username
            user.type = User.UserType.Normal
        user.set_password(password1)
        user.is_active = True
        user.save()
        UsedToken.mark_used(self.kwargs['token'], is_case_sensitive=True)
        InviteLink.mark_activated(self.kwargs['token'], user, timezone.now())
        login(request, user, backend='django.contrib.auth.backends.ModelBackend')
        OnUserSignUp(request, user).emit()
        return HttpResponseRedirect(redirect_to=settings.LOGIN_REDIRECT_URL)


class CheckWikidotUsernameView(BaseView):
    def get(self, request, *args, **kwargs):
        username = canonicalize_username(request.GET.get('username', '').strip())
        if not username:
            return JsonResponse({'is_wikidot': False})
        is_wikidot = User.objects.filter(
            wikidot_username=username,
            type=User.UserType.Wikidot
        ).exists()
        return JsonResponse({'is_wikidot': is_wikidot})


class SendWikidotCodeView(BaseView):
    def post(self, request, *args, **kwargs):
        raw_username = request.POST.get('username', '').strip()
        username = canonicalize_username(raw_username)
        if not username:
            return JsonResponse({'ok': False, 'error': '用户名不能为空'})

        # 再次确认是待认领的Wikidot账号
        wd_user = User.objects.filter(wikidot_username=username, type=User.UserType.Wikidot).first()
        if not wd_user:
            return JsonResponse({'ok': False, 'error': '该用户名不是待认领的 Wikidot 账号'})

        # 外部服务按 Wikidot 原名查人，发身份名查不到
        verify_name = raw_username

        try:
            r = requests.post(
                f"{WIKIT_VERIFY_API}/send",
                data={'user': verify_name},
                timeout=5
            )
            data = r.json()
            if data.get('status') == 'success':
                # 外部服务拿这个字符串当验证码的 key
                request.session[WD_VERIFY_NAME_KEY] = verify_name
                request.session[WD_VERIFY_FOR_KEY] = username
                return JsonResponse({'ok': True})
            else:
                return JsonResponse({'ok': False, 'error': data.get('message', '发送失败，请稍后重试')})
        except Exception:
            return JsonResponse({'ok': False, 'error': '无法连接到验证服务，请稍后重试'})


class SignupView(TemplateResponseMixin, ContextMixin, View):
    template_name = "signup/register.html"

    def get(self, request, *args, **kwargs):
        if not isinstance(request.user, AnonymousUser):
            return HttpResponseRedirect(redirect_to=settings.LOGIN_REDIRECT_URL)
        context = self.get_context_data()
        return self.render_to_response(context)

    def _verify_wikidot_code(self, username, code):
        if not code:
            return False, '请输入验证码'
        try:
            r = requests.post(
                f"{WIKIT_VERIFY_API}/verify",
                data={'user': username, 'code': code},
                timeout=5
            )
            data = r.json()
            if data.get('status') == 'success':
                return True, None
            return False, data.get('message', '验证码错误，请重试')
        except Exception:
            return False, '无法连接到验证服务，请稍后重试'

    def post(self, request, *args, **kwargs):
        context = self.get_context_data()
        data = request.POST

        raw_username = normalize_display_name(data.get('username', ''))
        password = data.get('password', '')
        password_confirm = data.get('password_confirm', '')

        # 认领时要能原样填 Wikidot 上的名字，所以只挡不可见字符
        try:
            validate_display_name(raw_username)
        except ValidationError as e:
            context.update({'error': e.messages[0], 'prefill_username': raw_username})
            return self.render_to_response(context)

        username = canonicalize_username(raw_username)
        display = raw_username if raw_username != username else None

        if RESERVED_USERNAME_RE.match(username):
            context.update({'error': '该用户名是保留形式，请换一个。', 'prefill_username': raw_username})
            return self.render_to_response(context)

        # 放在密码校验前：重渲染时要知道验证码区域展不展开
        wikidot_user = User.objects.filter(
            wikidot_username=username,
            type=User.UserType.Wikidot
        ).first() if username else None

        # 密码一致性校验
        if password != password_confirm:
            context.update({
                'error': '两次输入的密码不一致',
                'prefill_username': raw_username,
                'is_wikidot': bool(wikidot_user),
            })
            return self.render_to_response(context)

        try:
            validate_password(password, wikidot_user or User(username=username, display_name=display))
        except ValidationError as e:
            context.update({
                'error': ' '.join(e.messages),
                'prefill_username': raw_username,
                'is_wikidot': bool(wikidot_user),
            })
            return self.render_to_response(context)

        if wikidot_user:
            # Wikidot账号认领流程
            code = data.get('verification_code', '').strip()
            # session 没记录(过期/换标签页)才退回当前输入
            if request.session.get(WD_VERIFY_FOR_KEY) == username:
                verify_name = request.session.get(WD_VERIFY_NAME_KEY) or raw_username
            else:
                verify_name = raw_username
            ok, err = self._verify_wikidot_code(verify_name, code)
            if not ok:
                context.update({
                    'error': err,
                    'is_wikidot': True,
                    'prefill_username': raw_username,
                })
                return self.render_to_response(context)

            wikidot_user.username = username
            wikidot_user.type = User.UserType.Normal
            # 迁移时没带上 full_name 的账号在这里补
            if not wikidot_user.display_name and verify_name != username:
                wikidot_user.display_name = verify_name
            wikidot_user.set_password(password)
            wikidot_user.is_active = True
            wikidot_user.save()

            request.session.pop(WD_VERIFY_NAME_KEY, None)
            request.session.pop(WD_VERIFY_FOR_KEY, None)

            verified_role = _signup_role('verified_role')
            if verified_role:
                wikidot_user.roles.add(verified_role)

            login(request, wikidot_user, backend='django.contrib.auth.backends.ModelBackend')
            OnUserSignUp(request, wikidot_user).emit()
            return redirect('/')

        else:
            # 普通注册流程
            if username and User.objects.filter(username=username).exists():
                context.update({
                    # 大小写、空格、点都折成同一个身份名，不说清楚会显得莫名其妙
                    'error': '用户名已被使用（%s 与已有用户的身份名相同）' % username,
                    'prefill_username': raw_username,
                })
                return self.render_to_response(context)

            with transaction.atomic():
                # 兜底名要先有 pk 才能拼，先占一个临时的
                user = User.objects.create_user(username=username or str(uuid4()))
                if not username:
                    user.username = allocate_fallback_username(user.pk)
                if display:
                    user.display_name = display
                default_role = _signup_role('default_role')
                if default_role:
                    user.roles.add(default_role)
                user.set_password(password)
                user.is_active = True
                user.save()

            login(request, user, backend='django.contrib.auth.backends.ModelBackend')
            OnUserSignUp(request, user).emit()
            return redirect('/')


# 站点没配就回落到 reader，而 reader 也可能不存在（全新站点），所以两层都不能硬取。
def _signup_role(field):
    site = get_current_site()
    role = getattr(site, field, None) if site else None
    if role is not None:
        return role
    return Role.objects.filter(slug='reader').first()
