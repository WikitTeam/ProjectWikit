from web.models.site import get_current_site, get_site_system_theme_url


def site_branding(request):
    site = get_current_site()
    if site is None:
        return {}
    return {
        'site_auth_icon': '/local--files/%s' % site.auth_icon if site.auth_icon else '',
        'site_icon': '/local--files/%s' % site.icon if site.icon else '',
        'site_signup_notice': site.signup_notice,
        'site_system_theme_url': get_site_system_theme_url(),
    }
