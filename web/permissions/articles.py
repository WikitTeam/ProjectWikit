from web.permissions import BaseRolePermission


def is_perms_collection():
    return True


class ViewArticlesPermission(BaseRolePermission):
    name = '浏览文章'
    codename = 'view_articles'
    description = '允许浏览文章、其文件、标签、反向链接和父页面'
    group = '文章'


class VoteArticlesPermission(BaseRolePermission):
    name = '给文章投票'
    codename = 'rate_articles'
    description = '允许对文章进行评分、修改和取消评分'
    group = '文章'


class CreateArticlesPermission(BaseRolePermission):
    name = '创建文章'
    codename = 'create_articles'
    description = '允许创建文章'
    group = '文章'


class EditArticlesPermission(BaseRolePermission):
    name = '编辑文章'
    codename = 'edit_articles'
    description = '允许修改文章内容、标题以及回退版本'
    group = '文章'


class TagArticlesPermission(BaseRolePermission):
    name = '编辑文章标签'
    codename = 'tag_articles'
    description = '允许为文章添加和删除标签'
    group = '文章'


class MoveArticlesPermission(BaseRolePermission):
    name = '移动文章'
    codename = 'move_articles'
    description = '允许更改文章的地址和分类'
    group = '文章'


class LockArticlesPermission(BaseRolePermission):
    name = '锁定文章'
    codename = 'lock_articles'
    description = '允许对文章进行锁定和解锁'
    group = '文章'


class ManageArticleFilesPermission(BaseRolePermission):
    name = '管理文章文件'
    codename = 'manage_article_files'
    description = '允许上传、重命名和删除文章文件'
    group = '文章'


class DeleteArticlesPermission(BaseRolePermission):
    name = '删除文章'
    codename = 'delete_articles'
    description = '允许永久删除文章'
    group = '文章'


class ResetArticleVotesPermission(BaseRolePermission):
    name = '重置投票'
    codename = 'reset_article_votes'
    description = '允许重置文章的投票'
    group = '文章'


class CommentArticlesPermission(BaseRolePermission):
    name = '评论文章'
    codename = 'comment_articles'
    description = '允许评论文章'
    group = '文章'


class ViewArticleCommentsPermission(BaseRolePermission):
    name = '查看文章评论'
    codename = 'view_article_comments'
    description = '允许查看文章的评论'
    group = '文章'

class ManageArticleAuthorsPermission(BaseRolePermission):
    name = '管理文章作者'
    codename = 'manage_article_authors'
    description = '允许添加和删除文章作者'
    group = '文章'