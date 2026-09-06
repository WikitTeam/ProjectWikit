import django.utils.timezone
import web.fields.models
import web.models.users
import web.util
from django.conf import settings
from django.db import migrations, models


class Migration(migrations.Migration):

    dependencies = [
        ('auth', '0012_alter_user_first_name_max_length'),
        ('web', '0086_email_verification'),
    ]

    operations = [
        migrations.AlterModelOptions(
            name='usedtoken',
            options={'base_manager_name': 'prefetch_manager', 'verbose_name': '已使用的令牌', 'verbose_name_plural': '已使用的令牌列表'},
        ),
        migrations.AlterModelOptions(
            name='user',
            options={'verbose_name': '用户', 'verbose_name_plural': '用户列表'},
        ),
        migrations.RenameIndex(
            model_name='directmessage',
            new_name='web_directm_recipie_07e335_idx',
            old_name='dm_recip_read_idx',
        ),
        migrations.RenameIndex(
            model_name='directmessage',
            new_name='web_directm_sender__630662_idx',
            old_name='dm_sr_created_idx',
        ),
        migrations.RenameIndex(
            model_name='directmessage',
            new_name='web_directm_recipie_a11daa_idx',
            old_name='dm_rs_created_idx',
        ),
        migrations.RenameIndex(
            model_name='userreport',
            new_name='web_userrep_status_466a49_idx',
            old_name='report_status_ct_idx',
        ),
        migrations.RenameIndex(
            model_name='userreport',
            new_name='web_userrep_reporte_77ecac_idx',
            old_name='report_target_status_idx',
        ),
        migrations.AlterField(
            model_name='actionlogentry',
            name='origin_ip',
            field=models.GenericIPAddressField(blank=True, null=True, verbose_name='IP地址'),
        ),
        migrations.AlterField(
            model_name='article',
            name='authors',
            field=models.ManyToManyField(related_name='authored_by', to=settings.AUTH_USER_MODEL, verbose_name='作者'),
        ),
        migrations.AlterField(
            model_name='article',
            name='media_name',
            field=models.TextField(default=web.util.uuid4_str, unique=True, verbose_name='文件系统中文件文件夹的名称'),
        ),
        migrations.AlterField(
            model_name='externallink',
            name='link_type',
            field=models.TextField(choices=[('include', 'Include'), ('link', 'Link')], verbose_name='链接类型'),
        ),
        migrations.AlterField(
            model_name='forumsection',
            name='is_hidden',
            field=models.BooleanField(default=False, verbose_name='隐藏分类'),
        ),
        migrations.AlterField(
            model_name='forumthread',
            name='created_at',
            field=models.DateTimeField(auto_now_add=True, verbose_name='创建时间'),
        ),
        migrations.AlterField(
            model_name='site',
            name='password_help',
            field=models.TextField(blank=True, help_text='找回密码页上「邮箱收不到信？」展开后显示的内容。留空使用内置文案。支持 wikitext，不允许任何模块。', verbose_name='找回密码求助文案'),
        ),
        migrations.AlterField(
            model_name='user',
            name='api_key',
            field=models.CharField(blank=True, max_length=255, null=True, unique=True, verbose_name='API密钥'),
        ),
        migrations.AlterField(
            model_name='user',
            name='date_joined',
            field=models.DateTimeField(default=django.utils.timezone.now, verbose_name='date joined'),
        ),
        migrations.AlterField(
            model_name='user',
            name='email',
            field=models.EmailField(blank=True, max_length=254, verbose_name='email address'),
        ),
        migrations.AlterField(
            model_name='user',
            name='first_name',
            field=models.CharField(blank=True, max_length=150, verbose_name='first name'),
        ),
        migrations.AlterField(
            model_name='user',
            name='forum_inactive_until',
            field=models.DateTimeField(null=True, verbose_name='论坛权限禁用至'),
        ),
        migrations.AlterField(
            model_name='user',
            name='groups',
            field=models.ManyToManyField(blank=True, help_text='The groups this user belongs to. A user will get all permissions granted to each of their groups.', related_name='user_set', related_query_name='user', to='auth.group', verbose_name='groups'),
        ),
        migrations.AlterField(
            model_name='user',
            name='inactive_until',
            field=models.DateTimeField(null=True, verbose_name='禁用至'),
        ),
        migrations.AlterField(
            model_name='user',
            name='is_active',
            field=models.BooleanField(default=True, verbose_name='已启用'),
        ),
        migrations.AlterField(
            model_name='user',
            name='is_forum_active',
            field=models.BooleanField(default=True, verbose_name='论坛权限已启用'),
        ),
        migrations.AlterField(
            model_name='user',
            name='is_superuser',
            field=models.BooleanField(default=False, help_text='Designates that this user has all permissions without explicitly assigning them.', verbose_name='superuser status'),
        ),
        migrations.AlterField(
            model_name='user',
            name='last_login',
            field=models.DateTimeField(blank=True, null=True, verbose_name='last login'),
        ),
        migrations.AlterField(
            model_name='user',
            name='last_name',
            field=models.CharField(blank=True, max_length=150, verbose_name='last name'),
        ),
        migrations.AlterField(
            model_name='user',
            name='password',
            field=models.CharField(max_length=128, verbose_name='password'),
        ),
        migrations.AlterField(
            model_name='user',
            name='type',
            field=models.TextField(choices=[('normal', '普通用户'), ('wikidot', 'Wikidot 用户'), ('system', '系统用户'), ('bot', '机器人')], default='normal', verbose_name='用户类型'),
        ),
        migrations.AlterField(
            model_name='user',
            name='user_permissions',
            field=models.ManyToManyField(blank=True, help_text='Specific permissions for this user.', related_name='user_set', related_query_name='user', to='auth.permission', verbose_name='user permissions'),
        ),
        migrations.AlterField(
            model_name='user',
            name='username',
            field=web.fields.models.CITextField(error_messages={'unique': '用户名已存在'}, max_length=150, unique=True, validators=[web.models.users.StrictUsernameValidator()], verbose_name='用户名'),
        ),
        migrations.AlterField(
            model_name='vote',
            name='date',
            field=models.DateTimeField(auto_now_add=True, null=True, verbose_name='投票日期'),
        ),
    ]
