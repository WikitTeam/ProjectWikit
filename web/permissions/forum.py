from web.permissions import BaseRolePermission


def is_perms_collection():
    return True


class ViewForumPostsPermission(BaseRolePermission):
    name = '浏览论坛帖子'
    codename = 'view_forum_posts'
    description = '允许浏览论坛帖子'
    group = '论坛'


class CreateForumPostsPermission(BaseRolePermission):
    name = '创建论坛帖子'
    codename = 'create_forum_posts'
    description = '允许在论坛创建和编辑自己的帖子'
    group = '论坛'


class EditForumPostsPermission(BaseRolePermission):
    name = '编辑论坛帖子'
    codename = 'edit_forum_posts'
    description = '允许编辑他人的论坛帖子'
    group = '论坛'


class DeleteForumPostsPermission(BaseRolePermission):
    name = '删除论坛帖子'
    codename = 'delete_forum_posts'
    description = '允许删除论坛帖子'
    group = '论坛'


class ViewForumThreadsPermission(BaseRolePermission):
    name = '浏览论坛主题'
    codename = 'view_forum_threads'
    description = '允许浏览论坛主题'
    group = '论坛'


class CreateForumThreadsPermission(BaseRolePermission):
    name = '创建论坛主题'
    codename = 'create_forum_threads'
    description = '允许创建论坛主题'
    group = '论坛'


class EditForumThreadsPermission(BaseRolePermission):
    name = '编辑论坛主题'
    codename = 'edit_forum_threads'
    description = '允许编辑论坛主题的标题和描述'
    group = '论坛'


class PinForumThreadsPermission(BaseRolePermission):
    name = '置顶论坛主题'
    codename = 'pin_forum_threads'
    description = '允许置顶论坛主题'
    group = '论坛'


class LockForumThreadsPermission(BaseRolePermission):
    name = '锁定论坛主题'
    codename = 'lock_forum_threads'
    description = '允许锁定论坛主题'
    group = '论坛'


class MoveForumThreadsPermission(BaseRolePermission):
    name = '移动论坛主题'
    codename = 'move_forum_threads'
    description = '允许移动论坛主题'
    group = '论坛'


class ViewForumSectionsPermission(BaseRolePermission):
    name = '浏览论坛版块'
    codename = 'view_forum_sections'
    description = '允许浏览论坛版块'
    group = '论坛'


class ViewHiddenForumSectionsPermission(BaseRolePermission):
    name = '浏览隐藏的论坛版块'
    codename = 'view_hidden_forum_sections'
    description = '允许浏览隐藏的论坛版块'
    group = '论坛'


class ViewForumCategoriesPermission(BaseRolePermission):
    name = '浏览论坛分类'
    codename = 'view_forum_categories'
    description = '允许浏览论坛分类'
    group = '论坛'