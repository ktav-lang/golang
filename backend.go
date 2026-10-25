package ktav

import (
	"sync/atomic"

	"github.com/ktav-lang/golang/internal/engine"
)

func enginePureLoads(b []byte) ([]byte, error)             { return engine.PureLoads(b) }
func enginePureLoadsStrict(b []byte) ([]byte, error)       { return engine.PureLoadsStrict(b) }
func enginePureDumps(b []byte) ([]byte, error)             { return engine.PureDumps(b) }
func enginePureDumpsForceStrings(b []byte) ([]byte, error) { return engine.PureDumpsForceStrings(b) }
func enginePureEmitCanonical(b []byte) ([]byte, error)     { return engine.PureEmitCanonical(b) }
func enginePureFormatSource(b []byte) ([]byte, error)      { return engine.PureFormatSource(b) }
func enginePureCanonicalFromSource(b []byte) ([]byte, error) {
	return engine.PureCanonicalFromSource(b)
}

func enginePureEditValue(src, path, replacement []byte) ([]byte, error) {
	return engine.PureEditValue(src, path, replacement)
}

func enginePureEditValueSegments(src, path, replacement []byte) ([]byte, error) {
	return engine.PureEditValueSegments(src, path, replacement)
}

// Backend selects which implementation backs the package's public
// operations.
//
// The zero value is not used: the package starts on [BackendNative]
// (the historical behaviour) and every switch is explicit.
type Backend int

const (
	// BackendNative dispatches through the purego-loaded `ktav_cabi`
	// shared library (downloaded and cached on first use). This is the
	// default and preserves the historical behaviour exactly, including
	// a native load failure surfacing as an error rather than a
	// fallback.
	BackendNative Backend = iota
	// BackendPure dispatches to the in-process pure-Go parser and
	// writer. Pure mode never loads, downloads or requires the native
	// library, and never spawns a subprocess.
	BackendPure
)

// backendVTable is the dispatch seam: one set of function pointers the
// public operations call through a single atomic pointer read. The pure
// entries never touch the purego/loader code paths; the native entries
// are the ktav_ffi.go wrappers unchanged.
type backendVTable struct {
	loads             func(src []byte) ([]byte, error)
	loadsStrict       func(src []byte) ([]byte, error)
	dumps             func(src []byte) ([]byte, error)
	dumpsForceStrings func(src []byte) ([]byte, error)
	emitCanonical     func(src []byte) ([]byte, error)
	formatSource      func(src []byte) ([]byte, error)
	canonicalFromSrc  func(src []byte) ([]byte, error)
	editValue         func(src, path, replacement []byte) ([]byte, error)
	editValueSegments func(src, path, replacement []byte) ([]byte, error)
}

func nativeVTable() *backendVTable {
	return &backendVTable{
		loads:             loadsJSON,
		loadsStrict:       loadsStrictJSON,
		dumps:             dumpsJSON,
		dumpsForceStrings: dumpsForceStringsJSON,
		emitCanonical:     emitCanonicalJSON,
		formatSource:      formatSource,
		canonicalFromSrc:  canonicalFromSourceJSON,
		editValue:         editValueNative,
		editValueSegments: editValueSegmentsNative,
	}
}

func pureVTable() *backendVTable {
	return &backendVTable{
		loads:             enginePureLoads,
		loadsStrict:       enginePureLoadsStrict,
		dumps:             enginePureDumps,
		dumpsForceStrings: enginePureDumpsForceStrings,
		emitCanonical:     enginePureEmitCanonical,
		formatSource:      enginePureFormatSource,
		canonicalFromSrc:  enginePureCanonicalFromSource,
		editValue:         enginePureEditValue,
		editValueSegments: enginePureEditValueSegments,
	}
}

// backendPtr holds the active vtable. Switching is a pointer swap; each
// call reads it once, so there is no mutex in the hot path and no
// data race.
var backendPtr atomic.Pointer[backendVTable]

func init() { backendPtr.Store(nativeVTable()) }

func activeBackend() *backendVTable { return backendPtr.Load() }

// canonicalFromSourceJSON is the native backend's text→text canonical
// composition of the two FFI calls (the cabi exposes it as one symbol;
// the Go binding has always chained the two, byte-identically).
func canonicalFromSourceJSON(src []byte) ([]byte, error) {
	js, err := loadsJSON(src)
	if err != nil {
		return nil, err
	}
	return emitCanonicalJSON(js)
}

// UseBackend selects the implementation backing all public operations.
//
// There is no automatic fallback in either direction: pure mode never
// re-routes to the native library (and vice versa), so a defect fails
// loudly instead of silently switching implementations. Calling
// UseBackend again simply swaps the vtable; concurrent in-flight calls
// finish on the backend they started on.
func UseBackend(b Backend) error {
	switch b {
	case BackendNative:
		backendPtr.Store(nativeVTable())
	case BackendPure:
		backendPtr.Store(pureVTable())
	default:
		return engine.NewErrorf("ktav: unknown backend %d", int(b))
	}
	return nil
}

// UsePure selects the pure-Go backend. It never loads, downloads or
// requires the native `ktav_cabi` library and never spawns a
// subprocess; it is safe to call before any other package use in
// environments where no native artifact is present.
func UsePure() error { return UseBackend(BackendPure) }

// UseNative selects the native (purego cabi) backend — the default.
// The shared library is loaded lazily on first use, exactly as before.
func UseNative() error { return UseBackend(BackendNative) }

// Operations is a set of public operations bound to ONE fixed backend.
// Every method dispatches through the instance's own function table and
// never reads or writes the package-global default, so concurrent
// Operations values (and the package-level functions) stay fully
// isolated from each other.
type Operations struct {
	vt *backendVTable
}

// New creates an Operations bound to the given backend. BackendNative
// resolves the purego cabi lazily on first call (and fails with an
// error if the artifact is unavailable); BackendPure never touches the
// loader. The package-level functions remain the global-default API
// and are unaffected by any number of live Operations values.
func New(b Backend) (*Operations, error) {
	switch b {
	case BackendNative:
		return &Operations{vt: nativeVTable()}, nil
	case BackendPure:
		return &Operations{vt: pureVTable()}, nil
	default:
		return nil, engine.NewErrorf("ktav: unknown backend %d", int(b))
	}
}
