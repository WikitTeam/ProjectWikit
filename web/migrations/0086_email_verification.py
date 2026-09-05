import django.db.models.functions.text
from django.db import migrations, models


class Migration(migrations.Migration):

    dependencies = [
        ('web', '0085_site_password_help'),
    ]

    operations = [
        migrations.AddField(
            model_name='user',
            name='email_verified_at',
            field=models.DateTimeField(blank=True, null=True, verbose_name='邮箱验证时间'),
        ),
        migrations.AddField(
            model_name='user',
            name='pending_email',
            field=models.TextField(blank=True, default='', verbose_name='待验证的新邮箱'),
        ),
        migrations.AddField(
            model_name='user',
            name='previous_email',
            field=models.TextField(blank=True, default='', verbose_name='改绑前的邮箱'),
        ),
        migrations.AddField(
            model_name='user',
            name='email_changed_at',
            field=models.DateTimeField(blank=True, null=True, verbose_name='邮箱上次改绑时间'),
        ),
        migrations.AddField(
            model_name='user',
            name='username_changed_at',
            field=models.DateTimeField(blank=True, null=True, verbose_name='用户名上次修改时间'),
        ),
        migrations.AddField(
            model_name='site',
            name='email_policy',
            field=models.TextField(
                choices=[
                    ('at_signup', '注册时就必须验证'),
                    ('required', '可以注册，但未验证不能操作'),
                    ('optional', '只当标识，未验证只影响找回密码'),
                ],
                default='optional',
                help_text='注册一律要填邮箱。这里决定没验证的人还能做什么。',
                verbose_name='邮箱验证要求',
            ),
        ),
        migrations.RemoveConstraint(
            model_name='user',
            name='user_email_ci_uniqueness',
        ),
        migrations.AddConstraint(
            model_name='user',
            constraint=models.UniqueConstraint(
                django.db.models.functions.text.Lower('email'),
                condition=models.Q(email__isnull=False) & ~models.Q(email='')
                          & models.Q(email_verified_at__isnull=False),
                name='user_email_ci_uniqueness',
            ),
        ),
    ]
