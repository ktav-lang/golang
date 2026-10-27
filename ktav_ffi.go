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

// editValueNative invokes ktav_edit_value: source, textual path and
// tagged-wire replacement; output is UTF-8 source bytes.
func editValueNative(src, path, replacement []byte) ([]byte, error) {
	s, err := native.Load()
	if err != nil {
		return nil, err
	}
	return callEditFn(s, s.EditValue, src, path, replacement)
}

// editValueSegmentsNative invokes ktav_edit_value_segments: path is the
// UTF-8 JSON array of literal segment strings.
func editValueSegmentsNative(src, path, replacement []byte) ([]byte, error) {
	s, err := native.Load()
	if err != nil {
		return nil, err
	}
	return callEditFn(s, s.EditValueSegments, src, path, replacement)
}

// insertValueNative invokes ktav_insert_value: source, textual path
// (final segment is the NEW Object key) and tagged-wire replacement.
func insertValueNative(src, path, replacement []byte) ([]byte, error) {
	s, err := native.Load()
	if err != nil {
		return nil, err
	}
	return callEditFn(s, s.InsertValue, src, path, replacement)
}

// insertValueSegmentsNative invokes ktav_insert_value_segments: path is
// the UTF-8 JSON array of literal segment strings.
func insertValueSegmentsNative(src, path, replacement []byte) ([]byte, error) {
	s, err := native.Load()
	if err != nil {
		return nil, err
	}
	return callEditFn(s, s.InsertValueSegments, src, path, replacement)
}

// removeValueNative invokes ktav_remove_value: source + textual path
// only — no replacement operand.
func removeValueNative(src, path []byte) ([]byte, error) {
	s, err := native.Load()
	if err != nil {
		return nil, err
	}
	return callEdit2Fn(s, s.RemoveValue, src, path)
}

// removeValueSegmentsNative invokes ktav_remove_value_segments: path is
// the UTF-8 JSON array of literal segment strings.
func removeValueSegmentsNative(src, path []byte) ([]byte, error) {
	s, err := native.Load()
	if err != nil {
		return nil, err
	}
	return callEdit2Fn(s, s.RemoveValueSegments, src, path)
}

// renameKeyNative invokes ktav_rename_key: newKey is ONE literal UTF-8
// key — no JSON encoding, no path escaping.
func renameKeyNative(src, path, newKey []byte) ([]byte, error) {
	s, err := native.Load()
	if err != nil {
		return nil, err
	}
	return callEditFn(s, s.RenameKey, src, path, newKey)
}

// renameKeySegmentsNative invokes ktav_rename_key_segments.
func renameKeySegmentsNative(src, path, newKey []byte) ([]byte, error) {
	s, err := native.Load()
	if err != nil {
		return nil, err
	}
	return callEditFn(s, s.RenameKeySegments, src, path, newKey)
}

var emptyInputByte byte

// callEditFn invokes a C ABI function with signature
//
//	int fn(const u8 *src, usize src_len,
//	       const u8 *path, usize path_len,
//	       const u8 *replacement, usize replacement_len,
//	       u8 **out_buf, usize *out_len,
//	       char **out_err, usize *out_err_len)
//
// and returns the output bytes on success or a *Error on failure.
func callEditFn(s *native.Syms, fn uintptr, src, path, replacement []byte) ([]byte, error) {
	var (
		srcPtr    uintptr
		pathPtr   uintptr
		replPtr   uintptr
		outBuf    uintptr
		outLen    uintptr
		outErr    uintptr
		outErrLen uintptr
	)
	if len(src) > 0 {
		srcPtr = uintptr(unsafe.Pointer(&src[0]))
	} else {
		srcPtr = uintptr(unsafe.Pointer(&emptyInputByte))
	}
	if len(path) > 0 {
		pathPtr = uintptr(unsafe.Pointer(&path[0]))
	} else {
		pathPtr = uintptr(unsafe.Pointer(&emptyInputByte))
	}
	if len(replacement) > 0 {
		replPtr = uintptr(unsafe.Pointer(&replacement[0]))
	} else {
		replPtr = uintptr(unsafe.Pointer(&emptyInputByte))
	}

	rc, _, _ := purego.SyscallN(
		fn,
		srcPtr,
		uintptr(len(src)),
		pathPtr,
		uintptr(len(path)),
		replPtr,
		uintptr(len(replacement)),
		uintptr(unsafe.Pointer(&outBuf)),
		uintptr(unsafe.Pointer(&outLen)),
		uintptr(unsafe.Pointer(&outErr)),
		uintptr(unsafe.Pointer(&outErrLen)),
	)

	// Keep the inputs alive through the call — Go's escape analysis may
	// not see the C-side access.
	runtime.KeepAlive(src)
	runtime.KeepAlive(path)
	runtime.KeepAlive(replacement)

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

// callEdit2Fn invokes a C ABI function with signature
//
//	int fn(const u8 *src, usize src_len,
//	       const u8 *path, usize path_len,
//	       u8 **out_buf, usize *out_len,
//	       char **out_err, usize *out_err_len)
//
// and returns the output bytes on success or a *Error on failure.
func callEdit2Fn(s *native.Syms, fn uintptr, src, path []byte) ([]byte, error) {
	var (
		srcPtr    uintptr
		pathPtr   uintptr
		outBuf    uintptr
		outLen    uintptr
		outErr    uintptr
		outErrLen uintptr
	)
	if len(src) > 0 {
		srcPtr = uintptr(unsafe.Pointer(&src[0]))
	} else {
		srcPtr = uintptr(unsafe.Pointer(&emptyInputByte))
	}
	if len(path) > 0 {
		pathPtr = uintptr(unsafe.Pointer(&path[0]))
	} else {
		pathPtr = uintptr(unsafe.Pointer(&emptyInputByte))
	}

	rc, _, _ := purego.SyscallN(
		fn,
		srcPtr,
		uintptr(len(src)),
		pathPtr,
		uintptr(len(path)),
		uintptr(unsafe.Pointer(&outBuf)),
		uintptr(unsafe.Pointer(&outLen)),
		uintptr(unsafe.Pointer(&outErr)),
		uintptr(unsafe.Pointer(&outErrLen)),
	)

	// Keep the inputs alive through the call — Go's escape analysis may
	// not see the C-side access.
	runtime.KeepAlive(src)
	runtime.KeepAlive(path)

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
	} else {
		srcPtr = uintptr(unsafe.Pointer(&emptyInputByte))
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
