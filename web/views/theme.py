from django.http import HttpResponse, HttpResponseNotFound
from django.views import View

from web.models.site import get_theme_dir


class SiteThemeFileView(View):
    def get(self, request, slug, *args, **kwargs):
        safe = ''.join(c for c in slug if c.isalnum() or c in '-_')
        if not safe:
            return HttpResponseNotFound('theme not found')

        path = get_theme_dir() / (safe + '.css')
        if not path.exists():
            return HttpResponseNotFound('theme not found')

        with open(path, 'rb') as f:
            data = f.read()

        resp = HttpResponse(data, content_type='text/css; charset=utf-8')
        resp['Cache-Control'] = 'public, max-age=31536000, immutable'
        return resp
