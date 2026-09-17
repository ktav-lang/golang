package ktav

import (
	"runtime"
	"unsafe"

	"github.com/ebitengine/purego"

	"github.com/ktav-lang/golang/internal/native"
)

// NativeVersion returns the version string baked into the loaded
// `ktav_cabi` shared library. Useful for diagnostics when a stale cache
// or KTAV_LIB_PATH points at an out-of-sync build.
func NativeVersion() (string, error) {
	s, err := native.Load()
	if err != nil {
		return "", err
	}
	ret, _, _ := purego.SyscallN(s.Version)
	if ret == 0 {
		return "", nil
	}
	// Walk until NUL — `const char *` from Rust, static lifetime.
	base := unsafe.Pointer(ret)
	var n int
	for *(*byte)(unsafe.Add(base, n)) != 0 {
		n++
	}
	return string(unsafe.Slice((*byte)(base), n)), nil
}

// ─── FFI glue ─────────────────────────────────────────────────────────

func loadsJSON(src []byte) ([]byte, error) {
	s, err := native.Load()
	if err != nil {
		return nil, err
	}
	return callStringFn(s, s.Loads, src)
}

func loadsStrictJSON(src []byte) ([]byte, error) {
	s, err := native.Load()
	if err != nil {
		return nil, err
	}
	return callStringFn(s, s.LoadsStrict, src)
}

func dumpsJSON(src []byte) ([]byte, error) {
	s, err := native.Load()
	if err != nil {
		return nil, err
	}
	return callStringFn(s, s.Dumps, src)
}

func dumpsForceStringsJSON(src []byte) ([]byte, error) {
	s, err := native.Load()
	if err != nil {
		return nil, err
	}
	return callStringFn(s, s.DumpsForceStrings, src)
}

func emitCanonicalJSON(src []byte) ([]byte, error) {
	s, err := native.Load()
	if err != nil {
		return nil, err
	}
	return callStringFn(s, s.EmitCanonical, src)
}

func formatSource(src []byte) ([]byte, error) {
	s, err := native.Load()
	if err != nil {
		return nil, err
	}
	return callStringFn(s, s.Format, src)
}

// callStringFn invokes a C ABI function with signature
//
//	int fn(const u8 *src, usize src_len,
//	       u8 **out_buf, usize *out_len,
//	       char **out_err, usize *out_err_len)
//
// and returns the output bytes on success or a *Error on failure.
func callStringFn(s *native.Syms, fn uintptr, src []byte) ([]byte, error) {
	var (
		srcPtr    uintptr
		outBuf    uintptr
		outLen    uintptr
		outErr    uintptr
		outErrLen uintptr
	)
	if len(src) > 0 {
		srcPtr = uintptr(unsafe.Pointer(&src[0]))
	}

	rc, _, _ := purego.SyscallN(
		fn,
		srcPtr,
		uintptr(len(src)),
		uintptr(unsafe.Pointer(&outBuf)),
		uintptr(unsafe.Pointer(&outLen)),
		uintptr(unsafe.Pointer(&outErr)),
		uintptr(unsafe.Pointer(&outErrLen)),
	)

	// Keep src alive through the call — Go's escape analysis may not
	// see the C-side access.
	runtime.KeepAlive(src)

	if rc != 0 {
		raw := "ktav: unknown error"
		if outErr != 0 && outErrLen != 0 {
			raw = string(copyFromC(outErr, outErrLen))
			purego.SyscallN(s.Free, outErr, outErrLen)
		}
		return nil, errorFromEnvelope([]byte(raw))
	}

	var out []byte
	if outBuf != 0 && outLen != 0 {
		out = copyFromC(outBuf, outLen)
		purego.SyscallN(s.Free, outBuf, outLen)
	}
	return out, nil
}

func copyFromC(ptr, n uintptr) []byte {
	if n == 0 {
		return nil
	}
	src := unsafe.Slice((*byte)(unsafe.Pointer(ptr)), int(n))
	dst := make([]byte, int(n))
	copy(dst, src)
	return dst
}
