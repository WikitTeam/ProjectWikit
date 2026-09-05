from django.db import migrations, models


class Migration(migrations.Migration):

    dependencies = [
        ('web', '0084_forum_post_like_article_favourite'),
    ]

    operations = [
        migrations.AddField(
            model_name='site',
            name='password_help',
            field=models.TextField(
                blank=True,
                default='',
                help_text='找回密码页上「邮箱收不到信？」展开后显示的内容。'
                          '留空使用内置文案。支持 wikitext，不允许任何模块。',
                verbose_name='找回密码求助文案',
            ),
        ),
    ]
