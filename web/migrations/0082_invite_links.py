import auto_prefetch
import django.db.models.deletion
import django.db.models.manager
from django.conf import settings
from django.db import migrations, models


class Migration(migrations.Migration):

    dependencies = [
        ('web', '0081_assign_ticket_permissions'),
    ]

    operations = [
        migrations.CreateModel(
            name='InviteLink',
            fields=[
                ('id', models.BigAutoField(auto_created=True, primary_key=True, serialize=False, verbose_name='ID')),
                ('kind', models.TextField(choices=[('register', '邀请注册'), ('claim', '认领')], default='register', verbose_name='类型')),
                ('delivery', models.TextField(choices=[('link', '手动发链接'), ('email', '邮件发送')], default='link', verbose_name='发放方式')),
                ('email', models.TextField(blank=True, verbose_name='邮箱')),
                ('wikidot_username', models.TextField(blank=True, verbose_name='Wikidot 用户名')),
                ('token', models.TextField(verbose_name='令牌')),
                ('uidb64', models.TextField(verbose_name='账号编码')),
                ('created_at', models.DateTimeField(auto_now_add=True, verbose_name='创建时间')),
                ('activated_at', models.DateTimeField(blank=True, null=True, verbose_name='激活时间')),
                ('activated_username', models.TextField(blank=True, verbose_name='激活后用户名')),
            ],
            options={
                'verbose_name': '邀请链接',
                'verbose_name_plural': '邀请链接',
                'ordering': ['-created_at'],
                'abstract': False,
                'base_manager_name': 'prefetch_manager',
            },
            managers=[
                ('objects', django.db.models.manager.Manager()),
                ('prefetch_manager', django.db.models.manager.Manager()),
            ],
        ),
        migrations.AddField(
            model_name='invitelink',
            name='created_by',
            field=auto_prefetch.ForeignKey(blank=True, null=True, on_delete=django.db.models.deletion.SET_NULL, related_name='invite_links_created', to=settings.AUTH_USER_MODEL, verbose_name='创建人'),
        ),
        migrations.AddField(
            model_name='invitelink',
            name='target',
            field=auto_prefetch.ForeignKey(blank=True, null=True, on_delete=django.db.models.deletion.SET_NULL, related_name='invite_links', to=settings.AUTH_USER_MODEL, verbose_name='目标账号'),
        ),
        migrations.AddIndex(
            model_name='invitelink',
            index=models.Index(fields=['kind', 'activated_at'], name='web_invitel_kind_bfa536_idx'),
        ),
        migrations.AddIndex(
            model_name='invitelink',
            index=models.Index(fields=['token'], name='web_invitel_token_4bbe08_idx'),
        ),
    ]
