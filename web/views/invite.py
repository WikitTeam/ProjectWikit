from django.db import IntegrityError
from django.utils.http import urlsafe_base64_encode
from django.contrib.admin.views.decorators import staff_member_required
from django.contrib.auth.tokens import PasswordResetTokenGenerator
from django.utils.encoding import force_bytes
from django.core.mail import send_mail, BadHeaderError
from django.utils.decorators import method_decorator
from django.template.loader import render_to_string
from django.shortcuts import redirect, resolve_url
from django.contrib.auth import get_user_model
from django.views.generic import FormView
from django.contrib.admin import site
from django.contrib import messages

from web.forms import ClaimLinkForm, InviteForm
from web.models.invites import InviteLink
from web.models.site import get_current_site


User = get_user_model()


class TokenGenerator(PasswordResetTokenGenerator):
    def _make_hash_value(self, user, timestamp):
        return 'v2:' + str(user.pk) + str(timestamp) + str(user.is_active)


account_activation_token = TokenGenerator()


@method_decorator(staff_member_required, name='dispatch')
class InviteView(FormView):
    form_class = InviteForm
    template_name = "admin/web/user/user_action.html"
    email_template_name = "mails/invite_email.txt"
    user = None

    def get_initial(self):
        initial = super().get_initial()
        return initial

    def get_user(self) -> User | None:
        user_id = self.kwargs.get('id') or None
        if user_id:
            return User.objects.get(pk=user_id)
        return None

    def get_context_data(self, **kwargs):
        context = super().get_context_data(**kwargs)
        context["title"] = "邀请用户"
        context["submit_btn"] = "发送"
        user = self.get_user()
        if user:
            context["title"] = "激活 wd 用户：%s" % user.wikidot_username
            context["submit_btn"] = "激活"
        context.update(site.each_context(self.request))
        return context

    def get_success_url(self):
        return resolve_url("admin:index")

    def form_valid(self, form):
        email = form.cleaned_data['email']
        roles = form.cleaned_data['roles']
        user = self.get_user()
        if user:
            created = not user.email
        else:
            try:
                user, created = User.objects.get_or_create(email=email)
            except IntegrityError:
                created = False
        site = get_current_site()
        if created:
            user.roles.set(roles)
            user.is_active = False
            user.username = 'user-%d' % user.id
            user.email = email
            user.save()
            subject = f"邀请加入 {site.title}"
            c = {
                "email": user.email,
                'domain': self.request.get_host(),
                'site_name': site.title,
                "uid": urlsafe_base64_encode(force_bytes(user.pk)),
                "user": user,
                'token': account_activation_token.make_token(user),
                'protocol': self.request.scheme,
            }
            content = render_to_string(self.email_template_name, c)
            try:
                send_mail(subject, content, None, [user.email], fail_silently=False)
                record_invite_link(
                    self.request, user,
                    kind=InviteLink.Kind.Claim if user.type == User.UserType.Wikidot else InviteLink.Kind.Register,
                    delivery=InviteLink.Delivery.Email,
                    token=c['token'], uidb64=c['uid'],
                )
                messages.success(self.request, "邀请已成功发送")
            except BadHeaderError:
                messages.error(self.request, "邮件头格式错误")
        else:
            messages.error(self.request, "此邮箱已被站点成员使用")

        return redirect(self.get_success_url())


def record_invite_link(request, user, kind, delivery, token, uidb64):
    return InviteLink.objects.create(
        kind=kind,
        delivery=delivery,
        target=user,
        email=user.email or '',
        wikidot_username=user.wikidot_username or '',
        token=token,
        uidb64=uidb64,
        created_by=request.user if request.user.is_authenticated else None,
    )


def invite_url(request, link):
    site = get_current_site()
    host = site.domain if site else request.get_host()
    return '%s://%s%s' % (request.scheme, host, link.path)


@method_decorator(staff_member_required, name='dispatch')
class GenerateInviteLinkView(FormView):
    form_class = InviteForm
    template_name = "admin/web/user/user_action.html"

    def get_context_data(self, **kwargs):
        context = super().get_context_data(**kwargs)
        context["title"] = "创建邀请链接"
        context["submit_btn"] = "生成链接"
        context.update(site.each_context(self.request))
        return context

    # 链接以前只活在一条一刷新就没的提示里，现在落进表，所以生成完直接送到那张表上。
    def get_success_url(self):
        return resolve_url("admin:web_invitelink_changelist")

    def form_valid(self, form):
        email = form.cleaned_data['email']
        roles = form.cleaned_data['roles']

        try:
            user, created = User.objects.get_or_create(email=email)
            if not created:
                messages.error(self.request, "该邮箱已被站点成员使用")
                return redirect(resolve_url("admin:index"))

            user.roles.set(roles)
            user.is_active = False
            user.username = f'user-{user.id}'
            user.email = email
            user.save()

            link = record_invite_link(
                self.request, user,
                kind=InviteLink.Kind.Register,
                delivery=InviteLink.Delivery.Link,
                token=account_activation_token.make_token(user),
                uidb64=urlsafe_base64_encode(force_bytes(user.pk)),
            )
            messages.success(self.request, "邀请链接已生成：%s" % invite_url(self.request, link))

        except IntegrityError:
            messages.error(self.request, "创建用户失败")
            return redirect(resolve_url("admin:index"))

        return redirect(self.get_success_url())


@method_decorator(staff_member_required, name='dispatch')
class GenerateClaimLinkView(FormView):
    form_class = ClaimLinkForm
    template_name = "admin/web/user/user_action.html"

    def get_context_data(self, **kwargs):
        context = super().get_context_data(**kwargs)
        context["title"] = "创建认领链接"
        context["submit_btn"] = "生成链接"
        context["after_text"] = "只列出尚未被认领的 Wikidot 账号。认领后账号会转为普通账号，用户名沿用原 Wikidot 用户名。"
        context.update(site.each_context(self.request))
        return context

    def get_success_url(self):
        return resolve_url("admin:web_invitelink_changelist")

    def form_valid(self, form):
        user = form.cleaned_data['user']
        roles = form.cleaned_data['roles']
        if roles:
            user.roles.set(roles)

        link = record_invite_link(
            self.request, user,
            kind=InviteLink.Kind.Claim,
            delivery=InviteLink.Delivery.Link,
            token=account_activation_token.make_token(user),
            uidb64=urlsafe_base64_encode(force_bytes(user.pk)),
        )
        messages.success(self.request, "认领链接已生成：%s" % invite_url(self.request, link))
        return redirect(self.get_success_url())
