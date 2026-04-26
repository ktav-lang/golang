package ktav_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ktav-lang/golang/internal/native"
)

// Hardcoded paths anchored at the module root (= test working dir for
// package-level `go test`). Intentionally not configurable — this
// binding implements one specific spec version and ships the cabi
// build into the same workspace.
const (
	cabiBuildDir = "target/release"
	specTestsDir = "spec/versions/0.1/tests"
)

func TestMain(m *testing.M) {
	native.SetLibraryPath(filepath.Join(cabiBuildDir, cabiName()))
	os.Exit(m.Run())
}

func requireCabi(t *testing.T) {
	p := filepath.Join(cabiBuildDir, cabiName())
	if _, err := os.Stat(p); err != nil {
		t.Skipf("cabi not built (%s) — run `cargo build --release -p ktav-cabi`", p)
	}
}

func requireSpec(t *testing.T) string {
	if _, err := os.Stat(specTestsDir); err != nil {
		t.Skipf("spec submodule missing (%s) — run `git submodule update --init`", specTestsDir)
	}
	return specTestsDir
}

func cabiName() string {
	switch runtime.GOOS {
	case "windows":
		return "ktav_cabi.dll"
	case "darwin":
		return "libktav_cabi.dylib"
	default:
		return "libktav_cabi.so"
	}
}
