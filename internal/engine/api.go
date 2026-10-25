package engine

import "encoding/json"

// Operations provides the shared pure-Go implementation. Its zero value is ready to use.
type Operations struct{}

func New() *Operations { return &Operations{} }

func (o *Operations) Loads(src string) (any, error)                  { return loadsPublic(src, false) }
func (o *Operations) LoadsStrict(src string) (any, error)            { return loadsPublic(src, true) }
func (o *Operations) LoadsInto(src string, target any) error         { return loadsIntoPublic(src, target) }
func (o *Operations) Dumps(v any) (string, error)                    { return dumpsPublic(v, false, false) }
func (o *Operations) DumpsForceStrings(v any) (string, error)        { return dumpsPublic(v, true, false) }
func (o *Operations) EmitCanonical(v any) (string, error)            { return dumpsPublic(v, false, true) }
func (o *Operations) FormatSource(src string) (string, error)        { return formatPublic(src) }
func (o *Operations) CanonicalFromSource(src string) (string, error) { return canonicalPublic(src) }

// EditValue replaces the value addressed by the textual path
// (`key1.key2\.literal.0`) with `replacement`, splicing only the
// target's bytes. Source-scoped: comments, key spelling and EOLs outside
// the edit survive verbatim.
func (o *Operations) EditValue(src, textualPath string, replacement any) (string, error) {
	return editPublic(src, textualPath, replacement)
}

// EditValueSegments is EditValue with pre-decoded literal segments: no
// extra unescaping is performed. A numeric segment is a string key in an
// Object and a zero-based ASCII-digit index in an Array.
func (o *Operations) EditValueSegments(src string, segments []string, replacement any) (string, error) {
	return editSegmentsPublic(src, segments, replacement)
}

func PureLoads(b []byte) ([]byte, error)               { return pureLoadsTagged(b) }
func PureLoadsStrict(b []byte) ([]byte, error)         { return pureLoadsStrictTagged(b) }
func PureDumps(b []byte) ([]byte, error)               { return pureDumpsTagged(b) }
func PureDumpsForceStrings(b []byte) ([]byte, error)   { return pureDumpsForceStringsTagged(b) }
func PureEmitCanonical(b []byte) ([]byte, error)       { return pureEmitCanonicalTagged(b) }
func PureFormatSource(b []byte) ([]byte, error)        { return pureFormatSource(b) }
func PureCanonicalFromSource(b []byte) ([]byte, error) { return pureCanonicalFromSource(b) }

// PureEditValue is the byte seam the backend vtable dispatches to:
// UTF-8 source, UTF-8 textual path, tagged-wire replacement (scalar
// allowed); output is UTF-8 source bytes.
func PureEditValue(src, path, replacement []byte) ([]byte, error) {
	return pureEditValueBytes(src, path, replacement)
}

// PureEditValueSegments is the segments variant: path is the UTF-8 JSON
// array of literal segment strings.
func PureEditValueSegments(src, path, replacement []byte) ([]byte, error) {
	return pureEditValueSegmentsBytes(src, path, replacement)
}

// EncodeEditReplacementTagged encodes an edit replacement (any value,
// scalars allowed) into the tagged wire JSON WITHOUT the dumps-family
// document-root restriction. Used by the native-backed root package's
// EditValue/EditValueSegments.
func EncodeEditReplacementTagged(v any) ([]byte, error) {
	return encodeEditReplacementTagged(v)
}

// editWireReplacement converts a host Go value into the tagged wire JSON
// the edit seams consume (same numeric wire as Dumps, scalar roots
// allowed).
func editWireReplacement(replacement any) ([]byte, error) {
	return encodeEditReplacementTagged(replacement)
}

func editPublic(src, textualPath string, replacement any) (string, error) {
	wire, err := editWireReplacement(replacement)
	if err != nil {
		return "", err
	}
	out, err := pureEditValueBytes([]byte(src), []byte(textualPath), wire)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func editSegmentsPublic(src string, segments []string, replacement any) (string, error) {
	wire, err := editWireReplacement(replacement)
	if err != nil {
		return "", err
	}
	// A nil []string IS the empty path by contract (EditNotFound after
	// source validation), so it must marshal to `[]`, not JSON null
	// (which the transport would reject as a Message). Non-nil slices
	// marshal as-is; nothing is copied.
	wirePath, err := marshalSegmentsWire(segments)
	if err != nil {
		return "", err
	}
	out, err := pureEditValueSegmentsBytes([]byte(src), wirePath, wire)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// marshalSegmentsWire encodes the segments transport, mapping a nil
// slice to the empty JSON array. A literal `null` remains a transport
// error — that is a different input (a raw byte seam caller), not a Go
// nil slice.
func marshalSegmentsWire(segments []string) ([]byte, error) {
	if segments == nil {
		return []byte("[]"), nil
	}
	return json.Marshal(segments)
}

// Shared conversion and wire seams used by the native-backed root package.
func DecodeJSON(raw []byte) (any, error)       { return decodeJSON(raw) }
func FlattenTagged(raw []byte) ([]byte, error) { return flattenTagged(raw) }
func EncodeWriterTagged(v any, forceStrings bool) ([]byte, error) {
	return encodeWriterTagged(v, forceStrings)
}
func PrepareWriterValue(v any, path []string, forceStrings bool) (any, error) {
	return prepareWriterValue(v, path, forceStrings)
}
func InvalidUTF8Error(src string) *Error  { return invalidUTF8Error(src) }
func ErrorFromEnvelope(raw []byte) *Error { return errorFromEnvelope(raw) }
func UnrepresentableError(reason string, path []string) *Error {
	return unrepresentableError(reason, path)
}
func NewErrorf(format string, args ...any) *Error { return newErrorf(format, args...) }
func FirstInvalidUTF8Span(src string) (int, int, bool) {
	return firstInvalidUTF8Span(src)
}
func LineOfOffset(src string, offset int) int { return lineOfOffset(src, offset) }
