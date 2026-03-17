import copy

from django.template import Context
from jazzmin.templatetags import jazzmin
from jazzmin.settings import get_settings

from web import models

register = jazzmin.register


# 修复侧边菜单中的分类
@register.simple_tag(takes_context=True, name="get_side_menu")
def get_side_menu(context: Context, using: str = "available_apps") -> list[dict]:
    available_apps = copy.deepcopy(context.get(using, []))

    options = get_settings()

    structure = (
        ('用户与角色', (
            {'model': models.roles.RoleCategory},
            {'model': models.roles.Role},
            {'model': models.users.User},
        )),
        ('活动记录', (
            {'model': models.logs.ActionLogEntry},
            {
                'name': '可疑活动',
                'url': '/-/admin/web/actionlogentry/sus',
                'permissions': ['roles.view_sensitive_info']
            },
        )),
        ('结构管理', (
            {'model': models.site.Site},
            {'model': models.articles.Category},
            {'model': models.articles.TagsCategory},
            {'model': models.articles.Tag},
        )),
        ('论坛', (
            {'model': models.forum.ForumSection},
            {'model': models.forum.ForumCategory},
        ))
    )

    app_label_by_model = dict()
    model_meta_by_model = dict()

    # 遍历原始数组，因为它已经经过权限过滤
    for app in available_apps:
        for model in app.get("models", []):
            model_meta_by_model[model["model"]] = model
            app_label_by_model[model["model"]] = app["app_label"].lower()

    menu = []

    for section in structure:
        app = dict()
        app["icon"] = options["default_icon_parents"]
        app["name"] = section[0]
        app["models"] = []
        for model in section[1]:
            if 'model' in model:
                app_label = app_label_by_model.get(model["model"])
                model = model_meta_by_model.get(model["model"])
                if not model or not app_label:
                    continue
                model_str = "{app_label}.{model}".format(app_label=app_label, model=model["object_name"]).lower()
                if model_str in options.get("hide_models", []):
                    continue
                item = copy.deepcopy(model)
                item["url"] = item["admin_url"]
                item["model_str"] = model_str
                item["icon"] = options["icons"].get(model_str, options["default_icon_children"])
                app["models"].append(item)
            else:
                item = copy.deepcopy(model)
                if 'permissions' in item:
                    does_not_have_any = [x for x in item["permissions"] if not context.get('user').has_perm(x)]
                    if does_not_have_any:
                        continue
                    del item["permissions"]
                item["icon"] = item.get("icon") or options["default_icon_children"]
                app["models"].append(item)
        if not app["models"]:
            continue
        menu.append(app)

    return menu