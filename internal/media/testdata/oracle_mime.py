# Prints the three tables mimetable.go is generated from. Run it inside the
# Django container:
#
#   docker compose exec -T web python - < internal/media/testdata/oracle_mime.py
import json
import mimetypes

mimetypes.init()
print(json.dumps({
    'types': mimetypes.types_map,
    'encodings': mimetypes.encodings_map,
    'suffix': mimetypes.MimeTypes().suffix_map,
}, sort_keys=True, indent=1))
