from django.conf import settings
from django.core.cache import cache
from django.http import HttpResponse, HttpResponseRedirect
from django.views import View

from web.models.site import Theme, get_active_theme_meta


class SiteThemeView(View):
    def get(self, request, *args, **kwargs):
        meta = get_active_theme_meta()

        if meta.get('none'):
            return HttpResponseRedirect(settings.STATIC_URL + 'theme.css')

        if meta['mode'] == 'external':
            if meta['external_url']:
                return HttpResponseRedirect(meta['external_url'])
            return HttpResponse('', content_type='text/css; charset=utf-8')

        version = '%s-%s' % (meta['id'], meta['v'])
        body_key = 'theme_css_body_%s' % version
        css = cache.get(body_key)
        if css is None:
            theme = Theme.objects.filter(pk=meta['id']).only('css').first()
            css = (theme.css if theme else '') or ''
            cache.set(body_key, css, 86400)

        resp = HttpResponse(css, content_type='text/css; charset=utf-8')
        resp['Cache-Control'] = 'public, max-age=31536000, immutable'
        return resp
