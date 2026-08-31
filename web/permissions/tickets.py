from web.permissions import BaseRolePermission


def is_perms_collection():
    return True


class ViewUserTicketsPermission(BaseRolePermission):
    name = '查看用户工单'
    codename = 'view_user_tickets'
    description = '允许查看用户通过表单提交的工单'
    represent_django_perms = ['web.view_userticket', 'web.change_userticket']
    group = '工单管理'
    admin_only = True


class ReviewMembershipApplicationsPermission(BaseRolePermission):
    name = '审核入组申请'
    codename = 'review_membership_applications'
    description = '允许审核入组申请，并决定授予申请人哪个角色'
    group = '工单管理'
    admin_only = True
