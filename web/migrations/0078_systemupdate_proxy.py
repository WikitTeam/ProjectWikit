from django.db import migrations


class Migration(migrations.Migration):

    dependencies = [
        ('web', '0077_theme_and_site_options'),
    ]

    operations = [
        migrations.CreateModel(
            name='SystemUpdate',
            fields=[],
            options={
                'verbose_name': '系统更新',
                'verbose_name_plural': '系统更新',
                'proxy': True,
                'indexes': [],
                'constraints': [],
            },
            bases=('web.site',),
        ),
    ]
