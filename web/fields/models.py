__all__ = [
    'CIText',
    'CICharField',
    'CIEmailField',
    'CITextField',
    'CSSColorField'
]

import re

from django.core import validators
from django.db import models

from . import widgets


class CSSHexColorValidator(validators.RegexValidator):
    regex = r"^#([0-9a-fA-F]{3}){1,2}$|^#([0-9a-fA-F]{4}){1,2}\Z"
    message = '颜色格式无效，只允许十六进制值(#000/#000000/#00000000)。'
    flags = re.ASCII


class CIText:
    def get_internal_type(self):
        return "CI" + super().get_internal_type()

    def db_type(self, connection):
        return "citext"


class CICharField(CIText, models.CharField):
    pass


class CIEmailField(CIText, models.EmailField):
    pass


class CITextField(CIText, models.TextField):
    pass


class CSSColorField(models.CharField):
    default_validators = [CSSHexColorValidator()]

    def formfield(self, **kwargs):
        kwargs.update({'widget': widgets.ColorInput})
        return super().formfield(**kwargs)