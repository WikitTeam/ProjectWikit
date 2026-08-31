from django.http import HttpRequest, HttpResponse, HttpResponseRedirect
from django.utils.http import urlencode
from django.views import View

from web.models.roles import Role
from web.models.site import get_current_site
from web.models.tickets import UserTicket


MAX_SUBJECT_LENGTH = 200
MAX_BODY_LENGTH = 20000


def _back(request: HttpRequest, key: str, outcome: str) -> HttpResponse:
    # 文章渲染只看得见路径参数，查询串到不了模块手里。
    page = (request.POST.get('page') or '').strip().strip('/')
    if not page:
        return HttpResponseRedirect('/')
    return HttpResponseRedirect('/%s/%s/%s' % (page, key, outcome))


class SubmitTicketView(View):
    def post(self, request: HttpRequest):
        if not request.user.is_authenticated:
            return _back(request, 'applied', 'failed')

        body = (request.POST.get('body') or '').strip()
        if not body or len(body) > MAX_BODY_LENGTH:
            return _back(request, 'applied', 'failed')

        kind = request.POST.get('kind')
        if kind not in UserTicket.Kind.values:
            kind = UserTicket.Kind.Ticket

        UserTicket.objects.create(
            kind=kind,
            author=request.user,
            subject=(request.POST.get('subject') or '').strip()[:MAX_SUBJECT_LENGTH],
            body=body,
            source_page=(request.POST.get('page') or '').strip(),
        )
        return _back(request, 'applied', 'ok')


class MembershipByPasswordView(View):
    def post(self, request: HttpRequest):
        if not request.user.is_authenticated:
            return _back(request, 'membership', 'failed')

        site = get_current_site()
        if not site or not site.membership_password_enabled or not site.membership_password:
            return _back(request, 'membership', 'failed')
        if (request.POST.get('password') or '') != site.membership_password:
            return _back(request, 'membership', 'failed')

        role = site.membership_password_role
        if role is None:
            return _back(request, 'membership', 'failed')
        request.user.roles.add(role)
        return _back(request, 'membership', 'ok')
