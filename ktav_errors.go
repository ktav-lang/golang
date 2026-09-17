package ktav

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Error is the binding's error type, carrying the native ktav error
// envelope's nine fields as first-class members. A zero value ("" / 0 /
// nil) corresponds to an explicit JSON null in the envelope. Msg is a
// human-readable reconstruction — never the raw envelope JSON.
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

// envelopeJSON is the wire shape of the native error envelope
// (ktav::ErrorEnvelope::to_json): one JSON object, nine fields in
// order, absent info as explicit null. Nullable fields are pointers
// here so a JSON null maps to nil rather than a zero value.
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
}

// errorFromEnvelope parses the native error payload as the structured
// envelope JSON and reconstructs a human-readable Msg. If the bytes do
// not parse as a JSON object with a string `error` field (e.g. a stale
// pre-envelope native library still emitting plain message strings),
// it falls back to a Msg-only Error carrying the raw text.
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
	e.Msg = e.reconstructMessage()
	return e
}

// reconstructMessage renders the envelope as
//
//	ktav: <Class>[ <Reason>][ at line N][ (path "a"."b.c")]: <Body>
//
// omitting empty parts (and the trailing colon when Body is empty).
// Fragments conformance tests match on — reason codes like
// EmptyKeyName and Body text like "object or array" — survive here.
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
