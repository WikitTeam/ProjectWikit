from django.core.management.base import BaseCommand

from web.models.settings import Settings
from web.models.site import Site


class Command(BaseCommand):
    help = 'Creates a website'

    def add_arguments(self, parser):
        parser.add_argument('-s', '--slug', required=True, help='网站标识符 (e.g.: wikit-site, wikit, scp, backrooms)')
        parser.add_argument('-d', '--domain', required=True, help='网站域名 (e.g.: projwikit.unitreaty.org)')
        parser.add_argument('-D', '--media-domain', required=False, help='本地文件沙箱域名 (e.g.: files.projwikit.unitreaty.org)')
        parser.add_argument('-t', '--title', required=True, help='网站标题 (e.g.: SCP Foundation, The Backrooms)')
        parser.add_argument('-H', '--headline', required=True, help='网站副标题 (e.g.: 控制, 收容, 保护)')

    def handle(self, *args, **options):
        if Site.objects.exists():
            curr_site = Site.objects.get()
            print(f"无法创建多个网站！已存在 \"{curr_site.title}\" on {curr_site.domain}.")
            return
        
        media_domain = options["media_domain"] if options["media_domain"] else options["domain"]

        site = Site(
            title=options["title"],
            headline=options["headline"],
            slug=options["slug"],
            domain=options["domain"],
            media_domain=media_domain
        )
        site.save()

        settings = Settings(site=site)
        settings.save()
