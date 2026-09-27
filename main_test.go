package ktav_test

import (
	"fmt"
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
	specTestsDir = "spec/versions/0.8/tests"
)

func TestMain(m *testing.M) {
	if err := validateSpec08Corpus(specTestsDir); err != nil {
		fmt.Fprintf(os.Stderr, "conformance corpus guard failed: %v\n", err)
		os.Exit(1)
	}
	path := filepath.Join(cabiBuildDir, cabiName())
	if dir := os.Getenv("CARGO_TARGET_DIR"); dir != "" {
		path = filepath.Join(dir, "release", cabiName())
	}
	if override := os.Getenv("KTAV_LIB_PATH"); override != "" {
		path = override
	}
	native.SetLibraryPath(path)
	os.Exit(m.Run())
}

func requireCabi(t *testing.T) {
	p := filepath.Join(cabiBuildDir, cabiName())
	if dir := os.Getenv("CARGO_TARGET_DIR"); dir != "" {
		p = filepath.Join(dir, "release", cabiName())
	}
	if override := os.Getenv("KTAV_LIB_PATH"); override != "" {
		p = override
	}
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("cabi unavailable (%s): %v; build with `cargo build --release -p ktav-cabi`", p, err)
	}
}

func requireSpec(*testing.T) string {
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
