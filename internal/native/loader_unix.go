//go:build darwin || linux || freebsd

package native

import (
	"fmt"

	"github.com/ebitengine/purego"
)

func openLibrary(path string) (uintptr, error) {
	if path == "" {
		return 0, fmt.Errorf("empty library path")
	}
	return purego.Dlopen(path, purego.RTLD_NOW|purego.RTLD_GLOBAL)
}

func dlsym(handle uintptr, name string) (uintptr, error) {
	return purego.Dlsym(handle, name)
}
