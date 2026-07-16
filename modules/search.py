import json
import re

from modules._csrf_protection import csrf_safe_method
from renderer.utils import render_template_from_string
from web.controllers import search as search_controller


def allow_api():
    return True


def render(context, params):
    path_params = context.path_params or {}
    config = {
        'placeholder': params.get('placeholder') or '搜索文章标题或内容…',
        'tags': params.get('tags') or '',
        'category': params.get('category') or '',
        'q': (path_params.get('q') or params.get('q') or '').strip(),
    }
    return render_template_from_string(
        '<div class="w-search-module" data-config="{{config}}"></div>',
        config=json.dumps(config)
    )


@csrf_safe_method
def api_search(context, params):
    q = (params.get('q') or '').strip()
    author = (params.get('author') or '').strip() or None
    tags_raw = (params.get('tags') or '').strip()
    tags = [t for t in re.split(r'[,\s]+', tags_raw) if t] if tags_raw else None
    date_from = (params.get('datefrom') or '').strip() or None
    date_to = (params.get('dateto') or '').strip() or None

    try:
        offset = max(0, int(params.get('offset') or 0))
    except (TypeError, ValueError):
        offset = 0

    if not q and not author and not tags and not date_from and not date_to:
        return {'results': [], 'hasMore': False, 'total': 0}

    items, has_more, total = search_controller.search_articles_filtered(
        q, author=author, tags=tags, date_from=date_from, date_to=date_to,
        user=context.user, offset=offset, limit=20
    )
    return {'results': items, 'hasMore': has_more, 'total': total}
