"""Проверить прозрачность audit wrapper и передачу ошибок."""

import os
from pathlib import Path
import subprocess
import tempfile
import unittest


class ZigLinkCaptureTest(unittest.TestCase):
    def setUp(self):
        self.directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.directory.cleanup)
        self.root = Path(self.directory.name)
        self.real = self.root / 'actual-zig'
        self.real.write_text('#!/usr/bin/env bash\nprintf "%s\\n" "$@"\nprintf diagnostic >&2\nexit "${STATUS:-0}"\n')
        self.real.chmod(0o755)
        self.output = self.root / 'link.map'
        self.wrapper = Path(__file__).resolve().parents[1] / 'scripts/capture-zig-link.sh'

    def invoke(self, *args, status=0):
        return subprocess.run(
            ['bash', str(self.wrapper), *args],
            env={**os.environ, 'KTAV_REAL_ZIG': str(self.real), 'KTAV_LINK_MAP': str(self.output), 'STATUS': str(status)},
            capture_output=True,
        )

    def test_plain_invocation_does_not_create_map(self):
        result = self.invoke('version')
        self.assertEqual(result.returncode, 0)
        self.assertEqual(result.stdout, b'version\n')
        self.assertEqual(result.stderr, b'diagnostic')
        self.assertFalse(self.output.exists())

    def test_map_preserves_arguments_and_streams(self):
        result = self.invoke('cc', '-Wl,--print-map', 'path with spaces.o')
        self.assertEqual(result.returncode, 0)
        self.assertEqual(result.stdout, b'cc\n-Wl,--print-map\npath with spaces.o\n')
        self.assertEqual(self.output.read_bytes(), result.stdout)
        self.assertEqual(result.stderr, b'diagnostic')

    def test_failed_link_is_not_hidden_by_tee(self):
        result = self.invoke('cc', '-Wl,--print-map', status=23)
        self.assertEqual(result.returncode, 23)
        self.assertEqual(self.output.read_bytes(), result.stdout)
        self.assertEqual(result.stderr, b'diagnostic')

    def test_failed_plain_command_is_preserved(self):
        result = self.invoke('version', status=19)
        self.assertEqual(result.returncode, 19)
        self.assertFalse(self.output.exists())


if __name__ == '__main__':
    unittest.main()
