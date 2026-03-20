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
EMPTY_SOURCE_DIR = ROOT / '.tmp' / 'welink-empty-source'
DEFAULT_WORK_DIR = ROOT / '.tmp' / 'welink-decrypt-work'
DEFAULT_ANALYSIS_DIR = ROOT / 'decrypted'


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


def candidate_source_dirs(kind: str) -> list[Path]:
    home = Path.home()
    paths = [
        Path(os.getenv('WELINK_SOURCE_DATA_DIR', '')),
        Path(os.getenv('WELINK_DATA_DIR', '')),
        Path(os.getenv('WELINK_ANALYSIS_DATA_DIR', '')),
        ROOT / 'decrypted',
        ROOT.parent / 'decrypted',
        ROOT.parent / 'wechat-decrypt' / 'decrypted_with_wal',
        home / 'wechat-decrypt' / 'decrypted_with_wal',
        home / 'decrypted',
    ]
    if kind == 'windows':
        paths.extend([
            home / 'Documents' / 'WeChat Files',
            home / 'AppData' / 'Roaming' / 'Tencent' / 'WeChat',
        ])
    elif kind == 'macos':
        paths.extend([
            home / 'Library' / 'Containers' / 'com.tencent.xinWeChat' / 'Data' / 'Documents' / 'xwechat_files',
        ])
    return unique_existing([p for p in paths if str(p) not in {'', '.'}])


def candidate_work_dirs() -> list[Path]:
    paths = [
        Path(os.getenv('WELINK_WORK_DIR', '')),
        DEFAULT_WORK_DIR,
    ]
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
    DEFAULT_WORK_DIR.mkdir(parents=True, exist_ok=True)
    env_path = ROOT / '.env'
    msg_value = to_env_path(msg_dir) if msg_dir else './.tmp/welink-empty-msg'
    lines = [
        'WELINK_BACKEND_PORT=8080',
        'WELINK_FRONTEND_PORT=3000',
        'WELINK_GIN_MODE=release',
        f'WELINK_DATA_DIR={to_env_path(data_dir)}',
        f'WELINK_MSG_DIR={msg_value}',
        '',
    ]
    env_path.write_text('\n'.join(lines), encoding='utf-8')
    return env_path


def write_env_file_v2(
    mode: str,
    kind: str,
    analysis_data_dir: Path | None,
    source_data_dir: Path | None,
    work_dir: Path,
    msg_dir: Path | None,
) -> Path:
    EMPTY_MSG_DIR.mkdir(parents=True, exist_ok=True)
    EMPTY_SOURCE_DIR.mkdir(parents=True, exist_ok=True)
    work_dir.mkdir(parents=True, exist_ok=True)
    env_path = ROOT / '.env'
    msg_value = to_env_path(msg_dir) if msg_dir else './.tmp/welink-empty-msg'
    analysis_dir = analysis_data_dir or DEFAULT_ANALYSIS_DIR
    analysis_dir.mkdir(parents=True, exist_ok=True)
    analysis_value = to_env_path(analysis_dir)
    is_decrypt_first = mode == 'decrypt-first'
    is_manual_sync = mode == 'manual-sync'
    source_value = to_env_path(source_data_dir) if ((is_decrypt_first or is_manual_sync) and source_data_dir) else ''
    ingest_enabled = 'true' if is_decrypt_first else 'false'
    decrypt_enabled = 'true' if is_decrypt_first else 'false'
    decrypt_auto_start = decrypt_enabled
    sync_enabled = 'true' if is_decrypt_first else 'false'

    lines = [
        'WELINK_BACKEND_PORT=8080',
        'WELINK_FRONTEND_PORT=3000',
        'WELINK_GIN_MODE=release',
        # legacy compatibility
        f'WELINK_DATA_DIR={analysis_value}',
        f'WELINK_MSG_DIR={msg_value}',
        # new runtime/ingest/sync/decrypt envs
        f'WELINK_MODE={mode}',
        f'WELINK_PLATFORM={kind}',
        f'WELINK_INGEST_ENABLED={ingest_enabled}',
        f'WELINK_SOURCE_DATA_DIR={source_value}',
        f'WELINK_WORK_DIR={to_env_path(work_dir)}',
        f'WELINK_ANALYSIS_DATA_DIR={analysis_value}',
        f'WELINK_DECRYPT_ENABLED={decrypt_enabled}',
        f'WELINK_DECRYPT_AUTO_START={decrypt_auto_start}',
        f'WELINK_SYNC_ENABLED={sync_enabled}',
        'WELINK_SYNC_WATCH_WAL=true',
        f'WELINK_RUNTIME_ENGINE_TYPE={kind if kind != "other" else "welink"}',
        '',
    ]
    env_path.write_text('\n'.join(lines), encoding='utf-8')
    return env_path


def main() -> int:
    parser = argparse.ArgumentParser(description='Validate WeLink data paths and optionally generate .env')
    parser.add_argument('--mode', default=os.getenv('WELINK_MODE', 'analysis-only'), choices=['analysis-only', 'manual-sync', 'decrypt-first'])
    parser.add_argument('--platform', default='auto', choices=['auto', 'macos', 'windows', 'other'])
    parser.add_argument('--data-dir')
    parser.add_argument('--source-data-dir')
    parser.add_argument('--work-dir')
    parser.add_argument('--msg-dir')
    parser.add_argument('--write-env', action='store_true')
    args = parser.parse_args()

    kind = detect_platform(args.platform)
    mode = args.mode
    data_candidates = candidate_data_dirs(kind)
    source_candidates = candidate_source_dirs(kind)
    work_candidates = candidate_work_dirs()
    msg_candidates = candidate_msg_dirs(kind)
    analysis_data_dir = choose_path(args.data_dir, data_candidates)
    source_data_dir = choose_path(args.source_data_dir, source_candidates)
    work_dir = choose_path(args.work_dir, work_candidates) or DEFAULT_WORK_DIR
    msg_dir = choose_path(args.msg_dir, msg_candidates)

    print(f'[welink-doctor] platform: {kind}')
    print(f'[welink-doctor] mode: {mode}')
    print(f'[welink-doctor] repo: {ROOT}')

    if mode == 'analysis-only':
        if analysis_data_dir is None:
            print('[welink-doctor] ERROR: 没找到解密后的数据目录，请用 --data-dir 指定。')
            return 1

        ok, issues = validate_data_dir(analysis_data_dir)
        print(f'[welink-doctor] analysis data dir: {analysis_data_dir}')
        if not ok:
            for issue in issues:
                print(f'  - {issue}')
            return 1
        print('[welink-doctor] analysis data dir ok')
        print('[welink-doctor] docker recommendation: manual sync mode (no auto decrypt/watcher).')
        print('[welink-doctor] source data dir will remain empty in generated .env.')
    elif mode == 'manual-sync':
        if analysis_data_dir is None:
            print('[welink-doctor] ERROR: manual-sync 模式下需要 analysis data dir，请用 --data-dir 指定。')
            return 1
        ok, issues = validate_data_dir(analysis_data_dir)
        print(f'[welink-doctor] analysis data dir: {analysis_data_dir}')
        if not ok:
            for issue in issues:
                print(f'  - {issue}')
            return 1
        print('[welink-doctor] analysis data dir ok')

        if source_data_dir is None:
            print('[welink-doctor] ERROR: manual-sync 模式下需要 source data dir，请用 --source-data-dir 指定标准目录。')
            return 1

        ok, issues = validate_data_dir(source_data_dir)
        print(f'[welink-doctor] source data dir: {source_data_dir}')
        if not ok:
            for issue in issues:
                print(f'  - {issue}')
            return 1
        print('[welink-doctor] source data dir ok')
    else:
        if source_data_dir is None:
            print('[welink-doctor] ERROR: decrypt-first 模式下需要 source data dir，请用 --source-data-dir 指定。')
            return 1
        if not source_data_dir.exists() or not source_data_dir.is_dir():
            print(f'[welink-doctor] ERROR: source data dir 不存在或不是目录: {source_data_dir}')
            return 1
        print(f'[welink-doctor] source data dir: {source_data_dir}')
        print(f'[welink-doctor] work dir: {work_dir}')
        if analysis_data_dir is None:
            analysis_data_dir = DEFAULT_ANALYSIS_DIR
        if analysis_data_dir:
            ok, issues = validate_data_dir(analysis_data_dir)
            print(f'[welink-doctor] analysis data dir: {analysis_data_dir}')
            if ok:
                print('[welink-doctor] analysis data dir ok')
            else:
                print('[welink-doctor] analysis data dir currently not ready (expected before decrypt completes):')
                for issue in issues:
                    print(f'  - {issue}')
        else:
            print('[welink-doctor] analysis data dir: 未提供，将由 decrypt-first 流程生成后再分析。')

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
        env_path = write_env_file_v2(
            mode=mode,
            kind=kind,
            analysis_data_dir=analysis_data_dir,
            source_data_dir=source_data_dir,
            work_dir=work_dir,
            msg_dir=msg_dir,
        )
        print(f'[welink-doctor] wrote {env_path}')

    if mode == 'decrypt-first':
        print('[welink-doctor] next (decrypt-first):')
        print('  1) docker compose up --build')
        print('  2) 通过后端 system/decrypt 接口启动解密任务')
    elif mode == 'manual-sync':
        print('[welink-doctor] next (manual-sync):')
        print('  1) docker compose up --build')
        print('  2) 通过 /api/system/config-check 校验 source / analysis / work')
        print('  3) 在系统页点击“校验并同步标准目录”或手动重建')
    else:
        print('[welink-doctor] next (analysis-only/manual-sync):')
        print('  1) 外部工具先准备标准目录（contact/message，可选 sns）')
        print('  2) docker compose up --build')
        print('  3) 通过 /api/system/config-check 校验后再手动同步/重建')

    return 0


if __name__ == '__main__':
    sys.exit(main())
