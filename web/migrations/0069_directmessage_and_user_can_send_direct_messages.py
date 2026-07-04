import auto_prefetch
import django.db.models.deletion
import django.db.models.manager
from django.conf import settings
from django.db import migrations, models


class Migration(migrations.Migration):

    dependencies = [
        ('web', '0068_user_user_email_ci_uniqueness'),
    ]

    operations = [
        migrations.AddField(
            model_name='user',
            name='can_send_direct_messages',
            field=models.BooleanField(default=True, verbose_name='允许发送私信'),
        ),
        migrations.CreateModel(
            name='DirectMessage',
            fields=[
                ('id', models.BigAutoField(auto_created=True, primary_key=True, serialize=False, verbose_name='ID')),
                ('body', models.TextField(verbose_name='内容')),
                ('created_at', models.DateTimeField(auto_now_add=True, verbose_name='发送时间')),
                ('is_read', models.BooleanField(default=False, verbose_name='已读')),
                ('recipient', auto_prefetch.ForeignKey(on_delete=django.db.models.deletion.CASCADE, related_name='received_direct_messages', to=settings.AUTH_USER_MODEL, verbose_name='接收者')),
                ('sender', auto_prefetch.ForeignKey(on_delete=django.db.models.deletion.CASCADE, related_name='sent_direct_messages', to=settings.AUTH_USER_MODEL, verbose_name='发送者')),
            ],
            options={
                'verbose_name': '私信',
                'verbose_name_plural': '私信',
                'abstract': False,
                'base_manager_name': 'prefetch_manager',
            },
            managers=[
                ('objects', django.db.models.manager.Manager()),
                ('prefetch_manager', django.db.models.manager.Manager()),
            ],
        ),
        migrations.AddIndex(
            model_name='directmessage',
            index=models.Index(fields=['recipient', 'is_read'], name='dm_recip_read_idx'),
        ),
        migrations.AddIndex(
            model_name='directmessage',
            index=models.Index(fields=['sender', 'recipient', 'created_at'], name='dm_sr_created_idx'),
        ),
        migrations.AddIndex(
            model_name='directmessage',
            index=models.Index(fields=['recipient', 'sender', 'created_at'], name='dm_rs_created_idx'),
        ),
        migrations.CreateModel(
            name='DirectMessageBlock',
            fields=[
                ('id', models.BigAutoField(auto_created=True, primary_key=True, serialize=False, verbose_name='ID')),
                ('created_at', models.DateTimeField(auto_now_add=True, verbose_name='创建时间')),
                ('blocked', auto_prefetch.ForeignKey(on_delete=django.db.models.deletion.CASCADE, related_name='dm_blocks_in', to=settings.AUTH_USER_MODEL, verbose_name='被拉黑者')),
                ('blocker', auto_prefetch.ForeignKey(on_delete=django.db.models.deletion.CASCADE, related_name='dm_blocks_out', to=settings.AUTH_USER_MODEL, verbose_name='拉黑者')),
            ],
            options={
                'verbose_name': '私信拉黑',
                'verbose_name_plural': '私信拉黑',
                'abstract': False,
                'base_manager_name': 'prefetch_manager',
            },
            managers=[
                ('objects', django.db.models.manager.Manager()),
                ('prefetch_manager', django.db.models.manager.Manager()),
            ],
        ),
        migrations.AddConstraint(
            model_name='directmessageblock',
            constraint=models.UniqueConstraint(fields=('blocker', 'blocked'), name='dm_block_uniqueness'),
        ),
    ]
