import pkgutil
import sys
import logging

from types import ModuleType
from importlib.util import module_from_spec

from django.db import transaction

from web.util import check_function_exists_and_callable

_all_modules = {}
_initialized = False


class ModuleError(Exception):
    def __init__(self, message, *args):
        super().__init__(message, *args)
        self.message = message


def get_all_modules():
    global _initialized, _all_modules
    if _initialized:
        return _all_modules
    package = sys.modules[__name__]
    for importer, modname, ispkg in pkgutil.iter_modules(package.__path__):
        try:
            fullname = 'modules.%s' % modname
            if fullname in sys.modules:
                m = sys.modules[fullname]
            else:
                spec = importer.find_spec(fullname)
                m = module_from_spec(spec)
                spec.loader.exec_module(m)
        except:
            logging.error('加载模块 \'%s\' 失败：', modname.lower(), exc_info=True)
            continue
        if not check_function_exists_and_callable(m, 'render') and not check_function_exists_and_callable(m, 'allow_api'):
            continue
        _all_modules[modname.lower()] = m
    _initialized = True
    return _all_modules


def get_module(name_or_module) -> any:
    if type(name_or_module) == str:
        name = name_or_module.lower()
        modules = get_all_modules()
        return modules.get(name, None)
    if not isinstance(name_or_module, ModuleType):
        raise ValueError('预期传入字符串或模块')
    return name_or_module


def module_has_content(name_or_module):
    m = get_module(name_or_module)
    if m is None:
        return False
    if 'has_content' not in m.__dict__ or not callable(m.__dict__['has_content']):
        return False
    return m.__dict__['has_content']()


def module_allows_api(name_or_module):
    m = get_module(name_or_module)
    if m is None:
        return False
    if 'allow_api' not in m.__dict__ or not callable(m.__dict__['allow_api']):
        return False
    return m.__dict__['allow_api']()


@transaction.atomic
def render_module(name, context, params, content=None):
    if context and context.path_params.get('nomodule', 'false') == 'true':
        raise ModuleError('模块处理已禁用')
    m = get_module(name)
    if m is None:
        raise ModuleError('模块 \'%s\' 不存在' % name)
    try:
        render = m.__dict__.get('render', None)
        if render is None:
            raise ModuleError('模块 \'%s\' 不支持在页面上使用')
        if module_has_content(m):
            return render(context, params, content)
        else:
            return render(context, params)
    except ModuleError as e:
        raise
    except:
        logging.error('模块处理失败：%s，参数：%s，路径：%s，错误：', name, params, context.path_params if context else None, exc_info=True)
        raise ModuleError('处理模块 \'%s\' 时出错' % name)


@transaction.atomic
def handle_api(name, method, context, params):
    m = get_module(name)
    if m is None:
        raise ModuleError('模块 \'%s\' 不存在' % name)
    try:
        if module_allows_api(m):
            api_method = 'api_%s' % method
            if api_method not in m.__dict__ or not callable(m.__dict__[api_method]):
                raise ModuleError('模块 \'%s\' 的方法无效')
            return m.__dict__[api_method](context, params), getattr(m.__dict__[api_method], 'is_csrf_safe', False)
        else:
            raise ModuleError('模块 \'%s\' 不支持API')
    except ModuleError:
        raise
    except:
        logging.error('模块API调用失败：%s，API：%s，参数：%s，路径：%s，错误：', name, method, params, context.path_params if context else None, exc_info=True)
        raise ModuleError('处理模块 \'%s\' 时出错' % name)