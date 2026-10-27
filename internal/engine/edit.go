package engine

// Pure-Go source edit engine: lossless replacement of an existing value
// addressed by a path, splicing ONLY the target's byte range (plus any
// inline wrappers the readability rules force open). It mirrors the Rust
// reference (ktav::edit / document::parser::edit::{tree,splice,layout})
// decision-for-decision so native and pure backends agree byte-for-byte:
//
//   - positional tree (edit_tree.go) records byte ranges per value on
//     the SAME line-dispatch decisions as the real parser;
//   - the splice plan (edit_splice.go) reuses the source's marker zone
//     and whitespace, switching only the raw `:` marker when the
//     replacement's type demands it;
//   - the layout pass (edit_layout.go) expands the minimum chain of
//     inline wrappers when readability requires it;
//   - every successful patch is re-parsed and semantically verified
//     (exactly the target replaced, rest of the tree equal), so a silent
//     corruption is impossible (fail closed → EditUnsupported).
//
// Error classes are EditInvalidPath / EditNotFound / EditUnsupported;
// reason/line/line_text/span/path/canonical/spec_section stay null and
// Body carries the UTF-8 byte offset (InvalidPath) or the reason text
// (Unsupported). Source validation comes FIRST (real 0.9 parser
// envelopes with locations); only afterwards are the path/replacement
// transports decoded, and their failures surface as source-less
// Message envelopes — see edit_path.go for the transport strictness
// that keeps the pure side byte-compatible with the C ABI's serde.

import "strconv"

// ─── edit errors ──────────────────────────────────────────────────────

func editInvalidPath(offset int) *Error {
	return &Error{
		Msg:   "invalid edit path (bad escape or empty segment) at byte " + strconv.Itoa(offset),
		Class: "EditInvalidPath",
		Body:  strconv.Itoa(offset),
	}
}

func editNotFound() *Error {
	return &Error{
		Msg:   "edit path does not address an existing value",
		Class: "EditNotFound",
	}
}

func editUnsupported(reason string) *Error {
	return &Error{
		Msg:   "no lossless source edit: " + reason,
		Class: "EditUnsupported",
		Body:  reason,
	}
}

// ─── public entry (byte level) ────────────────────────────────────────

// pureEditValueBytes is the pure backend's byte seam: src is UTF-8 Ktav
// source, path the UTF-8 textual path, replacement the tagged wire JSON
// (scalar allowed). Output is UTF-8 Ktav source bytes.
func pureEditValueBytes(src, path, replacement []byte) ([]byte, error) {
	text := string(src)
	// 1. UTF-8 validation and a real parse of the source first: parse
	// errors surface as genuine 0.9 envelopes with locations.
	if err := invalidUTF8Error(text); err != nil {
		return nil, err
	}
	original, _, perr := parseDoc(text, false, false)
	if perr != nil {
		enrichLineText(text, perr)
		return nil, perr
	}
	// 2. textual path transport decode (source-less Message on invalid
	// UTF-8 — matching the C ABI's std::str::from_utf8 rejection).
	if terr := transportUTF8Error(path); terr != nil {
		return nil, terr
	}
	segments, perr2 := parseEditPath(string(path))
	if perr2 != nil {
		return nil, perr2
	}
	// 3. replacement wire decode (Message envelope on transport errors).
	if terr := transportUTF8Error(replacement); terr != nil {
		return nil, terr
	}
	if terr := jsonSurrogateEscapeError(replacement); terr != nil {
		return nil, terr
	}
	repl, werr := ingestWireValue(replacement)
	if werr != nil {
		return nil, werr
	}
	if werr = validateEditIntegerDomain(repl); werr != nil {
		return nil, werr
	}
	out, eerr := applyEdit(text, original.(*pvalue), segments, repl)
	if eerr != nil {
		return nil, eerr
	}
	return []byte(out), nil
}

// pureEditValueSegmentsBytes is the segments variant: path is the UTF-8
// JSON array of literal (already-decoded) segment strings.
func pureEditValueSegmentsBytes(src, path, replacement []byte) ([]byte, error) {
	text := string(src)
	if err := invalidUTF8Error(text); err != nil {
		return nil, err
	}
	original, _, perr := parseDoc(text, false, false)
	if perr != nil {
		enrichLineText(text, perr)
		return nil, perr
	}
	// Segments transport: serde-compatible strictness (invalid UTF-8,
	// encoded surrogates, unpaired surrogate escapes).
	if terr := transportUTF8Error(path); terr != nil {
		return nil, terr
	}
	if terr := jsonSurrogateEscapeError(path); terr != nil {
		return nil, terr
	}
	segments, serr := decodeSegmentsWire(path)
	if serr != nil {
		return nil, serr
	}
	if len(segments) == 0 {
		return nil, editNotFound()
	}
	if terr := transportUTF8Error(replacement); terr != nil {
		return nil, terr
	}
	if terr := jsonSurrogateEscapeError(replacement); terr != nil {
		return nil, terr
	}
	repl, werr := ingestWireValue(replacement)
	if werr != nil {
		return nil, werr
	}
	if werr = validateEditIntegerDomain(repl); werr != nil {
		return nil, werr
	}
	out, eerr := applyEdit(text, original.(*pvalue), segments, repl)
	if eerr != nil {
		return nil, eerr
	}
	return []byte(out), nil
}
