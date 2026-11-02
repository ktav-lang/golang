"""Сохранить проверенные Zig runtime notices и проверить actual link inputs."""

import gzip
import hashlib
import json
from pathlib import Path, PurePosixPath
import re
import subprocess
import sys

REVISION = '5ad91a646a753cc3eecd8751e61cf458dadd9ac4'
EXPECTED_MEMBERS = {
    ('libunwind.a', 'UnwindLevel1.o'),
    ('libunwind.a', 'UnwindLevel1-gcc-ext.o'),
    ('libunwind.a', 'UnwindRegistersSave.o'),
    ('libunwind.a', 'UnwindRegistersRestore.o'),
    ('libunwind.a', 'libunwind.o'),
    ('libc_nonshared.a', 'stat64-2.32.o'),
    ('libc_nonshared.a', 'fstat64-2.32.o'),
}


def _read(path, limit=4 * 1024 * 1024):
    with path.open('rb') as source:
        data = source.read(limit + 1)
    if len(data) > limit:
        raise ValueError('runtime evidence file exceeds limit')
    return data


def _sources(lib, catalog):
    if catalog.get('source_revision') != REVISION or catalog.get('zig_version') != '0.14.0':
        raise ValueError('unreviewed Zig runtime source revision')
    lib = lib.resolve()
    for item in catalog['files']:
        relative = PurePosixPath(item['path'])
        if relative.is_absolute() or '..' in relative.parts:
            raise ValueError('invalid runtime source path')
        if item['path'] == 'LICENSE':
            source = lib.parent / 'LICENSE'
            boundary = lib.parent
        elif relative.parts[0] == 'lib':
            source = lib.joinpath(*relative.parts[1:])
            boundary = lib
        else:
            raise ValueError('unexpected runtime source path')
        if not source.resolve().is_relative_to(boundary):
            raise ValueError('runtime source escapes Zig installation')
        data = _read(source)
        if hashlib.sha256(data).hexdigest() != item['sha256']:
            raise ValueError(f"Zig runtime source changed: {item['path']}")
    return catalog['files']


def _prepare(evidence):
    version = subprocess.check_output(['zig', 'version'], text=True).strip()
    if version != '0.14.0':
        raise ValueError('unreviewed Zig version')
    environment = json.loads(subprocess.check_output(['zig', 'env']))
    lib = Path(environment['lib_dir'])
    catalog_path = Path(__file__).resolve().parents[1] / 'ci/zig-runtime-catalog.json'
    catalog_bytes = _read(catalog_path)
    catalog = json.loads(catalog_bytes)
    files = _sources(lib, catalog)
    notice_path = evidence / 'NOTICE.json'
    bundle = json.loads(_read(notice_path, 64 * 1024 * 1024))
    if bundle.get('schema_version') != 1 or 'non_cargo_runtimes' in bundle:
        raise ValueError('unexpected native notice bundle state')
    bundle['non_cargo_runtimes'] = [{
        'name': 'Zig vendored LLVM libunwind and glibc nonshared runtime',
        'zig_version': version,
        'source_revision': REVISION,
        'coverage': 'Conservative source notices; actual archive members are checked after linking.',
        'catalog_sha256': hashlib.sha256(catalog_bytes).hexdigest(),
        'files': files,
        'credits': catalog['credits'],
    }]
    bundle['coverage'] += ' Includes verified Zig runtime source notices; linked archive members require the post-build map gate.'
    data = (json.dumps(bundle, ensure_ascii=False, sort_keys=True, indent=2) + '\n').encode('utf-8')
    compressed = gzip.compress(data, mtime=0)
    if len(compressed) > 4 * 1024 * 1024:
        raise ValueError('compressed runtime notices exceed resource limit')
    with (evidence / 'zig-runtime-catalog.json').open('xb') as out:
        out.write(catalog_bytes)
    notice_path.write_bytes(data)
    (evidence / 'NOTICE.json.gz').write_bytes(compressed)
    print(f'Included {len(files)} exact Zig runtime source records')


def _members(text):
    matches = re.findall(r'([^\s()]+[.]a)\(([^()\n]+)\)', text)
    members = {(PurePosixPath(archive).name, PurePosixPath(member).name) for archive, member in matches}
    if members != EXPECTED_MEMBERS:
        raise ValueError(f'unreviewed archive membership: {sorted(members ^ EXPECTED_MEMBERS)}')
    objects = set(re.findall(r'[^\s()]+[.]o(?=:\()', text))
    if not objects or any(not path.endswith('.rcgu.o') for path in objects):
        raise ValueError('unreviewed direct object input')
    return sorted(members)


def _verify(evidence):
    bundle = json.loads(_read(evidence / 'NOTICE.json', 64 * 1024 * 1024))
    runtimes = bundle.get('non_cargo_runtimes')
    if not isinstance(runtimes, list) or len(runtimes) != 1 or runtimes[0].get('source_revision') != REVISION:
        raise ValueError('Zig runtime notices missing or unreviewed')
    members = _members(_read(evidence / 'native-link.map', 16 * 1024 * 1024).decode('utf-8'))
    with (evidence / 'zig-runtime-members.json').open('x', encoding='utf-8') as out:
        json.dump({'source_revision': REVISION, 'archive_members': members}, out, sort_keys=True, indent=2)
    print('Verified all seven static Zig runtime archive members')


def main():
    if len(sys.argv) != 3 or sys.argv[1] not in ('prepare', 'verify'):
        raise SystemExit('usage: zig_runtime_notices.py prepare|verify EVIDENCE_DIRECTORY')
    try:
        (_prepare if sys.argv[1] == 'prepare' else _verify)(Path(sys.argv[2]))
    except (OSError, ValueError, KeyError, TypeError, subprocess.CalledProcessError) as error:
        raise SystemExit(str(error)) from error


if __name__ == '__main__':
    main()
