from web.permissions import BaseRolePermission


def is_perms_collection():
    return True


class ManageUsersPermission(BaseRolePermission):
    name = '管理用户'
    codename = 'manage_users'
    description = '允许创建和编辑用户'
    represent_django_perms = ['web.view_user', 'web.add_user', 'web.change_user']
    group = '管理后台'
    admin_only = True


class ManageRolesPermission(BaseRolePermission):
    name = '管理角色'
    codename = 'manage_roles'
    description = '允许创建、编辑和删除角色及角色分类'
    represent_django_perms = ['web.view_role', 'web.add_role', 'web.change_role', 'web.delete_role', 'web.view_rolecategory', 'web.add_rolecategory', 'web.change_rolecategory', 'web.delete_rolecategory']
    group = '管理后台'
    admin_only = True


class ManageSitePermission(BaseRolePermission):
    name = '管理站点'
    codename = 'manage_site'
    description = '允许修改站点的基本参数'
    represent_django_perms = ['web.view_site', 'web.change_site']
    group = '管理后台'
    admin_only = True


class ViewActionsLogPermission(BaseRolePermission):
    name = '查看操作记录'
    codename = 'view_actions_log'
    description = '允许查看操作日志'
    represent_django_perms = ['web.view_actionlogentry']
    group = '管理后台'
    admin_only = True
    

class ManageCaregoriesPermission(BaseRolePermission):
    name = '管理分类'
    codename = 'manage_categories'
    description = '允许创建、编辑和删除文章分类'
    represent_django_perms = ['web.view_category', 'web.add_category', 'web.change_category', 'web.delete_category']
    group = '管理后台'
    admin_only = True


class ManageTagsPermission(BaseRolePermission):
    name = '管理标签'
    codename = 'manage_tags'
    description = '允许创建、编辑和删除标签及标签分类'
    represent_django_perms = ['web.view_tag', 'web.add_tag', 'web.change_tag', 'web.delete_tag', 'web.view_tagscategory', 'web.add_tagscategory', 'web.change_tagscategory', 'web.delete_tagscategory']
    group = '管理后台'
    admin_only = True


class ManageForumPermission(BaseRolePermission):
    name = '管理论坛'
    codename = 'manage_forum'
    description = '允许创建、编辑和删除论坛版块和分类'
    represent_django_perms = ['web.view_forumcategory', 'web.add_forumcategory', 'web.change_forumcategory', 'web.delete_forumcategory', 'web.view_forumsection', 'web.add_forumsection', 'web.change_forumsection', 'web.delete_forumsection']
    group = '管理后台'
    admin_only = True


class ViewsensitiveInfoPermission(BaseRolePermission):
    name = '查看敏感信息'
    codename = 'view_sensitive_info'
    description = '允许查看用户的敏感信息，如邮箱或IP地址'
    group = '管理后台'
    admin_only = True


class ViewVotesTimestampPermission(BaseRolePermission):
    name = '查看投票时间'
    codename = 'view_votes_timestamp'
    description = '允许查看文章评分的日期和时间'
    group = '管理后台'
    admin_only = True