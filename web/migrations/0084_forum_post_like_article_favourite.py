import auto_prefetch
import django.db.models.deletion
import django.db.models.manager
from django.conf import settings
from django.db import migrations, models


class Migration(migrations.Migration):

    dependencies = [
        migrations.swappable_dependency(settings.AUTH_USER_MODEL),
        ('web', '0083_site_system_theme'),
    ]

    operations = [
        migrations.CreateModel(
            name='ForumPostLike',
            fields=[
                ('id', models.BigAutoField(auto_created=True, primary_key=True, serialize=False, verbose_name='ID')),
                ('created_at', models.DateTimeField(auto_now_add=True, verbose_name='点赞时间')),
                ('post', auto_prefetch.ForeignKey(on_delete=django.db.models.deletion.CASCADE, to='web.forumpost', verbose_name='帖子')),
                ('user', auto_prefetch.ForeignKey(on_delete=django.db.models.deletion.CASCADE, to=settings.AUTH_USER_MODEL, verbose_name='点赞者')),
            ],
            options={
                'verbose_name': '帖子点赞',
                'verbose_name_plural': '帖子点赞',
                'base_manager_name': 'prefetch_manager',
            },
            managers=[
                ('objects', django.db.models.manager.Manager()),
                ('prefetch_manager', django.db.models.manager.Manager()),
            ],
        ),
        migrations.CreateModel(
            name='ArticleFavourite',
            fields=[
                ('id', models.BigAutoField(auto_created=True, primary_key=True, serialize=False, verbose_name='ID')),
                ('created_at', models.DateTimeField(auto_now_add=True, verbose_name='收藏时间')),
                ('article', auto_prefetch.ForeignKey(on_delete=django.db.models.deletion.CASCADE, to='web.article', verbose_name='文章')),
                ('user', auto_prefetch.ForeignKey(on_delete=django.db.models.deletion.CASCADE, to=settings.AUTH_USER_MODEL, verbose_name='收藏者')),
            ],
            options={
                'verbose_name': '收藏',
                'verbose_name_plural': '收藏',
                'base_manager_name': 'prefetch_manager',
            },
            managers=[
                ('objects', django.db.models.manager.Manager()),
                ('prefetch_manager', django.db.models.manager.Manager()),
            ],
        ),
        migrations.AddConstraint(
            model_name='forumpostlike',
            constraint=models.UniqueConstraint(fields=('post', 'user'), name='web_forumpostlike_unique'),
        ),
        migrations.AddIndex(
            model_name='forumpostlike',
            index=models.Index(fields=['post', 'created_at'], name='web_forumpo_post_id_3cdbb2_idx'),
        ),
        migrations.AddConstraint(
            model_name='articlefavourite',
            constraint=models.UniqueConstraint(fields=('article', 'user'), name='web_articlefavourite_unique'),
        ),
        migrations.AddIndex(
            model_name='articlefavourite',
            index=models.Index(fields=['user', 'created_at'], name='web_article_user_id_e697e3_idx'),
        ),
    ]
