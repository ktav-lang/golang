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
//	integer scalar    int64 if it fits, else *big.Int
//	float scalar      float64
//	bare scalar       string
//	[ ... ]           []any
//	{ ... }           map[string]any (key order not preserved)
//
// Under spec 0.5, integer and float values are inferred from the
// scalar body's lexical form (bare `42` → Integer, `3.14` → Float).
// The typed markers `:i` / `:f` no longer exist. Integers that
// overflow i64 fall back to String (not *big.Int).
//
// Key order from the source is **not** preserved on either side: decode
// returns a plain `map[string]any`, and encode goes through
// `encoding/json`, which emits object keys in alphabetical order. If
// you need a fixed shape, use `LoadsInto` into a struct.
//
// On encode, Go *big.Int always emits an integer scalar; Go int /
// int64 / uint64 emit an integer scalar; Go float64 emits a float
// scalar. NaN / ±Inf are rejected. Top-level value must encode to a
// Ktav object (i.e. a map[string]any or struct) or a Ktav array
// (i.e. a []any or any other JSON-encodable slice). Top-level Arrays
// render as bare item-per-line — no surrounding `[...]` brackets, per
// spec § 5.0.1.
//
// [FormatSource] reformats Ktav source text (comments preserved,
// fixed point); [CanonicalFromSource] re-emits it canonically.
package ktav

import (
	"encoding/json"
)

// Loads parses a Ktav document and returns its Go representation (see
// package doc for the mapping).
func Loads(src string) (any, error) {
	js, err := loadsJSON([]byte(src))
	if err != nil {
		return nil, err
	}
	return decodeJSON(js)
}

// LoadsStrict parses a Ktav document using the strict parser. Strict mode
// rejects lossy numeric spellings while preserving the same Go type mapping
// as Loads.
func LoadsStrict(src string) (any, error) {
	js, err := loadsStrictJSON([]byte(src))
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
// encode to a JSON object (map[string]any, struct, etc.) or a JSON
// array (a slice). Top-level Arrays render as bare item-per-line —
// no surrounding `[...]` brackets, per spec § 5.0.1.
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
func DumpsForceStrings(v any) (string, error) {
	tagged, err := encodeTagged(v)
	if err != nil {
		return "", err
	}
	out, err := dumpsForceStringsJSON(tagged)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

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
func EmitCanonical(v any) (string, error) {
	tagged, err := encodeTagged(v)
	if err != nil {
		return "", err
	}
	out, err := emitCanonicalJSON(tagged)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// FormatSource formats Ktav source text into its normalised spelling,
// preserving every comment verbatim (spec § 3.4: a comment owns a whole
// line). Blank lines survive as a grouping hint, but a run of two or
// more collapses to exactly one and blank padding immediately inside a
// bracket is dropped, so formatting is a fixed point:
// FormatSource(FormatSource(x)) == FormatSource(x). Key order is never
// changed (spec § 5.9). For a document with no comments and no blank
// lines, the result equals CanonicalFromSource of the same text.
func FormatSource(src string) (string, error) {
	out, err := formatSource([]byte(src))
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// CanonicalFromSource parses a Ktav document and immediately emits it
// in canonical form (spec § 5.9), preserving the source's insertion
// order of object keys. This is equivalent to `ktav parse | ktav
// emit-canonical` on the command line.
func CanonicalFromSource(src string) (string, error) {
	js, err := loadsJSON([]byte(src))
	if err != nil {
		return "", err
	}
	out, err := emitCanonicalJSON(js)
	if err != nil {
		return "", err
	}
	return string(out), nil
}
