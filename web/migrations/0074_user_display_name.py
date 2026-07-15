import web.models.users
from django.db import migrations, models


class Migration(migrations.Migration):

    dependencies = [
        ('web', '0073_assign_view_user_reports_permission'),
    ]

    operations = [
        migrations.AddField(
            model_name='user',
            name='display_name',
            field=models.CharField(blank=True, max_length=150, null=True, validators=[web.models.users.StrictDisplayNameValidator()], verbose_name='显示名'),
        ),
    ]
