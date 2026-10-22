package ktav

// Lexical primitives for the pure backend: the § 3.3 whitespace class,
// the § 3.7 escape table, § 5.3.3 quoted-segment scanning, the § 4
// separator/dotted-path scanners, key validation, and the shared
// quote-aware/escape-aware inline-compound scanner. Single pass,
// byte-cursor based; no regex, no whole-input splits.

import "unicode/utf8"

// ─── § 3.3 whitespace ─────────────────────────────────────────────────

// isKtavWhitespace: the full § 3.3 set (25 code points), never a host
// whitespace primitive.
func isKtavWhitespaceR(r rune) bool {
	switch {
	case r < 0x80:
		return r == 0x09 || r == 0x0A || r == 0x0B || r == 0x0C || r == 0x0D || r == 0x20
	case r == 0x85 || r == 0xA0 || r == 0x1680 || (r >= 0x2000 && r <= 0x200A) ||
		r == 0x2028 || r == 0x2029 || r == 0x202F || r == 0x205F || r == 0x3000:
		return true
	}
	return false
}

// isInlineWhitespaceR: § 3.3 minus LF and CR (the line-bounded view).
func isInlineWhitespaceR(r rune) bool {
	return isKtavWhitespaceR(r) && r != 0x0A && r != 0x0D
}

// inlineWSByte: ASCII byte fast path of the inline view.
func inlineWSByte(b byte) bool { return b == ' ' || b == '\t' || b == 0x0B || b == 0x0C }

// inlineWSAt returns the byte length of the § 3.3 inline whitespace
// code point starting at s[i], or 0. A mid-code-point offset is never
// whitespace.
func inlineWSAt(s string, i int) int {
	if i >= len(s) {
		return 0
	}
	b := s[i]
	if inlineWSByte(b) {
		return 1
	}
	if b < 0x80 {
		return 0
	}
	r, size := utf8.DecodeRuneInString(s[i:])
	if r == utf8.RuneError && size <= 1 {
		return 0
	}
	if isInlineWhitespaceR(r) {
		return size
	}
	return 0
}

// trimInline returns s with leading/trailing § 3.3 inline whitespace
// removed (line-bounded view; lines are pre-split on LF/CR).
func trimInline(s string) string {
	start := 0
	for start < len(s) {
		n := inlineWSAt(s, start)
		if n == 0 {
			break
		}
		start += n
	}
	end := len(s)
	for end > start {
		// Step back one rune.
		r, size := utf8.DecodeLastRuneInString(s[start:end])
		_ = r
		if inlineWSAt(s, end-size) == 0 {
			break
		}
		end -= size
	}
	return s[start:end]
}

// ─── § 3.7 escapes ────────────────────────────────────────────────────

// scanEscape decodes one escape sequence at bytes[i] (which must be
// '\'). Returns the decoded string, the sequence's byte length, or the
// offending sequence text for BadEscapeSequence.
func scanEscape(bytes []byte, i int) (decoded string, length int, bad string) {
	if i+1 >= len(bytes) {
		return "", 0, "\\<end-of-line>"
	}
	next := bytes[i+1]
	switch next {
	case '\\':
		return "\\", 2, ""
	case ',':
		return ",", 2, ""
	case '}':
		return "}", 2, ""
	case ']':
		return "]", 2, ""
	case '{':
		return "{", 2, ""
	case '[':
		return "[", 2, ""
	case 'n':
		return "\n", 2, ""
	case 'r':
		return "\r", 2, ""
	case '.':
		return ".", 2, ""
	case ':':
		return ":", 2, ""
	case '"':
		return "\"", 2, ""
	case '\'':
		return "'", 2, ""
	case '`':
		return "`", 2, ""
	case 'u':
		if i+6 > len(bytes) || !allHexDigits(bytes[i+2:i+6]) {
			return "", 0, malformedUnicodeEscape(bytes, i)
		}
		value := hexVal4(bytes[i+2 : i+6])
		if value >= 0xD800 && value <= 0xDBFF {
			// High surrogate: must pair with an immediately following
			// low-surrogate `\uXXXX` (12 bytes total).
			if i+12 <= len(bytes) && bytes[i+6] == '\\' && bytes[i+7] == 'u' &&
				allHexDigits(bytes[i+8:i+12]) {
				low := hexVal4(bytes[i+8 : i+12])
				if low >= 0xDC00 && low <= 0xDFFF {
					combined := 0x10000 + (value-0xD800)*0x400 + (low - 0xDC00)
					return string(rune(combined)), 12, ""
				}
			}
			return "", 0, malformedUnicodeEscape(bytes, i)
		}
		if value >= 0xDC00 && value <= 0xDFFF {
			// Lone low surrogate.
			return "", 0, malformedUnicodeEscape(bytes, i)
		}
		return string(rune(value)), 6, ""
	default:
		if next < 0x80 {
			return "", 0, "\\" + string(rune(next))
		}
		return "", 0, "\\<0x" + hexByte(next) + ">"
	}
}

func allHexDigits(b []byte) bool {
	for _, c := range b {
		if !isASCIIHexDigit(c) {
			return false
		}
	}
	return len(b) > 0
}

func isASCIIHexDigit(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

func hexVal4(b []byte) uint32 {
	var v uint32
	for _, c := range b {
		v = v*16 + uint32(hexDigitVal(c))
	}
	return v
}

func hexDigitVal(c byte) uint32 {
	switch {
	case c >= '0' && c <= '9':
		return uint32(c - '0')
	case c >= 'a' && c <= 'f':
		return uint32(c-'a') + 10
	default:
		return uint32(c-'A') + 10
	}
}

func hexByte(b byte) string {
	const hex = "0123456789ABCDEF"
	return string([]byte{hex[b>>4], hex[b&0xF]})
}

func malformedUnicodeEscape(bytes []byte, i int) string {
	end := i + 6
	if end > len(bytes) {
		end = len(bytes)
	}
	seq := "\\u"
	for _, b := range bytes[i+2 : end] {
		if b < 0x80 {
			seq += string(rune(b))
		} else {
			seq += "<0x" + hexByte(b) + ">"
		}
	}
	return seq
}

// processEscapes applies the § 3.7 table to s. Returns the decoded
// text and whether at least one recognised escape was decoded (escape
// provenance forces String classification per § 5.2).
func processEscapes(s string) (out string, escaped bool, bad string) {
	if indexByteString(s, '\\') < 0 {
		return s, false, ""
	}
	bytes := []byte(s)
	buf := make([]byte, 0, len(bytes))
	i := 0
	for i < len(bytes) {
		if bytes[i] == '\\' {
			dec, n, badSeq := scanEscape(bytes, i)
			if badSeq != "" {
				return "", false, badSeq
			}
			buf = append(buf, dec...)
			i += n
			continue
		}
		_, size := utf8.DecodeRune(bytes[i:])
		if size == 0 {
			size = 1
		}
		buf = append(buf, bytes[i:i+size]...)
		i += size
	}
	return string(buf), true, ""
}

func indexByteString(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

// ─── quoted segments (§ 5.3.3) ────────────────────────────────────────

func isQuoteByte(b byte) bool { return b == '"' || b == '\'' || b == '`' }

// quotedSpanEnd returns the index of the segment's own unescaped closer
// for a segment opened at bytes[open], or ok=false when unterminated.
func quotedSpanEnd(bytes []byte, open int) (int, bool) {
	quote := bytes[open]
	j := open + 1
	for j < len(bytes) {
		if bytes[j] == '\\' {
			j += 2
			continue
		}
		if bytes[j] == quote {
			return j, true
		}
		j++
	}
	return 0, false
}

// ─── separator scan (§ 4) ─────────────────────────────────────────────

// Colon scan outcomes for the pair separator.
const (
	colonFound = iota
	colonAbsent
	colonUnterminatedQuote
)

// scanUnescapedColon finds the first unescaped `:` outside quoted
// segments (quote only opens a segment at a segment-start position:
// the very start or after an unescaped dot, plus surrounding inline
// whitespace).
func scanUnescapedColon(s string) int {
	i := 0
	segStart := true
	for i < len(s) {
		if segStart {
			for {
				n := inlineWSAt(s, i)
				if n == 0 {
					break
				}
				i += n
			}
			if i >= len(s) {
				return colonAbsent
			}
			if isQuoteByte(s[i]) {
				end, ok := quotedSpanEnd([]byte(s), i)
				if !ok {
					return colonUnterminatedQuote
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
			return colonFound
		case '.':
			segStart = true
			i++
		default:
			i++
		}
		if i > len(s) {
			return colonAbsent
		}
	}
	return colonAbsent
}

// keyIsSingleSegment reports whether the key contains no unescaped `.`
// separator (quoted-segment aware).
func keyIsSingleSegment(s string) bool {
	bytes := []byte(s)
	if !hasQuoteBytes(bytes) {
		i := 0
		for i < len(bytes) {
			switch bytes[i] {
			case '\\':
				i += 2
			case '.':
				return false
			default:
				i++
			}
		}
		return true
	}
	i := 0
	segStart := true
	for i < len(bytes) {
		if segStart {
			for inlineWSAt(s, i) > 0 {
				i += inlineWSAt(s, i)
			}
			if i < len(bytes) && isQuoteByte(bytes[i]) {
				end, ok := quotedSpanEnd(bytes, i)
				if !ok {
					return true // unterminated: one defensive segment
				}
				i = end + 1
				segStart = false
				continue
			}
			segStart = false
		}
		switch bytes[i] {
		case '\\':
			i += 2
		case '.':
			return false
		default:
			i++
		}
		if i > len(bytes) {
			break
		}
	}
	return true
}

func hasQuoteBytes(bytes []byte) bool {
	for _, b := range bytes {
		if isQuoteByte(b) {
			return true
		}
	}
	return false
}

// splitKeyPath splits a key on unescaped `.` bytes (quoted segments
// are opaque to the split).
func splitKeyPath(s string) []string {
	bytes := []byte(s)
	out := []string{}
	if !hasQuoteBytes(bytes) {
		start, i := 0, 0
		for i < len(bytes) {
			switch bytes[i] {
			case '\\':
				if i+1 < len(bytes) {
					i += 2
				} else {
					i++
				}
			case '.':
				out = append(out, s[start:i])
				start = i + 1
				i++
			default:
				i++
			}
		}
		out = append(out, s[start:])
		return out
	}
	start, i := 0, 0
	segStart := true
	for i < len(bytes) {
		if segStart {
			for inlineWSAt(s, i) > 0 {
				i += inlineWSAt(s, i)
			}
			if i < len(bytes) && isQuoteByte(bytes[i]) {
				end, ok := quotedSpanEnd(bytes, i)
				if !ok {
					break // defensive: remainder stays one segment
				}
				i = end + 1
				segStart = false
				continue
			}
			segStart = false
		}
		switch bytes[i] {
		case '\\':
			i += 2
		case '.':
			out = append(out, s[start:i])
			start = i + 1
			i++
			segStart = true
		default:
			i++
		}
		if i > len(bytes) {
			break
		}
	}
	out = append(out, s[start:])
	return out
}

// decodeKeySegment decodes one already-validated, already-trimmed key
// segment: a quoted segment's delimiters are stripped and only the
// interior is escape-processed (never trimmed).
func decodeKeySegment(input string) (string, string) {
	// Second return is the BadEscapeSequence payload ("" = ok).
	if len(input) > 0 && isQuoteByte(input[0]) {
		interior := input[1 : len(input)-1]
		if indexByteString(interior, '\\') < 0 {
			return interior, ""
		}
		out, _, bad := processEscapes(interior)
		return out, bad
	}
	if indexByteString(input, '\\') < 0 {
		return input, ""
	}
	out, _, bad := processEscapes(input)
	return out, bad
}

// ─── key validation (§ 4, § 5.3.1) ────────────────────────────────────

func isForbiddenRawControlByte(b byte) bool {
	return (b < 0x20 && b != '\t' && b != 0x0B && b != 0x0C) || b == 0x7F
}

// escapedLen measures the byte length of the escape at bytes[i]
// (validity is decode's job): 2 named, 6 `\uXXXX`, 12 surrogate pair,
// 1 dangling backslash, 2 fallback.
func escapedLen(bytes []byte, i int) int {
	if i+1 >= len(bytes) {
		return 1
	}
	if bytes[i+1] != 'u' || i+6 > len(bytes) {
		return 2
	}
	hex := bytes[i+2 : i+6]
	if !allHexDigits(hex) {
		return 2
	}
	value := hexVal4(hex)
	if value < 0xD800 || value > 0xDBFF {
		return 6
	}
	if i+12 > len(bytes) || bytes[i+6] != '\\' || bytes[i+7] != 'u' {
		return 6
	}
	lowHex := bytes[i+8 : i+12]
	if !allHexDigits(lowHex) {
		return 6
	}
	low := hexVal4(lowHex)
	if low >= 0xDC00 && low <= 0xDFFF {
		return 12
	}
	return 6
}

// keyValidity outcomes.
const (
	keyValid = iota
	keyEmpty
	keyInvalid
)

// checkKey validates one raw, outer-trimmed key segment before
// decoding: positional quoted vs bare form (§ 5.3.1 / § 5.3.3).
func checkKey(raw string) int {
	if raw == "" {
		return keyEmpty
	}
	if isQuoteByte(raw[0]) {
		return checkQuotedKey(raw)
	}
	if isBareKeyValid(raw) {
		return keyValid
	}
	return keyInvalid
}

func isBareKeyValid(raw string) bool {
	if raw == "" {
		return false
	}
	bytes := []byte(raw)
	i := 0
	for i < len(bytes) {
		if bytes[i] == '\\' {
			i += escapedLen(bytes, i)
			continue
		}
		switch bytes[i] {
		case ',', '{', '}', '[', ']', '(', ')':
			return false
		}
		if isForbiddenRawControlByte(bytes[i]) {
			return false
		}
		i++
	}
	return true
}

func checkQuotedKey(raw string) int {
	bytes := []byte(raw)
	closer, ok := quotedSpanEnd(bytes, 0)
	if !ok {
		return keyInvalid // unterminated; normally diagnosed earlier
	}
	if closer != len(bytes)-1 {
		return keyInvalid // content after the closer (§ 6.4)
	}
	i := 1
	for i < closer {
		if bytes[i] == '\\' {
			i += escapedLen(bytes, i)
			continue
		}
		if isForbiddenRawControlByte(bytes[i]) {
			return keyInvalid
		}
		i++
	}
	if closer == 1 {
		return keyEmpty // `""` / `''` / `` `` `` — empty quoted content
	}
	return keyValid
}

// ─── shared inline-compound scanner (§ 5.8) ───────────────────────────

// scanStop reasons for the shared byte-at-a-time inline scanner.
const (
	stopCloser = iota // depth reached 0 at idx; stopByte is the closer
	stopEofAfterWsSkip
	stopUnterminatedQuote
	stopBadEscape
	stopExhausted
)

// scanCfg carries the per-entry-point policy knobs of the shared
// scanner (mirrors the Rust monomorphised configs; see that module's
// preserved-quirk inventory for the rationale of each knob).
type scanCfg struct {
	trackQuotes    bool
	useStack       bool
	openerJump     bool
	decrementAny   bool // both closer kinds decrement shared depth
	closerLiteral  bool // `}`/`]` are ordinary content (split only)
	validateEsc    bool // scan only: BadEscapeSequence precedence
	trackRaw       bool // `::` raw marker + prev byte tracking
	commaSplits    bool
	commaCtxScope  bool // comma key context from current scope vs body kind
	commaClearsRaw bool
	colonSetsVS    bool
	colonClearsVS  bool // split: non-key colon clears value_start
	restoreInKey   bool
	restoreSeg     bool
	restoreRaw     bool
	mismatchSegF   bool
	clearVsOnClose bool
}

// Configs: scan (scanInlineCloser) and split (splitTopLevel), each in a
// quote-free fast and quote-aware variant.
var (
	cfgScanFast = scanCfg{useStack: true, decrementAny: true, validateEsc: true, trackRaw: true,
		commaCtxScope: true, commaClearsRaw: true, colonSetsVS: true,
		restoreInKey: true, clearVsOnClose: true}
	cfgScanQ = scanCfg{trackQuotes: true, useStack: true, decrementAny: true, validateEsc: true,
		trackRaw: true, commaCtxScope: true, commaClearsRaw: true, colonSetsVS: true,
		restoreInKey: true, restoreSeg: true, restoreRaw: true, mismatchSegF: true, clearVsOnClose: true}
	cfgSplitFast = scanCfg{openerJump: true, closerLiteral: true, commaSplits: true,
		colonSetsVS: true, colonClearsVS: true}
	cfgSplitQ = scanCfg{trackQuotes: true, openerJump: true, closerLiteral: true,
		commaSplits: true, colonSetsVS: true, colonClearsVS: true}
)

type scopeFrame struct {
	kind          byte // opener byte
	savedInKey    bool
	savedSegStart bool
	savedRaw      bool
}

type scanner struct {
	cfg      scanCfg
	input    string
	bytes    []byte
	open     byte
	close    byte
	bodyObj  bool
	lineNum  int
	span     kspan
	i        int
	depth    int
	inKey    bool
	segStart bool
	valStart bool
	raw      bool
	prev     byte
	segAt    int
	stack    []scopeFrame
	segments []string
	pairs    [][2]int // recorded (opener, closer) bounds
	bad      *Error
	stopIdx  int
	stopByte byte
}

func newScanner(cfg scanCfg, input string, open, close byte, bodyObj bool, lineNum int, span kspan) *scanner {
	return &scanner{cfg: cfg, input: input, bytes: []byte(input), open: open, close: close,
		bodyObj: bodyObj, lineNum: lineNum, span: span, prev: open}
}

// run executes the shared loop. Mirrors the Rust Scanner byte loop.
func (sc *scanner) run() int {
	cfg := sc.cfg
	input, bytes, ln := sc.input, sc.bytes, len(sc.bytes)
	open, close := sc.open, sc.close
	i, depth := sc.i, sc.depth
	inKey, segStart, valStart, raw := sc.inKey, sc.segStart, sc.valStart, sc.raw
	prev := sc.prev
	segAt := sc.segAt
	var stack []scopeFrame
	var segments []string
	var pairs [][2]int

	for {
		if i >= ln {
			sc.writeBack(i, depth, inKey, segStart, valStart, raw, prev, segAt, stack, segments, pairs)
			return stopExhausted
		}

		// (1) quoted key segment start.
		if cfg.trackQuotes && inKey && segStart {
			for inlineWSAt(input, i) > 0 {
				i += inlineWSAt(input, i)
			}
			if i >= ln {
				sc.writeBack(i, depth, inKey, segStart, valStart, raw, prev, segAt, stack, segments, pairs)
				return stopEofAfterWsSkip
			}
			if isQuoteByte(bytes[i]) {
				end, ok := quotedSpanEnd(bytes, i)
				if !ok {
					sc.writeBack(i, depth, inKey, segStart, valStart, raw, prev, segAt, stack, segments, pairs)
					return stopUnterminatedQuote
				}
				if cfg.trackRaw {
					prev = bytes[end]
				}
				i = end + 1
				segStart = false
				continue
			}
			segStart = false
		}

		b := bytes[i]
		switch {
		case b == '\\':
			if cfg.validateEsc {
				_, n, badSeq := scanEscape(bytes, i)
				if badSeq != "" {
					sc.bad = errBadEscapeSequence(sc.lineNum, sc.span, badSeq)
					sc.writeBack(i, depth, inKey, segStart, valStart, raw, prev, segAt, stack, segments, pairs)
					return stopBadEscape
				}
				valStart = false
				if cfg.trackRaw {
					prev = '\\'
				}
				i += n
				continue
			}
			valStart = false
			i += 2
			continue

		case b == ':':
			if inKey {
				inKey = false
				if cfg.colonSetsVS {
					valStart = true
				}
			} else if cfg.trackRaw && prev == ':' && valStart {
				raw = true
			} else if cfg.trackRaw {
				if valStart && !(i+1 < ln && bytes[i+1] == ':') {
					valStart = false
				}
			} else if cfg.colonClearsVS {
				valStart = false
			}

		case b == ',':
			if cfg.commaSplits {
				segments = append(segments, input[segAt:i])
				segAt = i + 1
			}
			scopeObject := sc.bodyObj
			if cfg.commaCtxScope {
				if len(stack) > 0 {
					scopeObject = stack[len(stack)-1].kind == '{'
				}
			}
			inKey = scopeObject
			if cfg.trackQuotes {
				segStart = scopeObject
			}
			if cfg.commaClearsRaw {
				raw = false
			}
			valStart = !scopeObject

		case b == '.' && cfg.trackQuotes && inKey:
			segStart = true

		case b == '{' || b == '[':
			gatedOpen := valStart && !raw
			if !gatedOpen {
				valStart = false
			} else if cfg.openerJump {
				var o, c byte = b, '}'
				if b == '[' {
					c = ']'
				}
				subCfg := cfgScanFast
				if hasQuoteBytes(bytes[i:]) {
					subCfg = cfgScanQ
				}
				sub := newScanner(subCfg, input[i:], o, c, o == '{', sc.lineNum, sc.span)
				sub.i, sub.depth = 1, 1
				sub.inKey, sub.segStart, sub.valStart = o == '{', o == '{', o == '['
				stop := sub.run()
				if stop == stopCloser && sub.stopByte == c {
					i += sub.i + 1
					valStart = false
					if cfg.trackQuotes {
						segStart = false
					}
					continue
				}
				// No closer inside the body: literal byte.
				valStart = false
				i++
				continue
			} else {
				if cfg.decrementAny || b == open {
					depth++
				}
				if cfg.useStack {
					stack = append(stack, scopeFrame{kind: b, savedInKey: inKey,
						savedSegStart: segStart, savedRaw: raw})
				}
				valStart = b == '['
				inKey = b == '{'
				if cfg.trackQuotes {
					segStart = b == '{'
				}
			}

		case (b == '}' || b == ']') && !cfg.closerLiteral:
			if !cfg.decrementAny && b != close {
				if cfg.trackQuotes {
					// The closer's own matching opener kind (mirrors the
					// Rust scanner: `]` pops a `[` scope, `}` a `{` one).
					want := byte('{')
					if b == ']' {
						want = '['
					}
					if len(stack) > 0 && stack[len(stack)-1].kind == want {
						f := stack[len(stack)-1]
						stack = stack[:len(stack)-1]
						inKey = f.savedInKey
						if cfg.restoreSeg {
							segStart = f.savedSegStart
						} else {
							segStart = false
						}
					} else {
						segStart = false
					}
				}
			} else {
				raw = false
				depth--
				if depth == 0 {
					sc.stopIdx, sc.stopByte = i, b
					sc.writeBack(i, depth, inKey, segStart, valStart, raw, prev, segAt, stack, segments, pairs)
					return stopCloser
				}
				// Nested closer: pop only when it matches the most
				// recently opened compound kind — the closer's own
				// matching opener (`]`↔`[`, `}`↔`{`), so crossed closers
				// don't corrupt tracking (R4-F2).
				want := byte('{')
				if b == ']' {
					want = '['
				}
				if len(stack) > 0 && stack[len(stack)-1].kind == want {
					f := stack[len(stack)-1]
					stack = stack[:len(stack)-1]
					if cfg.restoreInKey {
						inKey = f.savedInKey
					}
					if cfg.trackQuotes {
						if cfg.restoreSeg {
							segStart = f.savedSegStart
						} else {
							segStart = false
						}
						if cfg.restoreRaw {
							raw = f.savedRaw
						}
					}
				} else if cfg.mismatchSegF && cfg.trackQuotes {
					segStart = false
				}
				if cfg.clearVsOnClose {
					valStart = false
				}
			}

		default:
			if n := inlineWSAt(input, i); n > 0 {
				if cfg.trackRaw {
					prev = bytes[i+n-1]
				}
				i += n
				continue
			}
			valStart = false
		}
		prev = b
		i++
	}
}

// writeBack persists the loop state (segments/pairs/stopIdx are the
// only outputs the callers consume besides the stop reason).
func (sc *scanner) writeBack(i, depth int, inKey, segStart, valStart, raw bool, prev byte,
	segAt int, stack []scopeFrame, segments []string, pairs [][2]int) {
	sc.i, sc.depth, sc.inKey, sc.segStart = i, depth, inKey, segStart
	sc.valStart, sc.raw, sc.prev, sc.segAt = valStart, raw, prev, segAt
	sc.stack, sc.segments, sc.pairs = stack, segments, pairs
}

// scanInlineCloser is findMatchingClose plus escape validation with
// BadEscapeSequence precedence. Returns (idx, stopCloser, nil),
// (0, stop*, badErr) or (0, other-stop, nil).
func scanInlineCloser(input string, open, close byte, lineNum int, span kspan) (int, int, *Error) {
	bytes := []byte(input)
	if len(bytes) == 0 || bytes[0] != open {
		return 0, stopExhausted, nil
	}
	cfg := cfgScanFast
	if hasQuoteBytes(bytes) {
		cfg = cfgScanQ
	}
	sc := newScanner(cfg, input, open, close, open == '{', lineNum, span)
	sc.i, sc.depth = 1, 1
	sc.inKey, sc.segStart, sc.valStart = open == '{', open == '{', open == '['
	stop := sc.run()
	switch stop {
	case stopCloser:
		if sc.stopByte == close {
			return sc.stopIdx, stopCloser, nil
		}
		return 0, stopExhausted, nil // crossed closer: not a matching closer
	case stopBadEscape:
		return 0, stopBadEscape, sc.bad
	default:
		return 0, stop, nil
	}
}

// splitTopLevel splits an inline body on unescaped top-level commas.
func splitTopLevel(input string, lineNum int, span kspan, bodyObject bool) ([]string, *Error) {
	open, close := byte('{'), byte('}')
	if !bodyObject {
		open, close = '[', ']'
	}
	cfg := cfgSplitFast
	if bodyObject && hasQuoteBytes([]byte(input)) {
		cfg = cfgSplitQ
	}
	sc := newScanner(cfg, input, open, close, bodyObject, lineNum, span)
	sc.inKey, sc.segStart, sc.valStart = bodyObject, bodyObject, !bodyObject
	stop := sc.run()
	switch stop {
	case stopUnterminatedQuote:
		return nil, errUnterminatedInlineCompound(lineNum, span)
	case stopEofAfterWsSkip:
		if trimInline(input[sc.segAt:]) == "" {
			return sc.segments, nil // valid trailing comma
		}
		return append(sc.segments, input[sc.segAt:]), nil
	case stopExhausted:
		return append(sc.segments, input[sc.segAt:]), nil
	default:
		return sc.segments, nil // unreachable for split configs
	}
}

// malformedCloserNotAtEnd is § 5.2 rule 8's diagnostic.
func malformedCloserNotAtEnd(lineNum int, span kspan) *Error {
	return errMalformedInlineCompound(lineNum, span,
		"matching closer is not the last byte of the body; non-whitespace content follows the closed inline compound")
}
