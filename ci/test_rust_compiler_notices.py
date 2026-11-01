"""Проверить exact compiler grants без запуска compiler или native payload."""

import hashlib
import json
from pathlib import Path
import shutil
import sys
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / 'scripts'))
from rust_compiler_notices import collect, RUST_REVISION, PROFILE_SHA256

RUSTC = 'rustc 1.99.1 (fixture)\ncommit-hash: ' + RUST_REVISION + '\nrelease: 1.99.1\nLLVM version: 23.1.1\n'


class CompilerNoticesTests(unittest.TestCase):
    def source_copy(self, root):
        profile = ROOT / 'ci/rust-compiler-notices.json'
        (root / 'ci').mkdir()
        shutil.copyfile(profile, root / 'ci/rust-compiler-notices.json')
        for item in json.loads(profile.read_bytes())['materials']:
            path = root / item['path']
            path.parent.mkdir(parents=True, exist_ok=True)
            shutil.copyfile(ROOT / item['path'], path)

    def test_complete_original_collection(self):
        result = collect(ROOT, RUSTC)
        self.assertEqual(result['profile_sha256'], PROFILE_SHA256)
        self.assertEqual(result['compiler_builtins_license'], 'MIT AND Apache-2.0 WITH LLVM-exception')
        self.assertEqual(result['libm_license'], 'MIT')
        self.assertEqual(len(result['materials']), 14)
        self.assertEqual(sum(item['legal_document'] for item in result['materials']), 4)
        for item in result['materials']:
            raw = item['text'].encode('utf-8')
            self.assertEqual(len(raw), item['bytes'])
            self.assertEqual(hashlib.sha256(raw).hexdigest(), item['sha256'])
            self.assertEqual(hashlib.sha1(b'blob ' + str(len(raw)).encode() + bytes([0]) + raw).hexdigest(), item['git_blob'])

    def test_wrong_compiler_source_or_version_is_rejected(self):
        for text in [RUSTC.replace('1.99.1', '1.88.0'), RUSTC.replace(RUST_REVISION, '0' * 40)]:
            with self.assertRaisesRegex(ValueError, 'unreviewed Rust compiler'):
                collect(ROOT, text)

    def test_changed_profile_is_rejected(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            self.source_copy(root)
            (root / 'ci/rust-compiler-notices.json').write_bytes(b'{}')
            with self.assertRaisesRegex(ValueError, 'profile differs'):
                collect(root, RUSTC)

    def test_changed_original_is_rejected(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            self.source_copy(root)
            profile = json.loads((root / 'ci/rust-compiler-notices.json').read_bytes())
            path = root / profile['materials'][0]['path']
            path.write_bytes(path.read_bytes() + b'changed')
            with self.assertRaisesRegex(ValueError, 'original identity differs'):
                collect(root, RUSTC)

    def test_missing_original_is_rejected(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            self.source_copy(root)
            profile = json.loads((root / 'ci/rust-compiler-notices.json').read_bytes())
            (root / profile['materials'][0]['path']).unlink()
            with self.assertRaisesRegex(ValueError, 'missing or symlinked'):
                collect(root, RUSTC)

    def test_copyright_holders_and_legacy_credits_preserved(self):
        result = collect(ROOT, RUSTC)
        documents = {item['role']: item['text'] for item in result['materials'] if item['legal_document']}
        self.assertIn('Jorge Aparicio', documents['libm-LICENSE.txt'])
        self.assertIn('Rich Felker', documents['libm-LICENSE.txt'])
        self.assertIn('LLVM', documents['compiler-builtins-LICENSE.txt'])
        self.assertIn('compiler-rt-CREDITS.TXT', documents)


if __name__ == '__main__':
    unittest.main()
