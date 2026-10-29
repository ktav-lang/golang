"""Подготовить source-backed notices для самостоятельной C ABI библиотеки."""

from __future__ import annotations

import gzip
import json
import shutil
import subprocess
import sys
from pathlib import Path

from cargo_notice_bundle import _build


def _prepare(evidence: Path) -> None:
    root = Path.cwd().resolve()
    evidence = evidence.resolve()
    if evidence.is_relative_to(root):
        raise ValueError("notice evidence must be outside project sources")
    if evidence.exists() and (not evidence.is_dir() or any(evidence.iterdir())):
        raise ValueError("notice evidence directory must be empty")
    lock = Path("Cargo.lock")
    if not lock.is_file() or lock.is_symlink():
        raise ValueError("prepare an existing regular Cargo.lock first")
    original = lock.read_bytes()
    raw = subprocess.check_output(
        ["cargo", "metadata", "--locked", "--all-features", "--format-version", "1"]
    )
    metadata = json.loads(raw)
    crate = (root / "crates" / "cabi").resolve()
    own = [p for p in metadata["packages"] if Path(p["manifest_path"]).resolve().parent == crate]
    if len(own) != 1 or own[0].get("license") != "MIT OR Apache-2.0":
        raise ValueError("unexpected C ABI workspace license")
    for name in ("LICENSE-MIT", "LICENSE-APACHE"):
        original_license = root / name
        target = crate / name
        if target.is_symlink():
            raise ValueError("C ABI license copy must not be a symlink")
        if target.exists():
            if target.read_bytes() != original_license.read_bytes():
                raise ValueError("C ABI license copy differs from workspace license")
        else:
            shutil.copyfile(original_license, target)
    for package in metadata["packages"]:
        if evidence.is_relative_to(Path(package["manifest_path"]).resolve().parent):
            raise ValueError("notice evidence overlaps a Cargo package source")
    evidence.mkdir(parents=True, exist_ok=True)
    (evidence / "metadata.json").write_bytes(raw)
    (evidence / "Cargo.lock").write_bytes(original)
    (evidence / "rustc.txt").write_bytes(
        subprocess.check_output(["rustc", "--version", "--verbose"])
    )
    sysroot = subprocess.check_output(["rustc", "--print", "sysroot"], text=True).strip()
    shutil.copyfile(
        Path(sysroot) / "share" / "doc" / "rust" / "COPYRIGHT-library.html",
        evidence / "COPYRIGHT-library.html",
    )
    licenses = subprocess.check_output(
        [
            "cargo",
            "about",
            "generate",
            "--locked",
            "--all-features",
            "--fail",
            "--config",
            "ci/about.toml",
            "--format",
            "json",
        ]
    )
    (evidence / "licenses.json").write_bytes(licenses)
    data = _build(evidence)
    compressed = gzip.compress(data, mtime=0)
    if len(compressed) > 4 * 1024 * 1024:
        raise ValueError("compressed native notices exceed the resource size limit")
    if lock.read_bytes() != original:
        raise ValueError("Cargo.lock changed during notice preparation")
    (evidence / "NOTICE.json").write_bytes(data)
    (evidence / "NOTICE.json.gz").write_bytes(compressed)
    print(f"Prepared {len(data)} notice bytes, compressed to {len(compressed)} bytes")


def main() -> None:
    """Подготовить notices без изменения dependency resolution."""
    if len(sys.argv) != 2:
        raise SystemExit("usage: prepare_cabi_notices.py EMPTY_EVIDENCE_DIRECTORY")
    try:
        _prepare(Path(sys.argv[1]))
    except (OSError, ValueError, KeyError, TypeError, subprocess.CalledProcessError) as exc:
        raise SystemExit(str(exc)) from exc


if __name__ == "__main__":
    main()
