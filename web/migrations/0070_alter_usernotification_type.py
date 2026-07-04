from django.db import migrations, models


class Migration(migrations.Migration):

    dependencies = [
        ('web', '0069_directmessage_and_user_can_send_direct_messages'),
    ]

    operations = [
        migrations.AlterField(
            model_name='usernotification',
            name='type',
            field=models.TextField(choices=[('welcome', '欢迎消息'), ('new_post_reply', '帖子回复'), ('new_thread_post', '新帖子'), ('new_article_revision', '文章编辑'), ('forum_mention', '论坛提及'), ('direct_message', '私信')], verbose_name='通知类型'),
        ),
    ]
