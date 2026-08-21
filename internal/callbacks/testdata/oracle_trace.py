import json
import os

from web import threadvars
from web.models.site import Site
from renderer import callbacks_with_context, page_info_from_context
from renderer.parser import RenderContext
from ftml import ftml

CORPUS = json.loads(os.environ['TRACE_CORPUS'])

def q(s):
    return json.dumps(s, ensure_ascii=False)

out = []
with threadvars.context():
    threadvars.put('current_site', Site.objects.first())
    ctx = RenderContext()
    Base = type(callbacks_with_context(ctx))

    class Traced(Base):
        def __init__(self, context, log):
            super().__init__(context)
            self._log = log

        def module_has_body(self, name):
            self._log.append('module_has_body(%s)' % name)
            return super().module_has_body(name)

        def render_module(self, name, params, body):
            pairs = ', '.join('%s=%s' % (k, q(params[k])) for k in sorted(params))
            self._log.append('render_module(%s, {%s}, body=%s)' % (name, pairs, q(body)))
            return '<div class="module">%s|%s</div>' % (name, body)

        def render_user(self, username, avatar):
            self._log.append('render_user(%s, avatar=%s)' % (username, 'true' if avatar else 'false'))
            if username == 'kakushi':
                return '<span class="user">kakushi</span>'
            return '<span class="error-inline">no</span>'

        def get_i18n_message(self, message_id):
            self._log.append('get_i18n_message(%s)' % message_id)
            return super().get_i18n_message(message_id)

        def get_html_injected_code(self, html_id):
            self._log.append('get_html_injected_code(%s)' % html_id)
            return super().get_html_injected_code(html_id)

        def fetch_internal_links(self, page_refs):
            self._log.append('get_page_info([%s])' % ' '.join(page_refs))
            return [ftml.PartialPageInfo(full_name=r, exists=True, title=r)
                    for r in page_refs if r == 'exists']

        def fetch_includes(self, include_refs):
            parts = []
            for r in include_refs:
                v = dict(r.variables) if getattr(r, 'variables', None) else {}
                pairs = ', '.join('%s=%s' % (k, q(v[k])) for k in sorted(v))
                parts.append('%s{%s}' % (r.full_name, pairs))
            self._log.append('include_pages([%s])' % ' '.join(parts))
            return [ftml.FetchedPage(full_name=r.full_name,
                                     content='**被包含的内容**' if r.full_name == 'exists' else None)
                    for r in include_refs]

        def render_include_not_found(self, full_name):
            self._log.append('no_such_include(%s)' % full_name)
            return super().render_include_not_found(full_name)

        def evaluate_expression(self, expr):
            self._log.append('evaluate_expression(%s)' % q(expr))
            return super().evaluate_expression(expr)

        def normalize_page_name(self, full_name):
            self._log.append('normalize_page_name(%s)' % q(full_name))
            return super().normalize_page_name(full_name)

        def next_include_level(self):
            self._log.append('next_include_level()')
            return super().next_include_level()

    for name, src in CORPUS:
        log = []
        with threadvars.context():
            threadvars.put('current_site', Site.objects.first())
            ftml.render_html(src, Traced(ctx, log), page_info_from_context(ctx), 'article')
        out.append('== %s ==' % name)
        out.extend(log if log else ['(no callbacks)'])
        out.append('')

print('---TRACE-START---')
print('\n'.join(out))
