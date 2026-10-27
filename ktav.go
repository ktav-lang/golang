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
	"unicode/utf8"

	"github.com/ktav-lang/golang/internal/engine"
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
	return engine.DecodeJSON(js)
}

func loadsStrictVia(vt *backendVTable, src string) (any, error) {
	if err := invalidUTF8Error(src); err != nil {
		return nil, err
	}
	js, err := vt.loadsStrict([]byte(src))
	if err != nil {
		return nil, err
	}
	return engine.DecodeJSON(js)
}

func loadsIntoVia(vt *backendVTable, src string, target any) error {
	if err := invalidUTF8Error(src); err != nil {
		return err
	}
	js, err := vt.loads([]byte(src))
	if err != nil {
		return err
	}
	plain, err := engine.FlattenTagged(js)
	if err != nil {
		return err
	}
	return json.Unmarshal(plain, target)
}

func dumpsVia(vt *backendVTable, v any) (string, error) {
	tagged, err := engine.EncodeWriterTagged(v, false)
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
	tagged, err := engine.EncodeWriterTagged(v, true)
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
	tagged, err := engine.EncodeWriterTagged(v, false)
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

func editValueVia(vt *backendVTable, src, textualPath string, replacement any) (string, error) {
	// Edit replacements are NOT restricted to a document root (scalar
	// replacement is allowed); the dumps-family ScalarRoot guard does
	// not apply here.
	wire, err := engine.EncodeEditReplacementTagged(replacement)
	if err != nil {
		return "", err
	}
	out, err := vt.editValue([]byte(src), []byte(textualPath), wire)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func editValueSegmentsVia(vt *backendVTable, src string, segments []string, replacement any) (string, error) {
	// Scalar roots allowed (see editValueVia).
	wire, err := engine.EncodeEditReplacementTagged(replacement)
	if err != nil {
		return "", err
	}
	// A nil []string IS the empty path by contract (EditNotFound after
	// source validation), so it must marshal to `[]`, not JSON null.
	wirePath, err := marshalSegmentsWire(segments)
	if err != nil {
		return "", err
	}
	out, err := vt.editValueSegments([]byte(src), wirePath, wire)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// marshalSegmentsWire marshals pre-decoded path segments for the
// segments-family wire: a nil []string IS the empty path by contract
// and must marshal to `[]`, never JSON null (which the native transport
// would reject as a Message).
func marshalSegmentsWire(segments []string) ([]byte, error) {
	wire := []byte("[]")
	if segments != nil {
		var err error
		wire, err = json.Marshal(segments)
		if err != nil {
			return nil, engine.NewErrorf("ktav: path segments are not encodable: %v", err)
		}
	}
	return wire, nil
}

func marshalStrictSegmentsWire(segments []string) ([]byte, error) {
	for _, segment := range segments {
		if !utf8.ValidString(segment) {
			return nil, engine.TransportUTF8Error([]byte(segment))
		}
	}
	return marshalSegmentsWire(segments)
}

func insertValueVia(vt *backendVTable, src, textualPath string, replacement any) (string, error) {
	// Match the C ABI contract: source UTF-8 validation and parsing precede
	// replacement transport encoding. Use the selected backend; never fall back.
	if err := invalidUTF8Error(src); err != nil {
		return "", err
	}
	if _, err := vt.loads([]byte(src)); err != nil {
		return "", err
	}
	if _, err := engine.ValidateTextualEditPath([]byte(textualPath)); err != nil {
		return "", err
	}
	// Insert replacements are NOT restricted to a document root (scalar
	// replacement is allowed); the dumps-family ScalarRoot guard does
	// not apply here.
	wire, err := engine.EncodeEditReplacementTagged(replacement)
	if err != nil {
		return "", err
	}
	out, err := vt.insertValue([]byte(src), []byte(textualPath), wire)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func insertValueSegmentsVia(vt *backendVTable, src string, segments []string, replacement any) (string, error) {
	if err := invalidUTF8Error(src); err != nil {
		return "", err
	}
	if _, err := vt.loads([]byte(src)); err != nil {
		return "", err
	}
	wirePath, err := marshalStrictSegmentsWire(segments)
	if err != nil {
		return "", err
	}
	if err = engine.ValidateLiteralEditSegments(segments); err != nil {
		return "", err
	}
	// Scalar roots allowed (see insertValueVia).
	wire, err := engine.EncodeEditReplacementTagged(replacement)
	if err != nil {
		return "", err
	}
	out, err := vt.insertValueSegments([]byte(src), wirePath, wire)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func removeValueVia(vt *backendVTable, src, textualPath string) (string, error) {
	out, err := vt.removeValue([]byte(src), []byte(textualPath))
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func removeValueSegmentsVia(vt *backendVTable, src string, segments []string) (string, error) {
	if err := invalidUTF8Error(src); err != nil {
		return "", err
	}
	if _, err := vt.loads([]byte(src)); err != nil {
		return "", err
	}
	wirePath, err := marshalStrictSegmentsWire(segments)
	if err != nil {
		return "", err
	}
	out, err := vt.removeValueSegments([]byte(src), wirePath)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func renameKeyVia(vt *backendVTable, src, textualPath, newKey string) (string, error) {
	// newKey is ONE literal key — passed as raw UTF-8 bytes, no JSON
	// encoding, no path escaping.
	out, err := vt.renameKey([]byte(src), []byte(textualPath), []byte(newKey))
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func renameKeySegmentsVia(vt *backendVTable, src string, segments []string, newKey string) (string, error) {
	if err := invalidUTF8Error(src); err != nil {
		return "", err
	}
	if _, err := vt.loads([]byte(src)); err != nil {
		return "", err
	}
	wirePath, err := marshalStrictSegmentsWire(segments)
	if err != nil {
		return "", err
	}
	out, err := vt.renameKeySegments([]byte(src), wirePath, []byte(newKey))
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

// EditValue replaces the value addressed by textualPath in src with
// replacement and returns the patched SOURCE TEXT — a lossless splice of
// exactly the target's byte range, never a re-render of the document:
//
//	out, err := ktav.EditValue(src, "targets.storagehost.address", "10.0.0.1")
//	out, err := ktav.EditValue(src, `list\.item.2`, []any{"a", "b"})
//
// The textual path splits on '.', with `\.` a literal dot and `\\` a
// literal backslash; any other escape, a trailing lone backslash or an
// empty segment is an *Error with Class "EditInvalidPath" whose Body is
// the UTF-8 byte offset of the problem inside the path argument. A path
// that does not address an existing value (missing key, non-numeric or
// out-of-bounds array index, descent through a scalar, or an empty
// segment list) is Class "EditNotFound". Replacement of an existing
// value only: there is no create, remove or rename.
//
// replacement is any value Dumps accepts (the type mapping is the
// package's), including scalars. The host scalar keeps its type:
// arbitrary integers never degrade to floats.
//
// Dotted, inline and block forms are preserved, as are comments,
// whitespace, key spelling, BOM, and the document's CR/LF/CRLF/EOF
// bytes outside the effective edit. A no-op replacement (value equal to
// the existing one) returns src byte-identical, however noncanonical
// its spelling. An inline ancestor is expanded to block form only when
// readability requires it (multiline replacement, line growth beyond
// max(100 Unicode scalar values, the old physical line length), or ≥4
// inline escapes with ≥25% overhead); new block levels use four spaces
// and the source EOL.
//
// Errors of class "EditUnsupported" name a target/replacement pair with
// no lossless single-range patch: noncontiguous dotted-key/reopen
// targets, structural replacement that would drop interior comment
// lines, strings containing CR or a line equal to '))'. On success the
// patch has been re-parsed and semantically verified.
//
// EditValue dispatches through the active backend ([UseBackend]); the
// native and pure implementations agree byte-for-byte.
func EditValue(src, textualPath string, replacement any) (string, error) {
	return editValueVia(activeBackend(), src, textualPath, replacement)
}

// EditValueSegments is EditValue with pre-decoded literal segments: no
// additional unescaping is performed (`[]string{"a.b"}` addresses the
// key `a.b`, not the path `a\.b`). A numeric segment is a string key
// inside an Object and a zero-based ASCII-digit index inside an Array
// (leading zeros allowed, signs are not indexes). An empty segment list
// is EditNotFound.
func EditValueSegments(src string, segments []string, replacement any) (string, error) {
	return editValueSegmentsVia(activeBackend(), src, segments, replacement)
}

//	out, err := ktav.InsertValue(src, "targets.storagehost.address", "10.0.0.1")
//	out, err := ktav.InsertValue(src, `list\.item.2`, []any{"a", "b"})
//
// The textual path splits on '.', with `\.` a literal dot and `\\` a
// literal backslash; any other escape, a trailing lone backslash or an
// empty segment is an *Error with Class "EditInvalidPath" whose Body is
// the UTF-8 byte offset of the problem inside the path argument. The
// LAST path segment is a NEW Object key: the parent is addressed by the
// preceding segments and must be an Object. If the key already exists
// the error class is "EditAlreadyExists"; a missing parent, descent
// through a scalar, or a non-numeric/out-of-bounds array index is
// "EditNotFound"; inserting into an Array (or any structural case with
// no lossless single-range patch) is "EditUnsupported".
//
// replacement is any value Dumps accepts (the type mapping is the
// package's), including scalars.
//
// Dotted, inline and block forms are preserved, as are comments,
// whitespace, key spelling, BOM, and the document's CR/LF/CRLF/EOF
// bytes outside the inserted range. The inserted pair is placed at the
// end of the parent Object; empty ancestors are never pruned. On
// success the patch has been re-parsed and semantically verified.
func InsertValue(src, textualPath string, replacement any) (string, error) {
	return insertValueVia(activeBackend(), src, textualPath, replacement)
}

// InsertValueSegments is InsertValue with pre-decoded literal segments:
// no additional unescaping is performed (`[]string{"a.b"}` inserts the
// key `a.b`, not the path `a\.b`). The last segment is the NEW Object
// key; a numeric segment is a string key inside an Object and a
// zero-based ASCII-digit index inside an Array (leading zeros allowed,
// signs are not indexes). An empty segment list is EditNotFound.
func InsertValueSegments(src string, segments []string, replacement any) (string, error) {
	return insertValueSegmentsVia(activeBackend(), src, segments, replacement)
}

// RemoveValue deletes the value addressed by textualPath in src and
// returns the patched SOURCE TEXT — a lossless splice removing exactly
// the target's byte range (for an array element, its delimiter too),
// never a re-render of the document:
//
//	out, err := ktav.RemoveValue(src, "targets.storagehost.address")
//	out, err := ktav.RemoveValue(src, `list\.item.2`)
//
// The textual path splits on '.', with `\.` a literal dot and `\\` a
// literal backslash; invalid escapes or empty segments are an *Error
// with Class "EditInvalidPath". An empty textual path is Class
// "EditInvalidPath". A path that does not address an existing value
// (missing key, bad index, descent through a scalar, or an empty segment
// list) is Class "EditNotFound". Removing a leaf, a whole
// subtree or an array element is supported; empty ancestors are NOT
// pruned. Errors of class "EditUnsupported" name structural cases with
// no lossless single-range patch.
func RemoveValue(src, textualPath string) (string, error) {
	return removeValueVia(activeBackend(), src, textualPath)
}

// RemoveValueSegments is RemoveValue with pre-decoded literal segments:
// no additional unescaping is performed. A numeric segment is a string
// key inside an Object and a zero-based ASCII-digit index inside an
// Array (leading zeros allowed, signs are not indexes). An empty
// segment list is EditNotFound.
func RemoveValueSegments(src string, segments []string) (string, error) {
	return removeValueSegmentsVia(activeBackend(), src, segments)
}

// RenameKey renames the Object key addressed by textualPath in src, in
// place: the pair's value, order, position and formatting are
// preserved, only the key bytes are spliced:
//
//	out, err := ktav.RenameKey(src, "targets.storagehost.address", "endpoint")
//
// The textual path splits on '.', with `\.` a literal dot and `\\` a
// literal backslash; invalid escapes or empty segments are an *Error
// with Class "EditInvalidPath". newKey is ONE literal key: dots,
// backslashes and Unicode are part of the name, never path syntax, and
// it is inserted verbatim (no quoting beyond what the format requires).
// Renaming to the SAME key is a no-op returning src byte-identical; a
// sibling key equal to newKey yields Class "EditAlreadyExists"; a
// missing target or non-Object parent is "EditNotFound"; renaming an
// array index is "EditUnsupported".
func RenameKey(src, textualPath, newKey string) (string, error) {
	return renameKeyVia(activeBackend(), src, textualPath, newKey)
}

// RenameKeySegments is RenameKey with pre-decoded literal segments: no
// additional unescaping is performed; newKey is still ONE literal key
// (dots and backslashes are part of the name). An empty segment list is
// EditNotFound.
func RenameKeySegments(src string, segments []string, newKey string) (string, error) {
	return renameKeySegmentsVia(activeBackend(), src, segments, newKey)
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

// EditValue replaces the value addressed by textualPath, dispatching
// through this instance's backend only. See the package-level
// [EditValue] for the full contract.
func (o *Operations) EditValue(src, textualPath string, replacement any) (string, error) {
	return editValueVia(o.vt, src, textualPath, replacement)
}

// EditValueSegments is EditValue with pre-decoded literal segments,
// dispatching through this instance's backend only.
func (o *Operations) EditValueSegments(src string, segments []string, replacement any) (string, error) {
	return editValueSegmentsVia(o.vt, src, segments, replacement)
}

// InsertValue inserts a NEW key/value pair, dispatching through this
// instance's backend only. See the package-level [InsertValue] for the
// full contract.
func (o *Operations) InsertValue(src, textualPath string, replacement any) (string, error) {
	return insertValueVia(o.vt, src, textualPath, replacement)
}

// InsertValueSegments is InsertValue with pre-decoded literal segments,
// dispatching through this instance's backend only.
func (o *Operations) InsertValueSegments(src string, segments []string, replacement any) (string, error) {
	return insertValueSegmentsVia(o.vt, src, segments, replacement)
}

// RemoveValue deletes the value addressed by textualPath, dispatching
// through this instance's backend only. See the package-level
// [RemoveValue] for the full contract.
func (o *Operations) RemoveValue(src, textualPath string) (string, error) {
	return removeValueVia(o.vt, src, textualPath)
}

// RemoveValueSegments is RemoveValue with pre-decoded literal segments,
// dispatching through this instance's backend only.
func (o *Operations) RemoveValueSegments(src string, segments []string) (string, error) {
	return removeValueSegmentsVia(o.vt, src, segments)
}

// RenameKey renames the Object key addressed by textualPath, dispatching
// through this instance's backend only. See the package-level
// [RenameKey] for the full contract.
func (o *Operations) RenameKey(src, textualPath, newKey string) (string, error) {
	return renameKeyVia(o.vt, src, textualPath, newKey)
}

// RenameKeySegments is RenameKey with pre-decoded literal segments,
// dispatching through this instance's backend only.
func (o *Operations) RenameKeySegments(src string, segments []string, newKey string) (string, error) {
	return renameKeySegmentsVia(o.vt, src, segments, newKey)
}
