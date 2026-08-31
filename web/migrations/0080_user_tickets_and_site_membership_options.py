import auto_prefetch
import django.db.models.deletion
import django.db.models.manager
from django.conf import settings
from django.db import migrations, models


class Migration(migrations.Migration):

    dependencies = [
        ('web', '0079_theme_slug'),
    ]

    operations = [
        migrations.CreateModel(
            name='UserTicket',
            fields=[
                ('id', models.BigAutoField(auto_created=True, primary_key=True, serialize=False, verbose_name='ID')),
                ('kind', models.TextField(choices=[('ticket', '工单'), ('membershipapply', '入组申请')], default='ticket', verbose_name='类型')),
                ('subject', models.TextField(blank=True, verbose_name='标题')),
                ('body', models.TextField(verbose_name='正文')),
                ('source_page', models.TextField(blank=True, verbose_name='提交页面')),
                ('status', models.TextField(choices=[('pending', '待处理'), ('approved', '已通过'), ('rejected', '已驳回'), ('closed', '已关闭')], default='pending', verbose_name='状态')),
                ('admin_notes', models.TextField(blank=True, verbose_name='处理备注')),
                ('created_at', models.DateTimeField(auto_now_add=True, verbose_name='提交时间')),
                ('reviewed_at', models.DateTimeField(blank=True, null=True, verbose_name='处理时间')),
            ],
            options={
                'verbose_name': '用户提交',
                'verbose_name_plural': '用户提交',
                'abstract': False,
                'base_manager_name': 'prefetch_manager',
            },
            managers=[
                ('objects', django.db.models.manager.Manager()),
                ('prefetch_manager', django.db.models.manager.Manager()),
            ],
        ),
        migrations.AddField(
            model_name='site',
            name='auth_icon',
            field=models.ImageField(blank=True, null=True, upload_to='-/sites', verbose_name='登录/注册页图标'),
        ),
        migrations.AddField(
            model_name='site',
            name='default_role',
            field=models.ForeignKey(blank=True, null=True, on_delete=django.db.models.deletion.SET_NULL, related_name='+', to='web.role', verbose_name='普通注册获得的角色'),
        ),
        migrations.AddField(
            model_name='site',
            name='footer_license',
            field=models.TextField(blank=True, help_text='留空使用内置文案。支持 wikitext，并且只允许 [[module time]] 这一个模块。', verbose_name='页脚授权说明'),
        ),
        migrations.AddField(
            model_name='site',
            name='membership_password',
            field=models.TextField(blank=True, verbose_name='入组密码'),
        ),
        migrations.AddField(
            model_name='site',
            name='membership_password_enabled',
            field=models.BooleanField(default=False, verbose_name='启用密码入组'),
        ),
        migrations.AddField(
            model_name='site',
            name='membership_password_role',
            field=models.ForeignKey(blank=True, null=True, on_delete=django.db.models.deletion.SET_NULL, related_name='+', to='web.role', verbose_name='密码入组获得的角色'),
        ),
        migrations.AddField(
            model_name='site',
            name='signup_notice',
            field=models.TextField(blank=True, help_text='留空使用内置文案。显示在注册按钮下方。', verbose_name='注册页提示'),
        ),
        migrations.AddField(
            model_name='site',
            name='verified_role',
            field=models.ForeignKey(blank=True, null=True, on_delete=django.db.models.deletion.SET_NULL, related_name='+', to='web.role', verbose_name='认证账号获得的角色'),
        ),
        migrations.AddField(
            model_name='userticket',
            name='author',
            field=auto_prefetch.ForeignKey(blank=True, null=True, on_delete=django.db.models.deletion.SET_NULL, related_name='tickets_submitted', to=settings.AUTH_USER_MODEL, verbose_name='提交人'),
        ),
        migrations.AddField(
            model_name='userticket',
            name='granted_role',
            field=auto_prefetch.ForeignKey(blank=True, null=True, on_delete=django.db.models.deletion.SET_NULL, related_name='+', to='web.role', verbose_name='授予的角色'),
        ),
        migrations.AddField(
            model_name='userticket',
            name='reviewed_by',
            field=auto_prefetch.ForeignKey(blank=True, null=True, on_delete=django.db.models.deletion.SET_NULL, related_name='tickets_reviewed', to=settings.AUTH_USER_MODEL, verbose_name='处理人'),
        ),
        migrations.CreateModel(
            name='MembershipApplication',
            fields=[
            ],
            options={
                'verbose_name': '申请书',
                'verbose_name_plural': '申请书',
                'abstract': False,
                'proxy': True,
                'base_manager_name': 'prefetch_manager',
                'indexes': [],
                'constraints': [],
            },
            bases=('web.userticket',),
            managers=[
                ('objects', django.db.models.manager.Manager()),
                ('prefetch_manager', django.db.models.manager.Manager()),
            ],
        ),
        migrations.CreateModel(
            name='SupportTicket',
            fields=[
            ],
            options={
                'verbose_name': '用户工单',
                'verbose_name_plural': '用户工单',
                'abstract': False,
                'proxy': True,
                'base_manager_name': 'prefetch_manager',
                'indexes': [],
                'constraints': [],
            },
            bases=('web.userticket',),
            managers=[
                ('objects', django.db.models.manager.Manager()),
                ('prefetch_manager', django.db.models.manager.Manager()),
            ],
        ),
        migrations.AddIndex(
            model_name='userticket',
            index=models.Index(fields=['kind', 'status', 'created_at'], name='web_usertic_kind_b5423e_idx'),
        ),
        migrations.AddIndex(
            model_name='userticket',
            index=models.Index(fields=['author', 'kind'], name='web_usertic_author__6ccbb3_idx'),
        ),
    ]
