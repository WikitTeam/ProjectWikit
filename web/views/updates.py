from django.contrib import admin
from django.http import JsonResponse
from django.shortcuts import redirect
from django.views import View
from django.views.generic import TemplateView
from django.utils.decorators import method_decorator
from django.views.decorators.http import require_POST

from web.util import updates


PERMISSION = 'roles.manage_updates'


def _can_manage(request):
    return request.user.is_authenticated and request.user.has_perm(PERMISSION)


class UpdatesAdminView(TemplateView):
    template_name = 'admin/updates.html'

    def dispatch(self, request, *args, **kwargs):
        if not _can_manage(request):
            return redirect('admin:index')
        return super().dispatch(request, *args, **kwargs)

    def get_context_data(self, **kwargs):
        ctx = super().get_context_data(**kwargs)
        ctx.update(admin.site.each_context(self.request))
        ctx['overview'] = updates.get_overview()
        return ctx


class UpdatesStatusView(View):
    def get(self, request, *args, **kwargs):
        if not _can_manage(request):
            return JsonResponse({'error': 'forbidden'}, status=403)
        data = updates.get_overview()
        data['log'] = updates.get_log_tail()
        return JsonResponse(data)


@method_decorator(require_POST, name='dispatch')
class UpdatesTriggerView(View):
    def post(self, request, *args, **kwargs):
        if not _can_manage(request):
            return JsonResponse({'error': 'forbidden'}, status=403)
        ok, message = updates.request_update(request.user)
        return JsonResponse({'ok': ok, 'message': message}, status=200 if ok else 409)
