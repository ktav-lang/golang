package engine

// Wiring for the pure backend: parse entries (plain + trivia), the
// tagged-JSON bridge shared with ktav_wire.go, the wire ingest that
// mirrors the Rust cabi's payload canonicalisation, and the seam
// functions the vtable dispatches to. Pure code touches no loader /
// purego / FFI symbols anywhere in this file.

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"
)

// ─── parse entries ────────────────────────────────────────────────────

// splitLines iterates the document's lines (§ 3.2: LF, CR and CRLF all
// terminate; the final line may lack a terminator) and calls handle
// with each line's text, 1-based number and absolute byte start.
func splitLines(src string, skip int, handle func(line string, lineNum, lineStart int) *Error) *Error {
	hasCR := strings.IndexByte(src, '\r') >= 0
	lineNum := 0
	if !hasCR {
		lineStart := skip
		for {
			end := lineStart + strings.IndexByte(src[lineStart:], '\n')
			if end < lineStart {
				end = len(src)
			}
			lineNum++
			if err := handle(src[lineStart:end], lineNum, lineStart); err != nil {
				return err
			}
			if end == len(src) {
				return nil
			}
			lineStart = end + 1
		}
	}
	lineStart := skip
	for lineStart < len(src) {
		pos := lineStart + strings.IndexAny(src[lineStart:], "\n\r")
		if pos < lineStart {
			pos = len(src)
		}
		// Forward-progress invariant: the default MUST be len(src), so
		// a final line without a terminator never rewinds the cursor.
		nextStart := len(src)
		if pos < len(src) {
			if src[pos] == '\r' {
				if pos+1 < len(src) && src[pos+1] == '\n' {
					nextStart = pos + 2
				} else {
					nextStart = pos + 1
				}
			} else {
				nextStart = pos + 1
			}
		}
		lineNum++
		if err := handle(src[lineStart:pos], lineNum, lineStart); err != nil {
			return err
		}
		lineStart = nextStart
	}
	return nil
}

// bomSkip returns 3 when src starts with exactly one U+FEFF (§ 3.1).
func bomSkip(src string) int {
	if strings.HasPrefix(src, "\uFEFF") {
		return 3
	}
	return 0
}

// parseDoc runs the parser over src. Returns *pvalue (plain mode) or
// *fvalue plus leading trivia (trivia mode).
func parseDoc(src string, strict, trivia bool) (root any, leading []tline, err *Error) {
	p := newParser(src, strict, trivia)
	skip := bomSkip(src)
	if herr := splitLines(src, skip, p.handleLine); herr != nil {
		return nil, nil, herr
	}
	root, ferr := p.finish(len(src))
	if ferr != nil {
		return nil, nil, ferr
	}
	if trivia {
		return root, p.docLeading, nil
	}
	return root, nil, nil
}

// enrichLineText fills LineText for parse-time errors (the envelope's
// line_text field), mirroring errorFromEnvelope's source-derived value.
func enrichLineText(src string, e *Error) {
	if e != nil && e.Line > 0 && e.LineText == "" && e.Span != nil {
		e.LineText = lineTextAt(src, e.Span.Start)
	}
}

// ─── tagged-JSON bridge ───────────────────────────────────────────────

// marshalTagged renders the plain tree as tagged JSON ({"$i": digits} /
// {"$f": text} / plain values), preserving insertion order — matching
// the Rust side's value_to_json + serde_json preserve_order output.
//
// Iterative (explicit container frame stack): deep dotted-key documents
// (§ 5.3.2) produce unbounded-depth trees; the encoder must not recurse.
func marshalTagged(v *pvalue, b *bytes.Buffer) {
	type mframe struct {
		v       *pvalue
		i       int  // next child index
		started bool // opener written
	}
	stack := []*mframe{{v: v}}
	for len(stack) > 0 {
		f := stack[len(stack)-1]
		switch f.v.kind {
		case kNull:
			b.WriteString("null")
			stack = stack[:len(stack)-1]
		case kBool:
			if f.v.b {
				b.WriteString("true")
			} else {
				b.WriteString("false")
			}
			stack = stack[:len(stack)-1]
		case kInt:
			b.WriteString(`{"$i":`)
			writeJSONString(b, f.v.s)
			b.WriteByte('}')
			stack = stack[:len(stack)-1]
		case kFloat:
			b.WriteString(`{"$f":`)
			writeJSONString(b, f.v.s)
			b.WriteByte('}')
			stack = stack[:len(stack)-1]
		case kStr:
			writeJSONString(b, f.v.s)
			stack = stack[:len(stack)-1]
		case kArr:
			if !f.started {
				f.started = true
				b.WriteByte('[')
			}
			if f.i < len(f.v.arr) {
				if f.i > 0 {
					b.WriteByte(',')
				}
				child := f.v.arr[f.i]
				f.i++
				stack = append(stack, &mframe{v: child})
				continue
			}
			b.WriteByte(']')
			stack = stack[:len(stack)-1]
		case kObj:
			if !f.started {
				f.started = true
				b.WriteByte('{')
			}
			if f.i < len(f.v.obj.entries) {
				if f.i > 0 {
					b.WriteByte(',')
				}
				e := f.v.obj.entries[f.i]
				f.i++
				writeJSONString(b, e.key)
				b.WriteByte(':')
				stack = append(stack, &mframe{v: e.val})
				continue
			}
			b.WriteByte('}')
			stack = stack[:len(stack)-1]
		}
	}
}

func writeJSONString(b *bytes.Buffer, s string) {
	enc, err := json.Marshal(s)
	if err != nil {
		// json.Marshal of a string cannot fail (Go strings cannot hold
		// lone surrogates); kept total for safety.
		b.WriteString(`""`)
		return
	}
	b.Write(enc)
}

// messageErr builds the "Message"-class error the cabi's
// message_envelope produces for wire-contract violations.
func messageErr(msg string) *Error {
	return &Error{Msg: msg, Class: "Message", Body: msg}
}

// ingestWire decodes tagged JSON into the plain tree, canonicalising
// `$i` / `$f` payloads exactly like the Rust cabi's wire_to_value.
//
// The decode is ITERATIVE: an explicit container stack replaces the
// recursive token walk, so deeply nested tagged JSON (a deep dotted-key
// document round-tripping through Loads) cannot overflow the goroutine
// stack. encoding/json's Decoder.Token is itself a flat state machine
// (no per-nesting-level recursion), so the whole path is stack-bounded
// by this frame stack's heap allocation, not by goroutine stack growth.
func ingestWire(src []byte) (*pvalue, *Error) { return ingestWireOpt(src, false) }

// ingestWireValue is the edit-engine's wire ingest: the tagged transport
// may also be a bare scalar (edit replacements are not restricted to
// top-level documents).
func ingestWireValue(src []byte) (*pvalue, *Error) { return ingestWireOpt(src, true) }

func ingestWireOpt(src []byte, allowScalar bool) (*pvalue, *Error) {
	dec := json.NewDecoder(bytes.NewReader(src))
	dec.UseNumber()

	type cframe struct {
		obj    *pvalue // non-nil for object frames
		arr    *pvalue // non-nil for array frames
		key    string  // object frames: key awaiting its value
		hasKey bool
	}
	// attach places val into the innermost open container.
	attach := func(top *cframe, key string, val *pvalue) *Error {
		if top.arr != nil {
			top.arr.arr = append(top.arr.arr, val)
			return nil
		}
		if _, occupied := top.obj.obj.insertLeaf(key, val); occupied {
			return messageErr("input JSON: duplicate object key")
		}
		return nil
	}
	// scalarValue converts a non-delimiter token (with $i/$f payloads
	// canonicalised like the recursive reference decoder).
	scalarValue := func(tok json.Token) (*pvalue, *Error) {
		switch t := tok.(type) {
		case nil:
			return &pvalue{kind: kNull}, nil
		case bool:
			return pvBool(t), nil
		case json.Number:
			payload := string(t)
			if looksLikeFloatText(payload) {
				return pvFloat(ingestFloat(payload)), nil
			}
			return pvInt(ingestInteger(payload)), nil
		case string:
			return pvStr(t), nil
		default:
			return nil, messageErr("input JSON: unexpected token")
		}
	}

	var stack []*cframe
	root := &pvalue{kind: kNull}
	closed := false
	for !closed {
		tok, err := dec.Token()
		if err != nil {
			return nil, messageErr("input JSON: " + err.Error())
		}
		if delim, ok := tok.(json.Delim); ok {
			switch delim {
			case '[', '{':
				f := &cframe{arr: pvArr()}
				if delim == '{' {
					f.arr, f.obj = nil, pvObj()
				}
				stack = append(stack, f)
			default: // ']' or '}'
				if len(stack) == 0 {
					return nil, messageErr("input JSON: unexpected delimiter")
				}
				top := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				val := top.arr
				if val == nil {
					val = top.obj
					// Tagged scalar shapes are single-entry objects.
					if len(val.obj.entries) == 1 {
						e := val.obj.entries[0]
						if e.key == "$i" || e.key == "$f" {
							if e.val.kind != kStr {
								return nil, messageErr(e.key + " payload must be a string")
							}
							if e.key == "$i" {
								val = pvInt(ingestInteger(e.val.s))
							} else {
								val = pvFloat(ingestFloat(e.val.s))
							}
						}
					}
				}
				if len(stack) == 0 {
					root, closed = val, true
					break
				}
				top2 := stack[len(stack)-1]
				key := top2.key
				if ierr := attach(top2, key, val); ierr != nil {
					return nil, ierr
				}
				top2.hasKey = false
			}
			continue
		}
		// Non-delimiter token: object KEY when the top frame awaits one,
		// otherwise a scalar value.
		if len(stack) > 0 {
			top := stack[len(stack)-1]
			if top.obj != nil && !top.hasKey {
				key, ok := tok.(string)
				if !ok {
					return nil, messageErr("input JSON: non-string object key")
				}
				top.key, top.hasKey = key, true
				continue
			}
		}
		val, ierr := scalarValue(tok)
		if ierr != nil {
			return nil, ierr
		}
		if len(stack) == 0 {
			// Top-level scalar: rejected after the trailing-data check.
			root, closed = val, true
			break
		}
		top := stack[len(stack)-1]
		if ierr := attach(top, top.key, val); ierr != nil {
			return nil, ierr
		}
		top.hasKey = false
	}

	// Trailing data check (serde_json::from_slice rejects it).
	if _, err := dec.Token(); err == nil {
		return nil, messageErr("input JSON: invalid character after top-level value")
	}
	if !allowScalar && root.kind != kObj && root.kind != kArr {
		return nil, messageErr("top-level Ktav document must be an object or array")
	}
	return root, nil
}

// ingestInteger mirrors canonical_integer: strip +, redundant leading
// zeros, and signed zero; digits only.
func ingestInteger(s string) string {
	canonical, ok := canonicalIntegerText(s)
	if !ok {
		return s // unreachable from our own encoder; kept total
	}
	return canonical
}

// ingestFloat mirrors canonical_float: parse, ryu shortest round-trip;
// non-finite payloads pass through verbatim.
func ingestFloat(s string) string {
	if !looksLikeFloatText(s) {
		return s
	}
	f, err := strconvParseFloat(s)
	if err != nil && !isNonFiniteF(f) {
		return s
	}
	if isNonFiniteF(f) {
		return s
	}
	return formatRyu(f)
}

// ─── seam functions (pure side of the vtable) ─────────────────────────

func pureParseToTagged(src []byte, strict bool) ([]byte, error) {
	text := string(src)
	root, _, perr := parseDoc(text, strict, false)
	if perr != nil {
		enrichLineText(text, perr)
		return nil, perr
	}
	var b bytes.Buffer
	marshalTagged(root.(*pvalue), &b)
	return b.Bytes(), nil
}

func pureLoadsTagged(src []byte) ([]byte, error) { return pureParseToTagged(src, false) }

func pureLoadsStrictTagged(src []byte) ([]byte, error) { return pureParseToTagged(src, true) }

func pureDumpsTagged(tagged []byte) ([]byte, error) {
	return pureRenderTagged(tagged, modePretty, false)
}

func pureDumpsForceStringsTagged(tagged []byte) ([]byte, error) {
	return pureRenderTagged(tagged, modePretty, true)
}

func pureEmitCanonicalTagged(tagged []byte) ([]byte, error) {
	return pureRenderTagged(tagged, modeCanonical, false)
}

func pureRenderTagged(tagged []byte, mode writerMode, forceStrings bool) ([]byte, error) {
	v, ierr := ingestWire(tagged)
	if ierr != nil {
		return nil, ierr
	}
	w := &writer{mode: mode, forceStrings: forceStrings}
	w.renderRoot(v)
	if w.reason != "" {
		return nil, w.err()
	}
	out := []byte(w.b.String())
	return out, nil
}

// pureCanonicalFromSource is the text→text canonical path: the parse
// never materialises a host value, so every scalar spelling survives
// to the writer byte-exact.
func pureCanonicalFromSource(src []byte) ([]byte, error) {
	text := string(src)
	root, _, perr := parseDoc(text, false, false)
	if perr != nil {
		enrichLineText(text, perr)
		return nil, perr
	}
	w := &writer{mode: modeCanonical}
	w.renderRoot(root.(*pvalue))
	if w.reason != "" {
		return nil, w.err()
	}
	return []byte(w.b.String()), nil
}

// pureFormatSource parses with trivia preserved and re-emits through
// the canonical-structure formatter.
func pureFormatSource(src []byte) ([]byte, error) {
	text := string(src)
	root, leading, perr := parseDoc(text, false, true)
	if perr != nil {
		enrichLineText(text, perr)
		return nil, perr
	}
	fw := &fmtWriter{}
	if err := fw.emitDoc(root.(*fvalue), leading); err != nil {
		return nil, err
	}
	return []byte(fw.b.String()), nil
}

// small strconv shim keeps the numeric helpers decoupled from imports.
func strconvParseFloat(s string) (float64, error) { return strconv.ParseFloat(s, 64) }
