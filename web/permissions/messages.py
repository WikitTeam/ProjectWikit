from web.permissions import BaseRolePermission


def is_perms_collection():
    return True


class SendDirectMessagePermission(BaseRolePermission):
    name = '发送私信'
    codename = 'send_direct_message'
    description = '允许向其他用户发送站内私信'
    group = '用户互动'
