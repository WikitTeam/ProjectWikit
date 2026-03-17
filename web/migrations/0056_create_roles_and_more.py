import auto_prefetch
import django.db.models.deletion
import django.db.models.manager
import web.fields.models
import web.models.roles

from django.db import migrations, models
from django.apps.registry import Apps



def create_default_roles(apps: Apps, schema_editor):
    Role = apps.get_model('web', 'Role')
    Role.objects.create(
        slug='everyone',
        index=0
    )
    Role.objects.create(
        slug='registered',
        index=0
    )

def visualgroups_to_roles(apps: Apps, schema_editor):
    VisualUserGroup = apps.get_model('web', 'VisualUserGroup')
    Role = apps.get_model('web', 'Role')
    Vote = apps.get_model('web', 'Vote')
    User = apps.get_model('web', 'User')

    new_roles = []
    for n, group in enumerate(VisualUserGroup.objects.all().order_by('index')):
        new_roles.append(Role(
            slug=f'migrated_role_{n}',
            name=group.name,
            index=n,
            group_votes=True,
            inline_visual_mode='badge' if group.show_badge else 'hidden',
            profile_visual_mode='status',
            badge_text=group.badge,
            badge_bg=group.badge_bg,
            badge_text_color=group.badge_text_color,
            badge_show_border=group.badge_show_border,
        ))

    Role.objects.bulk_create(new_roles)

    for user in User.objects.filter(visual_group__isnull=False):
        user.roles.add(Role.objects.get(name=user.visual_group.name))

    votes_to_update = []
    for vote in Vote.objects.filter(visual_group__isnull=False):
        vote.role = Role.objects.get(name=vote.visual_group.name)
        votes_to_update.append(vote)

    Vote.objects.bulk_update(votes_to_update, ['role'])


class Migration(migrations.Migration):

    dependencies = [
        ('auth', '0012_alter_user_first_name_max_length'),
        ('web', '0055_alter_articlesearchindex_options_and_more'),
    ]

    operations = [
        migrations.CreateModel(
            name='RoleCategory',
            fields=[
                ('id', models.BigAutoField(auto_created=True, primary_key=True, serialize=False, verbose_name='ID')),
                ('name', models.CharField(verbose_name='名称')),
            ],
            options={
                'verbose_name': '角色分类',
                'verbose_name_plural': '角色分类',
            },
        ),
        migrations.AlterField(
            model_name='forumsection',
            name='is_hidden_for_users',
            field=models.BooleanField(default=False, verbose_name='仅管理员可见'),
        ),
        migrations.CreateModel(
            name='Role',
            fields=[
                ('id', models.BigAutoField(auto_created=True, primary_key=True, serialize=False, verbose_name='ID')),
                ('slug', models.CharField(unique=True, verbose_name='标识符')),
                ('name', models.CharField(blank=True, verbose_name='完整名称')),
                ('short_name', models.CharField(blank=True, verbose_name='简称')),
                ('index', models.PositiveIntegerField(db_index=True, default=0, editable=False, verbose_name='优先级')),
                ('is_staff', models.BooleanField(default=False, verbose_name='可访问管理后台')),
                ('group_votes', models.BooleanField(default=False, verbose_name='分组显示投票')),
                ('votes_title', models.CharField(blank=True, verbose_name='投票组标签')),
                ('inline_visual_mode', models.CharField(choices=[('hidden', '隐藏'), ('badge', '徽章'), ('icon', '图标')], default='hidden', verbose_name='用户名旁显示模式')),
                ('profile_visual_mode', models.CharField(choices=[('hidden', '隐藏'), ('badge', '徽章'), ('status', '状态')], default='hidden', verbose_name='个人资料显示模式')),
                ('color', web.fields.models.CSSColorField(default='#000000', verbose_name='颜色')),
                ('icon', models.FileField(blank=True, upload_to='-/roles', validators=[web.models.roles.svg_file_validator], verbose_name='图标')),
                ('badge_text', models.CharField(blank=True, verbose_name='徽章文本')),
                ('badge_bg', web.fields.models.CSSColorField(default='#808080', verbose_name='徽章背景色')),
                ('badge_text_color', web.fields.models.CSSColorField(default='#ffffff', verbose_name='文本颜色')),
                ('badge_show_border', models.BooleanField(default=False, verbose_name='显示边框')),
                ('permissions', models.ManyToManyField(blank=True, related_name='role_permissions_set', to='auth.permission', verbose_name='权限')),
                ('restrictions', models.ManyToManyField(blank=True, related_name='role_restrictions_set', to='auth.permission', verbose_name='限制')),
                ('category', auto_prefetch.ForeignKey(blank=True, null=True, on_delete=django.db.models.deletion.SET_NULL, to='web.rolecategory', verbose_name='分类')),
            ],
            options={
                'verbose_name': '角色',
                'verbose_name_plural': '角色',
                'ordering': ['index'],
                'abstract': False,
                'base_manager_name': 'prefetch_manager',
            },
            managers=[
                ('objects', django.db.models.manager.Manager()),
                ('prefetch_manager', django.db.models.manager.Manager()),
            ],
        ),
        migrations.AddField(
            model_name='user',
            name='roles',
            field=models.ManyToManyField(blank=True, related_name='users', related_query_name='user', to='web.role', verbose_name='角色'),
        ),
        migrations.AddField(
            model_name='vote',
            name='role',
            field=auto_prefetch.ForeignKey(null=True, on_delete=django.db.models.deletion.SET_NULL, to='web.role', verbose_name='角色'),
        ),
        migrations.CreateModel(
            name='RolePermissionsOverride',
            fields=[
                ('id', models.BigAutoField(auto_created=True, primary_key=True, serialize=False, verbose_name='ID')),
                ('permissions', models.ManyToManyField(blank=True, related_name='override_role_permissions_set', to='auth.permission')),
                ('restrictions', models.ManyToManyField(blank=True, related_name='override_role_restrictions_set', to='auth.permission')),
                ('role', auto_prefetch.ForeignKey(on_delete=django.db.models.deletion.CASCADE, to='web.role')),
            ],
            options={
                'abstract': False,
                'base_manager_name': 'prefetch_manager',
            },
            managers=[
                ('objects', django.db.models.manager.Manager()),
                ('prefetch_manager', django.db.models.manager.Manager()),
            ],
        ),
        migrations.AddField(
            model_name='category',
            name='permissions_override',
            field=models.ManyToManyField(to='web.rolepermissionsoverride'),
        ),

        # 创建默认角色
        migrations.RunPython(create_default_roles, migrations.RunPython.noop, atomic=True),

        # 从视觉用户组迁移到角色
        migrations.RunPython(visualgroups_to_roles, migrations.RunPython.noop, atomic=True),
    ]