#!/usr/bin/env python3
"""Build a file-based Go module proxy tree for the ktav-lang/golang module.

Layout follows `go help module-proxy` semantics for a proxy serving
module zip files:

    <out>/github.com/ktav-lang/golang/@v/list          — one version per line
    <out>/github.com/ktav-lang/golang/@v/v0.8.1.info   — {"Version","Time"}
    <out>/github.com/ktav-lang/golang/@v/v0.8.1.mod    — raw go.mod
    <out>/github.com/ktav-lang/golang/@v/v0.8.1.zip    — module zip

The zip layout is the one `go` requires for module zips downloaded from
a proxy: every path is `<module>@<version>/<path>` with the module's
go.mod at `github.com/ktav-lang/golang@v0.8.1/go.mod`.

Usage: module_proxy.py <module-root> <version> <out-dir>

Build-junk directories (worktrees, target, node_modules, .git, the spec
submodule, CI artifacts) are excluded from the zip.
"""

import json
import os
import sys
import time
import zipfile

EXCLUDE_DIRS = {
    ".git", ".github", "worktrees", "target", "node_modules",
    "spec", "testdata", "__pycache__",
}


def main(root: str, version: str, out: str) -> int:
    module = "github.com/ktav-lang/golang"
    atv = f"{module}@{version}"
    vdir = os.path.join(out, *module.split("/"), "@v")

    mod_src = os.path.join(root, "go.mod")
    if not os.path.isfile(mod_src):
        print(f"module_proxy: no go.mod under {root}", file=sys.stderr)
        return 1
    with open(mod_src, "rb") as f:
        mod_bytes = f.read()

    os.makedirs(vdir, exist_ok=True)
    with open(os.path.join(vdir, "list"), "w", newline="\n") as f:
        f.write(version + "\n")
    with open(os.path.join(vdir, version + ".info"), "w", newline="\n") as f:
        json.dump({"Version": version, "Time": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())}, f)
        f.write("\n")
    with open(os.path.join(vdir, version + ".mod"), "wb") as f:
        f.write(mod_bytes)

    zip_path = os.path.join(vdir, version + ".zip")
    files = 0
    with zipfile.ZipFile(zip_path, "w", zipfile.ZIP_DEFLATED) as z:
        for dirpath, dirnames, filenames in os.walk(root):
            dirnames[:] = [d for d in dirnames if d not in EXCLUDE_DIRS]
            for name in filenames:
                full = os.path.join(dirpath, name)
                rel = os.path.relpath(full, root).replace(os.sep, "/")
                z.write(full, f"{atv}/{rel}")
                files += 1
    # Sanity: the zip must carry the module's own go.mod at the root.
    with zipfile.ZipFile(zip_path) as z:
        names = z.namelist()
    if f"{atv}/go.mod" not in names:
        print("module_proxy: zip is missing <module>@<version>/go.mod", file=sys.stderr)
        return 1
    print(f"module_proxy: wrote {zip_path} ({files} files)")
    return 0


if __name__ == "__main__":
    if len(sys.argv) != 4:
        print(__doc__, file=sys.stderr)
        sys.exit(2)
    sys.exit(main(sys.argv[1], sys.argv[2], sys.argv[3]))
