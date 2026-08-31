from django.apps.registry import Apps
from django.db import migrations


NEW_PERMISSIONS = {
    'view_user_tickets': '查看用户工单',
    'review_membership_applications': '审核入组申请',
    'manage_permissions': '访问权限表',
}


def assign_ticket_permissions(apps: Apps, schema_editor):
    ContentType = apps.get_model('contenttypes', 'ContentType')
    Permission = apps.get_model('auth', 'Permission')
    Role = apps.get_model('web', 'Role')

    content_type, _ = ContentType.objects.get_or_create(
        app_label='web',
        model='roles',
    )

    created = {}
    for codename, name in NEW_PERMISSIONS.items():
        created[codename], _ = Permission.objects.get_or_create(
            codename=codename,
            content_type=content_type,
            defaults={'name': name},
        )

    view_reports = Permission.objects.filter(
        codename='view_user_reports', content_type=content_type,
    ).first()
    if view_reports:
        for role in Role.objects.filter(permissions=view_reports):
            role.permissions.add(created['view_user_tickets'])
            role.permissions.add(created['review_membership_applications'])

    # 在这条迁移之前，改权限的门是「管理角色」；不给这些角色补上「访问权限表」，
    # 升级之后他们会突然失去一项本来就有的能力。
    manage_roles = Permission.objects.filter(
        codename='manage_roles', content_type=content_type,
    ).first()
    if manage_roles:
        for role in Role.objects.filter(permissions=manage_roles):
            role.permissions.add(created['manage_permissions'])


def remove_permissions(apps: Apps, schema_editor):
    ContentType = apps.get_model('contenttypes', 'ContentType')
    Permission = apps.get_model('auth', 'Permission')

    content_type = ContentType.objects.filter(
        app_label='web', model='roles',
    ).first()
    if not content_type:
        return

    Permission.objects.filter(
        codename__in=list(NEW_PERMISSIONS),
        content_type=content_type,
    ).delete()


class Migration(migrations.Migration):

    dependencies = [
        ('web', '0080_user_tickets_and_site_membership_options'),
    ]

    operations = [
        migrations.RunPython(assign_ticket_permissions, remove_permissions, atomic=True),
    ]
