import re

from django.db import migrations


def _canon(name):
    # 与 web.models.users.canonicalize_username 保持一致：小写 + [空格 _ -] 折叠为单个 - + 去首尾 -
    return re.sub(r'[\s_-]+', '-', name or '').strip('-').lower()


def forwards(apps, schema_editor):
    User = apps.get_model('web', 'User')
    display_re = re.compile(r'^[\w -]+\Z', re.ASCII)
    skipped = []

    for u in User.objects.all().iterator():
        # 普通用户规范化 username；Wikidot 用户规范化 wikidot_username（其 username 是 uuid 占位，不动）
        if u.type == 'wikidot':
            field, original = 'wikidot_username', u.wikidot_username
        else:
            field, original = 'username', u.username

        if not original:
            continue

        canon = _canon(original)
        if not canon or canon == original:
            continue

        update_fields = []
        # 保留原名为显示名（仅当当前无显示名且原名符合显示名字符集）
        if not u.display_name and display_re.match(original):
            u.display_name = original
            update_fields.append('display_name')

        # 冲突检测（citext 大小写不敏感）：撞名就跳过，保持原值不动
        clash = User.objects.filter(**{'%s__iexact' % field: canon}).exclude(pk=u.pk).exists()
        if clash:
            skipped.append('%s (%s=%r -> %r 冲突，已跳过)' % (u.pk, field, original, canon))
            if update_fields:
                u.save(update_fields=update_fields)
            continue

        setattr(u, field, canon)
        update_fields.append(field)
        u.save(update_fields=update_fields)

    if skipped:
        print('\n以下用户因规范化后与他人冲突而未改动，请人工处理：')
        for line in skipped:
            print('  - ' + line)


def backwards(apps, schema_editor):
    # 不可逆（原始大小写/下划线已无法从 canon 还原）；显示名保留即可作为参考
    pass


class Migration(migrations.Migration):

    dependencies = [
        ('web', '0074_user_display_name'),
    ]

    operations = [
        migrations.RunPython(forwards, backwards),
    ]
