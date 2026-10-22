package engine

import (
	"strconv"
	"strings"
	"unicode/utf8"
)

// Emission helpers for the pure writers: § 5.9.10 key emission, § 5.9.7
// string-form selection and the raw-marker/numeric classification
// scans behind the writers (pure_write.go). Output is assembled in a
// local buffer first, so a rejection never emits partial output
// (§ 5.9.0).

const indentUnit = "    "

func pushIndent(b *strings.Builder, level int) {
	for i := 0; i < level; i++ {
		b.WriteString(indentUnit)
	}
}

// lastRune returns the final rune of s ("" for empty). Decoding goes
// through unicode/utf8: the width of the last rune is determined from
// its LEAD byte, never from the last byte (a continuation byte would
// mis-size the rune — the é panic this replaced).
func lastRune(s string) (rune, bool) {
	if s == "" {
		return 0, false
	}
	r, _ := utf8.DecodeLastRuneInString(s)
	return r, true
}

// ─── § 5.9.7 string-form selection ────────────────────────────────────

// stringNeedsMultiline: LF/CR/control bytes or § 3.3 edge whitespace
// route a body to a multi-line form (§ 5.9.7).
func stringNeedsMultiline(s string) bool {
	if s == "" {
		return false
	}
	r, _ := utf8.DecodeRuneInString(s)
	if isKtavWhitespaceR(r) {
		return true
	}
	if r2, ok := lastRune(s); ok && isKtavWhitespaceR(r2) {
		return true
	}
	for i := 0; i < len(s); i++ {
		if s[i] < 0x20 && s[i] != '\t' {
			return true
		}
	}
	return false
}

// chooseMultilineForm picks the multi-line string form that reproduces
// s byte-for-byte on re-parse, or the § 5.9.0 reason code when neither
// form can. preferStripped keeps the pretty writers' readable default.
func chooseMultilineForm(s string, preferStripped bool) (stripped bool, reason string) {
	soleSingle, soleDouble := false, false
	wsOnlyLine, indentedLine, trailingWSLine := false, false, false
	for _, line := range splitLF(s) {
		trimmed := trimInline(line)
		switch trimmed {
		case ")":
			soleSingle = true
		case "))":
			soleDouble = true
		}
		if line == "" {
			continue
		}
		if trimmed == "" {
			wsOnlyLine = true
		} else if inlineWSAt(line, 0) > 0 {
			indentedLine = true
		}
		if trimTrailingInline(line) != line {
			trailingWSLine = true
		}
	}
	strippedSafe := !soleSingle && !wsOnlyLine && !indentedLine && !trailingWSLine
	verbatimOK := !soleDouble
	if preferStripped && strippedSafe {
		return true, ""
	}
	if verbatimOK {
		return false, ""
	}
	hasCommonIndent := commonLeadingWSPrefix(splitLF(s)) != 0
	strippedLossless := !soleSingle && !wsOnlyLine && !hasCommonIndent && !trailingWSLine
	if strippedLossless {
		return true, ""
	}
	switch {
	case soleSingle:
		return false, "BothFormsRequired"
	case trailingWSLine:
		return false, "TrailingWhitespaceCollision"
	default:
		return false, "LeadingWhitespaceCollision"
	}
}

func splitLF(s string) []string {
	out := []string{}
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	return append(out, s[start:])
}

// needsRawMarker: would § 5.2 re-classify this one-line body as
// something other than a String?
func needsRawMarker(s string) bool {
	if s == "" {
		return false
	}
	examined := 0
	res := false
	if ws := inlineWSAt(s, 0); ws > 0 {
		t := trimLeadingInline(s)
		examined += ws + len(t)
		res = t != "" && (t[0] == '{' || t[0] == '[' || needsRawMarkerContent(t))
	} else {
		examined++
		if s[0] == '{' || s[0] == '[' {
			res = true
		} else {
			res = needsRawMarkerContent(s)
		}
	}
	// ONE probe at exit: the bytes this decision actually examined
	// (leading-whitespace run, first byte, and the trimmed body the
	// content classification walked).
	probe("rawmarker_scan", examined)
	return res
}

// needsRawMarkerContent classifies one one-line body and, through ONE
// probe call at exit, reports the bytes actually examined by the
// grammar scans — real per-byte work, so a hidden re-scan inside the
// classification shows up as a superlinear byte count under the
// installed probe. NOTE (honest cost): the examined counters add
// arithmetic per scanned byte INSIDE these two classification
// functions; when the hook is nil, the probe() call itself is one
// nil-test and no more, but the arithmetic is NOT removed (removing it
// would fork the semantics of the scans). It is bounded by one pass
// over the body.
func needsRawMarkerContent(s string) bool {
	examined := 0
	res := false
	// § 5.2 keywords and bare multi-line openers MUST use `::` so they
	// re-parse as String, not Bool/null/compound (Rust reference
	// render::helpers::needs_raw_marker_content: `()`/`(())` parse as
	// the empty String; other `(`-bodies like `(tail` re-parse as the
	// same String and need no marker).
	switch s {
	case "null", "true", "false", "(", "((", "()", "(())":
		examined = len(s)
		res = true
	default:
		res = matchesIntegerGrammar(s, &examined) || isFloatLiteralN(s, &examined)
	}
	probe("numeric_scan", examined)
	return res
}

// matchesIntegerGrammar is the § 3.6 integer grammar syntactically
// (value may overflow i64; lowercase base prefixes only). It adds the
// bytes it actually inspects to *examined.
func matchesIntegerGrammar(s string, examined *int) bool {
	b := []byte(s)
	i := 0
	if i < len(b) && (b[i] == '+' || b[i] == '-') {
		i++
		*examined++
	}
	if i >= len(b) {
		return false
	}
	if b[i] == '0' && i+1 < len(b) {
		*examined += 2
		switch b[i+1] {
		case 'x':
			n, ok := checkPrefixedDigits(b[i+2:], 16)
			*examined += n
			return ok
		case 'o':
			n, ok := checkPrefixedDigits(b[i+2:], 8)
			*examined += n
			return ok
		case 'b':
			n, ok := checkPrefixedDigits(b[i+2:], 2)
			*examined += n
			return ok
		}
	}
	n, ok := checkDecimalDigits(b[i:])
	*examined += n
	return ok
}

func checkPrefixedDigits(digits []byte, radix int) (int, bool) {
	n := 0
	if len(digits) == 0 || !isDigitForRadix(digits[0], radix) {
		return n, false
	}
	n++
	prevUnderscore := false
	for _, c := range digits[1:] {
		n++
		if c == '_' {
			if prevUnderscore {
				return n, false
			}
			prevUnderscore = true
			continue
		}
		prevUnderscore = false
		if !isDigitForRadix(c, radix) {
			return n, false
		}
	}
	return n, !prevUnderscore
}

func checkDecimalDigits(digits []byte) (int, bool) {
	n := 0
	if len(digits) == 0 || digits[0] < '0' || digits[0] > '9' {
		return n, false
	}
	n++
	prevUnderscore := false
	for _, c := range digits[1:] {
		n++
		if c == '_' {
			if prevUnderscore {
				return n, false
			}
			prevUnderscore = true
			continue
		}
		prevUnderscore = false
		if c < '0' || c > '9' {
			return n, false
		}
	}
	return n, !prevUnderscore
}

// itemNeedsRawMarker: the item-position collisions a bare array-item
// line has on top of the pair-value ones.
func itemNeedsRawMarker(s string) bool {
	return needsRawMarker(s) || strings.HasPrefix(s, "##") || strings.HasPrefix(s, "::") ||
		s == "]" || s == "}"
}

// ─── § 5.9.10 key emission ────────────────────────────────────────────

func pushUnicodeEscape(b *strings.Builder, r rune) {
	const hex = "0123456789ABCDEF"
	cp := uint32(r)
	if cp > 0xFFFF {
		b.WriteString("\\u")
		for shift := 28; shift >= 0; shift -= 4 {
			b.WriteByte(hex[(cp>>uint(shift))&0xF])
		}
		return
	}
	b.WriteString("\\u")
	for _, shift := range [4]uint{12, 8, 4, 0} {
		b.WriteByte(hex[(cp>>shift)&0xF])
	}
}

// pushEscapedKeySegment emits a key segment in bare or quoted form per
// § 5.9.10; rootFirstKey enables the § 5.9.12 rule (c) U+FEFF guard.
func pushEscapedKeySegment(key string, rootFirstKey bool, b *strings.Builder) {
	first := rune(-1)
	if key != "" {
		first, _ = utf8.DecodeRuneInString(key)
	}
	if first == '"' || first == '\'' || first == '`' ||
		(rootFirstKey && first == 0xFEFF) ||
		strings.HasPrefix(key, "##") {
		pushQuotedKey(key, b)
		return
	}
	if key != "" {
		// (a) edge § 3.3 whitespace other than LF/CR: quoting buys the
		// no-trim guarantee (quoted content is never trimmed).
		if isInlineWhitespaceR(first) {
			pushQuotedKey(key, b)
			return
		}
		if lastr, ok := lastRune(key); ok && isInlineWhitespaceR(lastr) {
			pushQuotedKey(key, b)
			return
		}
	}
	needsRewrite := false
	for i := 0; i < len(key); i++ {
		c := key[i]
		switch c {
		case '.', ':', ',', '{', '}', '[', ']', '(', ')':
			pushQuotedKey(key, b)
			return
		case '\\', '\n', '\r', 0x7F:
			needsRewrite = true
		default:
			if c < 0x20 && !inlineWSByte(c) {
				needsRewrite = true
			}
		}
	}
	if needsRewrite {
		pushBareKeyEscaped(key, b)
	} else {
		b.WriteString(key)
	}
}

func pushBareKeyEscaped(key string, b *strings.Builder) {
	// The last rune's start comes from its own width (lead-byte
	// derived, via utf8) — never from the final byte.
	_, lastSize := utf8.DecodeLastRuneInString(key)
	lastIdx := len(key) - lastSize
	for i := 0; i < len(key); {
		r, size := utf8.DecodeRuneInString(key[i:])
		atEdge := i == 0 || i == lastIdx
		switch {
		case r == '\\':
			b.WriteString("\\\\")
		case r == '\n':
			b.WriteString("\\n")
		case r == '\r':
			b.WriteString("\\r")
		case isKtavWhitespaceR(r):
			if atEdge {
				pushUnicodeEscape(b, r)
			} else {
				b.WriteRune(r)
			}
		case r == '.':
			b.WriteString("\\.")
		case r == ':':
			b.WriteString("\\:")
		case r == ',':
			b.WriteString("\\,")
		case r == '{':
			b.WriteString("\\{")
		case r == '}':
			b.WriteString("\\}")
		case r == '[':
			b.WriteString("\\[")
		case r == ']':
			b.WriteString("\\]")
		case r < 0x20 || r == 0x7F:
			pushUnicodeEscape(b, r)
		default:
			b.WriteRune(r)
		}
		i += size
	}
}

func pushQuotedKey(key string, b *strings.Builder) {
	b.WriteByte('"')
	for i := 0; i < len(key); {
		r, size := utf8.DecodeRuneInString(key[i:])
		switch {
		case r == '"':
			b.WriteString("\\\"")
		case r == '\\':
			b.WriteString("\\\\")
		case r == '\n':
			b.WriteString("\\n")
		case r == '\r':
			b.WriteString("\\r")
		case (r < 0x20 && r != '\t' && r != 0x0B && r != 0x0C) || r == 0x7F:
			pushUnicodeEscape(b, r)
		default:
			b.WriteRune(r)
		}
		i += size
	}
	b.WriteByte('"')
}

// leafFault reports a leaf-level unrepresentable reason, if any.
func leafFault(v *pvalue) string {
	switch v.kind {
	case kObj, kArr:
		return ""
	case kFloat:
		if f, err := strconv.ParseFloat(v.s, 64); err == nil && isNonFiniteF(f) {
			return "NonFiniteFloat"
		}
	case kStr:
		if strings.IndexByte(v.s, '\r') >= 0 {
			return "CRByte"
		}
		if stringNeedsMultiline(v.s) {
			if _, r := chooseMultilineForm(v.s, false); r != "" {
				return r
			}
		}
	}
	return ""
}
