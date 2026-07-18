import os
from pathlib import Path

import django.core.validators
from django.conf import settings
from django.db import migrations, models
from django.utils.text import slugify


def populate_slugs(apps, schema_editor):
    Theme = apps.get_model('web', 'Theme')
    used = set()
    for theme in Theme.objects.all().order_by('pk'):
        if theme.name == '默认主题':
            slug = 'default'
        else:
            slug = slugify(theme.name) or ('theme%d' % theme.pk)
        base = slug
        i = 1
        while slug in used:
            i += 1
            slug = '%s-%d' % (base, i)
        used.add(slug)
        theme.slug = slug
        theme.save(update_fields=['slug'])

        if theme.mode == 'inline':
            directory = Path(settings.MEDIA_ROOT) / 'theme'
            try:
                directory.mkdir(parents=True, exist_ok=True)
                with open(directory / (slug + '.css'), 'w', encoding='utf-8') as f:
                    f.write(theme.css or '')
            except OSError:
                pass


def noop(apps, schema_editor):
    pass


class Migration(migrations.Migration):

    dependencies = [
        ('web', '0078_systemupdate_proxy'),
    ]

    operations = [
        migrations.AddField(
            model_name='theme',
            name='slug',
            field=models.TextField(default='', verbose_name='标识名'),
            preserve_default=False,
        ),
        migrations.RunPython(populate_slugs, noop),
        migrations.AlterField(
            model_name='theme',
            name='slug',
            field=models.TextField(unique=True, validators=[django.core.validators.RegexValidator('^[A-Za-z0-9_-]+$', '标识名只能包含英文字母、数字、- 和 _')], verbose_name='标识名'),
        ),
    ]
