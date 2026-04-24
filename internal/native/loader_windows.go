//go:build windows

package native

import (
	"fmt"

	"golang.org/x/sys/windows"
)

func openLibrary(path string) (uintptr, error) {
	if path == "" {
		return 0, fmt.Errorf("empty library path")
	}
	h, err := windows.LoadLibrary(path)
	if err != nil {
		return 0, err
	}
	return uintptr(h), nil
}

func dlsym(handle uintptr, name string) (uintptr, error) {
	return windows.GetProcAddress(windows.Handle(handle), name)
}
