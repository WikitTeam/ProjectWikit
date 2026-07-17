from django.apps.registry import Apps
from django.db import migrations


def assign_manage_updates_permission(apps: Apps, schema_editor):
    ContentType = apps.get_model('contenttypes', 'ContentType')
    Permission = apps.get_model('auth', 'Permission')
    Role = apps.get_model('web', 'Role')

    content_type, _ = ContentType.objects.get_or_create(
        app_label='web',
        model='roles',
    )

    manage_updates_perm, _ = Permission.objects.get_or_create(
        codename='manage_updates',
        content_type=content_type,
        defaults={'name': '管理系统更新'},
    )

    manage_site_perm = Permission.objects.filter(
        codename='manage_site', content_type=content_type,
    ).first()
    if manage_site_perm:
        for role in Role.objects.filter(permissions=manage_site_perm):
            role.permissions.add(manage_updates_perm)


def remove_permission(apps: Apps, schema_editor):
    ContentType = apps.get_model('contenttypes', 'ContentType')
    Permission = apps.get_model('auth', 'Permission')

    content_type = ContentType.objects.filter(
        app_label='web', model='roles',
    ).first()
    if not content_type:
        return

    Permission.objects.filter(
        codename='manage_updates',
        content_type=content_type,
    ).delete()


class Migration(migrations.Migration):

    dependencies = [
        ('web', '0075_canonicalize_usernames'),
    ]

    operations = [
        migrations.RunPython(assign_manage_updates_permission, remove_permission, atomic=True),
    ]
