from django.contrib.auth import get_user_model
from django.http import HttpRequest

from renderer.utils import render_user_to_json
from web.controllers import messages
from web.views.api import APIError, APIView, takes_json, takes_url_params


User = get_user_model()


def _require_login(request: HttpRequest):
    if not request.user or not request.user.is_authenticated:
        raise APIError('请先登录', 401)


def _get_user_or_404(user_id: int):
    user = User.objects.filter(id=user_id).first()
    if not user:
        raise APIError('用户不存在', 404)
    return user


def _serialize_message(msg) -> dict:
    return {
        'id': msg.id,
        'sender_id': msg.sender_id,
        'recipient_id': msg.recipient_id,
        'body': msg.body,
        'created_at': msg.created_at.isoformat(),
        'is_read': msg.is_read,
    }


class SendMessageView(APIView):
    @takes_json
    def post(self, request: HttpRequest):
        _require_login(request)
        data = self.json_input or {}

        recipient_id = data.get('recipient_id')
        body = (data.get('body') or '').strip()

        if not recipient_id:
            raise APIError('缺少收件人', 400)
        if not body:
            raise APIError('消息内容不能为空', 400)

        recipient = _get_user_or_404(recipient_id)

        allowed, err = messages.can_send_message(request.user, recipient)
        if not allowed:
            raise APIError(err or '无法发送私信', 403)

        try:
            message = messages.send_message(request.user, recipient, body)
        except ValueError as e:
            raise APIError(str(e), 400)

        return self.render_json(200, _serialize_message(message))


class ConversationsListView(APIView):
    def get(self, request: HttpRequest):
        _require_login(request)

        result = []
        for entry in messages.list_conversations(request.user):
            partner = entry['partner']
            last = entry['last_message']
            result.append({
                'partner': render_user_to_json(partner),
                'last_message': {
                    'id': last.id,
                    'sender_id': last.sender_id,
                    'preview': last.preview(),
                    'created_at': last.created_at.isoformat(),
                },
                'unread_count': entry['unread_count'],
            })

        return self.render_json(200, {'conversations': result})


class ConversationView(APIView):
    @takes_url_params
    def get(self, request: HttpRequest, user_id: int, *, cursor: int = -1, limit: int = 30, mark_read: bool = False):
        _require_login(request)
        partner = _get_user_or_404(user_id)

        msgs, next_cursor = messages.get_conversation(request.user, partner, cursor=cursor, limit=limit)

        if mark_read:
            messages.mark_conversation_read(request.user, partner)

        return self.render_json(200, {
            'partner': render_user_to_json(partner),
            'messages': [_serialize_message(m) for m in msgs],
            'cursor': next_cursor,
        })


class MessagePermissionProbeView(APIView):
    def get(self, request: HttpRequest, user_id: int):
        _require_login(request)
        recipient = _get_user_or_404(user_id)
        allowed, err = messages.can_send_message(request.user, recipient)
        return self.render_json(200, {'allowed': allowed, 'reason': err})


class BlockUserView(APIView):
    @takes_json
    def post(self, request: HttpRequest, user_id: int):
        _require_login(request)
        target = _get_user_or_404(user_id)
        if target.id == request.user.id:
            raise APIError('不能拉黑自己', 400)
        messages.block_user(request.user, target)
        return self.render_json(200, {'status': 'ok', 'blocked': True})

    @takes_json
    def delete(self, request: HttpRequest, user_id: int):
        _require_login(request)
        target = _get_user_or_404(user_id)
        messages.unblock_user(request.user, target)
        return self.render_json(200, {'status': 'ok', 'blocked': False})
