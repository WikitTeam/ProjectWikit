import os

import django.db.models.deletion
from django.conf import settings
from django.db import migrations, models


def seed_default_theme(apps, schema_editor):
    Theme = apps.get_model('web', 'Theme')
    Site = apps.get_model('web', 'Site')

    css = ''
    path = os.path.join(settings.BASE_DIR, 'static', 'theme.css')
    try:
        with open(path, 'r', encoding='utf-8') as f:
            css = f.read()
    except OSError:
        css = ''

    Theme.objects.create(name='默认主题', mode='inline', css=css)


def unseed_default_theme(apps, schema_editor):
    Theme = apps.get_model('web', 'Theme')
    Site = apps.get_model('web', 'Site')

    site = Site.objects.first()
    if site is not None:
        site.active_theme = None
        site.save(update_fields=['active_theme'])

    Theme.objects.filter(name='默认主题').delete()


class Migration(migrations.Migration):

    dependencies = [
        ('web', '0076_assign_manage_updates_permission'),
    ]

    operations = [
        migrations.CreateModel(
            name='Theme',
            fields=[
                ('id', models.BigAutoField(auto_created=True, primary_key=True, serialize=False, verbose_name='ID')),
                ('name', models.TextField(verbose_name='主题名称')),
                ('mode', models.TextField(choices=[('inline', '内联CSS'), ('external', '外部链接')], default='inline', verbose_name='类型')),
                ('css', models.TextField(blank=True, default='', verbose_name='CSS 内容')),
                ('external_url', models.TextField(blank=True, default='', verbose_name='外部 CSS 链接')),
                ('updated_at', models.DateTimeField(auto_now=True, verbose_name='更新时间')),
            ],
            options={
                'verbose_name': '主题',
                'verbose_name_plural': '主题',
            },
        ),
        migrations.AddField(
            model_name='site',
            name='home_page',
            field=models.TextField(default='main', verbose_name='主页名称'),
        ),
        migrations.AddField(
            model_name='site',
            name='active_theme',
            field=models.ForeignKey(blank=True, null=True, on_delete=django.db.models.deletion.SET_NULL, related_name='+', to='web.theme', verbose_name='站点主题'),
        ),
        migrations.RunPython(seed_default_theme, unseed_default_theme),
    ]
