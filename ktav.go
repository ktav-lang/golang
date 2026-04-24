// Package ktav is the Go binding for the Ktav configuration format.
//
// The implementation loads a prebuilt `ktav_cabi` shared library via
// purego (no cgo required on the consumer side). On first call the
// library is downloaded from the matching GitHub Release and cached
// under the user cache directory; set $KTAV_LIB_PATH to point at a
// local build instead.
//
// # Type mapping
//
// Loads/Dumps convert between Ktav values and Go values as follows:
//
//	Ktav              Go
//	─────────────── ───────────────────────────
//	null              nil
//	true / false      bool
//	:i <digits>       int64 if it fits, else *big.Int
//	:f <number>       float64
//	bare scalar       string
//	[ ... ]           []any
//	{ ... }           map[string]any (insertion-ordered via orderedmap)
//
// On encode, Go *big.Int always emits `:i`; Go int / int64 / uint64
// emit `:i`; Go float64 emits `:f`; Go string emits a bare scalar.
// NaN / ±Inf are rejected. Top-level value must encode to a Ktav
// object (i.e. a map[string]any or struct).
package ktav

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"unsafe"

	"github.com/ebitengine/purego"

	"github.com/ktav-lang/golang/internal/native"
)

// Error is the Go-side counterpart of a Ktav parse/render error.
type Error struct{ Msg string }

func (e *Error) Error() string { return e.Msg }

// Loads parses a Ktav document and returns its Go representation (see
// package doc for the mapping).
func Loads(src string) (any, error) {
	js, err := loadsJSON([]byte(src))
	if err != nil {
		return nil, err
	}
	return decodeJSON(js)
}

// LoadsInto parses a Ktav document and JSON-unmarshals the tagged
// intermediate into `target`. Handy for struct-typed configs:
//
//	var cfg MyConfig
//	_ = ktav.LoadsInto(src, &cfg)
//
// `:i` scalars become JSON numbers or JSON strings (if they exceed
// json.Number precision); `:f` scalars become JSON numbers. Custom
// types wanting bigint precision should unmarshal into a json.Number
// field.
func LoadsInto(src string, target any) error {
	js, err := loadsJSON([]byte(src))
	if err != nil {
		return err
	}
	plain, err := flattenTagged(js)
	if err != nil {
		return err
	}
	return json.Unmarshal(plain, target)
}

// Dumps renders a Go value as a Ktav document. The top-level must
// encode to a JSON object (map[string]any, struct, etc.).
func Dumps(v any) (string, error) {
	tagged, err := encodeTagged(v)
	if err != nil {
		return "", err
	}
	out, err := dumpsJSON(tagged)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// ─── FFI glue ─────────────────────────────────────────────────────────

func loadsJSON(src []byte) ([]byte, error) {
	s, err := native.Load()
	if err != nil {
		return nil, err
	}
	return callStringFn(s, s.Loads, src)
}

func dumpsJSON(src []byte) ([]byte, error) {
	s, err := native.Load()
	if err != nil {
		return nil, err
	}
	return callStringFn(s, s.Dumps, src)
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
	runtimeKeepAlive(src)

	if rc != 0 {
		msg := "ktav: unknown error"
		if outErr != 0 && outErrLen != 0 {
			msg = string(copyFromC(outErr, outErrLen))
			purego.SyscallN(s.Free, outErr, outErrLen)
		}
		return nil, &Error{Msg: msg}
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

// runtimeKeepAlive is a thin shim for runtime.KeepAlive so the unsafe
// import is the only thing tying this file to the runtime.
func runtimeKeepAlive(x any) { _ = x }

// ─── tagged-JSON <-> Go conversion ────────────────────────────────────

// decodeJSON walks the Rust-emitted JSON (with $i / $f tags) into native
// Go values. Object key order is preserved via orderedMap.
func decodeJSON(raw []byte) (any, error) {
	if len(raw) == 0 {
		return nil, errors.New("ktav: empty decode input")
	}
	dec := json.NewDecoder(bytesReader(raw))
	dec.UseNumber()
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	return decodeValue(dec, tok)
}

func decodeValue(dec *json.Decoder, tok json.Token) (any, error) {
	switch t := tok.(type) {
	case json.Delim:
		switch t {
		case '{':
			return decodeObject(dec)
		case '[':
			return decodeArray(dec)
		default:
			return nil, fmt.Errorf("unexpected delim %q", t)
		}
	case bool:
		return t, nil
	case json.Number:
		if i, err := t.Int64(); err == nil {
			return i, nil
		}
		return t.Float64()
	case string:
		return t, nil
	case nil:
		return nil, nil
	default:
		return nil, fmt.Errorf("unexpected token %T", tok)
	}
}

func decodeArray(dec *json.Decoder) ([]any, error) {
	out := []any{}
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		v, err := decodeValue(dec, tok)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	if _, err := dec.Token(); err != nil { // consume ']'
		return nil, err
	}
	return out, nil
}

func decodeObject(dec *json.Decoder) (any, error) {
	type kv struct {
		k string
		v any
	}
	var entries []kv
	for dec.More() {
		ktok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		key, ok := ktok.(string)
		if !ok {
			return nil, fmt.Errorf("non-string object key: %v", ktok)
		}
		vtok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		v, err := decodeValue(dec, vtok)
		if err != nil {
			return nil, err
		}
		entries = append(entries, kv{key, v})
	}
	if _, err := dec.Token(); err != nil { // consume '}'
		return nil, err
	}

	// Detect the tagged wrappers: a single entry with key "$i" or "$f"
	// and a string payload.
	if len(entries) == 1 {
		e := entries[0]
		if e.k == "$i" || e.k == "$f" {
			if s, ok := e.v.(string); ok {
				if e.k == "$i" {
					return parseIntegerScalar(s)
				}
				f, err := parseFloatScalar(s)
				if err != nil {
					return nil, err
				}
				return f, nil
			}
			return nil, fmt.Errorf("%s payload must be a string", e.k)
		}
	}

	m := orderedMap{
		keys: make([]string, 0, len(entries)),
		data: make(map[string]any, len(entries)),
	}
	for _, e := range entries {
		m.keys = append(m.keys, e.k)
		m.data[e.k] = e.v
	}
	return m.asMap(), nil
}

// ─── tagged → untagged flattening (for LoadsInto) ─────────────────────

func flattenTagged(raw []byte) ([]byte, error) {
	var v any
	dec := json.NewDecoder(bytesReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	v = flattenAny(v)
	return json.Marshal(v)
}

func flattenAny(v any) any {
	switch t := v.(type) {
	case map[string]any:
		if len(t) == 1 {
			if s, ok := t["$i"].(string); ok {
				if n, err := parseIntegerScalar(s); err == nil {
					return n
				}
				return json.Number(s)
			}
			if s, ok := t["$f"].(string); ok {
				return json.Number(s)
			}
		}
		out := make(map[string]any, len(t))
		for k, val := range t {
			out[k] = flattenAny(val)
		}
		return out
	case []any:
		for i, x := range t {
			t[i] = flattenAny(x)
		}
		return t
	default:
		return v
	}
}

// ─── Go → tagged JSON (for Dumps) ─────────────────────────────────────

func encodeTagged(v any) ([]byte, error) {
	enc, err := toTagged(v)
	if err != nil {
		return nil, err
	}
	return json.Marshal(enc)
}

func toTagged(v any) (any, error) {
	switch t := v.(type) {
	case nil:
		return nil, nil
	case bool:
		return t, nil
	case string:
		return t, nil
	case int:
		return taggedI(fmt.Sprintf("%d", t)), nil
	case int8:
		return taggedI(fmt.Sprintf("%d", t)), nil
	case int16:
		return taggedI(fmt.Sprintf("%d", t)), nil
	case int32:
		return taggedI(fmt.Sprintf("%d", t)), nil
	case int64:
		return taggedI(fmt.Sprintf("%d", t)), nil
	case uint:
		return taggedI(fmt.Sprintf("%d", t)), nil
	case uint8:
		return taggedI(fmt.Sprintf("%d", t)), nil
	case uint16:
		return taggedI(fmt.Sprintf("%d", t)), nil
	case uint32:
		return taggedI(fmt.Sprintf("%d", t)), nil
	case uint64:
		return taggedI(fmt.Sprintf("%d", t)), nil
	case *big.Int:
		if t == nil {
			return nil, nil
		}
		return taggedI(t.String()), nil
	case big.Int:
		return taggedI(t.String()), nil
	case float32:
		return floatTag(float64(t))
	case float64:
		return floatTag(t)
	case json.Number:
		s := string(t)
		if s == "" {
			return nil, errors.New("empty json.Number")
		}
		if isIntegerLiteral(s) {
			return taggedI(s), nil
		}
		return taggedF(s), nil
	case []any:
		out := make([]any, len(t))
		for i, x := range t {
			y, err := toTagged(x)
			if err != nil {
				return nil, err
			}
			out[i] = y
		}
		return out, nil
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, x := range t {
			y, err := toTagged(x)
			if err != nil {
				return nil, err
			}
			out[k] = y
		}
		return out, nil
	default:
		// Fall back through encoding/json: lets callers pass structs
		// and custom maps via reflection.
		raw, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		var any1 any
		dec := json.NewDecoder(bytesReader(raw))
		dec.UseNumber()
		if err := dec.Decode(&any1); err != nil {
			return nil, err
		}
		return toTagged(any1)
	}
}

func taggedI(digits string) map[string]any {
	return map[string]any{"$i": digits}
}

func taggedF(text string) map[string]any {
	return map[string]any{"$f": text}
}

func floatTag(f float64) (any, error) {
	if isNaN(f) || isInf(f) {
		return nil, errors.New("ktav: NaN / Inf are not representable")
	}
	s := formatFloat(f)
	return taggedF(s), nil
}

// ─── misc small helpers ───────────────────────────────────────────────

func parseIntegerScalar(s string) (any, error) {
	if s == "" {
		return nil, errors.New("empty :i scalar")
	}
	// Try to fit into int64 first; otherwise return *big.Int so the
	// caller never silently loses precision.
	var sign int64 = 1
	rest := s
	if rest[0] == '-' {
		sign = -1
		rest = rest[1:]
	}
	var acc int64
	overflow := false
	for i := 0; i < len(rest); i++ {
		c := rest[i]
		if c < '0' || c > '9' {
			return nil, fmt.Errorf("non-digit in :i scalar: %q", s)
		}
		d := int64(c - '0')
		if acc > (1<<62)/10 {
			overflow = true
			break
		}
		acc = acc*10 + d
	}
	if !overflow {
		return acc * sign, nil
	}
	bi := new(big.Int)
	if _, ok := bi.SetString(s, 10); !ok {
		return nil, fmt.Errorf("bad integer literal: %q", s)
	}
	return bi, nil
}

func parseFloatScalar(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%g", &f)
	if err != nil {
		return 0, fmt.Errorf("bad float literal: %q", s)
	}
	return f, nil
}

func isIntegerLiteral(s string) bool {
	if s == "" {
		return false
	}
	i := 0
	if s[0] == '-' || s[0] == '+' {
		i++
	}
	if i == len(s) {
		return false
	}
	for ; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}
