import json

from modules import ModuleError
from renderer import RenderContext, render_template_from_string
from urllib.parse import quote
from web.controllers import articles


def has_content():
    return False


def allow_api():
    return True


def render(context: RenderContext, params):
    example = params.get('example', '')
    category = params.get('category', '')
    submit_text = params.get('submit', '创建页面')

    placeholder = f"比如说, {example}" if example else "比如说, new-page-1"

    config = json.dumps({'category': category})

    return render_template_from_string("""
    <div class="new-page-form w-newpage-module" data-config="{{ config }}">
      <form method="get" action="">
        <p>
          <div>页面名称:</div>
          <input name="new_fullname" type="text" placeholder="{{ placeholder }}" required="true">
        </p>
        <p>
          <input value="{{ submit_text }}" type="submit">
        </p>
      </form>
    </div>
    """,
    placeholder=placeholder,
    config=config,
    submit_text=submit_text
    )


def api_check(context, params):
    new_fullname = (params.get('new_fullname') or '').strip()
    category = (params.get('category') or '').strip()

    if not new_fullname:
        raise ModuleError('请输入页面名称')

    if category:
        full_name = f"{category}:{new_fullname}"
    else:
        full_name = new_fullname

    full_name = articles.normalize_article_name(full_name)

    if not articles.is_full_name_allowed(full_name):
        raise ModuleError('此页面名称不可用')

    existing = articles.get_article(full_name)
    if existing is not None:
        raise ModuleError('此页面已存在')

    return {'url': f"/{quote(full_name, safe=':')}/edit/true"}