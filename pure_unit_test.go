package ktav_test

// Focused negative controls for the pure backend's parser/writer states
// and boundaries. All expectations are pinned from the spec (§3–§6,
// §8.1) and the corpus oracles — never from the implementation. Every
// test runs through a per-instance ktav.Operations bound to BackendPure
// (ktav.New), so no test can contaminate the package-global default
// backend; the global-default API keeps its own test below.

import (
	"errors"
	"strings"
	"testing"

	ktav "github.com/ktav-lang/golang"
	"github.com/ktav-lang/golang/internal/native"
)

// pureErr asserts err is a *ktav.Error with exactly the given class (and
// reason, when wantReason is non-empty).
func pureErr(t *testing.T, err error, class, reason string) *ktav.Error {
	t.Helper()
	var e *ktav.Error
	if !errors.As(err, &e) {
		t.Fatalf("want *ktav.Error %s, got %T (%v)", class, err, err)
	}
	if e.Class != class {
		t.Fatalf("error class = %q, want %q (reason %q)", e.Class, class, e.Reason)
	}
	if reason != "" && e.Reason != reason {
		t.Fatalf("error reason = %q, want %q", e.Reason, reason)
	}
	return e
}

// requirePureFailure runs fn and expects an error of the given class.
func requirePureFailure(t *testing.T, label, class, reason string, fn func() (string, error)) {
	t.Helper()
	out, err := fn()
	if err == nil {
		t.Fatalf("%s: expected %s error, got ok:\n%s", label, class, out)
	}
	pureErr(t, err, class, reason)
}

// TestPureNegativeControls walks every major parser error state with a
// minimal hand-written document per §6 error class.
func TestPureNegativeControls(t *testing.T) {
	ops := pureOps(t)
	tests := []struct {
		name   string
		src    string
		class  string
		reason string
	}{
		// §5.0.1 rule 6 phase 2 + §5.1 rule 8: UnterminatedQuotedKey needs an
		// ESTABLISHED Object root — at the root level this line is not a pair
		// candidate (the quote-aware separator scan keeps the separator hidden
		// inside the unterminated quoted segment), so it falls to rule 7; the
		// vector is anchored with a first pair.
		{"unterminated_quoted_key", "anchor: ok\n\"open: 1\n", "UnterminatedQuotedKey", ""},
		// §5.0.1 rule 7: a bare non-pair first line makes the root an
		// implicit ARRAY, so the MissingSeparator class needs OBJECT
		// context — the second line is the bare one.
		{"missing_separator", "anchor: ok\njust some text\n", "MissingSeparator", ""},
		// §5.0.1 rule 7 alone: a bare root line yields a scalar array
		// item, NOT a MissingSeparator error (positive control).
		// {"implicit_array_bare_root", "just some text\n", ...} — covered
		// by TestPureImplicitArrayBareRoot below.
		// §5.0.1 rule 6 needs a non-empty raw prefix, so at the ROOT this line
		// is not a pair candidate (rule 7); inside an established Object the
		// empty prefix surfaces as EmptyKey (§6.5) — the vector is anchored.
		{"empty_key", "anchor: ok\n: value\n", "EmptyKey", ""},
		// §6.10: a glued plain `:` line is MissingSeparatorSpace inside an
		// established Object; at the ROOT it is not a pair candidate (§5.0.1
		// rule 6 <sep-end>) and falls to rule 7 — see TestPureRootGluedNotAPair.
		{"missing_separator_space", "anchor: ok\na:1\n", "MissingSeparatorSpace", ""},
		{"invalid_key", "a(b: 1\n", "InvalidKey", ""},
		{"duplicate_key", "a: 1\na: 2\n", "DuplicateKey", ""},
		{"dotted_conflict_leaf_first", "a: 1\na.b: 2\n", "KeyPathConflict", ""},
		{"dotted_conflict_dotted_first", "a.b: 1\na: 2\n", "KeyPathConflict", ""},
		{"bad_escape_in_key", `bad\tescape: 1` + "\n", "BadEscapeSequence", ""},
		// Escapes are processed ONLY in inline compound values (§3.7
		// list; code path parseInlineValueRaw → processEscapes) and in
		// keys — never in whole-line pair bodies. These vectors mirror
		// the corpus invalid/bad_escape/* fixtures, whose inputs are
		// inline values (`k: {a: \uD800}` → BadEscapeSequence).
		{"bad_escape_in_inline_value", `v: {k: no\tescape}` + "\n", "BadEscapeSequence", ""},
		{"lone_surrogate_high", "k: {v: \\uD800}\n", "BadEscapeSequence", ""},
		{"lone_surrogate_low", "k: {v: \\uDC00}\n", "BadEscapeSequence", ""},
		{"surrogate_pair_reversed", "k: {v: \\uDE00\\uD83D}\n", "BadEscapeSequence", ""},
		{"unterminated_inline_object", "k: {a: 1\n", "UnterminatedInlineCompound", ""},
		{"unterminated_inline_array", "k: [1, 2\n", "UnterminatedInlineCompound", ""},
		{"malformed_inline_empty_item", "k: [1,, 2]\n", "MalformedInlineCompound", ""},
		{"unbalanced_bracket", "}\n", "UnbalancedBracket", ""},
		{"orphan_after_inline_root", "{a: 1}\nstray: line\n", "OrphanLineAfterTopLevelInline", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ops.Loads(tt.src)
			if err == nil {
				t.Fatalf("expected %s, got ok\nsrc: %q", tt.class, tt.src)
			}
			e := pureErr(t, err, tt.class, tt.reason)
			if e.Span == nil || e.Span.Start >= e.Span.End {
				t.Fatalf("expected nonempty byte span over malformed bytes, got %#v", e.Span)
			}
			if e.Line < 1 {
				t.Fatalf("expected 1-based line, got %d", e.Line)
			}
		})
	}
}

// TestPureRootGluedNotAPair pins §5.0.1 rule 6/7 at the ROOT level: a
// glued plain `:` line (no <sep-end> whitespace), a line with an empty
// raw key prefix, and a line whose only separator hides inside an
// unterminated quoted segment are NOT pair candidates — the root is an
// implicit Array of bare-scalar items (rule 7), NOT an error.
func TestPureRootGluedNotAPair(t *testing.T) {
	ops := pureOps(t)
	for _, tt := range []struct{ name, src string }{
		{"glued_colon", "a:1\n"},
		{"empty_key_prefix", ": value\n"},
		{"separator_inside_unterminated_quote", `"open: 1` + "\n"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			v, err := ops.Loads(tt.src)
			if err != nil {
				t.Fatalf("Loads(%q): %v (§5.0.1 rule 7: implicit Array root, no error)", tt.src, err)
			}
			arr, ok := v.([]any)
			if !ok || len(arr) != 1 {
				t.Fatalf("Loads(%q) = %#v, want one-item implicit Array (§5.0.1 rule 7)", tt.src, v)
			}
		})
	}
}

// TestPureExplicitRootEOFRejection: explicit openers that hit EOF are
// UnclosedCompound (the envelope carries a span, no source line).
func TestPureExplicitRootEOFRejection(t *testing.T) {
	ops := pureOps(t)
	for _, tt := range []struct{ name, src string }{
		{"open_object_root", "{\n"},
		{"open_array_root", "[\n"},
		{"nested_open_object", "a: {\n    b: 1\n"},
		{"nested_open_array", "a: [\n    1\n"},
		{"open_multiline_string", "k: (\n    text\n"},
		{"open_verbatim_string", "k: ((\n    text\n"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ops.Loads(tt.src)
			e := pureErr(t, err, "UnclosedCompound", "")
			if e.Span == nil {
				t.Fatal("expected span on UnclosedCompound")
			}
		})
	}
}

// TestPureUnterminatedStrings pins the unterminated inline raw form as
// UnterminatedInlineCompound and the escape-precedence rule
// (BadEscapeSequence even before a missing closer, §6.13/corpus), plus
// the whole-line positive controls below.
func TestPureUnterminatedStrings(t *testing.T) {
	ops := pureOps(t)
	// An unterminated inline raw body is UnterminatedInlineCompound;
	// an invalid escape inside an inline value is BadEscapeSequence —
	// corpus invalid/bad_escape/unknown_sequence_before_missing_closer
	// pins that the bad escape wins even before the missing closer.
	requirePureFailure(t, "unterminated_inline_raw", "UnterminatedInlineCompound", "",
		func() (string, error) { return ops.CanonicalFromSource("k: {a: :: v\n") })
	requirePureFailure(t, "bad_escape_before_missing_closer", "BadEscapeSequence", "",
		func() (string, error) { return ops.CanonicalFromSource("cfg: {a: \\q\n") })

	// POSITIVE controls (§3.7 "Escape sequences are NOT processed in:
	// multi-line scalar values"): whole-line bodies keep backslash
	// sequences as literal data — corpus
	// valid/scalars/unicode_escape_not_processed_in_plain_body pins
	// `note: \u0041` → String "\\u0041" byte-for-byte.
	for _, tt := range []struct{ name, src, want string }{
		{"whole_line_u_escape", "v: \\uD800\n", "\\uD800"},
		{"whole_line_tab_escape", "v: no\\tescape\n", "no\\tescape"},
		{"whole_line_trailing_backslash", "k: ends\\\n", "ends\\"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			v, err := ops.Loads(tt.src)
			if err != nil {
				t.Fatalf("Loads(%q): %v", tt.src, err)
			}
			if got := v.(map[string]any)[strings.SplitN(tt.src, ":", 2)[0]]; got != tt.want {
				t.Fatalf("whole-line body = %#v, want literal %q", got, tt.want)
			}
		})
	}
}

// TestPureBOMPlacement: exactly one leading U+FEFF is skipped; a BOM
// anywhere else is data or an error (§3.1).
func TestPureBOMPlacement(t *testing.T) {
	ops := pureOps(t)

	// Leading BOM is consumed; the key is decoded without it.
	v, err := ops.Loads("\uFEFFa: 1\n")
	if err != nil {
		t.Fatalf("leading BOM rejected: %v", err)
	}
	if m := v.(map[string]any); m["a"] != int64(1) {
		t.Fatalf("leading BOM value = %#v", v)
	}

	// The canonical emission QUOTES a U+FEFF-prefixed FIRST root key
	// but keeps the character itself raw (corpus oracle
	// valid/bom_boundary/first_key_leading_bom pins the exact bytes:
	// "\xef\xbb\xbfhost": example.com\n).
	got, err := ops.CanonicalFromSource("\\uFEFFhost: example.com\n")
	if err != nil {
		t.Fatalf("CanonicalFromSource: %v", err)
	}
	if want := "\"\uFEFFhost\": example.com\n"; got != want {
		t.Fatalf("canonical BOM key\nwant: %q\ngot:  %q", want, got)
	}
}

// TestPureLineEndings: LF, CR and CRLF all terminate a line (§3.2),
// including MIXED documents and documents whose FINAL line has no
// terminator (the splitter must make forward progress and must not
// re-scan from byte zero — that would duplicate keys or loop).
func TestPureLineEndings(t *testing.T) {
	ops := pureOps(t)
	for _, tt := range []struct{ name, src string }{
		{"lf", "a: 1\nb: 2\n"},
		{"crlf", "a: 1\r\nb: 2\r\n"},
		{"cr", "a: 1\rb: 2\r"},
		{"mixed", "a: 1\nb: 2\r\nc: 3\r"},
		{"no_final_eol_lf", "a: 1\nb: 2"},
		{"no_final_eol_crlf", "a: 1\r\nb: 2"},
		{"no_final_eol_cr", "a: 1\rb: 2"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			v, err := ops.Loads(tt.src)
			if err != nil {
				t.Fatalf("Loads(%q): %v", tt.src, err)
			}
			m, ok := v.(map[string]any)
			if !ok || m["a"] != int64(1) || m["b"] != int64(2) {
				t.Fatalf("value = %#v", v)
			}
			if tt.name == "mixed" && m["c"] != int64(3) {
				t.Fatalf("mixed value = %#v", v)
			}
		})
	}
}

// TestPureInvalidUTF8RawBytes: InvalidUtf8 is detected before grammar
// (§6.15) with a byte span over the malformed bytes, at the START and
// in the MIDDLE of the document.
func TestPureInvalidUTF8RawBytes(t *testing.T) {
	ops := pureOps(t)
	tests := []struct {
		name  string
		src   string
		span  ktav.Span
		class string
	}{
		{"lone_continuation_at_start", "\x80a: 1\n", ktav.Span{Start: 0, End: 1}, "InvalidUtf8"},
		{"lone_continuation_in_middle", "a: 雪\ninvalid: \xff\n", ktav.Span{Start: 16, End: 17}, "InvalidUtf8"},
		// Byte offsets mirror the corpus fixture
		// invalid/invalid_utf8/grammar_error_before_bad_byte (0x80 at 29).
		{"grammar_error_before_bad_byte", "anchor: ok\njust-some-text\na: \x80", ktav.Span{Start: 29, End: 30}, "InvalidUtf8"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ops.Loads(tt.src)
			e := pureErr(t, err, tt.class, "")
			if e.Span == nil || *e.Span != tt.span {
				t.Fatalf("span = %#v, want %#v", e.Span, &tt.span)
			}
		})
	}
}

// TestPureNumericBoundaries pins §3.6/§5.2 rule 13/14 behaviour through
// the public API: leading zeros, i64 overflow → String, float-domain
// overflow → String vs underflow → Float 0.0 (§5.2 rule 14 pins
// underflow to ±0.0 as an ORDINARY Float), negative zero, and §5.9.8
// canonical thresholds. Every expected canonical byte below comes from
// the spec text (§5.9.8 names 0.01, 1e-3, 1.5e-3, -1e-3, 9999999.0,
// 1e7 and 120000000.0 → 1.2e8 verbatim) or from §5.2/§3.6 grammar
// rules — never from the implementation.
func TestPureNumericBoundaries(t *testing.T) {
	ops := pureOps(t)

	// Leading-zero decimals are Strings in BOTH modes (never lossy).
	for _, src := range []string{"n: 007\n", "n: 01.5\n"} {
		v, err := ops.Loads(src)
		if err != nil {
			t.Fatalf("Loads(%q): %v", src, err)
		}
		if got := v.(map[string]any)["n"]; got != strings.TrimSpace(strings.SplitN(src, ":", 2)[1]) {
			t.Fatalf("leading zero value = %#v, want String", got)
		}
		// Strict mode must NOT reject them (they are plain Strings).
		if _, err := ops.LoadsStrict(src); err != nil {
			t.Fatalf("LoadsStrict(%q): %v", src, err)
		}
	}

	// Integer overflow beyond int64 becomes a String (§5.2 rule 13;
	// corpus oracle valid/numbers/integer/big_overflow_to_string).
	v, err := ops.Loads("big: 9223372036854775808\n")
	if err != nil {
		t.Fatalf("Loads: %v", err)
	}
	if got := v.(map[string]any)["big"]; got != "9223372036854775808" {
		t.Fatalf("overflow value = %#v, want String \"9223372036854775808\"", got)
	}

	// Float-domain boundaries (§5.2 rule 14): overflow to non-finite is
	// a String; underflow to zero is an ordinary Float.
	if got, err := ops.Loads("v: 1e9999\n"); err != nil || got.(map[string]any)["v"] != "1e9999" {
		t.Fatalf("float overflow = %#v, %v; want String", got, err)
	}
	if got, err := ops.Loads("v: 1e-9999\n"); err != nil {
		t.Fatalf("float underflow: %v", err)
	} else if f, ok := got.(map[string]any)["v"].(float64); !ok || f != 0 {
		t.Fatalf("float underflow = %#v, want Float 0", got)
	}
	if got, err := ops.CanonicalFromSource("v: 1e-9999\n"); err != nil || got != "v: 0.0\n" {
		t.Fatalf("canonical(1e-9999) = %q, %v; want %q", got, err, "v: 0.0\n")
	}

	// Negative-zero Integer literal is lossy in strict mode (corpus
	// oracle strict-lossy/integer/negative_zero: body "-0" → "0"), but
	// Float -0.0 is NOT lossy and canonicalises to "-0.0" (§5.9.8 zero
	// rule: Float preserves the sign of zero).
	_, err = ops.LoadsStrict("offset: -0\n")
	pureErr(t, err, "LossyScalar", "")
	if _, err := ops.LoadsStrict("zero: -0.0\n"); err != nil {
		t.Fatalf("LoadsStrict(-0.0): %v", err)
	}
	for _, zero := range []struct{ src, want string }{
		{"zero: -0.0\n", "zero: -0.0\n"},
		{"zero: 0.0\n", "zero: 0.0\n"},
		{"zero: -0.0e0\n", "zero: -0.0\n"},
	} {
		got, err := ops.CanonicalFromSource(zero.src)
		if err != nil || got != zero.want {
			t.Fatalf("canonical(%q) = %q, %v; want %q", zero.src, got, err, zero.want)
		}
	}

	// §5.9.8 canonical float vectors — decimal region [1e-2, 1e7),
	// scientific outside it, shortest digits, whole values keep ".0".
	// Real newlines: whole-line bodies carry no escape processing, so
	// these MUST be genuine line-terminated pairs. Each vector is
	// derived from §5.9.8's rules/examples or cross-checked against a
	// corpus .canonical.ktav companion on disk (noted per line).
	canonicalTests := []struct {
		src, want, note string
	}{
		{"a: 0.1\n", "a: 0.1\n", "plain decimal (corpus float/dot: 0.5→0.5)"},
		{"a: 3.0\n", "a: 3.0\n", "whole values keep .0 (corpus float/dot: 1.0→1.0, 42.0→42.0)"},
		{"a: 0.01\n", "a: 0.01\n", "decimal-region lower bound (corpus float/notation_boundaries)"},
		{"a: 0.001\n", "a: 1e-3\n", "below 1e-2: scientific (corpus: small: 0.001 → 1e-3)"},
		{"a: 0.0015\n", "a: 1.5e-3\n", "scientific with shortest digits (corpus: mid_small)"},
		{"a: -0.001\n", "a: -1e-3\n", "negative scientific (corpus: negative_small)"},
		{"a: 9999999.0\n", "a: 9999999.0\n", "decimal-region upper bound (corpus: large_below)"},
		{"a: 1e7\n", "a: 1e7\n", "scientific from 1e7 (corpus: large_boundary: 10000000.0 → 1e7)"},
		{"a: 1e9\n", "a: 1e9\n", "scientific, shortest digits (corpus float/exponent: 1e9 → 1e9)"},
		{"a: 120000000.0\n", "a: 1.2e8\n", "§5.9.8 example"},
		{"a: 123456789.0\n", "a: 1.23456789e8\n", "at or above 1e7: scientific"},
		{"a: 10000000.5\n", "a: 1.00000005e7\n", "just above the threshold"},
		{"a: 5e3\n", "a: 5000.0\n", "exponent spelling becomes decimal (strict-lossy corpus: 5e3 → 5000.0)"},
		{"a: 1.7976931348623157e308\n", "a: 1.7976931348623157e308\n", "max finite (corpus float/max_finite)"},
		{"a: 5e-324\n", "a: 5e-324\n", "min subnormal (corpus float/min_positive_subnormal: 4.94…e-324 → 5e-324)"},
	}
	for _, tt := range canonicalTests {
		got, err := ops.CanonicalFromSource(tt.src)
		if err != nil {
			t.Fatalf("CanonicalFromSource(%q): %v", tt.src, err)
		}
		if got != tt.want {
			t.Fatalf("canonical(%q) = %q, want %q (%s)", tt.src, got, tt.want, tt.note)
		}
	}
}

// TestPureStrictLossySpellings: strict mode rejects every lossy
// spelling with the exact Body/Canonical payload (§8.1, corpus
// strict-lossy oracles), and lax mode accepts it with the intended value.
func TestPureStrictLossySpellings(t *testing.T) {
	ops := pureOps(t)
	tests := []struct {
		src       string
		key       string
		laxValue  any
		body      string
		canonical string
	}{
		{"v: 1.10\n", "v", 1.1, "1.10", "1.1"},
		{"v: 5e3\n", "v", 5000.0, "5e3", "5000.0"},
		{"v: 10000000.5\n", "v", 10000000.5, "10000000.5", "1.00000005e7"},
		{"v: +7\n", "v", int64(7), "+7", "7"},
		{"v: 0x1A2B\n", "v", int64(6699), "0x1A2B", "6699"},
		{"v: 0o755\n", "v", int64(493), "0o755", "493"},
		{"v: 0b1010\n", "v", int64(10), "0b1010", "10"},
		{"v: 1_000\n", "v", int64(1000), "1_000", "1000"},
		{"v: -0\n", "v", int64(0), "-0", "0"},
	}
	for _, tt := range tests {
		t.Run(strings.TrimSpace(tt.src), func(t *testing.T) {
			lax, err := ops.Loads(tt.src)
			if err != nil {
				t.Fatalf("lax Loads: %v", err)
			}
			if got := lax.(map[string]any)[tt.key]; got != tt.laxValue {
				t.Fatalf("lax value = %#v (%T), want %#v", got, got, tt.laxValue)
			}
			_, err = ops.LoadsStrict(tt.src)
			e := pureErr(t, err, "LossyScalar", "")
			if e.Body != tt.body || e.Canonical != tt.canonical {
				t.Fatalf("LossyScalar payload = {%q %q}, want {%q %q}", e.Body, e.Canonical, tt.body, tt.canonical)
			}
		})
	}
}

// TestPureDepthLimitBoundary: the declared inline-compound nesting limit
// is 128 (§5.8.4, implementation-defined; mirrors the reference).
// Nesting of exactly 128 compounds is accepted; 129 is rejected with
// MalformedInlineCompound. (Depth accounting per pure_parse.go: the
// outermost inline compound enters parseInlineArray/Object at depth 0.)
func TestPureDepthLimitBoundary(t *testing.T) {
	ops := pureOps(t)
	build := func(levels int) string {
		return "k: " + strings.Repeat("[", levels) + strings.Repeat("]", levels) + "\n"
	}
	if _, err := ops.Loads(build(128)); err != nil {
		t.Fatalf("depth-128 inline nesting rejected: %v", err)
	}
	_, err := ops.Loads(build(129))
	e := pureErr(t, err, "MalformedInlineCompound", "")
	if !strings.Contains(e.Body, "128") {
		t.Fatalf("depth-limit detail %q does not name the declared limit", e.Body)
	}
}

// TestPureEmptySourceAndScalarRoot: empty source parses to an empty
// object with zero canonical bytes; a scalar root parses in lax mode
// (where reachable) but is unrepresentable on every writer.
func TestPureEmptySourceAndScalarRoot(t *testing.T) {
	ops := pureOps(t)

	v, err := ops.Loads("")
	if err != nil {
		t.Fatalf("Loads(empty): %v", err)
	}
	if m, ok := v.(map[string]any); !ok || len(m) != 0 {
		t.Fatalf("empty source value = %#v, want empty object", v)
	}
	got, err := ops.CanonicalFromSource("")
	if err != nil || got != "" {
		t.Fatalf("canonical(empty) = %q, %v; want \"\"", got, err)
	}

	for _, label := range []string{"Dumps", "EmitCanonical"} {
		var err error
		if label == "Dumps" {
			_, err = ops.Dumps("scalar")
		} else {
			_, err = ops.EmitCanonical(int64(1))
		}
		e := pureErr(t, err, "UnrepresentableAt", "ScalarRoot")
		if e.SpecSection != "§5.9.0" {
			t.Fatalf("%s: SpecSection = %q, want §5.9.0", label, e.SpecSection)
		}
	}
}

// TestPureFormatterExactBytes pins the formatter's comment/trivia
// contract byte-exactly. Expectations derive from §3.4 (a comment owns
// a whole line), §5.9.1/§5.9.3 (root empty OBJECT emits zero bytes; the
// formatter must stay a fixed point, so trailing root trivia and the
// root empty ARRAY `[]` must survive), never from the implementation.
func TestPureFormatterExactBytes(t *testing.T) {
	ops := pureOps(t)
	tests := []struct{ name, src, want string }{
		// Empty root array stays an array: `[]` + LF (§5.9.3).
		{"empty_root_array", "[]\n", "[]\n"},
		// Root empty OBJECT formats to zero bytes, like its canonical
		// form (§5.9.3); `{}` on one line IS the empty object root.
		{"empty_root_object", "{}\n", ""},
		{"empty_root_object_multiline", "{\n}\n", ""},
		// Trailing whole-line comment after the root object's last pair.
		{"root_object_trailing_comment", "a: 1\n## tail\n", "a: 1\n## tail\n"},
		// Trailing comment after an explicit root's closing brace
		// survives (output shape may normalise, bytes are pinned).
		{"explicit_root_trailing_comment", "{a: 1}\n## tail\n", "a: 1\n## tail\n"},
		// Comment-only documents keep their comment.
		{"comment_only", "## only\n", "## only\n"},
		{"comment_only_after_blank_run", "## only\n\n\n", "## only\n"},
		// Explicit empty compounds as VALUES keep the closed-inline form.
		{"explicit_empty_compound_values", "a: {}\nb: []\n", "a: {}\nb: []\n"},
		// Wrapped root array whose first item is an EMPTY compound: the
		// §5.9.3 wrap keeps the array structure AND the trailing
		// whole-line comment, and the output is a fixed point
		// (regression: the wrapped-root-array closer once scheduled no
		// trailing trivia, dropping `## tail`).
		{"wrapped_root_array_empty_first_item_trailing_comment",
			"[\n    {}\n    ## tail\n]\n",
			"[\n    {}\n    ## tail\n]\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ops.FormatSource(tt.src)
			if err != nil {
				t.Fatalf("FormatSource(%q): %v", tt.src, err)
			}
			if got != tt.want {
				t.Fatalf("FormatSource(%q)\nwant: %q\ngot:  %q", tt.src, tt.want, got)
			}
			// Fixed-point property: formatting the output reproduces it.
			again, err := ops.FormatSource(got)
			if err != nil {
				t.Fatalf("FormatSource(output %q): %v", got, err)
			}
			if again != got {
				t.Fatalf("FormatSource not a fixed point\nfirst:  %q\nsecond: %q", got, again)
			}
		})
	}
}

// TestPureFormatterInvariants: the pure FormatSource keeps comments
// verbatim, collapses blank runs to one, is a fixed point, and equals
// CanonicalFromSource on comment-free documents (package contract).
func TestPureFormatterInvariants(t *testing.T) {
	ops := pureOps(t)

	src := "## leading comment\n\n\n\na: 1\n\n\n## inner comment\nb:   2\n## trailing\n"
	once, err := ops.FormatSource(src)
	if err != nil {
		t.Fatalf("FormatSource: %v", err)
	}
	for _, comment := range []string{"## leading comment", "## inner comment", "## trailing"} {
		if !strings.Contains(once, comment) {
			t.Fatalf("FormatSource dropped comment %q:\n%s", comment, once)
		}
	}
	if strings.Count(once, "\n\n") > 1 && strings.Contains(once, "\n\n\n") {
		t.Fatalf("FormatSource kept a blank run:\n%s", once)
	}
	twice, err := ops.FormatSource(once)
	if err != nil {
		t.Fatalf("FormatSource(fixed point input): %v", err)
	}
	if once != twice {
		t.Fatalf("FormatSource not a fixed point\nonce:\n%s\ntwice:\n%s", once, twice)
	}

	plain := "a: 1\nb:\n    c: 2\n"
	formatted, err := ops.FormatSource(plain)
	if err != nil {
		t.Fatalf("FormatSource(plain): %v", err)
	}
	canonical, err := ops.CanonicalFromSource(plain)
	if err != nil {
		t.Fatalf("CanonicalFromSource(plain): %v", err)
	}
	if formatted != canonical {
		t.Fatalf("comment-free formatting must equal canonical\nformat:   %q\ncanonical: %q", formatted, canonical)
	}
}

// TestPureImplicitArrayBareRoot: §5.0.1 rule 7 — a first line that is
// not a pair shape makes the root an implicit ARRAY, so a bare line is
// a scalar array ITEM, not a MissingSeparator error. (Positive control
// for the missing_separator vector in TestPureNegativeControls.)
func TestPureImplicitArrayBareRoot(t *testing.T) {
	ops := pureOps(t)
	v, err := ops.Loads("just some text\n")
	if err != nil {
		t.Fatalf("Loads: %v", err)
	}
	arr, ok := v.([]any)
	if !ok || len(arr) != 1 || arr[0] != "just some text" {
		t.Fatalf("value = %#v, want []any{\"just some text\"}", v)
	}
	got, err := ops.CanonicalFromSource("just some text\n")
	if err != nil || got != "just some text\n" {
		t.Fatalf("canonical = %q, %v; want bare item line", got, err)
	}
}

// TestPureDottedTriviaFormatting pins the formatter's dotted-key
// comment attachment (§3.4: a comment owns a whole line and documents
// the pair that follows; §5.3.2: dotted paths expand to pairs, the
// terminal pair carrying the line's trivia). Byte-exact expectations
// plus the fixed-point property; derived from the spec text, not the
// implementation.
func TestPureDottedTriviaFormatting(t *testing.T) {
	ops := pureOps(t)
	tests := []struct{ name, src, want string }{
		// Single-segment dotted pair: the comment attaches to the
		// terminal pair inside the expanded object.
		{"comment_before_dotted_pair", "## c\na.b: 1\n",
			"a: {\n    ## c\n    b: 1\n}\n"},
		// Multi-segment path: same rule, deeper expansion.
		{"comment_before_deep_dotted_pair", "## hdr\nx.y.z: 1\n",
			"x: {\n    y: {\n        ## hdr\n        z: 1\n    }\n}\n"},
		// Two dotted pairs reopening the same prefix, each with its own
		// preceding comment: both survive, in order, no duplicates —
		// descend() must never copy or re-emit a reopened slot's trivia.
		{"two_reopened_dotted_pairs", "## one\na.b: 1\n## two\na.c: 2\n",
			"a: {\n    ## one\n    b: 1\n    ## two\n    c: 2\n}\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ops.FormatSource(tt.src)
			if err != nil {
				t.Fatalf("FormatSource(%q): %v", tt.src, err)
			}
			if got != tt.want {
				t.Fatalf("FormatSource(%q)\nwant: %q\ngot:  %q", tt.src, tt.want, got)
			}
			again, err := ops.FormatSource(got)
			if err != nil {
				t.Fatalf("FormatSource(output): %v", err)
			}
			if again != got {
				t.Fatalf("not a fixed point\nfirst:  %q\nsecond: %q", got, again)
			}
		})
	}
}

// TestPureUnicodeEmission pins multi-byte UTF-8 through parse →
// canonical emission and key emission (§5.9.7/§5.9.10: non-ASCII
// printable runes are emitted raw; only §3.3 edge whitespace, control
// bytes and the structural set are escaped/quoted). Expectations are
// byte-exact and grounded in the corpus: valid/inline/escape/
// lowercase_unicode_hex (oracle "é", 2-byte), valid/scalars/
// string_max_code_point (U+10FFFF, 4-byte, whose .canonical.ktav emits
// the rune raw), plus spec-grammar-derived 3-byte and multibyte-key
// vectors. These exercise lastRune/DecodeLastRuneInString on the exact
// shapes that used to panic (continuation-byte tail mis-sized as a
// 4-byte rune).
func TestPureUnicodeEmission(t *testing.T) {
	ops := pureOps(t)
	tests := []struct{ name, src, want string }{
		// 2-byte é via a processed inline escape (corpus oracle "é").
		{"two_byte_escaped", "k: {v: \\u00e9}\n", "k: {\n    v: \u00e9\n}\n"},
		// 3-byte CJK rune, raw in and raw out (§5.9.7 plain emission).
		{"three_byte_raw", "k: {v: 雪}\n", "k: {\n    v: 雪\n}\n"},
		// 4-byte max code point: corpus valid/scalars/string_max_code_point
		// canonical companion is exactly this byte sequence.
		{"four_byte_max_code_point", "text: 􏿿\n", "text: 􏿿\n"},
		// Key ending in a 2-byte rune: bare emission (§5.9.10).
		{"key_ending_multibyte", "k: {\\u00e9: 1}\n", "k: {\n    é: 1\n}\n"},
		// Key whose LAST rune is §3.3 whitespace after a multibyte rune:
		// the edge-whitespace rule must quote (lastRune path).
		{"key_multibyte_then_space", "k: {\"aé \": 1}\n", "k: {\n    \"aé \": 1\n}\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ops.CanonicalFromSource(tt.src)
			if err != nil {
				t.Fatalf("CanonicalFromSource(%q): %v", tt.src, err)
			}
			if got != tt.want {
				t.Fatalf("canonical(%q)\nwant: %q\ngot:  %q", tt.src, tt.want, got)
			}
			// Fixed point: canonical output re-canonicalises to itself.
			again, err := ops.CanonicalFromSource(got)
			if err != nil || again != got {
				t.Fatalf("not a fixed point: %q, %v", again, err)
			}
			// The FORMATTER must handle every vector too: FormatSource
			// on the canonical bytes reproduces them (comment/Unicode
			// preservation through the trivia writer, same LIFO
			// closer/trivia contract as F2).
			formatted, err := ops.FormatSource(got)
			if err != nil {
				t.Fatalf("FormatSource(%q): %v", got, err)
			}
			if formatted != got {
				t.Fatalf("FormatSource(canonical)\nwant: %q\ngot:  %q", got, formatted)
			}
		})
	}
}

// TestPurePackageLevelGlobalAPI keeps the GLOBAL default API covered:
// UsePure/UseBackend swap the process-wide vtable and the package-level
// functions follow it. (Instance isolation from this global state is
// asserted by every other pure test, which never touches UseBackend.)
func TestPurePackageLevelGlobalAPI(t *testing.T) {
	withBackend(t, ktav.BackendPure)
	v, err := ktav.Loads("a: 1\n")
	if err != nil || v.(map[string]any)["a"] != int64(1) {
		t.Fatalf("package Loads = %#v, %v", v, err)
	}
	if _, err := ktav.LoadsStrict("a: 1\n"); err != nil {
		t.Fatalf("package LoadsStrict: %v", err)
	}
	if err := ktav.LoadsInto("a: 1\n", &struct{ A int64 }{}); err != nil {
		t.Fatalf("package LoadsInto: %v", err)
	}
	if out, err := ktav.Dumps(map[string]any{"a": int64(1)}); err != nil || out != "a: 1\n" {
		t.Fatalf("package Dumps = %q, %v", out, err)
	}
	if out, err := ktav.DumpsForceStrings(map[string]any{"a": int64(1)}); err != nil || out != "a:: 1\n" {
		t.Fatalf("package DumpsForceStrings = %q, %v", out, err)
	}
	if out, err := ktav.EmitCanonical(map[string]any{"a": int64(1)}); err != nil || out != "a: 1\n" {
		t.Fatalf("package EmitCanonical = %q, %v", out, err)
	}
	if out, err := ktav.FormatSource("a: 1\n"); err != nil || out != "a: 1\n" {
		t.Fatalf("package FormatSource = %q, %v", out, err)
	}
	if out, err := ktav.CanonicalFromSource("a: 1\n"); err != nil || out != "a: 1\n" {
		t.Fatalf("package CanonicalFromSource = %q, %v", out, err)
	}
	// New rejects an unknown backend.
	if _, err := ktav.New(ktav.Backend(42)); err == nil {
		t.Fatal("ktav.New(unknown backend) succeeded; want error")
	}
}

// TestPureNeverLoadsNativeLibrary is the packaging guard: with
// KTAV_FORCE_PURE=1 (the pure-backend CI job, where no cabi artifact is
// ever built) the whole pure suite must run without the loader ever
// dlopening anything. It is env-guarded rather than unconditional
// because in the native CI job earlier tests legitimately load the
// library first.
func TestPureNeverLoadsNativeLibrary(t *testing.T) {
	if !pureForced() {
		t.Skip("guarded: enforced by the pure-backend CI job (KTAV_FORCE_PURE=1, no cabi built)")
	}
	if native.Loaded() {
		t.Fatal("native library was loaded during a pure-forced test run")
	}
}

// TestPureWriterCloserOrderShapes pins the LIFO closer contract of the
// iterative writer through PUBLIC ops (CanonicalFromSource): every
// opener matches its closer exactly once, children render between the
// opener and the closer, and path bookkeeping keeps insertion order
// across dotted reopen. Each oracle is hand-traced from §5.9.2–§5.9.6
// (pairs and items at indent+1 = 4 spaces; §5.9.3 root-array wrap when
// the first item is a compound or empty compound), never from the
// implementation. Each case is also a fixed point.
func TestPureWriterCloserOrderShapes(t *testing.T) {
	ops := pureOps(t)
	tests := []struct{ name, src, want string }{
		// `a.b: 1` opens one nested object: leaf line at indent 1,
		// then ONE closer at indent 0 (§5.9.4).
		{"dotted_single", "a.b: 1\n", "a: {\n    b: 1\n}\n"},
		// Nested inline arrays keep one level per opener (§5.9.4/§5.9.6).
		{"nested_inline_arrays", "k: [[1], [2, 3]]\n",
			"k: [\n" +
				"    [\n" +
				"        1\n" +
				"    ]\n" +
				"    [\n" +
				"        2\n" +
				"        3\n" +
				"    ]\n" +
				"]\n"},
		// Root array whose first item is an empty compound: §5.9.3 wrap
		// (already wrapped in the source, so a fixed point).
		{"root_array_wrapped", "[\n    {}\n    2\n]\n",
			"[\n    {}\n    2\n]\n"},
		// Alternating scalar / object / array items (§5.9.6 branches).
		{"alternating_items", "x: [1, {a: 2}, [3]]\n",
			"x: [\n" +
				"    1\n" +
				"    {\n" +
				"        a: 2\n" +
				"    }\n" +
				"    [\n" +
				"        3\n" +
				"    ]\n" +
				"]\n"},
		// Dotted reopen: the reopened prefix `d` emits its accumulated
		// pairs in insertion order (e-block closed before g), §5.3.2.
		{"dotted_reopen", "d.e.f: 1\nd.g: 2\n",
			"d: {\n" +
				"    e: {\n" +
				"        f: 1\n" +
				"    }\n" +
				"    g: 2\n" +
				"}\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ops.CanonicalFromSource(tt.src)
			if err != nil {
				t.Fatalf("CanonicalFromSource(%q): %v", tt.src, err)
			}
			if got != tt.want {
				t.Fatalf("canonical(%q)\nwant: %q\ngot:  %q", tt.src, tt.want, got)
			}
			// Fixed point through the formatter as well.
			fmt, err := ops.FormatSource(tt.want)
			if err != nil {
				t.Fatalf("FormatSource(%q): %v", tt.want, err)
			}
			if fmt != tt.want {
				t.Fatalf("FormatSource not identity on canonical bytes\nwant: %q\ngot:  %q", tt.want, fmt)
			}
		})
	}
}
