// Package native resolves and dynamically loads the `ktav_cabi` shared
// library via purego. The Go caller never links to it — we dlopen at
// first use. Resolution order:
//
//  1. $KTAV_LIB_PATH, if set.
//  2. `<cacheDir>/ktav-go/<version>/<name>`, if already downloaded.
//  3. Downloaded from the matching GitHub Release asset into (2).
//
// "Version" is the `LibVersion` constant, synced with the Rust crate
// version at release time. A mismatch between a stale on-disk cache and
// the Go module version will force a fresh download.
package native

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

// LibVersion is the version of the companion `ktav_cabi` shared library
// this Go module expects. It is in lockstep with the Go module tag.
const LibVersion = "0.1.0"

// releaseAssetBase is the GitHub Release where prebuilt binaries live.
// Tagged `v<LibVersion>`; per-platform asset naming is handled below.
const releaseAssetBase = "https://github.com/ktav-lang/golang/releases/download/v"

// Syms holds the function pointers obtained from the loaded library.
type Syms struct {
	Loads   uintptr
	Dumps   uintptr
	Free    uintptr
	Version uintptr
}

var (
	once    sync.Once
	loaded  *Syms
	loadErr error
)

// Load resolves and loads the shared library exactly once, then returns
// the same Syms on every subsequent call.
func Load() (*Syms, error) {
	once.Do(func() {
		path, err := resolvePath()
		if err != nil {
			loadErr = err
			return
		}
		handle, err := openLibrary(path)
		if err != nil {
			loadErr = fmt.Errorf("open %s: %w", path, err)
			return
		}
		s := &Syms{}
		if err := bindSym(handle, "ktav_loads", &s.Loads); err != nil {
			loadErr = err
			return
		}
		if err := bindSym(handle, "ktav_dumps", &s.Dumps); err != nil {
			loadErr = err
			return
		}
		if err := bindSym(handle, "ktav_free", &s.Free); err != nil {
			loadErr = err
			return
		}
		if err := bindSym(handle, "ktav_version", &s.Version); err != nil {
			loadErr = err
			return
		}
		loaded = s
	})
	return loaded, loadErr
}

func bindSym(handle uintptr, name string, out *uintptr) error {
	sym, err := dlsym(handle, name)
	if err != nil {
		return fmt.Errorf("symbol %s: %w", name, err)
	}
	*out = sym
	return nil
}

// resolvePath returns an on-disk path to the shared library, downloading
// it into the user cache if necessary.
func resolvePath() (string, error) {
	if p := os.Getenv("KTAV_LIB_PATH"); p != "" {
		if _, err := os.Stat(p); err != nil {
			return "", fmt.Errorf("KTAV_LIB_PATH=%q: %w", p, err)
		}
		return p, nil
	}

	name, err := assetName()
	if err != nil {
		return "", err
	}

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("UserCacheDir: %w", err)
	}
	dir := filepath.Join(cacheDir, "ktav-go", "v"+LibVersion)
	target := filepath.Join(dir, name)

	if _, err := os.Stat(target); err == nil {
		return target, nil
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("mkdir %s: %w", dir, err)
	}
	url := releaseAssetBase + LibVersion + "/" + name
	if err := download(url, target); err != nil {
		return "", fmt.Errorf("fetch %s: %w", url, err)
	}
	return target, nil
}

func assetName() (string, error) {
	os_, arch := runtime.GOOS, runtime.GOARCH
	key := os_ + "-" + arch
	switch key {
	case "linux-amd64":
		return "libktav_cabi-linux-amd64.so", nil
	case "linux-arm64":
		return "libktav_cabi-linux-arm64.so", nil
	case "darwin-amd64":
		return "libktav_cabi-darwin-amd64.dylib", nil
	case "darwin-arm64":
		return "libktav_cabi-darwin-arm64.dylib", nil
	case "windows-amd64":
		return "ktav_cabi-windows-amd64.dll", nil
	case "windows-arm64":
		return "ktav_cabi-windows-arm64.dll", nil
	default:
		return "", fmt.Errorf("unsupported platform: %s/%s", os_, arch)
	}
}

func download(url, dst string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %s", resp.Status)
	}

	tmp := dst + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	_, err = io.Copy(f, resp.Body)
	closeErr := f.Close()
	if err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return closeErr
	}
	return os.Rename(tmp, dst)
}
