import django.db.models.deletion
from django.db import migrations, models


class Migration(migrations.Migration):

    dependencies = [
        ('web', '0082_invite_links'),
    ]

    operations = [
        migrations.AddField(
            model_name='site',
            name='system_theme',
            field=models.ForeignKey(
                blank=True,
                help_text='登录、注册、个人资料这类页面额外加载的 CSS。留空则只用内置样式。',
                null=True,
                on_delete=django.db.models.deletion.SET_NULL,
                related_name='+',
                to='web.theme',
                verbose_name='系统页主题',
            ),
        ),
    ]
