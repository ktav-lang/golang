package ktav

import (
	"encoding/hex"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestWriterRejectionsCarryStructuredFields(t *testing.T) {
	for name, write := range map[string]func() error{
		"Dumps":             func() error { _, err := Dumps(42); return err },
		"DumpsForceStrings": func() error { _, err := DumpsForceStrings("root"); return err },
		"EmitCanonical":     func() error { _, err := EmitCanonical(true); return err },
	} {
		t.Run(name, func(t *testing.T) {
			got := write()
			err, ok := got.(*Error)
			if !ok {
				t.Fatalf("expected *Error, got %T", got)
			}
			if err.Class != "UnrepresentableAt" || err.Reason != "ScalarRoot" || err.SpecSection != "§5.9.0" {
				t.Fatalf("unexpected scalar-root envelope: %#v", err)
			}
			if err.Path == nil || len(err.Path) != 0 || err.Span != nil || err.Line != 0 {
				t.Fatalf("unexpected scalar-root location: %#v", err)
			}
		})
	}

	for _, number := range []struct {
		name  string
		value float64
	}{{"NaN", math.NaN()}, {"PositiveInfinity", math.Inf(1)}, {"NegativeInfinity", math.Inf(-1)}} {
		t.Run(number.name, func(t *testing.T) {
			value := map[string]any{"配置": []any{map[string]any{"nested.key": number.value}}}
			for name, write := range map[string]func() error{
				"Dumps":         func() error { _, err := Dumps(value); return err },
				"EmitCanonical": func() error { _, err := EmitCanonical(value); return err },
			} {
				t.Run(name, func(t *testing.T) {
					got := write()
					err, ok := got.(*Error)
					if !ok {
						t.Fatalf("expected *Error, got %T", got)
					}
					if err.Class != "UnrepresentableAt" || err.Reason != "NonFiniteFloat" || err.SpecSection != "§5.9.0" {
						t.Fatalf("unexpected non-finite envelope: %#v", err)
					}
					if len(err.Path) != 2 || err.Path[0] != "配置" || err.Path[1] != "nested.key" || err.Span != nil || err.Line != 0 {
						t.Fatalf("unexpected non-finite path: %#v", err)
					}
				})
			}
		})
	}

	for _, test := range []struct {
		name  string
		value map[string]any
		want  string
	}{
		{"empty-key-before-float", map[string]any{"a": map[string]any{"": 1}, "z": math.NaN()}, "EmptyKeyName"},
		{"float-before-empty-key", map[string]any{"a": math.NaN(), "z": map[string]any{"": 1}}, "NonFiniteFloat"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := Dumps(test.value)
			ktavErr, ok := err.(*Error)
			if !ok {
				t.Fatalf("expected *Error, got %T", err)
			}
			if ktavErr.Class != "UnrepresentableAt" || ktavErr.Reason != test.want || ktavErr.SpecSection != "§5.9.0" {
				t.Fatalf("unexpected precedence envelope: %#v", ktavErr)
			}
			if test.want == "EmptyKeyName" && (len(ktavErr.Path) != 2 || ktavErr.Path[0] != "a" || ktavErr.Path[1] != "") {
				t.Fatalf("unexpected empty-key path: %#v", ktavErr.Path)
			}
		})
	}
}

func TestDumpsForceStringsPreservesNonFiniteFloatText(t *testing.T) {
	for _, test := range []struct {
		value float64
		want  string
	}{{math.NaN(), "NaN"}, {math.Inf(1), "+Inf"}, {math.Inf(-1), "-Inf"}} {
		out, err := DumpsForceStrings(map[string]any{"value": test.value})
		if err != nil {
			t.Fatalf("DumpsForceStrings(%v): %v", test.value, err)
		}
		got, err := Loads(out)
		if err != nil {
			t.Fatalf("Loads(%q): %v", out, err)
		}
		if actual := got.(map[string]any)["value"]; actual != test.want {
			t.Errorf("round-trip value = %#v, want %q", actual, test.want)
		}
	}
}

func TestInvalidUTF8EntryPointsCarryStructuredFields(t *testing.T) {
	src := "valid: 雪\ninvalid: \xff"
	checks := map[string]func() error{
		"Loads":               func() error { _, err := Loads(src); return err },
		"LoadsStrict":         func() error { _, err := LoadsStrict(src); return err },
		"LoadsInto":           func() error { return LoadsInto(src, new(any)) },
		"FormatSource":        func() error { _, err := FormatSource(src); return err },
		"CanonicalFromSource": func() error { _, err := CanonicalFromSource(src); return err },
	}
	for name, check := range checks {
		t.Run(name, func(t *testing.T) {
			got := check()
			err, ok := got.(*Error)
			if !ok {
				t.Fatalf("expected *Error, got %T", got)
			}
			if err.Class != "InvalidUtf8" || err.Reason != "" || err.SpecSection != "§6.15" {
				t.Fatalf("unexpected invalid-UTF-8 envelope: %#v", err)
			}
			start := len("valid: 雪\ninvalid: ")
			if err.Span == nil || err.Span.Start != start || err.Span.End != start+1 || err.Line != 2 {
				t.Fatalf("unexpected invalid-UTF-8 span: %#v", err)
			}
		})
	}
}

func TestInvalidUTF8LocationMatchesCoreReference(t *testing.T) {
	// Expected values mirror the Rust C-ABI envelope tests: std valid_up_to/
	// error_len plus the continuation-byte extension, with the line counted
	// over raw bytes per spec §3.2/§6.15.
	for _, test := range []struct {
		name             string
		src              string
		line, start, end int
	}{
		// CRLF keys; 80 continuation byte after "bad: " lands on line 3.
		{"multibyte-crlf", "name: שלום\r\nport: 8080\r\nbad: \x80", 3, 33, 34},
		// E1 lead with 80 continuation at EOF: span reaches end of input.
		{"truncated-at-eof", "a: 1\n\xe1\x80", 2, 5, 7},
		// E1 lead with 80 continuation then 'A': two-byte prefix swallowed.
		{"interrupted", "a: \xe1\x80\x41\n", 1, 3, 5},
		// C0 lead is overlong encoding: two-byte span.
		{"overlong-2byte", "a: \xc0\xaf\n", 1, 3, 5},
		// ED lead with A0: surrogate encoding, three-byte span.
		{"surrogate-3byte", "a: \xed\xa0\x80\n", 1, 3, 6},
		// F4 lead with 90: out-of-range four-byte span.
		{"out-of-range-4byte", "a: \xf4\x90\x80\x80\n", 1, 3, 7},
		// BOM-prefixed doc with bare CR terminators; 80 between CRs lands on line 3.
		{"bom-lone-cr", "\xef\xbb\xbfa: 1\rb: 2\r\x80\rc: 3", 3, 13, 14},
		// UTF-8 error fires before any grammar error in the document.
		{"beats-grammar", "anchor: ok\njust-some-text\na: \x80", 3, 29, 30},
		// Sole byte is an illegal lead.
		{"first-byte", "\x80", 1, 0, 1},
		// All continuation bytes: whole input swallowed.
		{"continuation-run", "\x80\x80\x80", 1, 0, 3},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, got := Loads(test.src)
			err, ok := got.(*Error)
			if !ok {
				t.Fatalf("expected *Error, got %T", got)
			}
			if err.Class != "InvalidUtf8" || err.Reason != "" || err.SpecSection != "§6.15" {
				t.Fatalf("unexpected invalid-UTF-8 envelope: %#v", err)
			}
			if err.Line != test.line || err.Span == nil || err.Span.Start != test.start || err.Span.End != test.end {
				t.Fatalf("unexpected invalid-UTF-8 location: line %d, span %#v", err.Line, err.Span)
			}
		})
	}
}

func TestLineOfOffsetCountsRawTerminators(t *testing.T) {
	// Expected values replicate the Rust C ABI line_of_offset tests:
	// 1-based raw-byte lines, each LF / lone CR / CR LF pair counted as
	// exactly one terminator per spec §3.2, no BOM skipping.
	for _, test := range []struct {
		name string
		src  string
		pos  int
		want int
	}{
		{"empty", "", 0, 1},
		{"abc-at-0", "abc", 0, 1},
		{"abc-at-3", "abc", 3, 1},
		{"abc-beyond-end", "abc", 99, 1},
		{"lf-before", "a\nb", 1, 1},
		{"lf-after", "a\nb", 2, 2},
		{"cr-before", "a\rb", 1, 1},
		{"cr-after", "a\rb", 2, 2},
		{"crlf-at-cr", "a\r\nb", 1, 1},
		{"crlf-at-lf", "a\r\nb", 2, 1},
		{"crlf-after", "a\r\nb", 3, 2},
		{"mixed-at-0", "a\n\r\n\rb", 0, 1},
		{"mixed-at-2", "a\n\r\n\rb", 2, 2},
		{"mixed-at-4", "a\n\r\n\rb", 4, 3},
		{"mixed-at-5", "a\n\r\n\rb", 5, 4},
		{"trailing-lf", "a\n", 2, 2},
		{"trailing-crlf-full", "a\r\n", 3, 2},
		{"trailing-crlf-at-lf", "a\r\n", 2, 1},
		{"bom-before", "\xef\xbb\xbfa\nb", 1, 1},
		{"bom-after", "\xef\xbb\xbfa\nb", 5, 2},
		{"bom-lf", "\xef\xbb\xbf\n", 4, 2},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := lineOfOffset(test.src, test.pos); got != test.want {
				t.Fatalf("lineOfOffset(%q, %d) = %d, want %d", test.src, test.pos, got, test.want)
			}
		})
	}
}

func TestFirstInvalidUTF8SpanMatchesReference(t *testing.T) {
	// Same vectors and expectations as TestInvalidUTF8LocationMatchesCoreReference.
	for _, test := range []struct {
		name             string
		src              string
		line, start, end int
	}{
		{"multibyte-crlf", "name: שלום\r\nport: 8080\r\nbad: \x80", 3, 33, 34},
		{"truncated-at-eof", "a: 1\n\xe1\x80", 2, 5, 7},
		{"interrupted", "a: \xe1\x80\x41\n", 1, 3, 5},
		{"overlong-2byte", "a: \xc0\xaf\n", 1, 3, 5},
		{"surrogate-3byte", "a: \xed\xa0\x80\n", 1, 3, 6},
		{"out-of-range-4byte", "a: \xf4\x90\x80\x80\n", 1, 3, 7},
		{"bom-lone-cr", "\xef\xbb\xbfa: 1\rb: 2\r\x80\rc: 3", 3, 13, 14},
		{"beats-grammar", "anchor: ok\njust-some-text\na: \x80", 3, 29, 30},
		{"first-byte", "\x80", 1, 0, 1},
		{"continuation-run", "\x80\x80\x80", 1, 0, 3},
	} {
		t.Run(test.name, func(t *testing.T) {
			start, end, ok := firstInvalidUTF8Span(test.src)
			if !ok {
				t.Fatalf("firstInvalidUTF8Span(%q) reported ok=false", test.src)
			}
			if start != test.start || end != test.end {
				t.Fatalf("firstInvalidUTF8Span(%q) = [%d,%d), want [%d,%d)", test.src, start, end, test.start, test.end)
			}
			if start >= end {
				t.Fatalf("span [%d,%d) is empty", start, end)
			}
			if got := lineOfOffset(test.src, start); got != test.line {
				t.Fatalf("lineOfOffset(%q, %d) = %d, want %d", test.src, start, got, test.line)
			}
		})
	}
	t.Run("valid-input", func(t *testing.T) {
		if _, _, ok := firstInvalidUTF8Span("valid: 雪\nplain"); ok {
			t.Fatal("firstInvalidUTF8Span reported ok=true for valid input")
		}
	})
}

func TestFirstInvalidUTF8SpanAgreesWithDecodeRuneScan(t *testing.T) {
	// Independent oracle: the first index where Go's decoder yields
	// (RuneError, size 1), or -1 when the whole string decodes.
	oracle := func(src string) int {
		for i := 0; i < len(src); {
			r, size := utf8.DecodeRuneInString(src[i:])
			if r == utf8.RuneError && size == 1 {
				return i
			}
			i += size
		}
		return -1
	}
	for _, test := range []struct {
		name string
		src  string
	}{
		{"multibyte-crlf", "name: שלום\r\nport: 8080\r\nbad: \x80"},
		{"truncated-at-eof", "a: 1\n\xe1\x80"},
		{"interrupted", "a: \xe1\x80\x41\n"},
		{"overlong-2byte", "a: \xc0\xaf\n"},
		{"surrogate-3byte", "a: \xed\xa0\x80\n"},
		{"out-of-range-4byte", "a: \xf4\x90\x80\x80\n"},
		{"bom-lone-cr", "\xef\xbb\xbfa: 1\rb: 2\r\x80\rc: 3"},
		{"beats-grammar", "anchor: ok\njust-some-text\na: \x80"},
		{"first-byte", "\x80"},
		{"continuation-run", "\x80\x80\x80"},
		{"ff-fe", "\xff\xfe"},
		{"c0-lead", "a: \xc0"},
		{"valid", "valid: 雪\nplain"},
	} {
		t.Run(test.name, func(t *testing.T) {
			want := oracle(test.src)
			start, _, ok := firstInvalidUTF8Span(test.src)
			if want < 0 {
				if ok {
					t.Fatalf("firstInvalidUTF8Span(%q) reported ok=true for decodable input", test.src)
				}
				if !utf8.ValidString(test.src) {
					t.Fatalf("utf8.ValidString(%q) = false for decodable input", test.src)
				}
				return
			}
			if !ok {
				t.Fatalf("firstInvalidUTF8Span(%q) reported ok=false, oracle first bad index %d", test.src, want)
			}
			if start != want {
				t.Fatalf("firstInvalidUTF8Span(%q) start = %d, want oracle index %d", test.src, start, want)
			}
			if utf8.ValidString(test.src) {
				t.Fatalf("utf8.ValidString(%q) = true for undecodable input", test.src)
			}
		})
	}
}

func TestFirstInvalidUTF8SpanAgainstRustReference(t *testing.T) {
	if path := os.Getenv("KTAV_UTF8_DIFF_FILE"); path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read reference: %v", err)
		}
		reference := make(map[string][2]int)
		for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
			fields := strings.Fields(line)
			if len(fields) != 3 {
				t.Fatalf("malformed reference line %q", line)
			}
			b, err := hex.DecodeString(fields[0])
			if err != nil {
				t.Fatalf("hex %q: %v", fields[0], err)
			}
			start, e1 := strconv.Atoi(fields[1])
			end, e2 := strconv.Atoi(fields[2])
			if e1 != nil || e2 != nil {
				t.Fatalf("bad offsets %q", line)
			}
			if _, exists := reference[fields[0]]; exists {
				t.Fatalf("duplicate reference %s", fields[0])
			}
			if utf8.Valid(b) {
				t.Fatalf("reference input %s is valid UTF-8", fields[0])
			}
			reference[fields[0]] = [2]int{start, end}
		}
		alphabet := []byte{0x41, 0x7f, 0x80, 0x8f, 0x90, 0x9f, 0xa0, 0xbf, 0xc0, 0xc1, 0xc2, 0xdf, 0xe0, 0xe1, 0xed, 0xee, 0xf0, 0xf1, 0xf4, 0xf5, 0xff}
		count, mismatches := 0, 0
		examples := make([]string, 0, 10)
		for n := 1; n <= 4; n++ {
			total := 1
			for i := 0; i < n; i++ {
				total *= len(alphabet)
			}
			for v := 0; v < total; v++ {
				x := v
				b := make([]byte, n)
				for i := n - 1; i >= 0; i-- {
					b[i] = alphabet[x%len(alphabet)]
					x /= len(alphabet)
				}
				if utf8.Valid(b) {
					continue
				}
				count++
				h := hex.EncodeToString(b)
				want, exists := reference[h]
				if !exists {
					mismatches++
					if len(examples) < 10 {
						examples = append(examples, fmt.Sprintf("missing %s", h))
					}
					continue
				}
				start, end, ok := firstInvalidUTF8Span(string(b))
				if !ok || start != want[0] || end != want[1] {
					mismatches++
					if len(examples) < 10 {
						examples = append(examples, fmt.Sprintf("%s got [%d,%d) ok=%v want [%d,%d)", h, start, end, ok, want[0], want[1]))
					}
				}
			}
		}
		for h := range reference {
			b, _ := hex.DecodeString(h)
			if len(b) < 1 || len(b) > 4 {
				mismatches++
				continue
			} /* extra coverage checked by count and input lookup */
		}
		if count != len(reference) {
			mismatches++
			if len(examples) < 10 {
				examples = append(examples, fmt.Sprintf("reference count %d, invalid count %d", len(reference), count))
			}
		}
		t.Logf("compared %d invalid inputs; mismatch count %d; examples: %v", count, mismatches, examples)
		if mismatches != 0 {
			t.Errorf("Rust differential mismatch count %d; examples: %v", mismatches, examples)
		}
		return
	}
	for _, tc := range []struct {
		name, src  string
		start, end int
	}{
		{"e1-41-after-prefix", "a: \xe1\x41", 3, 4},
		{"f0-90-41-after-prefix", "a: \xf0\x90\x41", 3, 5},
		{"e0-80-after-prefix", "a: \xe0\x80", 3, 5},
		{"80", "a: \x80", 3, 4},
		{"8080", "a: \x80\x80", 3, 5},
		{"c0", "a: \xc0", 3, 4},
		{"c1", "a: \xc1", 3, 4},
		{"f5", "a: \xf5", 3, 4},
		{"ff", "a: \xff", 3, 4},
		{"c241", "a: \xc2\x41", 3, 4},
		{"df", "a: \xdf", 3, 4},
		{"df41", "a: \xdf\x41", 3, 4},
		{"e0", "a: \xe0", 3, 4},
		{"e0a0", "a: \xe0\xa0", 3, 5},
		{"e0a041", "a: \xe0\xa0\x41", 3, 5},
		{"e1", "a: \xe1", 3, 4},
		{"e180", "a: \xe1\x80", 3, 5},
		{"e18041", "a: \xe1\x80\x41", 3, 5},
		{"ed", "a: \xed", 3, 4},
		{"eda0", "a: \xed\xa0", 3, 5},
		{"ed80", "a: \xed\x80", 3, 5},
		{"ee", "a: \xee", 3, 4},
		{"ee41", "a: \xee\x41", 3, 4},
		{"f0", "a: \xf0", 3, 4},
		{"f090", "a: \xf0\x90", 3, 5},
		{"f0-90-41-after-prefix", "a: \xf0\x90\x41", 3, 5},
		{"f0908041", "a: \xf0\x90\x80\x41", 3, 6},
		{"f1", "a: \xf1", 3, 4},
		{"f180", "a: \xf1\x80", 3, 5},
		{"f18041", "a: \xf1\x80\x41", 3, 5},
		{"f1808041", "a: \xf1\x80\x80\x41", 3, 6},
		{"f4", "a: \xf4", 3, 4},
		{"f48f", "a: \xf4\x8f", 3, 5},
		{"f48f41", "a: \xf4\x8f\x41", 3, 5},
		{"f48f8041", "a: \xf4\x8f\x80\x41", 3, 6},
		{"f580", "a: \xf5\x80", 3, 5},
		{"c2-cont-run", "a: \xc2\x80\x80", 5, 6},
		{"f5-run", "a: \xf5\x80\x80", 3, 6},
		{"c0-run", "a: \xc0\x80", 3, 5},
		{"f0-interrupt-third", "a: \xf0\x90A", 3, 5},
		{"e1-interrupt-third", "a: \xe1\x80A", 3, 5},
	} {
		t.Run(tc.name, func(t *testing.T) {
			start, end, ok := firstInvalidUTF8Span(tc.src)
			if !ok || start != tc.start || end != tc.end {
				t.Fatalf("%x got [%d,%d) ok=%v want [%d,%d)", []byte(tc.src), start, end, ok, tc.start, tc.end)
			}
		})
	}
}
