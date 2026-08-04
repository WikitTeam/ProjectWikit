import json
from django.http import HttpResponse

from renderer import render_template_from_string
from renderer.utils import render_user_to_json

from web.models.site import get_site_theme_url


def reactive_view(request, *args, **kwargs):
    config = {
        'user': render_user_to_json(request.user)
    }

    return HttpResponse(

        render_template_from_string(
        """
            <!DOCTYPE html>
            <html lang="en">
            <head>
                <meta charset="UTF-8">
                <meta name="viewport" content="width=device-width, initial-scale=1.0">
                <link rel="stylesheet" type="text/css" href="/-/static/fontawesome/css/all.css">
                <link rel="stylesheet" type="text/css" href="/-/static/wikidot-base.css">
                <link rel="stylesheet" type="text/css" href="{{ theme_url }}">
                <link rel="stylesheet" type="text/css" href="/-/static/app.css">
                <link rel="preconnect" href="https://fonts.googleapis.com">
                <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
                <link href="https://fonts.googleapis.com/css2?family=Inter+Tight:wght@400;500;600&family=JetBrains+Mono:wght@400;500&display=swap" rel="stylesheet">
                <link rel="stylesheet" type="text/css" href="/-/static/wikit-tokens.css">
                <link rel="stylesheet" type="text/css" href="/-/static/wikit-shell.css">
                <script src="/-/static/app.js" type="text/javascript"></script>
            </head>
            <body>
                <div id="reactive-root" data-config="{{ config }}"></div>
                <div id="w-modals"></div>
            </body>
            </html>
        """,
        config=json.dumps(config),
        theme_url=get_site_theme_url()
    ))