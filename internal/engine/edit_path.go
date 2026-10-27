package engine

// Edit path transports: the textual dotted path and the segments JSON
// array. Both byte seams validate transports exactly like the C ABI:
// Go's utf8.Valid already uses the same definition as Rust's
// std::str::from_utf8 (ill-formed UTF-8 AND encoded surrogates are
// rejected by the stdlib), so the only extra check the pure side needs
// is for unpaired surrogate ESCAPES, which encoding/json silently
// normalises to U+FFFD where serde_json refuses them.

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"
)

// ValidateLiteralEditSegments ensures decoded segments are non-empty.
func ValidateLiteralEditSegments(segments []string) error {
	if len(segments) == 0 {
		return editNotFound()
	}
	return nil
}

// ValidateTextualEditPath validates a textual path before replacement conversion.
func ValidateTextualEditPath(path []byte) ([]string, error) {
	if err := transportUTF8Error(path); err != nil {
		return nil, err
	}
	segments, err := parseEditPath(string(path))
	if err != nil {
		return nil, err
	}
	if len(segments) == 0 {
		return nil, editNotFound()
	}
	return segments, nil
}

// parseEditPath splits dotted paths; `\.` escapes a dot and `\\` escapes
// a backslash. Any other escape, trailing lone backslash, or empty segment
// returns EditInvalidPath with the byte offset where the problem was detected.
func parseEditPath(path string) ([]string, *Error) {
	if path == "" {
		return nil, editInvalidPath(0)
	}
	var segments []string
	segStart := 0
	var cur *strings.Builder
	i := 0
	for i < len(path) {
		switch path[i] {
		case '.':
			if segStart == i && cur == nil {
				return nil, editInvalidPath(i)
			}
			if cur != nil {
				segments = append(segments, cur.String())
				cur = nil
			} else {
				segments = append(segments, path[segStart:i])
			}
			segStart = i + 1
			i++
		case '\\':
			if i+1 < len(path) && (path[i+1] == '.' || path[i+1] == '\\') {
				if cur == nil {
					cur = &strings.Builder{}
					cur.WriteString(path[segStart:i])
				}
				if path[i+1] == '.' {
					cur.WriteByte('.')
				} else {
					cur.WriteByte('\\')
				}
				i += 2
			} else {
				return nil, editInvalidPath(i)
			}
		default:
			chLen := editUTF8Len(path[i])
			if cur != nil {
				cur.WriteString(path[i : i+chLen])
			}
			i += chLen
		}
	}
	if segStart == len(path) && cur == nil {
		// Empty last segment (trailing '.').
		return nil, editInvalidPath(len(path))
	}
	if cur != nil {
		segments = append(segments, cur.String())
	} else {
		segments = append(segments, path[segStart:])
	}
	return segments, nil
}

// editUTF8Len is the UTF-8 sequence length by lead byte (1 for ASCII and
// for stray continuation bytes — input is valid UTF-8 by then).
func editUTF8Len(b byte) int {
	switch {
	case b < 0x80:
		return 1
	case b >= 0xC0 && b <= 0xDF:
		return 2
	case b >= 0xE0 && b <= 0xEF:
		return 3
	case b >= 0xF0 && b <= 0xF7:
		return 4
	default:
		return 1
	}
}

// parseEditIndex is the array-index segment rule: ASCII digits only
// (leading zeros allowed, signs are not indexes), zero-based.
func parseEditIndex(seg string) (int, *Error) {
	if seg == "" {
		return 0, editNotFound()
	}
	for i := 0; i < len(seg); i++ {
		if seg[i] < '0' || seg[i] > '9' {
			return 0, editNotFound()
		}
	}
	n, err := strconv.Atoi(seg)
	if err != nil {
		return 0, editNotFound()
	}
	return n, nil
}

// transportUTF8Error rejects transport bytes the C ABI's
// std::str::from_utf8 would reject. Go's utf8.Valid uses the same
// definition: it already refuses ill-formed UTF-8 AND encoded surrogates
// (U+D800..U+DFFF are never valid UTF-8 in Go), so this is exactly the
// one check the pure seams need — no extra rune walk, no extra
// protection beyond the stdlib.
func TransportUTF8Error(raw []byte) *Error { return transportUTF8Error(raw) }

func transportUTF8Error(raw []byte) *Error {
	if !utf8.Valid(raw) {
		return messageErr("input transport is not valid UTF-8")
	}
	return nil
}

// jsonSurrogateEscapeError rejects unpaired \uD800..\uDFFF escapes:
// serde_json (the native transport) refuses them, while encoding/json
// silently normalises each half to U+FFFD — a silent divergence on the
// pure side. Paired escapes decode to the same rune on both sides and
// pass. ESCAPE PARITY: a `\u` preceded by an odd run of backslashes is
// escaped data — `\\uD800` is the literal text `\uD800`, not an escape,
// and must NOT be rejected; the scanner therefore consumes `\\` as one
// unit and only inspects `\u` at true escape positions.
func jsonSurrogateEscapeError(raw []byte) *Error {
	for i := 0; i < len(raw); {
		if raw[i] != '\\' {
			i++
			continue
		}
		if i+1 >= len(raw) {
			return nil // trailing lone backslash: the JSON decoder rejects it
		}
		switch raw[i+1] {
		case '\\':
			// Literal backslash: whatever `\u` follows is plain text.
			i += 2
		case 'u':
			if i+6 > len(raw) {
				return nil // truncated escape: the JSON decoder rejects it
			}
			cp, ok := jsonHex4(raw[i+2 : i+6])
			if !ok {
				return nil // malformed hex: the JSON decoder rejects it
			}
			switch {
			case cp >= 0xD800 && cp <= 0xDBFF: // high surrogate needs a pair
				if i+12 > len(raw) || raw[i+6] != '\\' || raw[i+7] != 'u' {
					return messageErr("input transport JSON contains an unpaired surrogate escape")
				}
				lo, ok := jsonHex4(raw[i+8 : i+12])
				if !ok || lo < 0xDC00 || lo > 0xDFFF {
					return messageErr("input transport JSON contains an unpaired surrogate escape")
				}
				i += 12 // consume the paired low escape too
			case cp >= 0xDC00 && cp <= 0xDFFF: // lone low surrogate
				return messageErr("input transport JSON contains an unpaired surrogate escape")
			default:
				i += 6
			}
		default:
			// Any other escape (\n, \", ...): skip both bytes.
			i += 2
		}
	}
	return nil
}

func jsonHex4(b []byte) (rune, bool) {
	var cp rune
	for _, c := range b {
		cp <<= 4
		switch {
		case c >= '0' && c <= '9':
			cp |= rune(c - '0')
		case c >= 'a' && c <= 'f':
			cp |= rune(c-'a') + 10
		case c >= 'A' && c <= 'F':
			cp |= rune(c-'A') + 10
		default:
			return 0, false
		}
	}
	return cp, true
}

// decodeSegmentsWire parses the transport JSON array of literal strings.
// Non-string elements and any trailing data are transport errors (the
// source-less Message envelope); the caller applies the UTF-8/surrogate
// transport validation first.
func decodeSegmentsWire(raw []byte) ([]string, *Error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	tok, err := dec.Token()
	if err != nil {
		return nil, messageErr("input JSON: " + err.Error())
	}
	if delim, ok := tok.(json.Delim); !ok || delim != '[' {
		return nil, messageErr("input JSON: path must be a JSON array of strings")
	}
	var segments []string
	for dec.More() {
		t, err := dec.Token()
		if err != nil {
			return nil, messageErr("input JSON: " + err.Error())
		}
		s, ok := t.(string)
		if !ok {
			return nil, messageErr("input JSON: path segments must be strings")
		}
		segments = append(segments, s)
	}
	if _, err := dec.Token(); err != nil { // closing ']'
		return nil, messageErr("input JSON: " + err.Error())
	}
	// After the closing bracket ONLY a clean EOF is acceptable: any
	// other error from the next Token is malformed trailing garbage,
	// not the end of input — it must fail, never pass as success.
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return nil, messageErr("input JSON: invalid trailing data after the path array")
	}
	return segments, nil
}
