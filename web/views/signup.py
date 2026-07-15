from django.conf import settings
from django.contrib.auth.models import AnonymousUser
from django.contrib.auth.models import AbstractUser as _UserType
from django.utils.http import urlsafe_base64_decode
from django.utils.encoding import force_str
from django.contrib.auth import get_user_model
from django.http import HttpRequest, HttpResponseRedirect, JsonResponse
from django.contrib.auth import login
from django.views.generic.base import TemplateResponseMixin, ContextMixin, View
from django.views import View as BaseView

import re
import requests

from web.models.users import UsedToken, canonicalize_username
from .invite import account_activation_token
from web.events import EventBase
from django.shortcuts import redirect
from web.models.roles import Role


User = get_user_model()
WIKIT_VERIFY_API = "https://wikit.unitreaty.org/projwikit"


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
            raw = request.POST.get('username', '').strip()
            if not re.match(r'^[\w -]+\Z', raw, re.ASCII):
                context.update({'username': raw, 'error': '用户名只能包含英文字母、数字、空格及符号 [_-]。'})
                return self.render_to_response(context)
            username = canonicalize_username(raw)  # 归一为身份用户名
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
        username = canonicalize_username(request.POST.get('username', '').strip())
        if not username:
            return JsonResponse({'ok': False, 'error': '用户名不能为空'})

        # 再次确认是待认领的Wikidot账号
        wd_user = User.objects.filter(wikidot_username=username, type=User.UserType.Wikidot).first()
        if not wd_user:
            return JsonResponse({'ok': False, 'error': '该用户名不是待认领的 Wikidot 账号'})

        # 外部验证服务按原始大小写的 Wikidot 名(full_name/显示名)识别，不能发小写身份名
        verify_name = wd_user.display_name or wd_user.wikidot_username

        try:
            r = requests.post(
                f"{WIKIT_VERIFY_API}/send",
                data={'user': verify_name},
                timeout=5
            )
            data = r.json()
            if data.get('status') == 'success':
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

        raw_username = data.get('username', '').strip()
        password = data.get('password', '')
        password_confirm = data.get('password_confirm', '')

        # 输入按显示名校验（允许字母/数字/空格/_/-）
        if not re.match(r'^[\w -]+\Z', raw_username, re.ASCII):
            context.update({'error': '用户名只能包含英文字母、数字、空格及符号 [_-]。'})
            return self.render_to_response(context)

        # 归一为身份用户名（小写，空格/_ 转 -）；带展示价值时保留原样为显示名
        username = canonicalize_username(raw_username)
        display = raw_username if raw_username != username else None

        # 密码一致性校验
        if password != password_confirm:
            context.update({'error': '两次输入的密码不一致'})
            return self.render_to_response(context)

        # 判断是否为待认领的Wikidot账号
        wikidot_user = User.objects.filter(
            wikidot_username=username,
            type=User.UserType.Wikidot
        ).first()

        if wikidot_user:
            # Wikidot账号认领流程（显示名沿用迁移时的 full_name，不覆盖）
            code = data.get('verification_code', '').strip()
            # 外部验证服务按原始大小写的 Wikidot 名识别，与发送验证码时保持一致
            verify_name = wikidot_user.display_name or wikidot_user.wikidot_username
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
            wikidot_user.set_password(password)
            wikidot_user.is_active = True
            wikidot_user.save()

            login(request, wikidot_user, backend='django.contrib.auth.backends.ModelBackend')
            OnUserSignUp(request, wikidot_user).emit()
            return redirect('/')

        else:
            # 普通注册流程
            if User.objects.filter(username=username).exists():
                context.update({'error': '用户名已被使用'})
                return self.render_to_response(context)

            user = User.objects.create_user(username=username)
            if display:
                user.display_name = display
            reader_role = Role.objects.get(slug='reader')
            user.roles.add(reader_role)
            user.set_password(password)
            user.is_active = True
            user.save()

            login(request, user, backend='django.contrib.auth.backends.ModelBackend')
            OnUserSignUp(request, user).emit()
            return redirect('/')
