from django.core.management.base import BaseCommand, CommandError

from web import seeds, threadvars
from web.models.site import Site


class Command(BaseCommand):
    help = 'Seeds the database'

    def add_arguments(self, parser):
        parser.add_argument("-a", "--archive", required=False, help="从指定的 wikitCLI 备份归档进行迁移")
        # parser.add_argument("-o", "--fetch-from", required=False, help="Fetch from existing site, running on RuFoundation Engine.\nMake sure that the site engine version matches your version")
        parser.add_argument("-s", "--scope", choices=['all', 'pages', 'forum'], default='all',
                            help="迁移范围（仅归档模式）：all=全部，pages=仅页面（含文件/标签/评分/父页面），forum=仅论坛。默认 all")
        parser.add_argument("-t", "--force-tags", action='store_true',
                            help="强制迁移标签：绕过站点“禁止用户创建标签”设置，导入时按需新建标签")
        parser.add_argument("--no-votes", action='store_true',
                            help="跳过页面评分（votings）的迁移")
        parser.add_argument("--update-existing", action='store_true',
                            help="对已存在的文章也重新同步标签与评分（默认已存在文章整体跳过；仅补充缺失的评分，不覆盖已有投票）")

    def handle(self, *args, **options):
        if not Site.objects.exists():
            raise CommandError("You must create new site to run this command")

        site = Site.objects.get()

        with threadvars.context():
            threadvars.put('current_site', site)
            if options["archive"]:
                from web.seeds import wikit_archive
                wikit_archive.run(
                    options["archive"],
                    scope=options["scope"],
                    force_tags=options["force_tags"],
                    import_votes=not options["no_votes"],
                    update_existing=options["update_existing"],
                )
            else:
                seeds.run()
