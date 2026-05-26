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

from web.models.users import UsedToken
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
        if user.type == User.UserType.Wikidot:
            username = user.wikidot_username
            context.update({'is_wikidot': True})
        else:
            username = request.POST.get('username', '').strip()
        context.update({'username': username})
        password1 = request.POST.get('password', '')
        password2 = request.POST.get('password2', '')
        if not re.match(r"^[\w.-]+\Z", username, re.ASCII):
            context.update({'error': '无效用户名。允许的字符：A-Z、a-z、0-9、-、_。'})
            return self.render_to_response(context)
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
        username = request.GET.get('username', '').strip()
        if not username:
            return JsonResponse({'is_wikidot': False})
        is_wikidot = User.objects.filter(
            wikidot_username=username,
            type=User.UserType.Wikidot
        ).exists()
        return JsonResponse({'is_wikidot': is_wikidot})


class SendWikidotCodeView(BaseView):
    def post(self, request, *args, **kwargs):
        username = request.POST.get('username', '').strip()
        if not username:
            return JsonResponse({'ok': False, 'error': '用户名不能为空'})

        # 再次确认是待认领的Wikidot账号
        if not User.objects.filter(wikidot_username=username, type=User.UserType.Wikidot).exists():
            return JsonResponse({'ok': False, 'error': '该用户名不是待认领的 Wikidot 账号'})

        try:
            r = requests.post(
                f"{WIKIT_VERIFY_API}/send",
                data={'user': username},
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

        username = data.get('username', '').strip()
        password = data.get('password', '')
        password_confirm = data.get('password_confirm', '')

        # 用户名合法性检查
        if not re.match(r'^[\w.-]+\Z', username, re.ASCII):
            context.update({'error': '用户名只能包含英文字母、数字及符号 [.-_]（不含括号）。'})
            return self.render_to_response(context)

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
            # Wikidot账号认领流程
            code = data.get('verification_code', '').strip()
            ok, err = self._verify_wikidot_code(username, code)
            if not ok:
                context.update({
                    'error': err,
                    'is_wikidot': True,
                    'prefill_username': username,
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
            reader_role = Role.objects.get(slug='reader')
            user.roles.add(reader_role)
            user.set_password(password)
            user.is_active = True
            user.save()

            login(request, user, backend='django.contrib.auth.backends.ModelBackend')
            OnUserSignUp(request, user).emit()
            return redirect('/')
