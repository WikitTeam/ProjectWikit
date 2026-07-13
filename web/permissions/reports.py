from web.permissions import BaseRolePermission


def is_perms_collection():
    return True


class ViewUserReportsPermission(BaseRolePermission):
    name = '查看用户检举'
    codename = 'view_user_reports'
    description = '允许查看用户提交的检举记录'
    represent_django_perms = ['web.view_userreport', 'web.change_userreport']
    group = '工单管理'
    admin_only = True


class ViewReportedFullConversationPermission(BaseRolePermission):
    name = '查看被检举会话全部记录'
    codename = 'view_reported_full_conversation'
    description = '允许在处理检举时查看完整会话，而非仅用户勾选的消息'
    group = '工单管理'
    admin_only = True
