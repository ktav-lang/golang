package ktav_test

// Cross-backend read/write round trips plus INDEPENDENT canonical
// oracle assertions. The hand-written expectations below are transcribed
// from the spec corpus oracles on disk (spec/versions/0.9/tests/...) and
// §5.9 itself — not from either implementation — so two equally wrong
// backends cannot pass by agreeing. Cross-backend byte equality is
// asserted too, but explicitly as a secondary check. Every backend leg
// runs through its own ktav.Operations instance (ktav.New), so pure and
// native legs never share global state.

import (
	"math"
	"path/filepath"
	"testing"
)

// representativeValidFixtures are corpus stems (relative to
// spec/versions/0.9/tests/valid/) covering each major syntax family;
// used for native↔pure value round trips. Every stem is verified to
// exist under tests/valid/ on disk.
var representativeValidFixtures = []string{
	"valid/objects/nested",
	"valid/objects/empty_multiline",
	"valid/dotted_keys/reopen_after_sibling",
	"valid/dotted_keys/expansion",
	"valid/numbers/float/notation_boundaries",
	"valid/numbers/float/negative_zero",
	"valid/numbers/float/min_positive_subnormal",
	"valid/numbers/integer/binary",
	"valid/multiline/verbatim_basic",
	"valid/multiline/stripped_basic",
	"valid/quoted_keys/colon_no_escape_needed",
	"valid/key_escaping/unicode_escape_surrogate_pair",
	"valid/key_escaping/canonical_reescape",
	"valid/top_level_array/pair_shaped_first_item",
	"valid/top_level_array/empty_compound_first_item",
	"valid/bom_boundary/first_key_leading_bom",
	"valid/comments/own_line",
}

// TestPureRoundTripReadAcrossBackends: native Loads → pure Dumps → pure
// Loads must reproduce the value; pure Loads → native Dumps → native
// Loads likewise. Pure→pure round trips run unconditionally; native legs
// are skipped (with the pure legs still running) when the cabi artifact
// is absent.
func TestPureRoundTripReadAcrossBackends(t *testing.T) {
	specRoot := requireSpec(t)
	pure := pureOps(t)
	for _, stem := range representativeValidFixtures {
		t.Run(stem, func(t *testing.T) {
			src, err := readFixtureSource(filepath.Join(specRoot, filepath.FromSlash(stem)+".ktav"))
			if err != nil {
				t.Fatal(err)
			}

			// Pure leg (unconditional): pure Loads → pure Dumps → pure Loads.
			pureValue, err := pure.Loads(string(src))
			if err != nil {
				t.Fatalf("pure Loads: %v", err)
			}
			pureDump, err := pure.Dumps(pureValue)
			if err != nil {
				t.Fatalf("pure Dumps: %v", err)
			}
			pureAgain, err := pure.Loads(pureDump)
			if err != nil {
				t.Fatalf("pure Loads(Dumps): %v\n--- dumped ---\n%s", err, pureDump)
			}
			if !structEqual(pureValue, pureAgain) {
				t.Fatalf("pure→pure round trip diverged\nfirst:  %#v\nsecond: %#v", pureValue, pureAgain)
			}

			if !hasCabi() {
				return // native legs skipped; pure legs above already ran
			}

			native := nativeOps(t)

			// native → pure: parse natively, write purely, read purely.
			nativeValue, err := native.Loads(string(src))
			if err != nil {
				t.Fatalf("native Loads: %v", err)
			}
			pureRewrite, err := pure.Dumps(nativeValue)
			if err != nil {
				t.Fatalf("pure Dumps(native value): %v", err)
			}
			pureRead, err := pure.Loads(pureRewrite)
			if err != nil {
				t.Fatalf("pure Loads of pure rewrite: %v\n--- dumped ---\n%s", err, pureRewrite)
			}
			if !structEqual(nativeValue, pureRead) {
				t.Fatalf("native→pure round trip diverged\nnative: %#v\npure:   %#v", nativeValue, pureRead)
			}

			// pure → native: parse purely, write natively, read natively.
			pureValue2, err := pure.Loads(string(src))
			if err != nil {
				t.Fatalf("pure Loads: %v", err)
			}
			nativeRewrite, err := native.Dumps(pureValue2)
			if err != nil {
				t.Fatalf("native Dumps(pure value): %v", err)
			}
			nativeRead, err := native.Loads(nativeRewrite)
			if err != nil {
				t.Fatalf("native Loads of native rewrite: %v\n--- dumped ---\n%s", err, nativeRewrite)
			}
			if !structEqual(pureValue2, nativeRead) {
				t.Fatalf("pure→native round trip diverged\npure:   %#v\nnative: %#v", pureValue2, nativeRead)
			}
		})
	}
}

// canonicalOracleTable is the independent §5.9 oracle: source → exact
// canonical bytes, transcribed from corpus oracles on disk
// (valid/numbers/float/*, valid/bom_boundary/first_key_leading_bom,
// valid/key_escaping/*, valid/dotted_keys/*, valid/top_level_array/*,
// and — for the nested_array value shape — real corpus canonical bytes
// such as valid/arrays/mixed_heterogeneous.canonical.ktav, where a
// non-empty array value emits `key: [`, items at indent+1, `]`).
// This is the PRIMARY expectation for both backends.
var canonicalOracleTable = []struct {
	name string
	src  string
	want string
}{
	{"float_decimal_spellings", "a: 0.5\nb: 1.0\nc: 3.14\nd: -0.5\ne: 1.5\nf: 42.0\n",
		"a: 0.5\nb: 1.0\nc: 3.14\nd: -0.5\ne: 1.5\nf: 42.0\n"},
	{"float_notation_boundaries", "small: 1e-3\ndecimal_boundary: 0.01\nmid_small: 1.5e-3\nnegative_small: -1e-3\nlarge_below: 9999999.0\nlarge_boundary: 1e7\n",
		"small: 1e-3\ndecimal_boundary: 0.01\nmid_small: 1.5e-3\nnegative_small: -1e-3\nlarge_below: 9999999.0\nlarge_boundary: 1e7\n"},
	{"float_max_finite", "max: 1.7976931348623157e308\n", "max: 1.7976931348623157e308\n"},
	{"float_min_normal", "min_normal: 2.2250738585072014e-308\n", "min_normal: 2.2250738585072014e-308\n"},
	{"float_min_subnormal", "min_subnormal: 5e-324\n", "min_subnormal: 5e-324\n"},
	{"float_rounding_tie", "tie: 9.007199254740992e15\ntie2: 9007199254740993.0\n",
		"tie: 9.007199254740992e15\ntie2: 9.007199254740992e15\n"},
	{"float_overflow_becomes_string", "overflow:: 1e9999\n", "overflow:: 1e9999\n"},
	{"negative_zero_float", "zero: -0.0\n", "zero: -0.0\n"},
	{"positive_zero_float", "zero: 0.0\n", "zero: 0.0\n"},
	{"leading_zero_decimal_is_string", "ver:: 01.5\nexp:: 05e3\nsub_one: 0.5\nordinary: 1.5\n",
		"ver:: 01.5\nexp:: 05e3\nsub_one: 0.5\nordinary: 1.5\n"},
	{"nested_object_indent", "server: {\n    host: 127.0.0.1\n    port: 8080\n    timeouts: {\n        read: 30\n        write: 10\n    }\n}\n",
		"server: {\n    host: 127.0.0.1\n    port: 8080\n    timeouts: {\n        read: 30\n        write: 10\n    }\n}\n"},
	{"dotted_reopen_merge", "a.b: 1\nc: 2\na.d: 3\n",
		"a: {\n    b: 1\n    d: 3\n}\nc: 2\n"},
	{"dotted_expansion", "server.host: 127.0.0.1\nserver.port: 8080\nserver.endpoints.api: /v1\nserver.endpoints.admin: /admin\napp.debug: true\n",
		"server: {\n    host: 127.0.0.1\n    port: 8080\n    endpoints: {\n        api: /v1\n        admin: /admin\n    }\n}\napp: {\n    debug: true\n}\n"},
	{"quoted_key_no_escape_needed", "\"a:b\": 1\n", "\"a:b\": 1\n"},
	{"escaped_dot_key_requoted", "a\\.b: v\n", "\"a.b\": v\n"},
	{"surrogate_pair_inline", "k: {a: \\uD83D\\uDE00}\n", "k: {\n    a: \U0001F600\n}\n"},
	{"root_array_wraps_compound_first_item", "[\n    {}\n    2\n]\n", "[\n    {}\n    2\n]\n"},
	{"root_array_bare_scalars", "foo\nbar\nbaz\n", "foo\nbar\nbaz\n"},
	{"root_raw_marker_string", ":: host: localhost\n", ":: host: localhost\n"},
	// Corpus oracle valid/bom_boundary/first_key_leading_bom: the BOM
	// is quoted but emitted RAW (\xef\xbb\xbf), not re-escaped.
	{"bom_key_is_quoted_raw_bom", "\\uFEFFhost: example.com\n", "\"\uFEFFhost\": example.com\n"},
	{"verbatim_string_preserved", "body: ((\nline1\nline2\n))\n", "body: ((\nline1\nline2\n))\n"},
	{"stripped_reemitted_as_verbatim", "body: (\n    {\n      \"qwe\": 1\n    }\n)\n",
		"body: ((\n{\n  \"qwe\": 1\n}\n))\n"},
	{"empty_document_is_zero_bytes", "", ""},
	{"empty_array_root", "[]\n", "[]\n"},
	{"keywords_lowercase_and_empty_compounds", "a: {}\nb: []\nc: null\nd: true\ne: false\n",
		"a: {}\nb: []\nc: null\nd: true\ne: false\n"},
	{"canonical_output_is_idempotent_input", "a: {\n    b: 1\n}\n", "a: {\n    b: 1\n}\n"},
	// Non-empty array pair value: §5.9.4 explicit-bracket multi-line
	// form (real corpus bytes: valid/arrays/nested.canonical.ktav).
	// "two" keeps its QUOTES: §5.3.3 ("Keys only") makes value-position
	// quote characters ordinary String content, never stripped, and
	// §5.9.7 emits a quote-leading body as-is after the separator (no
	// raw marker — cf. corpus inline/unescaped_quote_value_discriminator,
	// whose canonical bytes emit the quote-leading body plainly).
	{"array_value_explicit_brackets", "list: [1, \"two\", null, true]\n",
		"list: [\n    1\n    \"two\"\n    null\n    true\n]\n"},
}

// TestPureCanonicalIndependentOracle runs the hand-written §5.9 table
// through BOTH backends (each via its own instance). The expected bytes
// are the primary oracle; the cross-backend byte equality afterwards is
// explicitly secondary.
func TestPureCanonicalIndependentOracle(t *testing.T) {
	pure := pureOps(t)
	for _, tt := range canonicalOracleTable {
		t.Run(tt.name, func(t *testing.T) {
			gotPure, err := pure.CanonicalFromSource(tt.src)
			if err != nil {
				t.Fatalf("pure CanonicalFromSource(%q): %v", tt.src, err)
			}
			if gotPure != tt.want {
				t.Fatalf("pure canonical mismatch (independent oracle)\nsource: %q\nwant:   %q\ngot:    %q", tt.src, tt.want, gotPure)
			}
			if !hasCabi() {
				return
			}
			native := nativeOps(t)
			gotNative, err := native.CanonicalFromSource(tt.src)
			if err != nil {
				t.Fatalf("native CanonicalFromSource(%q): %v", tt.src, err)
			}
			if gotNative != tt.want {
				t.Fatalf("native canonical mismatch (independent oracle)\nsource: %q\nwant:   %q\ngot:    %q", tt.src, tt.want, gotNative)
			}
			// SECONDARY check: the two backends must at least agree
			// with each other (meaningful only because each was also
			// held to the independent oracle above).
			if gotNative != gotPure {
				t.Fatalf("cross-backend canonical divergence\npure:   %q\nnative: %q", gotPure, gotNative)
			}
		})
	}
}

// TestPureEmitCanonicalGoValueOracle pins EmitCanonical's documented
// Go-value contract (alphabetical map keys, canonical scalars, §5.9.4
// explicit-bracket arrays) with hand-written bytes — independent of any
// backend. The nested_array bytes mirror real corpus canonical bytes
// (e.g. valid/arrays/mixed_heterogeneous.canonical.ktav).
func TestPureEmitCanonicalGoValueOracle(t *testing.T) {
	ops := pureOps(t)
	tests := []struct {
		name  string
		value any
		want  string
	}{
		{"alphabetical_order", map[string]any{"b": int64(1), "a": 2.5}, "a: 2.5\nb: 1\n"},
		{"negative_zero", map[string]any{"z": math.Copysign(0, -1)}, "z: -0.0\n"},
		{"shortest_float", map[string]any{"pi": 3.14, "big": 1e7, "tiny": 1e-3},
			"big: 1e7\npi: 3.14\ntiny: 1e-3\n"},
		{"nested_array", map[string]any{"list": []any{int64(1), "two", nil, true}},
			"list: [\n    1\n    two\n    null\n    true\n]\n"},
		{"empty_object_leaf", map[string]any{"e": map[string]any{}}, "e: {}\n"},
		{"empty_array_leaf", map[string]any{"e": []any{}}, "e: []\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ops.EmitCanonical(tt.value)
			if err != nil {
				t.Fatalf("pure EmitCanonical: %v", err)
			}
			if got != tt.want {
				t.Fatalf("pure EmitCanonical mismatch\nwant: %q\ngot:  %q", tt.want, got)
			}
		})
	}
}

// TestPureDumpsForceStringsCoercion checks the force-string contract on
// the pure writer: every leaf scalar becomes a `::` String and the
// output re-parses as strings only.
func TestPureDumpsForceStringsCoercion(t *testing.T) {
	ops := pureOps(t)
	out, err := ops.DumpsForceStrings(map[string]any{
		"i": int64(-42), "f": 2.5, "b": true, "n": nil, "s": "plain",
		"nested": []any{int64(7)},
	})
	if err != nil {
		t.Fatalf("pure DumpsForceStrings: %v", err)
	}
	back, err := ops.Loads(out)
	if err != nil {
		t.Fatalf("pure Loads(force-string output %q): %v", out, err)
	}
	m, ok := back.(map[string]any)
	if !ok {
		t.Fatalf("want map, got %T", back)
	}
	for key, want := range map[string]string{
		"i": "-42", "f": "2.5", "b": "true", "n": "null", "s": "plain",
	} {
		if got := m[key]; got != want {
			t.Errorf("key %s = %#v, want String %q", key, got, want)
		}
	}
	nested, ok := m["nested"].([]any)
	if !ok || len(nested) != 1 || nested[0] != "7" {
		t.Errorf("nested = %#v, want []any{\"7\"}", m["nested"])
	}
}
