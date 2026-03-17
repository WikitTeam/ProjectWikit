from django.contrib.admin.views.decorators import staff_member_required
from django.utils.decorators import method_decorator
from django.shortcuts import redirect, resolve_url
from django.contrib.auth import get_user_model
from django.views.generic import FormView
from django.contrib.admin import site
from django.contrib import messages

from web.forms import CreateBotForm


User = get_user_model()


@method_decorator(staff_member_required, name='dispatch')
class CreateBotView(FormView):
    form_class = CreateBotForm
    template_name = "admin/web/user/user_action.html"

    def get_initial(self):
        initial = super(CreateBotView, self).get_initial()
        return initial

    def get_context_data(self, **kwargs):
        context = super(CreateBotView, self).get_context_data(**kwargs)
        context["title"] = "创建机器人"
        context["submit_btn"] = "创建"
        context.update(site.each_context(self.request))
        return context

    def get_success_url(self):
        return resolve_url("admin:index")

    def form_valid(self, form):
        username = form.cleaned_data['username']
        if not User.objects.filter(username=username).exists():
            bot = User(username=username, type="bot")
            bot.save()
            messages.success(self.request, "机器人创建成功")
        else:
            messages.error(self.request, "用户名已被占用")
        return redirect(self.get_success_url())