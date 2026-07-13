from datetime import timedelta

from django.contrib.auth import get_user_model
from django.db.models import Q
from django.http import HttpRequest
from django.utils import timezone

from renderer.utils import render_user_to_json
from web.models.messages import DirectMessage
from web.models.reports import UserReport
from web.views.api import APIError, APIView, takes_json


User = get_user_model()

MAX_REASON_LENGTH = 2000
MAX_MESSAGES_PER_REPORT = 100
MAX_REPORTS_PER_TARGET_24H = 5


def _require_login(request: HttpRequest):
    if not request.user or not request.user.is_authenticated:
        raise APIError('请先登录', 401)


class SubmitReportView(APIView):
    @takes_json
    def post(self, request: HttpRequest):
        _require_login(request)
        data = self.json_input or {}

        reported_id = data.get('reported_id')
        message_ids = data.get('message_ids') or []
        reason = (data.get('reason') or '').strip()

        if not reported_id:
            raise APIError('缺少被检举人', 400)
        if not isinstance(message_ids, list) or not message_ids:
            raise APIError('请至少选择一条消息', 400)
        if len(message_ids) > MAX_MESSAGES_PER_REPORT:
            raise APIError(f'单次最多检举 {MAX_MESSAGES_PER_REPORT} 条消息', 400)
        if not reason:
            raise APIError('请填写检举理由', 400)
        if len(reason) > MAX_REASON_LENGTH:
            raise APIError(f'检举理由不能超过 {MAX_REASON_LENGTH} 个字符', 400)
        if reported_id == request.user.id:
            raise APIError('不能检举自己', 400)

        reported = User.objects.filter(id=reported_id).first()
        if not reported:
            raise APIError('被检举人不存在', 404)

        since = timezone.now() - timedelta(hours=24)
        recent_count = UserReport.objects.filter(
            reporter=request.user, reported=reported, created_at__gte=since,
        ).count()
        if recent_count >= MAX_REPORTS_PER_TARGET_24H:
            raise APIError('24 小时内对同一用户的检举次数过多，请稍后再试', 429)

        messages = list(
            DirectMessage.objects
            .filter(id__in=message_ids)
            .filter(
                Q(sender=request.user, recipient=reported)
                | Q(sender=reported, recipient=request.user)
            )
            .select_related('sender')
            .order_by('created_at')
        )
        if len(messages) != len(set(message_ids)):
            raise APIError('部分消息无效或不属于该会话', 400)

        snapshots = [{
            'id': m.id,
            'sender_id': m.sender_id,
            'sender_name': str(m.sender) if m.sender else '(已删除)',
            'body': m.body,
            'created_at': m.created_at.isoformat(),
        } for m in messages]

        report = UserReport.objects.create(
            reporter=request.user,
            reported=reported,
            reason=reason,
            reported_messages=snapshots,
        )

        return self.render_json(200, {'status': 'ok', 'report_id': report.id})


class AdminReportFullConversationView(APIView):
    def get(self, request: HttpRequest, report_id: int):
        _require_login(request)

        if not request.user.has_perm('roles.view_user_reports'):
            raise APIError('权限不足', 403)
        if not request.user.has_perm('roles.view_reported_full_conversation'):
            raise APIError('权限不足', 403)

        report = UserReport.objects.filter(id=report_id).first()
        if not report:
            raise APIError('检举不存在', 404)
        if not report.reporter or not report.reported:
            raise APIError('会话双方之一已被删除，无法查看完整记录', 410)

        messages = list(
            DirectMessage.objects
            .filter(
                Q(sender=report.reporter, recipient=report.reported)
                | Q(sender=report.reported, recipient=report.reporter)
            )
            .select_related('sender')
            .order_by('created_at')
        )

        return self.render_json(200, {
            'reporter': render_user_to_json(report.reporter),
            'reported': render_user_to_json(report.reported),
            'messages': [{
                'id': m.id,
                'sender_id': m.sender_id,
                'sender_name': str(m.sender) if m.sender else '(已删除)',
                'body': m.body,
                'created_at': m.created_at.isoformat(),
            } for m in messages],
        })
