import os
import json

from pathlib import Path
from datetime import datetime, timezone


MAILBOX = Path(os.environ.get('UPDATE_MAILBOX', '/mailbox'))


def _read_json(name):
    try:
        with open(MAILBOX / name, 'r', encoding='utf-8') as f:
            return json.load(f)
    except (FileNotFoundError, ValueError, OSError):
        return None


def get_current_version():
    data = _read_json('deployed.json')
    if not data:
        return None
    return data.get('ref') or data.get('sha')


def get_latest_release():
    return _read_json('release_cache.json')


def get_update_status():
    data = _read_json('update.status')
    if not data:
        return {'state': 'idle'}
    return data


def is_update_running():
    if (MAILBOX / 'update.request').exists() or (MAILBOX / 'update.request.processing').exists():
        return True
    return get_update_status().get('state') == 'running'


def is_update_available():
    latest = get_latest_release()
    if not latest or not latest.get('tag'):
        return False
    return latest.get('tag') != get_current_version()


def get_log_tail(max_lines=200):
    try:
        with open(MAILBOX / 'update.log', 'r', encoding='utf-8', errors='replace') as f:
            lines = f.readlines()
        return ''.join(lines[-max_lines:])
    except (FileNotFoundError, OSError):
        return ''


def request_update(user):
    if is_update_running():
        return False, '已有更新正在进行中'

    latest = get_latest_release()
    target = latest.get('tag') if latest else ''

    try:
        MAILBOX.mkdir(parents=True, exist_ok=True)
        tmp = MAILBOX / 'update.request.tmp'
        with open(tmp, 'w', encoding='utf-8') as f:
            f.write((target or '').strip())
        os.replace(tmp, MAILBOX / 'update.request')
    except OSError as e:
        return False, f'无法写入更新请求：{e}'

    return True, target or f'origin 最新提交'


def get_overview():
    return {
        'current': get_current_version(),
        'latest': get_latest_release(),
        'available': is_update_available(),
        'running': is_update_running(),
        'status': get_update_status(),
        'checked_at': (get_latest_release() or {}).get('fetched_at'),
        'now': datetime.now(timezone.utc).isoformat(),
    }
