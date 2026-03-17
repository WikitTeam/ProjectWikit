import json
from django.contrib.auth.models import AbstractUser as _UserType
from django.utils.safestring import mark_safe

from renderer import single_pass_render
from renderer.parser import RenderContext
from renderer.utils import render_template_from_string, render_user_to_html
from modules.forumthread import render_posts

from web import threadvars
from web.models.logs import ActionLogEntry
from web.models.users import User


def add_action_log(user: _UserType, type: ActionLogEntry.ActionType, meta):
    with threadvars.context():
        ActionLogEntry(
            user=user,
            stale_username=user.username,
            type=type,
            meta=meta,
            origin_ip = threadvars.get('current_client_ip')
        ).save()


def _mark_safe_all(data: dict):
    return {k: mark_safe(v) if isinstance(v, str) else v for k, v in data.items()}


def _make_post_preview(post_id, author_id, title, source):
    context = RenderContext(None, None, {}, None)
    post_info = {
        'id': post_id,
        'name':title,
        'author': render_user_to_html(User.objects.filter(id=author_id).first()),
        'content': single_pass_render(source, context, 'message'),
        'created_at': '',
        'updated_at':  '',
        'replies': [],
        'rendered_replies': False,
        'options_config': json.dumps({
            'hasRevisions': False,
            'canReply': False,
            'canEdit': False,
            'canDelete': False,
        })
    }
    return render_posts([post_info])

def _render_post_edit_preview(m):
    body = render_template_from_string(
        '''
        <div class='w-tabview'>
            <ul class='yui-nav post-versions'>
                <li class=''><a href='javascript:;'>旧版</a></li>
                |
                <li class='selected' title='active'><a href='javascript:;'>新版</a></li>
            </ul>
            <div class='yui-content'>
                <div class='w-tabview-tab' style='display: none;'>
                    {{prev_post}}
                </div>
                <div class='w-tabview-tab' style='display: block;'>
                    {{post}}
                </div>
            </div>
        </div>
        ''',
        prev_post=_make_post_preview(m['post']['id'], m['post']['author'], m['prev_title'], m['prev_source']),
        post=_make_post_preview(m['post']['id'], m['post']['author'], m['title'], m['source'])
    )
    return body


def get_action_log_entry_description(log_entry: ActionLogEntry):
    ActionType = ActionLogEntry.ActionType
    m = log_entry.meta
    try:
        match log_entry.type:
            case ActionType.Vote:
                m = _mark_safe_all(m)
                if m['is_new']:
                    return f'向页面 {m['article']} 添加了评分 {m['new_vote']:.1f}'
                elif m['is_remove']:
                    return f'从页面 {m['article']} 移除了评分 {m['old_vote']:.1f}'
                elif m['is_change']:
                    return f'页面 {m['article']} 的评分已从 {m['old_vote']:.1f} 更改为 {m['new_vote']:.1f}'
                else:
                    return f'页面 {m['article']} 的评分被重复移除（原本不存在）'
            case ActionType.NewArticle:
                m = _mark_safe_all(m)
                return f'创建了新页面：{m['article']}'
            case ActionType.RemoveArticle:
                m = _mark_safe_all(m)
                return f'页面：{m['article']}，删除时评分：{m['rating']}，投票数：{m['votes']}，人气值：{m['popularity']}'
            case ActionType.EditArticle:
                m = _mark_safe_all(m)
                msg = [f'页面：{m['article']}，版本：{m['rev_number']}']
                meta = m['log_entry_meta']
                match m['edit_type']:
                    case 'tags':
                        added_tags = meta['added_tags']
                        removed_tags = meta['removed_tags']
                        if added_tags:
                            msg.append(f'添加标签：{', '.join([t['name'] for t in added_tags])}')
                        if removed_tags:
                            msg.append(f'移除标签：{', '.join([t['name'] for t in removed_tags])}')
                    case 'title':
                        msg.append(f'标题已从 {meta['prev_title']} 更改为 {meta['title']}')
                    case 'name':
                        msg.append(f'地址已从 {meta['prev_name']} 更改为 {meta['name']}')
                    case 'votes_deleted':
                        msg.append(f'评分已重置')
                    case _:
                        msg.append(f'编辑类型：{m['edit_type']}')
                if m['comment']:
                    msg.append(f'注释：{m['comment']}')
                return '，'.join(msg)
            case ActionType.NewForumPost:
                return _make_post_preview(m['post']['id'], m['post']['author'], m['title'], m['source'])
            case ActionType.EditForumPost:
                return _render_post_edit_preview(m)
            case ActionType.RemoveForumPost:
                return _make_post_preview(m['post']['id'], m['post']['author'], m['title'], m['source'])
            case _:
                return None
    except:
        return f'处理日志 #{log_entry.id} 时出错'