package ktav

// Number handling for the pure backend: the § 3.6 integer/float
// grammars, § 5.2 rules 13–14 classification (minimum i64 domain,
// binary64 floats), the shortest-round-trip float text (ryu-compatible
// output built from Go's shortest digits), and § 5.9.8 canonical float
// rendering.

import (
	"math"
	"strconv"
)

// ─── § 5.2 classification helpers ─────────────────────────────────────

// hasRedundantLeadingZero: a base-10 digit run whose first digit is `0`
// while at least one further digit follows (sign-aware, underscore-
// tolerant). Such a body is never a number (§ 5.2 rules 13–14).
func hasRedundantLeadingZero(s string) bool {
	b := []byte(s)
	start := 0
	if len(b) > 0 && (b[0] == '+' || b[0] == '-') {
		start = 1
	}
	if start >= len(b) || b[start] != '0' {
		return false
	}
	if start+1 < len(b) {
		n := b[start+1]
		return (n >= '0' && n <= '9') || n == '_'
	}
	return false
}

// tryParseInteger parses a § 3.6 integer literal (lowercase base
// prefixes only, underscores between digits only) into an i64. Returns
// ok=false when the grammar doesn't match or the value overflows i64.
func tryParseInteger(s string) (int64, bool) {
	b := []byte(s)
	if len(b) == 0 {
		return 0, false
	}
	i := 0
	negative := false
	if b[i] == '-' {
		i++
		negative = true
	} else if b[i] == '+' {
		i++
	}
	if i >= len(b) {
		return 0, false
	}
	if b[i] == '0' && i+1 < len(b) {
		switch b[i+1] {
		case 'x':
			return parsePrefixedInt(b[i+2:], 16, negative)
		case 'o':
			return parsePrefixedInt(b[i+2:], 8, negative)
		case 'b':
			return parsePrefixedInt(b[i+2:], 2, negative)
		}
	}
	return parseDecimalInt(b[i:], negative)
}

func digitValueFor(b byte, radix int) (uint64, bool) {
	var v uint64
	switch {
	case b >= '0' && b <= '9':
		v = uint64(b - '0')
	case b >= 'a' && b <= 'f':
		v = uint64(b-'a') + 10
	case b >= 'A' && b <= 'F':
		v = uint64(b-'A') + 10
	default:
		return 0, false
	}
	if v >= uint64(radix) {
		return 0, false
	}
	return v, true
}

func parsePrefixedInt(digits []byte, radix int, negative bool) (int64, bool) {
	if len(digits) == 0 || !isDigitForRadix(digits[0], radix) {
		return 0, false
	}
	var val uint64
	prevUnderscore := false
	for _, c := range digits {
		if c == '_' {
			if prevUnderscore {
				return 0, false
			}
			prevUnderscore = true
			continue
		}
		prevUnderscore = false
		d, ok := digitValueFor(c, radix)
		if !ok {
			return 0, false
		}
		val, ok = mulAddU64(val, d, uint64(radix))
		if !ok {
			return 0, false
		}
	}
	if prevUnderscore {
		return 0, false
	}
	return finishInt(val, negative)
}

func isDigitForRadix(b byte, radix int) bool {
	_, ok := digitValueFor(b, radix)
	return ok
}

func mulAddU64(val, d, radix uint64) (uint64, bool) {
	if val > (^uint64(0)-d)/radix {
		return 0, false
	}
	return val*radix + d, true
}

func parseDecimalInt(digits []byte, negative bool) (int64, bool) {
	if len(digits) == 0 || digits[0] < '0' || digits[0] > '9' {
		return 0, false
	}
	var val uint64
	prevUnderscore := false
	count := 0
	for _, c := range digits {
		if c == '_' {
			if prevUnderscore || count == 0 {
				return 0, false
			}
			prevUnderscore = true
			continue
		}
		if c < '0' || c > '9' {
			return 0, false
		}
		prevUnderscore = false
		var ok bool
		val, ok = mulAddU64(val, uint64(c-'0'), 10)
		if !ok {
			return 0, false
		}
		count++
	}
	if prevUnderscore || count == 0 {
		return 0, false
	}
	return finishInt(val, negative)
}

func finishInt(val uint64, negative bool) (int64, bool) {
	const maxU = uint64(1<<63 - 1)
	minMag := maxU + 1
	if negative {
		if val > minMag {
			return 0, false
		}
		if val == 0 {
			return 0, true
		}
		if val == minMag {
			return -1 << 63, true
		}
		return -int64(val), true
	}
	if val > maxU {
		return 0, false
	}
	return int64(val), true
}

// isFloatLiteral checks the § 3.6 float grammar syntactically:
// sign? dec_part "." dec_part exponent?  |  sign? dec_part exponent.
func isFloatLiteral(s string) bool {
	examined := 0
	return isFloatLiteralN(s, &examined)
}

// isFloatLiteralN is isFloatLiteral with a byte-examiner accumulator
// (probe instrumentation; see needsRawMarkerContent).
func isFloatLiteralN(s string, examined *int) bool {
	b := []byte(s)
	i := 0
	if i < len(b) && (b[i] == '+' || b[i] == '-') {
		i++
		*examined++
	}
	n, ok := scanDecPart(b, i)
	if !ok {
		*examined += i
		return false
	}
	*examined += n - i
	i = n
	if i < len(b) && b[i] == '.' {
		i++
		n, ok := scanDecPart(b, i)
		if !ok {
			*examined += i
			return false
		}
		*examined += n - i
		i = n
		if i < len(b) && (b[i] == 'e' || b[i] == 'E') {
			n, ok := scanExponent(b, i)
			if !ok {
				*examined += i
				return false
			}
			*examined += n - i
			i = n
		}
		return i == len(b)
	}
	if i < len(b) && (b[i] == 'e' || b[i] == 'E') {
		n, ok := scanExponent(b, i)
		if !ok {
			*examined += i
			return false
		}
		*examined += n - i
		i = n
		return i == len(b)
	}
	*examined += i
	return false
}

func scanDecPart(b []byte, i int) (int, bool) {
	if i >= len(b) || b[i] < '0' || b[i] > '9' {
		return i, false
	}
	i++
	prevUnderscore := false
	for i < len(b) {
		if b[i] == '_' {
			if prevUnderscore {
				return i, false
			}
			prevUnderscore = true
			i++
			continue
		}
		if b[i] >= '0' && b[i] <= '9' {
			prevUnderscore = false
			i++
			continue
		}
		break
	}
	if prevUnderscore {
		return i, false
	}
	return i, true
}

func scanExponent(b []byte, i int) (int, bool) {
	if i >= len(b) || (b[i] != 'e' && b[i] != 'E') {
		return i, false
	}
	i++
	if i < len(b) && (b[i] == '+' || b[i] == '-') {
		i++
	}
	return scanDecPart(b, i)
}

// parseFloatValue parses a float literal already validated by
// isFloatLiteral into a float64. ok=false on overflow/NaN (which send
// the body to rule 15 / String).
func parseFloatValue(s string) (float64, bool) {
	cleaned := s
	if indexByteString(s, '_') >= 0 {
		buf := make([]byte, 0, len(s))
		for i := 0; i < len(s); i++ {
			if s[i] != '_' {
				buf = append(buf, s[i])
			}
		}
		cleaned = string(buf)
	}
	f, err := strconv.ParseFloat(cleaned, 64)
	if err != nil {
		// Go reports ErrRange both for overflow (result ±Inf) and for
		// underflow (result ±0). Per § 5.2 rule 14 only the overflow
		// case leaves the declared Float domain (the literal falls
		// through to rule 15 / String); underflow to zero is finite,
		// so such a literal is an ordinary Float.
		rangeErr, ok := err.(*strconv.NumError)
		if !ok || rangeErr.Err != strconv.ErrRange || math.IsInf(f, 0) {
			return 0, false
		}
	}
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, false
	}
	return f, true
}

// ─── § 5.9.8 canonical float rendering ────────────────────────────────

// formatRyu renders f in the ryu crate's "pretty" shortest-round-trip
// form (the Rust reference's stored Float text), built from Go's
// shortest round-trip digits (strconv.FormatFloat(f, 'e', -1, 64) —
// Go's roundTiesToEven shortest form, a Ryu-class result):
//   - decimal notation when 0 < kk <= 16 (point inside/right after the
//     digits, trailing ".0" for whole values) or -5 < kk <= 0 ("0.00…d");
//   - scientific `d[.ddd]e±E` otherwise, where kk = E+1.
func formatRyu(f float64) string {
	if f == 0 {
		// `f == 0` is true for -0.0 too; the sign bit decides.
		if math.Signbit(f) {
			return "-0.0"
		}
		return "0.0"
	}
	neg := math.Signbit(f)
	mant := strconv.FormatFloat(f, 'e', -1, 64)
	// mant is like "-1.2345e+06": [sign] d[.ddd] e±EE.
	ePos := indexByteString(mant, 'e')
	digits := make([]byte, 0, 17)
	for i := 0; i < ePos; i++ {
		if mant[i] != '-' && mant[i] != '.' {
			digits = append(digits, mant[i])
		}
	}
	exp, _ := strconv.Atoi(mant[ePos+1:])
	return ryuFromDigits(neg, string(digits), exp)
}

// ryuFromDigits assembles the pretty form from shortest digits D
// (first digit non-zero) and decimal exponent E (value = d.ddd × 10^E).
func ryuFromDigits(neg bool, digits string, exp int) string {
	n := len(digits)
	kk := exp + 1 // 1-based decimal point position within digits
	var out []byte
	if neg {
		out = append(out, '-')
	}
	switch {
	case kk > 0 && kk <= 16:
		if kk >= n {
			out = append(out, digits...)
			for i := 0; i < kk-n; i++ {
				out = append(out, '0')
			}
			out = append(out, '.', '0')
		} else {
			out = append(out, digits[:kk]...)
			out = append(out, '.')
			out = append(out, digits[kk:]...)
		}
	case kk > -5 && kk <= 0:
		out = append(out, '0', '.')
		for i := 0; i < -kk; i++ {
			out = append(out, '0')
		}
		out = append(out, digits...)
	default:
		// Scientific: d[.ddd]eE, exponent without a plus sign.
		out = append(out, digits[0])
		if n > 1 {
			out = append(out, '.')
			out = append(out, digits[1:]...)
		}
		out = append(out, 'e')
		out = strconv.AppendInt(out, int64(exp), 10)
	}
	return string(out)
}

// canonicalFloatResult values for canonicalFloatText.
const (
	cfOK = iota
	cfNonFinite
	cfPassthrough // payload is not an f64 lexical form: keep verbatim
)

// canonicalFloatText converts a stored ryu payload to the § 5.9.8
// canonical form: pass through the decimal region [1e-2, 1e7), build
// the shortest scientific form outside it, keep signed zero as-is.
func canonicalFloatText(s string) (string, int) {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return s, cfPassthrough
	}
	if isNonFiniteF(f) {
		return "", cfNonFinite
	}
	if f == 0 {
		return s, cfOK // "0.0" / "-0.0" stored at wire ingest
	}
	abs := f
	neg := false
	if math.Signbit(f) {
		abs = -f
		neg = true
	}
	if abs >= 1e-2 && abs < 1e7 {
		return s, cfOK // decimal region: ryu text is already canonical
	}
	// Scientific form: shortest digits + adjusted exponent (§ 5.9.8).
	mant := strconv.FormatFloat(f, 'e', -1, 64)
	ePos := indexByteString(mant, 'e')
	digits := make([]byte, 0, 17)
	for i := 0; i < ePos; i++ {
		if mant[i] != '-' && mant[i] != '.' {
			digits = append(digits, mant[i])
		}
	}
	exp, _ := strconv.Atoi(mant[ePos+1:])
	out := make([]byte, 0, len(digits)+8)
	if neg {
		out = append(out, '-')
	}
	out = append(out, digits[0])
	if len(digits) > 1 {
		out = append(out, '.')
		out = append(out, digits[1:]...)
	}
	out = append(out, 'e')
	out = strconv.AppendInt(out, int64(exp), 10)
	return string(out), cfOK
}

func isNonFiniteF(f float64) bool { return math.IsNaN(f) || math.IsInf(f, 0) }

// canonicalIntegerText normalises an `$i` wire payload to the canonical
// Integer text (no +, no redundant leading zeros, no signed zero).
func canonicalIntegerText(s string) (string, bool) {
	sign := ""
	digits := s
	if len(s) > 0 && (s[0] == '-' || s[0] == '+') {
		if s[0] == '-' {
			sign = "-"
		}
		digits = s[1:]
	}
	if digits == "" {
		return "", false
	}
	for i := 0; i < len(digits); i++ {
		if digits[i] < '0' || digits[i] > '9' {
			return "", false
		}
	}
	trimmed := digits
	for len(trimmed) > 0 && trimmed[0] == '0' {
		trimmed = trimmed[1:]
	}
	if trimmed == "" {
		return "0", true
	}
	return sign + trimmed, true
}

// looksLikeFloatText: a float lexical form ('.' or exponent) rather
// than an integer literal.
func looksLikeFloatText(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '.' || c == 'e' || c == 'E' {
			return true
		}
	}
	return false
}
