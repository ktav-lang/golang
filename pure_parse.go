package ktav

// The pure parser: line splitting, root-kind detection (§ 5.0.1), line
// dispatch (§ 5.1), pair/array-item/multiline/inline handling, dotted
// insertion with § 6.2/§ 6.3 outcomes, and the trivia-preserving mode
// that backs FormatSource. One pass over the input; the LINE-level
// state machine is iterative with an explicit frame stack. The only
// recursion is inline-compound parsing (§ 5.8), bounded by the
// declared 128-level cap; dotted keys never recurse (see
// insertValue/insertDotted: linear split + O(1) hash-index descend).

import (
	"strings"
	"unicode/utf8"
)

// kspan is a half-open byte-offset range in the source.
type kspan struct{ start, end int }

// ─── error constructors (envelope parity with the native library) ─────

const emDash = "—"

// lineTextAt returns the source line containing byte offset pos
// (LF-based search; one trailing CR stripped), or "" for empty source.
func lineTextAt(src string, pos int) string {
	if src == "" {
		return ""
	}
	if pos > len(src) {
		pos = len(src)
	}
	start := strings.LastIndexByte(src[:pos], '\n') + 1
	end := pos + strings.IndexByte(src[pos:], '\n')
	if end < pos {
		end = len(src)
	}
	line := src[start:end]
	return strings.TrimSuffix(line, "\r")
}

// structErr builds a parse-time error with the envelope fields.
func structErr(class string, line int, span kspan, spec, msg, body, canonical string, path []string) *Error {
	return &Error{
		Msg:         msg,
		Class:       class,
		Line:        line,
		Span:        &Span{Start: span.start, End: span.end},
		SpecSection: spec,
		Body:        body,
		Canonical:   canonical,
		Path:        path,
	}
}

func errMissingSeparatorSpace(line int, span kspan) *Error {
	return structErr("MissingSeparatorSpace", line, span, "§6.10",
		"Syntax error: Line "+itoa(line)+": MissingSeparatorSpace: separator must be followed by whitespace or end of line",
		"", "", nil)
}

func errLossyScalar(line int, body, canonical string, span kspan) *Error {
	return structErr("LossyScalar", line, span, "§3.6/§5.2",
		"Syntax error: Line "+itoa(line)+": LossyScalar: '"+body+"' would be inferred as a number and silently canonicalised to '"+canonical+
			"'; append '::' to keep it a String or write the canonical form",
		body, canonical, nil)
}

func errDuplicateKey(line int, key string, span kspan) *Error {
	return structErr("DuplicateKey", line, span, "§6.2",
		"Syntax error: Line "+itoa(line)+": duplicate key '"+key+"'", "", "", decodedKeyPath(key))
}

func errKeyPathConflictOverwrite(line int, path, existing, newKind string, span kspan) *Error {
	return structErr("KeyPathConflict", line, span, "§6.3",
		"Syntax error: Line "+itoa(line)+": conflict at '"+path+"' "+emDash+" cannot overwrite "+existing+" with "+newKind,
		"", "", decodedKeyPath(path))
}

func errKeyPathConflictBlocked(line int, path string, span kspan) *Error {
	return structErr("KeyPathConflict", line, span, "§6.3",
		"Syntax error: Line "+itoa(line)+": conflict at '"+path+"' "+emDash+" an existing value blocks the path",
		"", "", decodedKeyPath(path))
}

func errEmptyKey(line int, span kspan) *Error {
	return structErr("EmptyKey", line, span, "§6.5",
		"Syntax error: Empty key at line "+itoa(line), "", "", nil)
}

func errInvalidKey(line int, key string, span kspan) *Error {
	return structErr("InvalidKey", line, span, "§6.4",
		"Syntax error: Invalid key at line "+itoa(line)+": '"+key+"'", "", "", splitKeyPath(key))
}

func errUnclosedCompound(kind string, span kspan) *Error {
	msg := "Syntax error: Unclosed object at end of input"
	switch kind {
	case "array":
		msg = "Syntax error: Unclosed array at end of input"
	case "multiline":
		msg = "Syntax error: Unclosed multi-line string at end of input"
	}
	return structErr("UnclosedCompound", 0, span, "§6.1", msg, "", "", nil)
}

func errUnbalancedBracket(line int, span kspan, expected byte, found byte) *Error {
	opener := "{"
	if expected == '[' {
		opener = "["
	}
	return structErr("UnbalancedBracket", line, span, "§6.1",
		"Syntax error: Line "+itoa(line)+": UnbalancedBracket: '"+string(found)+"' without matching '"+opener+"'",
		"", "", nil)
}

func errMissingSeparator(line int, span kspan) *Error {
	return structErr("MissingSeparator", line, span, "§6.6",
		"Syntax error: Line "+itoa(line)+": MissingSeparator: object entries must be 'key: value' pairs",
		"", "", nil)
}

func errUnterminatedInlineCompound(line int, span kspan) *Error {
	return structErr("UnterminatedInlineCompound", line, span, "§6.11",
		"Syntax error: Line "+itoa(line)+": UnterminatedInlineCompound: inline compound not closed on the same line",
		"", "", nil)
}

func errUnterminatedQuotedKey(line int, span kspan) *Error {
	return structErr("UnterminatedQuotedKey", line, span, "§6.16",
		"Syntax error: Line "+itoa(line)+": UnterminatedQuotedKey: quoted key segment not closed on the same line",
		"", "", nil)
}

func errMalformedInlineCompound(line int, span kspan, detail string) *Error {
	return structErr("MalformedInlineCompound", line, span, "§6.12",
		"Syntax error: Line "+itoa(line)+": MalformedInlineCompound: "+detail, detail, "", nil)
}

func errBadEscapeSequence(line int, span kspan, sequence string) *Error {
	return structErr("BadEscapeSequence", line, span, "§6.13",
		"Syntax error: Line "+itoa(line)+": BadEscapeSequence: invalid escape sequence '"+sequence+"'",
		sequence, "", nil)
}

func errOrphanLine(line int, span kspan) *Error {
	return structErr("OrphanLineAfterTopLevelInline", line, span, "§6.14",
		"Syntax error: Line "+itoa(line)+": OrphanLineAfterTopLevelInline: content after root-level inline compound",
		"", "", nil)
}

// decodedKeyPath decodes a raw key-path text into exact segments
// (DuplicateKey / KeyPathConflict envelope path); a decode failure
// falls back to the raw spelling so construction stays total.
func decodedKeyPath(raw string) []string {
	segments := splitKeyPath(raw)
	out := make([]string, 0, len(segments))
	for _, seg := range segments {
		trimmed := trimInline(seg)
		dec, bad := decodeKeySegment(trimmed)
		if bad != "" {
			out = append(out, seg)
		} else {
			out = append(out, dec)
		}
	}
	return out
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// ─── multiline collection (§ 5.6) ─────────────────────────────────────

// leadingWSRunLen returns the byte length of the leading § 3.3
// whitespace run of s (code-point aligned).
func leadingWSRunLen(s string) int {
	i := 0
	for i < len(s) {
		n := inlineWSAt(s, i)
		if n == 0 {
			break
		}
		i += n
	}
	return i
}

// commonLeadingWSPrefix is the byte length of the longest leading-
// whitespace prefix shared code-point-for-code-point by every
// non-blank line (blank lines do not participate).
func commonLeadingWSPrefix(lines []string) int {
	first := -1
	var firstRun string
	runs := make([]string, 0, len(lines))
	for _, l := range lines {
		if trimInline(l) == "" {
			continue
		}
		run := l[:leadingWSRunLen(l)]
		runs = append(runs, run)
		if first < 0 {
			first = len(runs) - 1
			firstRun = run
		}
	}
	if firstRun == "" {
		return 0
	}
	minLen := len(firstRun)
	for _, r := range runs {
		if len(r) < minLen {
			minLen = len(r)
		}
	}
	// Compare code-point-for-code-point: advance over whole runes while
	// they match; ASCII bytes compare 1:1.
	i := 0
	for i < minLen {
		if firstRun[i] < 0x80 {
			match := true
			for _, r := range runs {
				if r[i] != firstRun[i] {
					match = false
					break
				}
			}
			if !match {
				break
			}
			i++
			continue
		}
		r, size := decodeRune(firstRun[i:])
		allMatch := true
		for _, run := range runs {
			r2, size2 := decodeRune(run[i:])
			if r2 != r || size2 != size {
				allMatch = false
				break
			}
		}
		if !allMatch {
			break
		}
		i += size
	}
	return i
}

// decodeRune decodes the rune at s[0] via unicode/utf8. Callers only
// ever pass positions that start a rune: every call site either walks
// forward by the returned size (starting at 0 or after a whole rune) or
// runs on text already gated by the §6.15 InvalidUtf8 check, so an
// invalid lead byte (0x80–0xBF, 0xF8+) cannot reach this function in
// practice; utf8 still makes that case total (RuneError, size 1)
// instead of mis-decoding a continuation byte as a 4-byte rune.
func decodeRune(s string) (rune, int) {
	return utf8.DecodeRuneInString(s)
}

// multilineFinish joins collected lines per § 5.6: verbatim joins
// byte-for-byte; stripped dedents by the common leading-whitespace
// prefix, trims each line's trailing whitespace, and blanks blank lines.
func multilineFinish(verbatim bool, lines []string) string {
	if !verbatim { // stripped
		if len(lines) == 1 {
			only := lines[0]
			if trimInline(only) == "" {
				return ""
			}
			return trimTrailingInline(only[leadingWSRunLen(only):])
		}
		commonLen := commonLeadingWSPrefix(lines)
		var b strings.Builder
		for i, l := range lines {
			if i > 0 {
				b.WriteByte('\n')
			}
			blank := trimInline(l) == ""
			if blank {
				continue
			}
			content := l
			if commonLen > 0 && len(l) >= commonLen {
				content = l[commonLen:]
			}
			b.WriteString(trimTrailingInline(content))
		}
		return b.String()
	}
	// Verbatim: join byte-for-byte with single LF.
	return strings.Join(lines, "\n")
}

// trimTrailingInline removes trailing § 3.3 inline whitespace.
func trimTrailingInline(s string) string {
	end := len(s)
	for end > 0 {
		// Step back one whitespace code point at a time.
		if inlineWSAt(s, end-1) > 0 && s[end-1] < 0x80 {
			end--
			continue
		}
		if s[end-1] >= 0x80 {
			// Walk back to the rune start.
			k := end - 1
			for k > 0 && s[k]&0xC0 == 0x80 {
				k--
			}
			if inlineWSAt(s, k) > 0 {
				end = k
				continue
			}
		}
		break
	}
	return s[:end]
}

// ─── scalar classification (§ 5.2) ────────────────────────────────────

// classifyScalar classifies a trimmed, escape-free body per § 5.2
// rules 10–15 (keywords, redundant zero, integer, float, String).
// Returns (kind, text, lossyCanonical) where lossyCanonical is set in
// strict mode when the spelling differs from the canonical form.
func classifyScalar(body string, strict bool, line int, span kspan) (*pvalue, *Error) {
	switch body {
	case "null":
		return &pvalue{kind: kNull}, nil
	case "true":
		return pvBool(true), nil
	case "false":
		return pvBool(false), nil
	}
	if hasRedundantLeadingZero(body) {
		return pvStr(body), nil
	}
	if val, ok := tryParseInteger(body); ok {
		canonical := itoa64(val)
		if strict && canonical != body {
			return nil, errLossyScalar(line, body, canonical, span)
		}
		return pvInt(canonical), nil
	}
	if isFloatLiteral(body) {
		if val, ok := parseFloatValue(body); ok {
			canonical := formatRyu(val)
			if strict {
				rendered, st := canonicalFloatText(canonical)
				if st == cfOK {
					if rendered != body {
						return nil, errLossyScalar(line, body, rendered, span)
					}
					return pvFloat(canonical), nil
				}
			}
			if canonical == body {
				return pvFloat(body), nil
			}
			return pvFloat(canonical), nil
		}
	}
	return pvStr(body), nil
}

// itoa64 renders v in decimal. The magnitude is formatted through
// uint64 so math.MinInt64 (`-9223372036854775808`, § 5.2 integers)
// survives the negation — int64(-MinInt64) overflows and the former
// int-based loop emitted just "-" (or "") for it.
func itoa64(v int64) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	var m uint64
	if neg {
		m = uint64(-(v + 1)) + 1 // no overflow at MinInt64
	} else {
		m = uint64(v)
	}
	var buf [20]byte
	i := len(buf)
	for m > 0 {
		i--
		buf[i] = byte('0' + m%10)
		m /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// ─── dotted insertion (§ 5.3.2, § 5.5, § 6.2, § 6.3) ──────────────────

// insertValue inserts a leaf value at a (possibly dotted) path inside
// table (*omap), validating each raw segment before decoding.
func insertValue(table *omap, path string, value *pvalue, line int, span kspan) *Error {
	if keyIsSingleSegment(path) {
		trimmed := trimInline(path)
		if trimmed == "" {
			return errEmptyKey(line, span)
		}
		if st := checkKey(trimmed); st != keyValid {
			if st == keyEmpty {
				return errEmptyKey(line, span)
			}
			return errInvalidKey(line, path, span)
		}
		decoded, bad := decodeKeySegment(trimmed)
		if bad != "" {
			return errBadEscapeSequence(line, span, bad)
		}
		existing, occupied := table.insertLeaf(decoded, value)
		if !occupied {
			return nil
		}
		if existing.isObject() != value.isObject() {
			return errKeyPathConflictOverwrite(line, path, existing.kindLabel(), value.kindLabel(), span)
		}
		return errDuplicateKey(line, path, span)
	}
	return insertDotted(table, path, value, line, span)
}

func insertDotted(table *omap, fullPath string, value *pvalue, line int, span kspan) *Error {
	segments := splitKeyPath(fullPath)
	cur := table
	for idx, seg := range segments {
		trimmed := trimInline(seg)
		if trimmed == "" {
			return errEmptyKey(line, span)
		}
		if st := checkKey(trimmed); st != keyValid {
			if st == keyEmpty {
				return errEmptyKey(line, span)
			}
			return errInvalidKey(line, fullPath, span)
		}
		decoded, bad := decodeKeySegment(trimmed)
		if bad != "" {
			return errBadEscapeSequence(line, span, bad)
		}
		if idx+1 == len(segments) {
			if _, occupied := cur.insertLeaf(decoded, value); occupied {
				return errDuplicateKey(line, fullPath, span)
			}
			return nil
		}
		next, blocked := cur.descend(decoded)
		if blocked {
			return errKeyPathConflictBlocked(line, fullPath, span)
		}
		cur = next
	}
	return nil
}

// ─── inline compound parsing (§ 5.8) ──────────────────────────────────

// maxInlineDepth is the declared nesting limit (§ 5.8.4: implementation-
// defined; mirrors the reference implementation's 128).
const maxInlineDepth = 128

// parseInlineObject parses a balanced inline object body including the
// outer `{`/`}`.
func parseInlineObject(input string, line int, span kspan, strict bool, depth int) (*pvalue, *Error) {
	if depth >= maxInlineDepth {
		return nil, errMalformedInlineCompound(line, span, "nesting depth exceeds limit (128)")
	}
	inner := input[1 : len(input)-1]
	if trimInline(inner) == "" {
		return pvObj(), nil
	}
	segments, err := splitTopLevel(inner, line, span, true)
	if err != nil {
		return nil, err
	}
	m := newOmap()
	n := len(segments)
	for i, seg := range segments {
		trimmed := trimInline(seg)
		if trimmed == "" {
			if i == n-1 {
				break // trailing comma is OK
			}
			return nil, errMalformedInlineCompound(line, span,
				"empty pair segment (leading comma, double comma, or missing pair)")
		}
		colon := scanUnescapedColon(trimmed)
		if colon != colonFound {
			return nil, errMalformedInlineCompound(line, span,
				"inline object pair missing ':' separator in '"+trimmed+"'")
		}
		// Recompute the byte offset (scanUnescapedColon returns a
		// status; walk the bytes for the position).
		colonPos := indexUnescapedColon(trimmed)
		rawKey := trimmed[:colonPos]
		afterColon := trimmed[colonPos+1:]
		isRaw := false
		valueBody := afterColon
		if len(afterColon) > 0 && afterColon[0] == ':' {
			isRaw = true
			valueBody = afterColon[1:]
		}
		key := trimInline(rawKey)
		if key == "" {
			return nil, errEmptyKey(line, span)
		}
		var value *pvalue
		if isRaw {
			processed, _, bad := processEscapes(trimInline(valueBody))
			if bad != "" {
				return nil, errBadEscapeSequence(line, span, bad)
			}
			value = pvStr(processed)
		} else {
			value, err = parseInlineValue(valueBody, line, span, depth, strict)
			if err != nil {
				return nil, err
			}
		}
		if err := insertValue(m, key, value, line, span); err != nil {
			return nil, err
		}
	}
	return &pvalue{kind: kObj, obj: m}, nil
}

// indexUnescapedColon returns the byte offset of the first unescaped
// `:` outside quoted segments (precondition: scanUnescapedColon == found).
func indexUnescapedColon(s string) int {
	i := 0
	segStart := true
	for i < len(s) {
		if segStart {
			for inlineWSAt(s, i) > 0 {
				i += inlineWSAt(s, i)
			}
			if i < len(s) && isQuoteByte(s[i]) {
				end, ok := quotedSpanEnd([]byte(s), i)
				if !ok {
					return -1
				}
				i = end + 1
				segStart = false
				continue
			}
			segStart = false
		}
		switch s[i] {
		case '\\':
			i += 2
		case ':':
			return i
		case '.':
			segStart = true
			i++
		default:
			i++
		}
		if i > len(s) {
			break
		}
	}
	return -1
}

func parseInlineArray(input string, line int, span kspan, strict bool, depth int) (*pvalue, *Error) {
	if depth >= maxInlineDepth {
		return nil, errMalformedInlineCompound(line, span, "nesting depth exceeds limit (128)")
	}
	inner := input[1 : len(input)-1]
	if trimInline(inner) == "" {
		return pvArr(), nil
	}
	segments, err := splitTopLevel(inner, line, span, false)
	if err != nil {
		return nil, err
	}
	items := []*pvalue{}
	n := len(segments)
	for i, seg := range segments {
		trimmed := trimInline(seg)
		if trimmed == "" {
			if i == n-1 {
				break
			}
			return nil, errMalformedInlineCompound(line, span,
				"empty inline-array item (leading comma, double comma, or empty position)")
		}
		v, err := parseInlineValueRaw(trimmed, line, span, depth, strict)
		if err != nil {
			return nil, err
		}
		items = append(items, v)
	}
	return &pvalue{kind: kArr, arr: items}, nil
}

// parseInlineValue parses an inline pair value (body not yet trimmed).
func parseInlineValue(body string, line int, span kspan, depth int, strict bool) (*pvalue, *Error) {
	trimmed := trimInline(body)
	if trimmed == "" {
		return pvStr(""), nil
	}
	return parseInlineValueRaw(trimmed, line, span, depth, strict)
}

// parseInlineValueRaw parses an already-trimmed inline value: only the
// FIRST byte decides compound vs scalar (§ 5.8.5).
func parseInlineValueRaw(trimmed string, line int, span kspan, depth int, strict bool) (*pvalue, *Error) {
	first := trimmed[0]
	if first == '{' {
		closeIdx, stop, badErr := scanInlineCloser(trimmed, '{', '}', line, span)
		if badErr != nil {
			return nil, badErr
		}
		switch {
		case stop == stopCloser && closeIdx == len(trimmed)-1:
			if trimInline(trimmed[1:len(trimmed)-1]) == "" {
				// Empty inline object: the compound itself sits at
				// depth+1 (§ 5.8.4) — the cap applies here too, or a
				// chain whose innermost level is empty never counts it.
				if depth+1 >= maxInlineDepth {
					return nil, errMalformedInlineCompound(line, span, "nesting depth exceeds limit (128)")
				}
				return pvObj(), nil
			}
			return parseInlineObject(trimmed, line, span, strict, depth+1)
		case stop == stopCloser:
			return nil, malformedCloserNotAtEnd(line, span)
		default:
			return nil, errUnterminatedInlineCompound(line, span)
		}
	}
	if first == '[' {
		closeIdx, stop, badErr := scanInlineCloser(trimmed, '[', ']', line, span)
		if badErr != nil {
			return nil, badErr
		}
		switch {
		case stop == stopCloser && closeIdx == len(trimmed)-1:
			if trimInline(trimmed[1:len(trimmed)-1]) == "" {
				// Empty inline array: same depth+1 accounting as the
				// empty object above (§ 5.8.4).
				if depth+1 >= maxInlineDepth {
					return nil, errMalformedInlineCompound(line, span, "nesting depth exceeds limit (128)")
				}
				return pvArr(), nil
			}
			return parseInlineArray(trimmed, line, span, strict, depth+1)
		case stop == stopCloser:
			return nil, malformedCloserNotAtEnd(line, span)
		default:
			return nil, errUnterminatedInlineCompound(line, span)
		}
	}
	if trimmed == "()" || trimmed == "(())" {
		return pvStr(""), nil
	}
	processed, escaped, bad := processEscapes(trimmed)
	if bad != "" {
		return nil, errBadEscapeSequence(line, span, bad)
	}
	if escaped {
		return pvStr(processed), nil
	}
	return classifyScalar(processed, strict, line, span)
}

// classifyValueStart dispatches a pair value / array item body
// (§ 5.2 rules 1–15).
const (
	vsScalar = iota
	vsNull
	vsBool
	vsInt
	vsFloat
	vsEmptyObject
	vsEmptyArray
	vsOpenObject
	vsOpenArray
	vsOpenMultilineStripped
	vsOpenMultilineVerbatim
	vsInline
)

type valueStart struct {
	kind int
	b    bool
	s    string // scalar text / inline body
	inl  *pvalue
}

func classifyValueStart(text string, line int, span kspan, strict bool) (valueStart, *Error) {
	return classifyValueStartDepth(text, line, span, strict, 0)
}

// classifyValueStartDepth is classifyValueStart with the inline-nesting
// depth threaded through (§ 5.8.4): line-level callers enter at 0; the
// depth travels into dispatchInlineCompound so nested compounds keep
// counting toward the declared 128 cap.
func classifyValueStartDepth(text string, line int, span kspan, strict bool, depth int) (valueStart, *Error) {
	trimmed := trimInline(text)
	if trimmed == "{" {
		return valueStart{kind: vsOpenObject}, nil
	}
	if trimmed == "[" {
		return valueStart{kind: vsOpenArray}, nil
	}
	if len(trimmed) > 0 && trimmed[0] == '{' {
		if trimmed[len(trimmed)-1] == '}' && trimInline(trimmed[1:len(trimmed)-1]) == "" {
			return valueStart{kind: vsEmptyObject}, nil
		}
		return dispatchInlineCompoundDepth(trimmed, '{', '}', line, span, strict, depth)
	}
	if len(trimmed) > 0 && trimmed[0] == '[' {
		if trimmed[len(trimmed)-1] == ']' && trimInline(trimmed[1:len(trimmed)-1]) == "" {
			return valueStart{kind: vsEmptyArray}, nil
		}
		return dispatchInlineCompoundDepth(trimmed, '[', ']', line, span, strict, depth)
	}
	switch trimmed {
	case "(":
		return valueStart{kind: vsOpenMultilineStripped}, nil
	case "((":
		return valueStart{kind: vsOpenMultilineVerbatim}, nil
	case "()", "(())":
		return valueStart{kind: vsScalar, s: ""}, nil
	}
	switch trimmed {
	case "null":
		return valueStart{kind: vsNull}, nil
	case "true":
		return valueStart{kind: vsBool, b: true}, nil
	case "false":
		return valueStart{kind: vsBool, b: false}, nil
	}
	if hasRedundantLeadingZero(trimmed) {
		return valueStart{kind: vsScalar, s: trimmed}, nil
	}
	if val, ok := tryParseInteger(trimmed); ok {
		canonical := itoa64(val)
		if strict && canonical != trimmed {
			return valueStart{}, errLossyScalar(line, trimmed, canonical, span)
		}
		return valueStart{kind: vsInt, s: canonical}, nil
	}
	if isFloatLiteral(trimmed) {
		if val, ok := parseFloatValue(trimmed); ok {
			canonical := formatRyu(val)
			if strict {
				rendered, st := canonicalFloatText(canonical)
				if st == cfOK {
					if rendered != trimmed {
						return valueStart{}, errLossyScalar(line, trimmed, rendered, span)
					}
					return valueStart{kind: vsFloat, s: canonical}, nil
				}
			}
			if canonical == trimmed {
				return valueStart{kind: vsFloat, s: trimmed}, nil
			}
			return valueStart{kind: vsFloat, s: canonical}, nil
		}
	}
	return valueStart{kind: vsScalar, s: trimmed}, nil
}

func dispatchInlineCompound(trimmed string, open, close byte, line int, span kspan, strict bool) (valueStart, *Error) {
	return dispatchInlineCompoundDepth(trimmed, open, close, line, span, strict, 0)
}

func dispatchInlineCompoundDepth(trimmed string, open, close byte, line int, span kspan, strict bool, depth int) (valueStart, *Error) {
	closeIdx, stop, badErr := scanInlineCloser(trimmed, open, close, line, span)
	if badErr != nil {
		return valueStart{}, badErr
	}
	switch {
	case stop == stopCloser && closeIdx == len(trimmed)-1:
		var v *pvalue
		var err *Error
		if open == '{' {
			v, err = parseInlineObject(trimmed, line, span, strict, depth)
		} else {
			v, err = parseInlineArray(trimmed, line, span, strict, depth)
		}
		if err != nil {
			return valueStart{}, err
		}
		return valueStart{kind: vsInline, inl: v}, nil
	case stop == stopCloser:
		return valueStart{}, malformedCloserNotAtEnd(line, span)
	default:
		return valueStart{}, errUnterminatedInlineCompound(line, span)
	}
}

// isPairShape is § 5.0.1 rule 6's phase-1 pair-candidate test (and the
// writers' Array-root first-item safeguard): a first unescaped `:` or
// `::` separator with a non-empty raw prefix, a plain `:` satisfied by
// whitespace or end-of-line.
func isPairShape(trimmed string) bool {
	colonPos := indexUnescapedColon(trimmed)
	if colonPos < 0 {
		return false
	}
	keyPart := trimTrailingInline(trimmed[:colonPos])
	if keyPart == "" {
		return false
	}
	after := trimmed[colonPos+1:]
	if len(after) > 0 && after[0] == ':' {
		return true
	}
	return after == "" || inlineWSAt(after, 0) > 0
}

// classifyRootKind is § 5.0.1 rules 2–7 (rules 1 and 8 handled by the
// caller). kind: 0 inline object, 1 inline array, 2 explicit object,
// 3 explicit array, 4 implicit object, 5 implicit array.
func classifyRootKind(trimmed string, line int, span kspan, strict bool) (int, *pvalue, *Error) {
	if trimmed == "{" {
		return 2, nil, nil
	}
	if trimmed == "[" {
		return 3, nil, nil
	}
	if len(trimmed) > 0 && trimmed[0] == '{' {
		return diagnoseRootInline(trimmed, '{', '}', line, span, strict)
	}
	if len(trimmed) > 0 && trimmed[0] == '[' {
		return diagnoseRootInline(trimmed, '[', ']', line, span, strict)
	}
	if isPairShape(trimmed) {
		return 4, nil, nil
	}
	return 5, nil, nil
}

func diagnoseRootInline(trimmed string, open, close byte, line int, span kspan, strict bool) (int, *pvalue, *Error) {
	closeIdx, stop, badErr := scanInlineCloser(trimmed, open, close, line, span)
	if badErr != nil {
		return 0, nil, badErr
	}
	switch {
	case stop == stopCloser && closeIdx == len(trimmed)-1:
		kind := 0
		if open == '[' {
			kind = 1
		}
		v, err := dispatchInlineRoot(trimmed, open, line, span, strict)
		if err != nil {
			return 0, nil, err
		}
		return kind, v, nil
	case stop == stopCloser:
		return 0, nil, malformedCloserNotAtEnd(line, span)
	default:
		return 0, nil, errUnterminatedInlineCompound(line, span)
	}
}

func dispatchInlineRoot(trimmed string, open byte, line int, span kspan, strict bool) (*pvalue, *Error) {
	if open == '{' {
		return parseInlineObject(trimmed, line, span, strict, 0)
	}
	return parseInlineArray(trimmed, line, span, strict, 0)
}
