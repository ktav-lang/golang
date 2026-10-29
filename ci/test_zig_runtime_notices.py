"""Проверить fail-closed source identity и состав runtime archives."""

import hashlib
import importlib.util
from pathlib import Path
import tempfile
import unittest

spec = importlib.util.spec_from_file_location(
    'zig_runtime_notices', Path(__file__).resolve().parents[1] / 'scripts/zig_runtime_notices.py'
)
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)


class ZigRuntimeNoticeTest(unittest.TestCase):
    def map_text(self, extra=''):
        lines = [' 0 0 1 1 /build/library.rcgu.o:(.text)']
        lines += [f' 0 0 1 1 /cache/{archive}(/cache/{member}):(.text)' for archive, member in module.EXPECTED_MEMBERS]
        return '\n'.join(lines) + extra

    def test_exact_archive_members(self):
        self.assertEqual(set(module._members(self.map_text())), module.EXPECTED_MEMBERS)

    def test_unknown_archive_rejected(self):
        with self.assertRaisesRegex(ValueError, 'archive membership'):
            module._members(self.map_text('\n 0 0 1 1 /cache/libgcc.a(other.o):(.text)'))

    def test_unknown_member_rejected(self):
        with self.assertRaisesRegex(ValueError, 'archive membership'):
            module._members(self.map_text('\n 0 0 1 1 /cache/libunwind.a(new.o):(.text)'))

    def test_missing_member_rejected(self):
        with self.assertRaisesRegex(ValueError, 'archive membership'):
            module._members(self.map_text().replace('stat64-2.32.o', 'changed.o'))

    def test_direct_crt_rejected(self):
        with self.assertRaisesRegex(ValueError, 'direct object'):
            module._members(self.map_text('\n 0 0 1 1 /cache/crtbeginS.o:(.text)'))

    def test_source_bytes_match_catalog(self):
        with tempfile.TemporaryDirectory() as directory:
            lib = Path(directory) / 'lib'
            source = lib / 'libunwind/source.c'
            source.parent.mkdir(parents=True)
            source.write_bytes(b'reviewed source')
            catalog = {'source_revision': module.REVISION, 'zig_version': '0.14.0', 'files': [
                {'path': 'lib/libunwind/source.c', 'sha256': hashlib.sha256(b'reviewed source').hexdigest()}
            ]}
            self.assertEqual(module._sources(lib, catalog), catalog['files'])
            source.write_bytes(b'changed source')
            with self.assertRaisesRegex(ValueError, 'source changed'):
                module._sources(lib, catalog)

    def test_wrong_revision_rejected(self):
        with self.assertRaisesRegex(ValueError, 'revision'):
            module._sources(Path('.'), {'source_revision': '0' * 40, 'zig_version': '0.14.0'})

    def test_path_escape_rejected(self):
        with self.assertRaisesRegex(ValueError, 'path'):
            module._sources(Path('.'), {
                'source_revision': module.REVISION, 'zig_version': '0.14.0',
                'files': [{'path': 'lib/../outside', 'sha256': '0' * 64}],
            })


if __name__ == '__main__':
    unittest.main()
