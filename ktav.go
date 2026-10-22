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
//	integer scalar    int64 when it fits the implementation's range
//	float scalar      float64
//	bare scalar       string
//	[ ... ]           []any
//	{ ... }           map[string]any (key order not preserved)
//
// Under spec 0.5, integer and float values are inferred from the
// scalar body's lexical form (bare `42` → Integer, `3.14` → Float).
// The typed markers `:i` / `:f` no longer exist. Under spec § 5.2,
// integer spellings outside the native int64 range are inferred as
// Strings. Go *big.Int values can be encoded, but an out-of-range value
// round-trips through Loads as a String.
//
// Key order from the source is **not** preserved on either side: decode
// returns a plain `map[string]any`, and encode goes through
// `encoding/json`, which emits object keys in alphabetical order. If
// you need a fixed shape, use `LoadsInto` into a struct.
//
// On encode, Go *big.Int always emits an integer scalar; Go int /
// int64 / uint64 emit an integer scalar; Go float64 emits a float
// scalar. NaN / ±Inf are rejected by Dumps and EmitCanonical, but
// coerced to strings by DumpsForceStrings. The top-level value must
// encode to a Ktav object (i.e. a map[string]any or struct) or a Ktav array
// (i.e. a []any or any other JSON-encodable slice). Root arrays follow
// spec § 5.9.3: delimiters are added when needed to disambiguate the
// first item.
//
// [FormatSource] reformats Ktav source text (comments preserved,
// fixed point); [CanonicalFromSource] re-emits it canonically.
package ktav

import (
	"encoding/json"
)

// loadsVia / loadsStrictVia / ... are the shared bodies of the
// package-level functions and the per-instance Operations methods: the
// ONLY difference is which vtable they dispatch through.
func loadsVia(vt *backendVTable, src string) (any, error) {
	if err := invalidUTF8Error(src); err != nil {
		return nil, err
	}
	js, err := vt.loads([]byte(src))
	if err != nil {
		return nil, err
	}
	return decodeJSON(js)
}

func loadsStrictVia(vt *backendVTable, src string) (any, error) {
	if err := invalidUTF8Error(src); err != nil {
		return nil, err
	}
	js, err := vt.loadsStrict([]byte(src))
	if err != nil {
		return nil, err
	}
	return decodeJSON(js)
}

func loadsIntoVia(vt *backendVTable, src string, target any) error {
	if err := invalidUTF8Error(src); err != nil {
		return err
	}
	js, err := vt.loads([]byte(src))
	if err != nil {
		return err
	}
	plain, err := flattenTagged(js)
	if err != nil {
		return err
	}
	return json.Unmarshal(plain, target)
}

func dumpsVia(vt *backendVTable, v any) (string, error) {
	tagged, err := encodeWriterTagged(v, false)
	if err != nil {
		return "", err
	}
	out, err := vt.dumps(tagged)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func dumpsForceStringsVia(vt *backendVTable, v any) (string, error) {
	tagged, err := encodeWriterTagged(v, true)
	if err != nil {
		return "", err
	}
	out, err := vt.dumpsForceStrings(tagged)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func emitCanonicalVia(vt *backendVTable, v any) (string, error) {
	tagged, err := encodeWriterTagged(v, false)
	if err != nil {
		return "", err
	}
	out, err := vt.emitCanonical(tagged)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func formatSourceVia(vt *backendVTable, src string) (string, error) {
	if err := invalidUTF8Error(src); err != nil {
		return "", err
	}
	out, err := vt.formatSource([]byte(src))
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func canonicalFromSourceVia(vt *backendVTable, src string) (string, error) {
	if err := invalidUTF8Error(src); err != nil {
		return "", err
	}
	out, err := vt.canonicalFromSrc([]byte(src))
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// Loads parses a Ktav document and returns its Go representation (see
// package doc for the mapping).
func Loads(src string) (any, error) { return loadsVia(activeBackend(), src) }

// LoadsStrict parses a Ktav document using the strict parser. Strict mode
// rejects lossy numeric spellings while preserving the same Go type mapping
// as Loads.
func LoadsStrict(src string) (any, error) { return loadsStrictVia(activeBackend(), src) }

// LoadsInto parses a Ktav document and JSON-unmarshals the tagged
// intermediate into `target`. Handy for struct-typed configs:
//
//	var cfg MyConfig
//	_ = ktav.LoadsInto(src, &cfg)
//
// Numeric scalars are flattened to JSON numbers, preserving their text
// through json.Number where the destination type supports it.
func LoadsInto(src string, target any) error { return loadsIntoVia(activeBackend(), src, target) }

// Dumps renders a Go value as a Ktav document. The top-level must
// encode to a JSON object (map[string]any, struct, etc.) or a JSON
// array (a slice). Root arrays follow spec § 5.9.3: delimiters are
// added when needed to disambiguate the first item.
func Dumps(v any) (string, error) { return dumpsVia(activeBackend(), v) }

// DumpsForceStrings renders a Go value as a Ktav document with **every
// scalar coerced to a String**: integers, floats, booleans, and null
// are flattened to their textual form via the raw-marker `::`.
// Compounds (Object / Array) preserve their structure; only leaf
// scalars are coerced.
//
// The output round-trips back through `Loads` as the same set of
// String scalars — useful for environments or downstream consumers
// that don't understand inferred numeric types, or for diffs where
// you want the textual form to be the canonical source of truth.
//
// Top-level shape rules match `Dumps`: object or array.
func DumpsForceStrings(v any) (string, error) { return dumpsForceStringsVia(activeBackend(), v) }

// EmitCanonical renders a Go value as a **canonical** Ktav document
// (spec § 5.9 — byte-deterministic output, no inline compounds,
// canonical float/integer normalisation). Two calls with identical
// values always produce identical bytes.
//
// Note: because Go maps are unordered, object key order in the output
// is determined by the tagged-JSON encoding (alphabetical for
// map[string]any). To preserve source key order, use
// [CanonicalFromSource] instead.
//
// Top-level shape rules match `Dumps`: object or array.
func EmitCanonical(v any) (string, error) { return emitCanonicalVia(activeBackend(), v) }

// FormatSource formats Ktav source text into its normalised spelling,
// preserving every comment verbatim (spec § 3.4: a comment owns a whole
// line). Blank lines survive as a grouping hint, but a run of two or
// more collapses to exactly one and blank padding immediately inside a
// bracket is dropped. Applying FormatSource to its output produces the
// same result. Key order is never changed (spec § 5.9). For documents
// without comments or blank lines, the result equals CanonicalFromSource
// of the same text.
func FormatSource(src string) (string, error) { return formatSourceVia(activeBackend(), src) }

// CanonicalFromSource parses a Ktav document and immediately emits it
// in canonical form (spec § 5.9), preserving the source's insertion
// order of object keys. This is equivalent to `ktav parse | ktav
// emit-canonical` on the command line.
func CanonicalFromSource(src string) (string, error) {
	return canonicalFromSourceVia(activeBackend(), src)
}

// ─── per-instance operations (no global state) ───────────────────────

// Loads parses a Ktav document and returns its Go representation,
// dispatching through this instance's backend only.
func (o *Operations) Loads(src string) (any, error) { return loadsVia(o.vt, src) }

// LoadsStrict parses with the strict parser, dispatching through this
// instance's backend only.
func (o *Operations) LoadsStrict(src string) (any, error) { return loadsStrictVia(o.vt, src) }

// LoadsInto parses and JSON-unmarshals the tagged intermediate into
// target, dispatching through this instance's backend only.
func (o *Operations) LoadsInto(src string, target any) error {
	return loadsIntoVia(o.vt, src, target)
}

// Dumps renders a Go value as a Ktav document, dispatching through
// this instance's backend only.
func (o *Operations) Dumps(v any) (string, error) { return dumpsVia(o.vt, v) }

// DumpsForceStrings renders with every leaf scalar coerced to a
// String, dispatching through this instance's backend only.
func (o *Operations) DumpsForceStrings(v any) (string, error) {
	return dumpsForceStringsVia(o.vt, v)
}

// EmitCanonical renders a Go value as a canonical Ktav document,
// dispatching through this instance's backend only.
func (o *Operations) EmitCanonical(v any) (string, error) { return emitCanonicalVia(o.vt, v) }

// FormatSource reformats Ktav source text (comments preserved, fixed
// point), dispatching through this instance's backend only.
func (o *Operations) FormatSource(src string) (string, error) { return formatSourceVia(o.vt, src) }

// CanonicalFromSource parses and immediately emits canonical form,
// dispatching through this instance's backend only.
func (o *Operations) CanonicalFromSource(src string) (string, error) {
	return canonicalFromSourceVia(o.vt, src)
}
