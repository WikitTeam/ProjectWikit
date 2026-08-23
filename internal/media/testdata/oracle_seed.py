# Puts the files and the attachment row oracle_media.py compares against in
# place. Run it inside the Django container:
#
#   docker compose exec -T web python manage.py shell < internal/media/testdata/oracle_seed.py
#
# files/ is bind-mounted from the repository, so pwikit sees the same bytes.
from pathlib import Path

from django.conf import settings

from web.models.articles import Article
from web.models.files import File

PROBE = bytes(i % 251 for i in range(300))
ARTICLE = 'main'
ATTACHMENT = 'probe attach.pdf'
MEDIA_NAME = '11111111-2222-3333-4444-555555555555'

root = Path(settings.MEDIA_ROOT)
probe_dir = root / '-' / 'probe'
probe_dir.mkdir(parents=True, exist_ok=True)
for name in ('a.pdf', 'a.txt', 'a.txt.gz', 'a.bin'):
    (probe_dir / name).write_bytes(PROBE)
(probe_dir / 'empty').write_bytes(b'')

article = Article.objects.get(name=ARTICLE, category='_default')
attachment, _ = File.objects.get_or_create(
    article=article, name=ATTACHMENT, deleted_at=None,
    defaults=dict(media_name=MEDIA_NAME, mime_type='application/pdf', size=len(PROBE)),
)
article_dir = root / 'media' / File.escape_media_name(article.media_name)
article_dir.mkdir(parents=True, exist_ok=True)
(article_dir / File.escape_media_name(attachment.media_name)).write_bytes(PROBE)

print('article.media_name =', article.media_name)
print('file.media_name =', attachment.media_name)
