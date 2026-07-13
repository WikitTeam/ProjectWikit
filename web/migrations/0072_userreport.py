import auto_prefetch
import django.db.models.deletion
import django.db.models.manager
from django.conf import settings
from django.db import migrations, models


class Migration(migrations.Migration):

    dependencies = [
        ('web', '0071_assign_send_dm_permission'),
    ]

    operations = [
        migrations.CreateModel(
            name='UserReport',
            fields=[
                ('id', models.BigAutoField(auto_created=True, primary_key=True, serialize=False, verbose_name='ID')),
                ('reason', models.TextField(verbose_name='举报理由')),
                ('reported_messages', models.JSONField(blank=True, default=list, verbose_name='举报的消息（快照）')),
                ('status', models.TextField(choices=[('pending', '待处理'), ('reviewed', '已处理'), ('dismissed', '已驳回')], default='pending', verbose_name='状态')),
                ('admin_notes', models.TextField(blank=True, verbose_name='处理备注')),
                ('created_at', models.DateTimeField(auto_now_add=True, verbose_name='提交时间')),
                ('reviewed_at', models.DateTimeField(blank=True, null=True, verbose_name='处理时间')),
                ('reported', auto_prefetch.ForeignKey(blank=True, null=True, on_delete=django.db.models.deletion.SET_NULL, related_name='reports_received', to=settings.AUTH_USER_MODEL, verbose_name='被举报人')),
                ('reporter', auto_prefetch.ForeignKey(blank=True, null=True, on_delete=django.db.models.deletion.SET_NULL, related_name='reports_submitted', to=settings.AUTH_USER_MODEL, verbose_name='举报人')),
                ('reviewed_by', auto_prefetch.ForeignKey(blank=True, null=True, on_delete=django.db.models.deletion.SET_NULL, related_name='reports_reviewed', to=settings.AUTH_USER_MODEL, verbose_name='处理人')),
            ],
            options={
                'verbose_name': '用户检举',
                'verbose_name_plural': '用户检举',
                'abstract': False,
                'base_manager_name': 'prefetch_manager',
            },
            managers=[
                ('objects', django.db.models.manager.Manager()),
                ('prefetch_manager', django.db.models.manager.Manager()),
            ],
        ),
        migrations.AddIndex(
            model_name='userreport',
            index=models.Index(fields=['status', 'created_at'], name='report_status_ct_idx'),
        ),
        migrations.AddIndex(
            model_name='userreport',
            index=models.Index(fields=['reported', 'status'], name='report_target_status_idx'),
        ),
    ]
