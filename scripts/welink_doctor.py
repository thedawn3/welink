#!/usr/bin/env python3
from __future__ import annotations

import argparse
import glob
import os
import platform
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
EMPTY_MSG_DIR = ROOT / '.tmp' / 'welink-empty-msg'


def detect_platform(value: str) -> str:
    if value != 'auto':
        return value
    system = platform.system().lower()
    if 'windows' in system:
        return 'windows'
    if 'darwin' in system or 'mac' in system:
        return 'macos'
    return 'other'


def unique_existing(paths: list[Path]) -> list[Path]:
    seen = set()
    result = []
    for path in paths:
        key = str(path)
        if key in seen:
            continue
        seen.add(key)
        if path.exists():
            result.append(path)
    return result


def candidate_data_dirs(kind: str) -> list[Path]:
    home = Path.home()
    paths = [
        Path(os.getenv('WELINK_DATA_DIR', '')),
        Path(os.getenv('WELINK_DECRYPTED_DIR', '')),
        ROOT / 'decrypted',
        ROOT.parent / 'decrypted',
        ROOT.parent / 'wechat-decrypt' / 'decrypted_with_wal',
        home / 'wechat-decrypt' / 'decrypted_with_wal',
        home / 'decrypted',
    ]
    if kind == 'windows':
        paths.extend([
            home / 'Documents' / 'wechat-decrypt' / 'decrypted_with_wal',
            home / 'Downloads' / 'wechat-decrypt' / 'decrypted_with_wal',
        ])
    return unique_existing([p for p in paths if str(p) not in {'', '.'}])


def candidate_msg_dirs(kind: str) -> list[Path]:
    home = Path.home()
    paths = [Path(os.getenv('WELINK_MSG_DIR', ''))]
    if kind == 'macos':
        paths.extend(Path(p) for p in glob.glob(str(home / 'Library/Containers/com.tencent.xinWeChat/Data/Documents/xwechat_files/*/msg')))
    elif kind == 'windows':
        paths.extend(Path(p) for p in glob.glob(str(home / 'Documents/WeChat Files/*/msg')))
        paths.extend(Path(p) for p in glob.glob(str(home / 'AppData/Roaming/Tencent/WeChat/*/msg')))
    return unique_existing([p for p in paths if str(p) not in {'', '.'}])


def validate_data_dir(path: Path) -> tuple[bool, list[str]]:
    issues = []
    if not path.exists():
        return False, [f'目录不存在: {path}']
    contact_db = path / 'contact' / 'contact.db'
    if not contact_db.is_file():
        issues.append(f'缺少 contact/contact.db: {contact_db}')
    message_dir = path / 'message'
    message_dbs = sorted(message_dir.glob('message_*.db')) if message_dir.is_dir() else []
    if not message_dbs:
        issues.append(f'缺少 message/message_*.db: {message_dir}')
    return not issues, issues


def validate_msg_dir(path: Path) -> tuple[bool, list[str]]:
    if not path:
        return True, []
    if not path.exists():
        return False, [f'媒体目录不存在: {path}']
    if not path.is_dir():
        return False, [f'媒体路径不是目录: {path}']
    return True, []


def choose_path(explicit: str | None, candidates: list[Path]) -> Path | None:
    if explicit:
        return Path(explicit).expanduser()
    return candidates[0] if candidates else None


def to_env_path(path: Path) -> str:
    return path.resolve().as_posix()


def write_env_file(data_dir: Path, msg_dir: Path | None) -> Path:
    EMPTY_MSG_DIR.mkdir(parents=True, exist_ok=True)
    env_path = ROOT / '.env'
    msg_value = to_env_path(msg_dir) if msg_dir else './.tmp/welink-empty-msg'
    env_path.write_text(
        '\n'.join([
            'WELINK_BACKEND_PORT=8080',
            'WELINK_FRONTEND_PORT=3000',
            'WELINK_GIN_MODE=release',
            f'WELINK_DATA_DIR={to_env_path(data_dir)}',
            f'WELINK_MSG_DIR={msg_value}',
            '',
        ]),
        encoding='utf-8',
    )
    return env_path


def main() -> int:
    parser = argparse.ArgumentParser(description='Validate WeLink data paths and optionally generate .env')
    parser.add_argument('--platform', default='auto', choices=['auto', 'macos', 'windows', 'other'])
    parser.add_argument('--data-dir')
    parser.add_argument('--msg-dir')
    parser.add_argument('--write-env', action='store_true')
    args = parser.parse_args()

    kind = detect_platform(args.platform)
    data_candidates = candidate_data_dirs(kind)
    msg_candidates = candidate_msg_dirs(kind)
    data_dir = choose_path(args.data_dir, data_candidates)
    msg_dir = choose_path(args.msg_dir, msg_candidates)

    print(f'[welink-doctor] platform: {kind}')
    print(f'[welink-doctor] repo: {ROOT}')

    if data_dir is None:
        print('[welink-doctor] ERROR: 没找到解密后的数据目录，请用 --data-dir 指定。')
        return 1

    ok, issues = validate_data_dir(data_dir)
    print(f'[welink-doctor] data dir: {data_dir}')
    if not ok:
        for issue in issues:
            print(f'  - {issue}')
        return 1

    print('[welink-doctor] data dir ok')

    if msg_dir:
        msg_ok, msg_issues = validate_msg_dir(msg_dir)
        print(f'[welink-doctor] msg dir: {msg_dir}')
        if not msg_ok:
            for issue in msg_issues:
                print(f'  - {issue}')
            return 1
        print('[welink-doctor] msg dir ok')
    else:
        print('[welink-doctor] msg dir: 未发现，将使用 ./.tmp/welink-empty-msg 占位。')

    if args.write_env:
        env_path = write_env_file(data_dir, msg_dir)
        print(f'[welink-doctor] wrote {env_path}')

    print('[welink-doctor] next: docker compose up --build')
    return 0


if __name__ == '__main__':
    sys.exit(main())
