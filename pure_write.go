package ktav

// The pure writers: § 5.9.10 key emission, § 5.9.7 string-form
// selection, the § 5.9 canonical/pretty writers and the trivia
// formatter — all ITERATIVE (explicit work stacks). Output is
// assembled in a local buffer first, so a rejection never emits
// partial output (§ 5.9.0).

import (
	"strconv"
	"strings"
	"unicode/utf8"
)

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
	probe("rawmarker_scan", len(s))
	if inlineWSAt(s, 0) > 0 {
		t := trimLeadingInline(s)
		return t != "" && (t[0] == '{' || t[0] == '[' || needsRawMarkerContent(t))
	}
	if s[0] == '{' || s[0] == '[' {
		return true
	}
	return needsRawMarkerContent(s)
}

func needsRawMarkerContent(s string) bool {
	probe("numeric_scan", 1)
	switch s {
	case "null", "true", "false", "(", "((", "()", "(())":
		return true
	}
	return matchesIntegerGrammar(s) || isFloatLiteral(s)
}

// matchesIntegerGrammar is the § 3.6 integer grammar syntactically
// (value may overflow i64; lowercase base prefixes only).
func matchesIntegerGrammar(s string) bool {
	b := []byte(s)
	i := 0
	if i < len(b) && (b[i] == '+' || b[i] == '-') {
		i++
	}
	if i >= len(b) {
		return false
	}
	if b[i] == '0' && i+1 < len(b) {
		switch b[i+1] {
		case 'x':
			return checkPrefixedDigits(b[i+2:], 16)
		case 'o':
			return checkPrefixedDigits(b[i+2:], 8)
		case 'b':
			return checkPrefixedDigits(b[i+2:], 2)
		}
	}
	return checkDecimalDigits(b[i:])
}

func checkPrefixedDigits(digits []byte, radix int) bool {
	if len(digits) == 0 || !isDigitForRadix(digits[0], radix) {
		return false
	}
	prevUnderscore := false
	for _, c := range digits[1:] {
		if c == '_' {
			if prevUnderscore {
				return false
			}
			prevUnderscore = true
			continue
		}
		prevUnderscore = false
		if !isDigitForRadix(c, radix) {
			return false
		}
	}
	return !prevUnderscore
}

func checkDecimalDigits(digits []byte) bool {
	if len(digits) == 0 || digits[0] < '0' || digits[0] > '9' {
		return false
	}
	prevUnderscore := false
	for _, c := range digits[1:] {
		if c == '_' {
			if prevUnderscore {
				return false
			}
			prevUnderscore = true
			continue
		}
		prevUnderscore = false
		if c < '0' || c > '9' {
			return false
		}
	}
	return !prevUnderscore
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

// ─── shared string emission primitives ────────────────────────────────

// emitStringBody renders a one-line string body with the pair
// separator; returns a § 5.9.0/§ 5.9.7 reason on rejection.
func emitStringInPairTo(b *strings.Builder, s string, indent int, preferStripped bool) string {
	if strings.IndexByte(s, '\r') >= 0 {
		return "CRByte"
	}
	if stringNeedsMultiline(s) {
		return emitMultilineTo(b, s, indent, true, preferStripped)
	}
	if needsRawMarker(s) {
		b.WriteString(":: ")
	} else {
		b.WriteString(": ")
	}
	b.WriteString(s)
	b.WriteByte('\n')
	return ""
}

func emitStringAsItemTo(b *strings.Builder, s string, indent int, isRootArrayFirst, preferStripped bool) string {
	if strings.IndexByte(s, '\r') >= 0 {
		return "CRByte"
	}
	if stringNeedsMultiline(s) {
		return emitMultilineTo(b, s, indent, false, preferStripped)
	}
	if itemNeedsRawMarker(s) ||
		(isRootArrayFirst && (isPairShape(s) || strings.HasPrefix(s, "\uFEFF"))) {
		b.WriteString(":: ")
	}
	b.WriteString(s)
	b.WriteByte('\n')
	return ""
}

// emitMultilineTo renders a multi-line string body; preferStripped is
// the pretty writers' readable default, canonical prefers verbatim.
func emitMultilineTo(b *strings.Builder, s string, indent int, isPair, preferStripped bool) string {
	stripped, reason := chooseMultilineForm(s, preferStripped)
	if reason != "" {
		return reason
	}
	if isPair {
		if stripped {
			b.WriteString(": (\n")
		} else {
			b.WriteString(": ((\n")
		}
	} else {
		if stripped {
			b.WriteString("(\n")
		} else {
			b.WriteString("((\n")
		}
	}
	if stripped && preferStripped {
		// Pretty stripped form: content lines at indent+1 so the
		// parser's common-indent dedent removes exactly the
		// writer-added prefix.
		for _, line := range splitLF(s) {
			if line != "" {
				pushIndent(b, indent+1)
				b.WriteString(line)
			}
			b.WriteByte('\n')
		}
	} else {
		// Verbatim and canonical stripped: content at indent 0.
		b.WriteString(s)
		b.WriteByte('\n')
	}
	pushIndent(b, indent)
	if stripped {
		b.WriteString(")\n")
	} else {
		b.WriteString("))\n")
	}
	return ""
}

// ─── pretty / canonical writer over the plain tree ────────────────────

type writerMode int

const (
	modePretty    writerMode = iota // Dumps / DumpsForceStrings
	modeCanonical                   // EmitCanonical / CanonicalFromSource
)

type writer struct {
	mode         writerMode
	forceStrings bool
	b            strings.Builder
	root         *pvalue
	reason       string // first § 5.9.0/§ 5.9.7 rejection, if any
}

func (w *writer) fail(reason string) {
	if w.reason == "" {
		w.reason = reason
	}
}

// renderRoot renders a root value (must be Object or Array, § 5.9.0).
// Emission is ITERATIVE — an explicit LIFO work stack replaces the
// recursive pair/item walk, so unbounded-depth trees (deep dotted
// keys, § 5.3.2; uncapped by spec and reference alike) can never
// overflow the goroutine stack. Each task is a closer line or one
// node; compounds expand into children plus a closer.
func (w *writer) renderRoot(v *pvalue) *Error {
	w.root = v
	if v.kind != kObj && v.kind != kArr {
		w.fail("ScalarRoot")
		return w.err()
	}
	type wtask struct {
		v            *pvalue
		key          string
		indent       int
		isItem       bool   // array-item emission (vs pair value)
		rootFirstKey bool   // § 5.9.10 rule (c) U+FEFF guard
		rootArrFirst bool   // § 5.9.3 first root-array-item safeguard
		closer       string // non-empty: emit this indent-prefixed line
	}
	stack := []wtask{}
	push := func(t wtask) {
		if w.reason == "" {
			stack = append(stack, t)
		}
	}
	// expandObj schedules an object's pairs (reversed for LIFO order).
	expandObj := func(m *omap, indent int, isRoot bool) {
		for i := len(m.entries) - 1; i >= 0; i-- {
			e := m.entries[i]
			push(wtask{v: e.val, key: e.key, indent: indent, rootFirstKey: isRoot && i == 0})
		}
	}
	expandArr := func(items []*pvalue, indent int, rootSafeguard bool) {
		for i := len(items) - 1; i >= 0; i-- {
			push(wtask{v: items[i], indent: indent, isItem: true, rootArrFirst: rootSafeguard && i == 0})
		}
	}

	switch {
	case v.kind == kObj:
		expandObj(v.obj, 0, true)
	case len(v.arr) == 0:
		w.b.WriteString("[]\n") // § 5.9.3 empty Array root
	case v.arr[0].kind == kObj || v.arr[0].kind == kArr:
		// § 5.9.3: first item is a compound — wrap the root array.
		w.b.WriteString("[\n")
		expandArr(v.arr, 1, false)
		push(wtask{closer: "]\n"})
	default:
		expandArr(v.arr, 0, true)
	}

	for len(stack) > 0 && w.reason == "" {
		t := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		probe("write_task", 1)
		if t.closer != "" {
			pushIndent(&w.b, t.indent)
			w.b.WriteString(t.closer)
			continue
		}
		if t.isItem {
			// ── item position (§ 5.9.6) ──
			pushIndent(&w.b, t.indent)
			if w.forceStrings {
				if text, ok := forcedStringText(t.v); ok {
					w.fail(emitStringAsItemTo(&w.b, text, t.indent, t.rootArrFirst, w.mode == modePretty))
					continue
				}
			}
			switch t.v.kind {
			case kNull:
				w.b.WriteString("null\n")
			case kBool:
				if t.v.b {
					w.b.WriteString("true\n")
				} else {
					w.b.WriteString("false\n")
				}
			case kInt:
				w.b.WriteString(t.v.s)
				w.b.WriteByte('\n')
			case kFloat:
				text, st := canonicalFloatText(t.v.s)
				if st == cfNonFinite {
					w.fail("NonFiniteFloat")
				} else if w.mode == modePretty {
					w.b.WriteString(t.v.s)
					w.b.WriteByte('\n')
				} else {
					w.b.WriteString(text)
					w.b.WriteByte('\n')
				}
			case kStr:
				w.fail(emitStringAsItemTo(&w.b, t.v.s, t.indent, t.rootArrFirst, w.mode == modePretty))
			case kArr:
				if len(t.v.arr) == 0 {
					w.b.WriteString("[]\n")
				} else {
					w.b.WriteString("[\n")
					expandArr(t.v.arr, t.indent+1, false)
					push(wtask{closer: "]\n", indent: t.indent})
				}
			case kObj:
				if len(t.v.obj.entries) == 0 {
					w.b.WriteString("{}\n")
				} else {
					w.b.WriteString("{\n")
					expandObj(t.v.obj, t.indent+1, false)
					push(wtask{closer: "}\n", indent: t.indent})
				}
			}
			continue
		}
		// ── pair position (§ 5.9.2) ──
		key, v := t.key, t.v
		if key == "" {
			w.fail("EmptyKeyName")
			continue
		}
		pushIndent(&w.b, t.indent)
		pushEscapedKeySegment(key, t.rootFirstKey, &w.b)
		if w.forceStrings {
			if text, ok := forcedStringText(v); ok {
				w.fail(emitStringInPairTo(&w.b, text, t.indent, w.mode == modePretty))
				continue
			}
		}
		switch v.kind {
		case kNull:
			w.b.WriteString(": null\n")
		case kBool:
			if v.b {
				w.b.WriteString(": true\n")
			} else {
				w.b.WriteString(": false\n")
			}
		case kInt:
			w.b.WriteString(": ")
			w.b.WriteString(v.s)
			w.b.WriteByte('\n')
		case kFloat:
			text, st := canonicalFloatText(v.s)
			if st == cfNonFinite {
				w.fail("NonFiniteFloat")
			} else if w.mode == modePretty {
				w.b.WriteString(": ")
				w.b.WriteString(v.s)
				w.b.WriteByte('\n')
			} else {
				w.b.WriteString(": ")
				w.b.WriteString(text)
				w.b.WriteByte('\n')
			}
		case kStr:
			w.fail(emitStringInPairTo(&w.b, v.s, t.indent, w.mode == modePretty))
		case kArr:
			if len(v.arr) == 0 {
				w.b.WriteString(": []\n")
			} else {
				w.b.WriteString(": [\n")
				expandArr(v.arr, t.indent+1, false)
				push(wtask{closer: "]\n", indent: t.indent})
			}
		case kObj:
			if len(v.obj.entries) == 0 {
				w.b.WriteString(": {}\n")
			} else {
				w.b.WriteString(": {\n")
				expandObj(v.obj, t.indent+1, false)
				push(wtask{closer: "}\n", indent: t.indent})
			}
		}
	}
	return w.err()
}

func (w *writer) err() *Error {
	if w.reason == "" {
		return nil
	}
	return attachPath(w.root, w.reason)
}

// forcedStringText is the ForceStrings coercion: every leaf scalar
// takes its textual form; compounds keep their structure.
func forcedStringText(v *pvalue) (string, bool) {
	switch v.kind {
	case kNull:
		return "null", true
	case kBool:
		if v.b {
			return "true", true
		}
		return "false", true
	case kInt, kFloat, kStr:
		return v.s, true
	}
	return "", false
}

// ─── formatter emission (canonical structure + trivia) ────────────────

type fmtWriter struct {
	b      strings.Builder
	indent int
	rootF  *fvalue
	lead   []tline
	reason string
}

func (fw *fmtWriter) fail(reason string) {
	if fw.reason == "" {
		fw.reason = reason
	}
}

func (fw *fmtWriter) err() *Error {
	if fw.reason == "" {
		return nil
	}
	return attachPath(fw.rootF.toPlain(), fw.reason)
}

func (fw *fmtWriter) emitTrivia(t []tline, indent int) {
	for _, line := range t {
		if line.blank {
			fw.b.WriteByte('\n')
		} else {
			pushIndent(&fw.b, indent)
			fw.b.WriteString(line.comment)
			fw.b.WriteByte('\n')
		}
	}
}

func (fw *fmtWriter) emitDoc(root *fvalue, leading []tline) *Error {
	fw.rootF = root
	fw.lead = leading
	fw.emitTrivia(leading, 0)

	// Iterative emission (depth safety): an explicit LIFO work stack
	// replaces the recursive object/array walk, so deep dotted-key
	// nesting (§ 5.3.2, uncapped) can never overflow the stack. Trivia
	// slots ride on the tasks so attachment points are unchanged.
	const (
		ftPair = iota // key+f: emit slot trivia, key line, value
		ftItem        // f: emit item trivia, item line
		ftCloser      // closer line, preceded by pre-trivia at indent+1
		ftTrivia      // bare trivia run
	)
	type fmtask struct {
		kind         int
		key          string
		f            *fvalue
		indent       int
		rootFirstKey bool
		rootArrFirst bool
		closer       string
		trivia       []tline // ftPair/ftItem: slot trivia; ftCloser: pre-trivia
	}
	stack := []fmtask{}
	push := func(t fmtask) {
		if fw.reason == "" {
			stack = append(stack, t)
		}
	}
	expandObj := func(f *fvalue, indent int, isRoot bool) {
		for i := len(f.comp.pairs) - 1; i >= 0; i-- {
			pr := f.comp.pairs[i]
			push(fmtask{kind: ftPair, key: pr.key, f: pr.val, indent: indent,
				trivia: pr.trivia, rootFirstKey: isRoot && i == 0})
		}
	}
	expandItems := func(f *fvalue, indent int) {
		for i := len(f.comp.items) - 1; i >= 0; i-- {
			it := f.comp.items[i]
			push(fmtask{kind: ftItem, f: it.val, indent: indent, trivia: it.trivia})
		}
	}

	switch root.v.kind {
	case kObj:
		expandObj(root, 0, true)
		// Trailing whole-line trivia after the root object's last pair
		// must survive; otherwise FormatSource is not a fixed point.
		push(fmtask{kind: ftTrivia, trivia: root.comp.trailing})
	case kArr:
		if len(root.comp.items) == 0 {
			if len(root.comp.trailing) == 0 {
				// Root empty array: § 5.9.3 pins `[]` + LF (an empty
				// root OBJECT emits zero bytes, but an empty array
				// must stay an array on re-parse).
				fw.b.WriteString("[]\n")
			} else {
				// Trailing trivia only: wrap so the comments survive
				// the round trip without inventing items.
				fw.b.WriteString("[\n")
				fw.emitTrivia(root.comp.trailing, 1)
				fw.b.WriteString("]\n")
			}
			break
		}
		if root.comp.items[0].val.v.kind == kObj || root.comp.items[0].val.v.kind == kArr {
			fw.b.WriteString("[\n")
			expandItems(root, 1)
			push(fmtask{kind: ftCloser, closer: "]\n"})
			break
		}
		for i := len(root.comp.items) - 1; i >= 0; i-- {
			it := root.comp.items[i]
			push(fmtask{kind: ftItem, f: it.val, trivia: it.trivia, rootArrFirst: i == 0})
		}
		push(fmtask{kind: ftTrivia, trivia: root.comp.trailing})
	default:
		fw.fail("ScalarRoot")
	}

	for len(stack) > 0 && fw.reason == "" {
		t := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		probe("write_task", 1)
		switch t.kind {
		case ftTrivia:
			fw.emitTrivia(t.trivia, t.indent)
		case ftCloser:
			fw.emitTrivia(t.trivia, t.indent+1)
			pushIndent(&fw.b, t.indent)
			fw.b.WriteString(t.closer)
		case ftItem:
			fw.emitTrivia(t.trivia, t.indent)
			f := t.f
			pushIndent(&fw.b, t.indent)
			switch f.v.kind {
			case kNull:
				fw.b.WriteString("null\n")
			case kBool:
				if f.v.b {
					fw.b.WriteString("true\n")
				} else {
					fw.b.WriteString("false\n")
				}
			case kInt:
				fw.b.WriteString(f.v.s)
				fw.b.WriteByte('\n')
			case kFloat:
				text, st := canonicalFloatText(f.v.s)
				if st == cfNonFinite {
					fw.fail("NonFiniteFloat")
				} else {
					fw.b.WriteString(text)
					fw.b.WriteByte('\n')
				}
			case kStr:
				if strings.IndexByte(f.v.s, '\r') >= 0 {
					fw.fail("CRByte")
				} else if f.v.s == "" {
					fw.b.WriteString("::\n")
				} else if stringNeedsMultiline(f.v.s) {
					fw.fail(emitMultilineTo(&fw.b, f.v.s, t.indent, false, false))
				} else {
					if itemNeedsRawMarker(f.v.s) ||
						(t.rootArrFirst && (isPairShape(f.v.s) || strings.HasPrefix(f.v.s, "\uFEFF"))) {
						fw.b.WriteString(":: ")
					}
					fw.b.WriteString(f.v.s)
					fw.b.WriteByte('\n')
				}
			case kArr:
				if len(f.comp.items) == 0 && len(f.comp.trailing) == 0 {
					fw.b.WriteString("[]\n")
				} else {
					fw.b.WriteString("[\n")
					expandItems(f, t.indent+1)
					push(fmtask{kind: ftCloser, closer: "]\n", indent: t.indent, trivia: f.comp.trailing})
				}
			case kObj:
				if len(f.comp.pairs) == 0 && len(f.comp.trailing) == 0 {
					fw.b.WriteString("{}\n")
				} else {
					fw.b.WriteString("{\n")
					expandObj(f, t.indent+1, false)
					push(fmtask{kind: ftCloser, closer: "}\n", indent: t.indent, trivia: f.comp.trailing})
				}
			}
		case ftPair:
			fw.emitTrivia(t.trivia, t.indent)
			key, f, indent := t.key, t.f, t.indent
			if key == "" {
				fw.fail("EmptyKeyName")
				continue
			}
			pushIndent(&fw.b, indent)
			pushEscapedKeySegment(key, t.rootFirstKey, &fw.b)
			switch f.v.kind {
			case kNull:
				fw.b.WriteString(": null\n")
			case kBool:
				if f.v.b {
					fw.b.WriteString(": true\n")
				} else {
					fw.b.WriteString(": false\n")
				}
			case kInt:
				fw.b.WriteString(": ")
				fw.b.WriteString(f.v.s)
				fw.b.WriteByte('\n')
			case kFloat:
				text, st := canonicalFloatText(f.v.s)
				if st == cfNonFinite {
					fw.fail("NonFiniteFloat")
				} else {
					fw.b.WriteString(": ")
					fw.b.WriteString(text)
					fw.b.WriteByte('\n')
				}
			case kStr:
				if strings.IndexByte(f.v.s, '\r') >= 0 {
					fw.fail("CRByte")
				} else if f.v.s == "" {
					fw.b.WriteString(":\n")
				} else if stringNeedsMultiline(f.v.s) {
					fw.fail(emitMultilineTo(&fw.b, f.v.s, indent, true, false))
				} else {
					if needsRawMarker(f.v.s) {
						fw.b.WriteString(":: ")
					}
					fw.b.WriteString(f.v.s)
					fw.b.WriteByte('\n')
				}
			case kArr:
				if len(f.comp.items) == 0 && len(f.comp.trailing) == 0 {
					fw.b.WriteString(": []\n")
				} else {
					fw.b.WriteString(": [\n")
					expandItems(f, indent+1)
					push(fmtask{kind: ftCloser, closer: "]\n", indent: indent, trivia: f.comp.trailing})
				}
			case kObj:
				if len(f.comp.pairs) == 0 && len(f.comp.trailing) == 0 {
					fw.b.WriteString(": {}\n")
				} else {
					fw.b.WriteString(": {\n")
					expandObj(f, indent+1, false)
					push(fmtask{kind: ftCloser, closer: "}\n", indent: indent, trivia: f.comp.trailing})
				}
			}
		}
	}
	return fw.err()
}

// ─── § 5.9.0 cold path: locate the offending node ─────────────────────

// attachPath turns a writer's path-less rejection into the error with
// the decoded key path to the offending pair (document order, § 5.9.0).
// The walk is iterative (explicit frame stack): it runs on arbitrary
// programmatic values and must be safe at unbounded depth.
func attachPath(v *pvalue, reason string) *Error {
	if v == nil || (v.kind != kObj && v.kind != kArr) {
		return unrepresentableError("ScalarRoot", nil)
	}
	type apframe struct {
		v        *pvalue
		arr      []*pvalue // array frames
		i        int
		isObj    bool
		pathBase int // len(path) at push time; pop restores it
	}
	var path []string
	stack := []apframe{{v: v, arr: v.arr, isObj: v.kind == kObj}}
	found := ""
	for len(stack) > 0 && found == "" {
		f := &stack[len(stack)-1]
		switch {
		case f.isObj:
			if f.i < len(f.v.obj.entries) {
				e := f.v.obj.entries[f.i]
				f.i++
				path = append(path, e.key)
				if e.key == "" {
					found = "EmptyKeyName"
					break
				}
				if r := leafFault(e.val); r != "" {
					found = r
					break
				}
				if e.val.kind == kObj || e.val.kind == kArr {
					stack = append(stack, apframe{v: e.val, arr: e.val.arr,
						isObj: e.val.kind == kObj, pathBase: len(path)})
				} else {
					path = path[:len(path)-1]
				}
			} else {
				path = path[:f.pathBase]
				stack = stack[:len(stack)-1]
			}
		default: // array frame
			if f.i < len(f.arr) {
				item := f.arr[f.i]
				f.i++
				if r := leafFault(item); r != "" {
					found = r
					break
				}
				if item.kind == kObj || item.kind == kArr {
					stack = append(stack, apframe{v: item, arr: item.arr,
						isObj: item.kind == kObj, pathBase: len(path)})
				}
			} else {
				path = path[:f.pathBase]
				stack = stack[:len(stack)-1]
			}
		}
	}
	if found != "" {
		return unrepresentableError(found, path)
	}
	return unrepresentableError(reason, nil)
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
