from django.contrib.auth.models import AbstractUser as _UserType
from django.db.models import Q, Max

from web.controllers.notifications import send_user_notification
from web.models.messages import DirectMessage, DirectMessageBlock
from web.models.notifications import UserNotification, UserNotificationMapping


MAX_BODY_LENGTH = 4000


def is_blocked(blocker: _UserType, blocked: _UserType) -> bool:
    if not blocker or not blocked or blocker.is_anonymous or blocked.is_anonymous:
        return False
    return DirectMessageBlock.objects.filter(blocker=blocker, blocked=blocked).exists()


def can_send_message(sender: _UserType, recipient: _UserType) -> tuple[bool, str | None]:
    if not sender or sender.is_anonymous:
        return False, '请先登录'
    if not recipient or recipient.is_anonymous:
        return False, '收件人不存在'
    if sender.id == recipient.id:
        return False, '不能给自己发私信'
    if not recipient.is_active:
        return False, '该用户已被禁用'
    if not sender.can_send_direct_messages:
        return False, '你的私信功能已被管理员禁用'
    if not sender.has_perm('roles.send_direct_message'):
        return False, '你没有发送私信的权限'
    if is_blocked(recipient, sender):
        return False, '你已被对方拉黑'
    return True, None


def send_message(sender: _UserType, recipient: _UserType, body: str) -> DirectMessage:
    body = (body or '').strip()
    if not body:
        raise ValueError('消息内容不能为空')
    if len(body) > MAX_BODY_LENGTH:
        raise ValueError(f'消息内容不能超过 {MAX_BODY_LENGTH} 个字符')

    message = DirectMessage.objects.create(
        sender=sender,
        recipient=recipient,
        body=body,
    )

    send_user_notification(
        recipient,
        UserNotification.NotificationType.DirectMessage,
        meta={
            'sender_id': sender.id,
            'sender_name': str(sender),
            'message_id': message.id,
            'preview': message.preview(),
        },
    )

    return message


def list_conversations(user: _UserType) -> list[dict]:
    if not user or user.is_anonymous:
        return []

    partner_last = (
        DirectMessage.objects
        .filter(Q(sender=user) | Q(recipient=user))
        .annotate(partner_id=_partner_id_expr(user))
        .values('partner_id')
        .annotate(last_id=Max('id'))
    )
    last_ids = [row['last_id'] for row in partner_last]

    messages = (
        DirectMessage.objects
        .filter(id__in=last_ids)
        .select_related('sender', 'recipient')
        .order_by('-created_at')
    )

    result = []
    for msg in messages:
        partner = msg.recipient if msg.sender_id == user.id else msg.sender
        unread_count = DirectMessage.objects.filter(
            sender=partner, recipient=user, is_read=False,
        ).count()
        result.append({
            'partner': partner,
            'last_message': msg,
            'unread_count': unread_count,
        })
    return result


def _partner_id_expr(user: _UserType):
    from django.db.models import Case, When, F
    return Case(
        When(sender=user, then=F('recipient_id')),
        default=F('sender_id'),
    )


def get_conversation(user: _UserType, partner: _UserType, cursor: int = -1, limit: int = 30) -> tuple[list[DirectMessage], int]:
    if not user or user.is_anonymous:
        return [], -1

    qs = DirectMessage.objects.filter(
        Q(sender=user, recipient=partner) | Q(sender=partner, recipient=user)
    ).order_by('-id')

    if cursor != -1:
        qs = qs.filter(id__lt=cursor)

    messages = list(qs[:limit])
    next_cursor = messages[-1].id if messages else -1
    return messages, next_cursor


def mark_conversation_read(user: _UserType, partner: _UserType) -> int:
    if not user or user.is_anonymous or not partner:
        return 0

    updated = DirectMessage.objects.filter(
        sender=partner, recipient=user, is_read=False,
    ).update(is_read=True)

    if updated:
        UserNotificationMapping.objects.filter(
            recipient=user,
            notification__type=UserNotification.NotificationType.DirectMessage,
            notification__meta__sender_id=partner.id,
            is_viewed=False,
        ).update(is_viewed=True)

    return updated


def block_user(blocker: _UserType, blocked: _UserType) -> bool:
    if not blocker or blocker.is_anonymous or not blocked or blocked.is_anonymous:
        return False
    if blocker.id == blocked.id:
        return False
    _, created = DirectMessageBlock.objects.get_or_create(blocker=blocker, blocked=blocked)
    return created


def unblock_user(blocker: _UserType, blocked: _UserType) -> bool:
    if not blocker or blocker.is_anonymous or not blocked or blocked.is_anonymous:
        return False
    deleted, _ = DirectMessageBlock.objects.filter(blocker=blocker, blocked=blocked).delete()
    return deleted > 0
