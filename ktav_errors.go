package ktav

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Error is the binding's error type, carrying the nine other native
// ktav error envelope fields as first-class members. A zero value
// ("" / 0 / nil) corresponds to an explicit JSON null in the envelope.
// Msg is the envelope's own `message` field (ktav 0.8.0+), taken
// verbatim — never the raw envelope JSON, and never reassembled from
// the other fields. Against a pre-0.8.0 native library, which never
// wrote `message`, Msg falls back to a local reconstruction.
type Error struct {
	Msg         string
	Class       string // envelope "error": ktav::Error variant name
	Reason      string // machine-readable reason code ("" when null)
	Line        int    // 1-based source line; 0 when the envelope has null
	LineText    string
	Span        *Span
	Path        []string // exact decoded key segments, never a joined string
	Body        string
	Canonical   string
	SpecSection string
}

// Span is a byte-offset range into the source document carried by the
// structured error envelope.
type Span struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

func (e *Error) Error() string { return e.Msg }

func newError(msg string) *Error { return &Error{Msg: msg} }

func newErrorf(format string, args ...any) *Error {
	return &Error{Msg: fmt.Sprintf(format, args...)}
}

func invalidUTF8Error(src string) *Error {
	if utf8.ValidString(src) {
		return nil
	}
	start, end, invalid := firstInvalidUTF8Span(src)
	if !invalid {
		return nil
	}
	return &Error{
		Msg:         "InvalidUtf8: input is not valid UTF-8",
		Class:       "InvalidUtf8",
		Line:        lineOfOffset(src, start),
		Span:        &Span{Start: start, End: end},
		SpecSection: "§6.15",
	}
}

// firstInvalidUTF8Span locates the first ill-formed UTF-8 sequence in src,
// mirroring the Rust C ABI reference (std str::from_utf8 valid_up_to/
// error_len semantics plus the envelope's continuation-byte extension)
// so both layers reject with identical Line/Span diagnostics (§6.15).
// It returns the half-open byte span [start, end) and ok=true, or
// ok=false when src is entirely well-formed.
func firstInvalidUTF8Span(src string) (start, end int, ok bool) {
	for i := 0; i < len(src); {
		b := src[i]
		if b < utf8.RuneSelf {
			i++
			continue
		}
		var size int
		var lo, hi byte // allowed range for the second byte
		switch {
		case b >= 0xC2 && b <= 0xDF:
			size, lo, hi = 2, 0x80, 0xBF
		case b == 0xE0:
			size, lo, hi = 3, 0xA0, 0xBF
		case (b >= 0xE1 && b <= 0xEC) || b == 0xEE || b == 0xEF:
			size, lo, hi = 3, 0x80, 0xBF
		case b == 0xED:
			size, lo, hi = 3, 0x80, 0x9F
		case b == 0xF0:
			size, lo, hi = 4, 0x90, 0xBF
		case b >= 0xF1 && b <= 0xF3:
			size, lo, hi = 4, 0x80, 0xBF
		case b == 0xF4:
			size, lo, hi = 4, 0x80, 0x8F
		default:
			// Continuation byte in lead position or an illegal lead
			// (C0, C1, F5..FF): a one-byte valid prefix.
			return i, extendContinuation(src, i+1), true
		}
		if i+1 >= len(src) {
			// No second byte is present, so the sequence is truncated.
			return i, len(src), true
		}
		if src[i+1] < lo || src[i+1] > hi {
			// Only the lead is a valid prefix (error_len 1).
			return i, extendContinuation(src, i+1), true
		}
		for j := 2; j < size; j++ {
			if i+j >= len(src) {
				// Input ends before this required continuation byte.
				return i, len(src), true
			}
			if src[i+j] < 0x80 || src[i+j] > 0xBF {
				// Sequence breaks after j good bytes (error_len j);
				// no continuation extension here.
				return i, i + j, true
			}
		}
		i += size
	}
	return 0, 0, false
}

// extendContinuation advances e while src[e] is a UTF-8 continuation
// byte, matching the C ABI envelope's extension of an error_len-1 span
// over any following continuation bytes.
func extendContinuation(src string, e int) int {
	for e < len(src) && src[e] >= 0x80 && src[e] <= 0xBF {
		e++
	}
	return e
}

// lineOfOffset returns the 1-based raw-byte line containing byte offset
// pos, counting each LF, lone CR, and CR LF pair as one terminator per
// spec §3.2 (no BOM skipping, §6.15). It mirrors the Rust C ABI
// line_of_offset byte-for-byte so both layers report identical lines.
func lineOfOffset(src string, pos int) int {
	line := 1
	bound := min(pos, len(src))
	for i := 0; i < bound; {
		switch src[i] {
		case '\n':
			line++
			i++
		case '\r':
			if i+1 < len(src) && src[i+1] == '\n' {
				if i+2 <= pos {
					line++
				}
				i += 2
			} else {
				line++
				i++
			}
		default:
			i++
		}
	}
	return line
}

func unrepresentableError(reason string, path []string) *Error {
	var message string
	spec := "§5.9.0"
	switch reason {
	case "ScalarRoot":
		message = "ScalarRoot: the document root is not an Object or an Array (spec § 5.9.0)"
	case "EmptyKeyName":
		message = "EmptyKeyName: an Object pair's name is the empty string (spec § 5.9.0)"
	case "NonFiniteFloat":
		message = "NonFiniteFloat: a Float is NaN or ±Infinity (spec § 5.9.0)"
	case "CRByte":
		spec = "§5.9.7"
		message = "CRByte: a String contains a CR byte (spec § 5.9.0 / § 5.9.7)"
	case "BothFormsRequired":
		spec = "§5.9.7"
		message = "BothFormsRequired: the multi-line String needs both forms, a segment trimming to '))' and a segment trimming to ')' (spec § 5.9.7)"
	case "TrailingWhitespaceCollision":
		spec = "§5.9.7"
		message = "TrailingWhitespaceCollision: a segment trims to '))' and some content line has trailing whitespace (spec § 5.9.7)"
	case "LeadingWhitespaceCollision":
		spec = "§5.9.7"
		message = "LeadingWhitespaceCollision: a segment trims to '))' and every non-blank segment shares leading whitespace at the same position (spec § 5.9.7)"
	}
	if reason != "ScalarRoot" && len(path) > 0 {
		quoted, _ := json.Marshal(path)
		message += " at " + string(quoted)
	}
	return &Error{
		Msg:         message,
		Class:       "UnrepresentableAt",
		Reason:      reason,
		Path:        append([]string{}, path...),
		SpecSection: spec,
	}
}

// envelopeJSON is the wire shape of the native error envelope
// (ktav::ErrorEnvelope::to_json): one JSON object, ten fields since
// ktav 0.8.0 (message was appended), absent info as explicit null
// (message itself is never null — every error renders). Nullable
// fields are pointers here so a JSON null maps to nil rather than a
// zero value.
type envelopeJSON struct {
	Error       *string   `json:"error"`
	Reason      *string   `json:"reason"`
	Line        *uint32   `json:"line"`
	LineText    *string   `json:"line_text"`
	Span        *Span     `json:"span"`
	Path        *[]string `json:"path"`
	Body        *string   `json:"body"`
	Canonical   *string   `json:"canonical"`
	SpecSection *string   `json:"spec_section"`
	Message     *string   `json:"message"`
}

// errorFromEnvelope parses the native error payload as the structured
// envelope JSON and takes Msg from the envelope's own `message` field
// verbatim (falling back to a local reconstruction against a pre-0.8.0
// native library that never wrote it). If the bytes do not parse as a
// JSON object with a string `error` field (e.g. a stale pre-envelope
// native library still emitting plain message strings), it falls back
// to a Msg-only Error carrying the raw text.
func errorFromEnvelope(raw []byte) *Error {
	var env envelopeJSON
	if err := json.Unmarshal(raw, &env); err != nil || env.Error == nil {
		// Stale native library: error text is not envelope JSON.
		return &Error{Msg: string(raw)}
	}
	e := &Error{
		Class: *env.Error,
		Span:  env.Span,
	}
	if env.Reason != nil {
		e.Reason = *env.Reason
	}
	if env.Line != nil {
		e.Line = int(*env.Line)
	}
	if env.LineText != nil {
		e.LineText = *env.LineText
	}
	if env.Path != nil {
		e.Path = *env.Path
	}
	if env.Body != nil {
		e.Body = *env.Body
	}
	if env.Canonical != nil {
		e.Canonical = *env.Canonical
	}
	if env.SpecSection != nil {
		e.SpecSection = *env.SpecSection
	}
	if env.Message != nil {
		e.Msg = *env.Message
	} else {
		e.Msg = e.reconstructMessage()
	}
	return e
}

// reconstructMessage is the pre-0.8.0 fallback, used only when the
// envelope has no `message` field. Renders the envelope as
//
//	ktav: <Class>[ <Reason>][ at line N][ (path "a"."b.c")]: <Body>
//
// omitting empty parts (and the trailing colon when Body is empty).
// Fragments conformance tests match on — reason codes like
// EmptyKeyName and Body text like "object or array" — survive here.
// Against a real 0.8.0 core, those same fragments survive too: the
// core's own Display text embeds the class/reason name up front (e.g.
// "NonFiniteFloat: a Float is NaN or ±Infinity ..."), which is why
// switching to verbatim `message` did not need any test changes here.
func (e *Error) reconstructMessage() string {
	var b strings.Builder
	b.WriteString("ktav: ")
	b.WriteString(e.Class)
	if e.Reason != "" {
		b.WriteByte(' ')
		b.WriteString(e.Reason)
	}
	if e.Line > 0 {
		b.WriteString(" at line ")
		b.WriteString(strconv.Itoa(e.Line))
	}
	if len(e.Path) > 0 {
		b.WriteString(" (path ")
		for i, seg := range e.Path {
			if i > 0 {
				b.WriteByte('.')
			}
			b.WriteString(strconv.Quote(seg))
		}
		b.WriteByte(')')
	}
	if e.Body != "" {
		b.WriteString(": ")
		b.WriteString(e.Body)
	}
	return b.String()
}
