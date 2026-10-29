"""Проверить compact notice resource без загрузки native кода."""

from __future__ import annotations

import gzip
import importlib.util
import unittest
from pathlib import Path

SCRIPT = Path(__file__).resolve().parents[1] / "scripts" / "check_embedded_notices.py"
spec = importlib.util.spec_from_file_location("embedded_notices", SCRIPT)
assert spec is not None
assert spec.loader is not None
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)


def _resource(data: bytes) -> bytes:
    compressed = gzip.compress(data, mtime=0)
    return module.MARKER + len(compressed).to_bytes(8, "little") + compressed


class EmbeddedNoticesTest(unittest.TestCase):
    def test_roundtrip_preserves_utf8_notice(self) -> None:
        expected = '{"copyright": "© contributors", "license": "MIT"}'.encode()
        resource = _resource(expected)
        actual, size = module._extract(b"native-prefix" + resource + b"native-suffix")
        self.assertEqual(actual, expected)
        self.assertEqual(size, len(resource) - len(module.MARKER) - 8)

    def test_missing_resource_is_rejected(self) -> None:
        with self.assertRaisesRegex(ValueError, "missing or duplicate"):
            module._extract(b"native without notices")

    def test_duplicate_resource_is_rejected(self) -> None:
        resource = _resource(b"notice")
        with self.assertRaisesRegex(ValueError, "missing or duplicate"):
            module._extract(resource + resource)

    def test_truncated_length_is_rejected(self) -> None:
        with self.assertRaisesRegex(ValueError, "truncated embedded notice length"):
            module._extract(module.MARKER + b"short")

    def test_truncated_payload_is_rejected(self) -> None:
        with self.assertRaisesRegex(ValueError, "invalid embedded notice size"):
            module._extract(module.MARKER + (100).to_bytes(8, "little") + b"short")

    def test_oversized_claim_is_rejected_without_large_allocation(self) -> None:
        length = module.MAX_GZIP + 1
        with self.assertRaisesRegex(ValueError, "invalid embedded notice size"):
            module._extract(module.MARKER + length.to_bytes(8, "little"))

    def test_corrupt_gzip_is_rejected(self) -> None:
        resource = bytearray(_resource(b"notice"))
        resource[-1] ^= 1
        with self.assertRaises((OSError, EOFError)):
            module._extract(bytes(resource))


if __name__ == "__main__":
    unittest.main()
