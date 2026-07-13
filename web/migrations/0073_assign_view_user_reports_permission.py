from django.apps.registry import Apps
from django.db import migrations


def assign_view_user_reports_permission(apps: Apps, schema_editor):
    ContentType = apps.get_model('contenttypes', 'ContentType')
    Permission = apps.get_model('auth', 'Permission')
    Role = apps.get_model('web', 'Role')

    content_type, _ = ContentType.objects.get_or_create(
        app_label='web',
        model='roles',
    )

    view_perm, _ = Permission.objects.get_or_create(
        codename='view_user_reports',
        content_type=content_type,
        defaults={'name': '查看用户检举'},
    )

    Permission.objects.get_or_create(
        codename='view_reported_full_conversation',
        content_type=content_type,
        defaults={'name': '查看被举报会话全部记录'},
    )

    manage_users_perm = Permission.objects.filter(
        codename='manage_users', content_type=content_type,
    ).first()
    if manage_users_perm:
        for role in Role.objects.filter(permissions=manage_users_perm):
            role.permissions.add(view_perm)


def remove_permissions(apps: Apps, schema_editor):
    ContentType = apps.get_model('contenttypes', 'ContentType')
    Permission = apps.get_model('auth', 'Permission')

    content_type = ContentType.objects.filter(
        app_label='web', model='roles',
    ).first()
    if not content_type:
        return

    Permission.objects.filter(
        codename__in=['view_user_reports', 'view_reported_full_conversation'],
        content_type=content_type,
    ).delete()


class Migration(migrations.Migration):

    dependencies = [
        ('web', '0072_userreport'),
    ]

    operations = [
        migrations.RunPython(assign_view_user_reports_permission, remove_permissions, atomic=True),
    ]
