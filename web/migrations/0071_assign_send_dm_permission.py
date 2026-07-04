from django.apps.registry import Apps
from django.db import migrations


def assign_send_dm_permission(apps: Apps, schema_editor):
    ContentType = apps.get_model('contenttypes', 'ContentType')
    Permission = apps.get_model('auth', 'Permission')
    Role = apps.get_model('web', 'Role')

    content_type, _ = ContentType.objects.get_or_create(
        app_label='web',
        model='roles',
    )

    perm, _ = Permission.objects.get_or_create(
        codename='send_direct_message',
        content_type=content_type,
        defaults={'name': '发送私信'},
    )

    for role in Role.objects.exclude(slug__in=['everyone', 'registered', 'reader']):
        role.permissions.add(perm)


def remove_send_dm_permission(apps: Apps, schema_editor):
    ContentType = apps.get_model('contenttypes', 'ContentType')
    Permission = apps.get_model('auth', 'Permission')

    content_type = ContentType.objects.filter(
        app_label='web',
        model='roles',
    ).first()
    if not content_type:
        return

    Permission.objects.filter(codename='send_direct_message', content_type=content_type).delete()


class Migration(migrations.Migration):

    dependencies = [
        ('web', '0070_alter_usernotification_type'),
    ]

    operations = [
        migrations.RunPython(assign_send_dm_permission, remove_send_dm_permission, atomic=True),
    ]
