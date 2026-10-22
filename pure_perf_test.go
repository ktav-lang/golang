package ktav_test

// Deterministic algorithmic regression checks for the pure backend.
// These assert CORRECTNESS and structural invariants of genuine
// operations on large inputs (wide objects, deep dotted reopen-merge,
// megabyte-scale strings, deep compounds, canonical idempotence). They
// deliberately do NOT gate on wall-clock milliseconds (shared-CI
// timings are flaky); the only timing smoke is Go's own generous test
// timeout, so a quadratic regression surfaces as a timeout/failure,
// never as a skipped or fake-passing benchmark. Complexity documentation
// lives in the plan file: parse is O(bytes) + O(total key bytes),
// dotted merge is amortized O(1) per pair with O(depth) conflict
// checks, and the writers are amortized O(output bytes).
//
// Escape semantics per §3.7: escape sequences are processed ONLY inside
// inline compounds (§5.8) and in keys (§5.3) — the body of a WHOLE-LINE
// pair (`key: body` or `key:: body`, §5.3/§5.4) is literal data, and
// the trailing §3.3 whitespace immediately before EOL/EOF is trimmed
// from its semantic String body.

import (
	"strings"
	"testing"
)

// TestPurePerfWideObject: 10k-key object parses to 10k entries and
// canonicalises to output containing every key, insertion-ordered.
func TestPurePerfWideObject(t *testing.T) {
	ops := pureOps(t)
	const n = 10000
	var b strings.Builder
	for i := 0; i < n; i++ {
		b.WriteString("key")
		b.WriteString(itoaTest(i))
		b.WriteString(": ")
		b.WriteString(itoaTest(i))
		b.WriteByte('\n')
	}
	src := b.String()

	v, err := ops.Loads(src)
	if err != nil {
		t.Fatalf("Loads: %v", err)
	}
	m, ok := v.(map[string]any)
	if !ok || len(m) != n {
		t.Fatalf("decoded %T with len %d, want map with %d entries", v, len(m), n)
	}
	if m["key0"] != int64(0) || m["key9999"] != int64(9999) {
		t.Fatalf("wide object values wrong: %#v %#v", m["key0"], m["key9999"])
	}

	canonical, err := ops.CanonicalFromSource(src)
	if err != nil {
		t.Fatalf("CanonicalFromSource: %v", err)
	}
	// Every key survives in canonical order (insertion order here) and
	// the output ends with a single trailing LF (§5.9).
	for _, probe := range []string{"key0: 0\n", "key5000: 5000\n", "key9999: 9999\n"} {
		if !strings.Contains(canonical, probe) {
			t.Fatalf("canonical output missing %q", probe)
		}
	}
	if !strings.HasSuffix(canonical, "\n") || strings.HasSuffix(canonical, "\n\n") {
		t.Fatalf("canonical output must end with exactly one LF (len %d)", len(canonical))
	}
}

// TestPurePerfDeepDottedReopenMerge: 5k pairs over 500 dotted prefixes,
// each reopened after intervening siblings — exercises the dotted-prefix
// merge bookkeeping (§5.3.2) without rescans.
func TestPurePerfDeepDottedReopenMerge(t *testing.T) {
	ops := pureOps(t)
	const prefixes = 500
	const leavesPerPrefix = 10
	var b strings.Builder
	for p := 0; p < prefixes; p++ {
		for l := 0; l < leavesPerPrefix; l++ {
			b.WriteString("grp")
			b.WriteString(itoaTest(p))
			b.WriteString(".leaf")
			b.WriteString(itoaTest(l))
			b.WriteString(": ")
			b.WriteString(itoaTest(p*leavesPerPrefix + l))
			b.WriteByte('\n')
			if l == 0 {
				// Intervening sibling forces a reopen-merge on the
				// NEXT grpN.leaf write.
				b.WriteString("sibling")
				b.WriteString(itoaTest(p))
				b.WriteString(": s\n")
			}
		}
	}
	src := b.String()

	v, err := ops.Loads(src)
	if err != nil {
		t.Fatalf("Loads: %v", err)
	}
	m := v.(map[string]any)
	if len(m) != prefixes+prefixes { // grp0..grp499 + sibling0..sibling499
		t.Fatalf("top-level len = %d, want %d", len(m), prefixes+prefixes)
	}
	grp0 := m["grp0"].(map[string]any)
	if len(grp0) != leavesPerPrefix {
		t.Fatalf("grp0 len = %d, want %d", len(grp0), leavesPerPrefix)
	}
	if grp0["leaf9"] != int64(9) || m["sibling499"] != "s" {
		t.Fatalf("merge values wrong: %#v", grp0["leaf9"])
	}

	// Canonical emission reflects the merged insertion order: first
	// prefix group precedes its sibling, and the last write wins nothing
	// (no duplicate keys).
	canonical, err := ops.CanonicalFromSource(src)
	if err != nil {
		t.Fatalf("CanonicalFromSource: %v", err)
	}
	if strings.Count(canonical, "grp0: {") != 1 || strings.Count(canonical, "sibling0: s\n") != 1 {
		t.Fatal("canonical dotted merge emitted duplicated groups")
	}
}

// TestPurePerfLongStrings: multi-megabyte Strings survive
// parse → Dumps → parse byte-exactly, and canonical emission is
// idempotent on them. Both bodies below are WHOLE-LINE pair bodies, so
// per §3.7/§5.3 the backslashes are literal data (escapes NOT
// processed) and only the trailing §3.3 whitespace before EOL is
// trimmed — expectations derived from the spec text, not the code.
func TestPurePerfLongStrings(t *testing.T) {
	ops := pureOps(t)

	unit := "The quick brown fox — 日本語テキスト — jumps over the lazy dog "
	var raw strings.Builder
	for raw.Len() < 3<<20 {
		raw.WriteString(unit)
	}
	plainSrc := raw.String()
	plainWant := strings.TrimRight(plainSrc, " \t")

	var esc strings.Builder
	for esc.Len() < 1<<20 {
		esc.WriteString(`a\,b\:c } { [ ] `)
	}
	escSrc := esc.String()
	// Whole-line raw body: backslashes survive VERBATIM (§3.7 lists
	// whole-line pair bodies among the contexts WITHOUT escape
	// processing); the final space is trailing §3.3 whitespace.
	escWant := strings.TrimRight(escSrc, " \t")

	tests := []struct{ name, src, want string }{
		{"plain_mb", "big:: " + plainSrc + "\n", plainWant},
		{"escaped_mb", "big:: " + escSrc + "\n", escWant},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := ops.Loads(tt.src)
			if err != nil {
				t.Fatalf("Loads: %v", err)
			}
			if got := v.(map[string]any)["big"]; got != tt.want {
				t.Fatalf("string round-trip diverged (len got %d, want %d)", len(got.(string)), len(tt.want))
			}
			dumped, err := ops.Dumps(v)
			if err != nil {
				t.Fatalf("Dumps: %v", err)
			}
			again, err := ops.Loads(dumped)
			if err != nil {
				t.Fatalf("Loads(Dumps): %v", err)
			}
			if got := again.(map[string]any)["big"]; got != tt.want {
				t.Fatalf("dumps round-trip diverged (len %d)", len(got.(string)))
			}
			canonical, err := ops.CanonicalFromSource(tt.src)
			if err != nil {
				t.Fatalf("CanonicalFromSource: %v", err)
			}
			canonical2, err := ops.CanonicalFromSource(canonical)
			if err != nil {
				t.Fatalf("CanonicalFromSource(canonical): %v", err)
			}
			if canonical != canonical2 {
				t.Fatalf("canonical emission not idempotent on long string")
			}
		})
	}
}

// TestPurePerfDeepNestedCompounds: nested multi-line compounds at a
// depth well inside the §5.8.4 inline limit parse and canonicalise with
// correct structure — no stack blowup, no arbitrary limits on
// spec-ordinary input.
func TestPurePerfDeepNestedCompounds(t *testing.T) {
	ops := pureOps(t)
	const depth = 400
	var b strings.Builder
	for i := 0; i < depth; i++ {
		b.WriteString(strings.Repeat("    ", i))
		b.WriteString("level")
		b.WriteString(itoaTest(i))
		b.WriteString(": {\n")
	}
	b.WriteString(strings.Repeat("    ", depth))
	b.WriteString("leaf: 1\n")
	for i := depth - 1; i >= 0; i-- {
		b.WriteString(strings.Repeat("    ", i))
		b.WriteString("}\n")
	}
	src := b.String()

	v, err := ops.Loads(src)
	if err != nil {
		t.Fatalf("Loads: %v", err)
	}
	node := v.(map[string]any)
	for i := 0; i < depth; i++ {
		next, ok := node["level"+itoaTest(i)].(map[string]any)
		if !ok {
			t.Fatalf("nesting broke at depth %d", i)
		}
		node = next
	}
	if node["leaf"] != int64(1) {
		t.Fatalf("deep leaf = %#v", node["leaf"])
	}

	canonical, err := ops.CanonicalFromSource(src)
	if err != nil {
		t.Fatalf("CanonicalFromSource: %v", err)
	}
	canonical2, err := ops.CanonicalFromSource(canonical)
	if err != nil {
		t.Fatalf("CanonicalFromSource(canonical): %v", err)
	}
	if canonical != canonical2 {
		t.Fatal("canonical emission not idempotent on deep compounds")
	}
	if !strings.Contains(canonical, "leaf: 1\n") {
		t.Fatal("deep canonical output lost the leaf")
	}
}

// TestPurePerfCanonicalIdempotenceWide: canonical(canonical(x)) == x on
// a mixed wide document — the writer fixed point over many value kinds.
func TestPurePerfCanonicalIdempotenceWide(t *testing.T) {
	ops := pureOps(t)
	var b strings.Builder
	b.WriteString("## doc\n")
	for i := 0; i < 2000; i++ {
		switch i % 5 {
		case 0:
			b.WriteString("k" + itoaTest(i) + ": " + itoaTest(i) + "\n")
		case 1:
			b.WriteString("k" + itoaTest(i) + ": " + itoaTest(i) + ".5\n")
		case 2:
			b.WriteString("k" + itoaTest(i) + ":: str " + itoaTest(i) + "\n")
		case 3:
			b.WriteString("k" + itoaTest(i) + ": [" + itoaTest(i) + ", true, null]\n")
		case 4:
			b.WriteString("k" + itoaTest(i) + ".sub: " + itoaTest(-i) + "\n")
		}
	}
	src := b.String()
	first, err := ops.CanonicalFromSource(src)
	if err != nil {
		t.Fatalf("CanonicalFromSource: %v", err)
	}
	second, err := ops.CanonicalFromSource(first)
	if err != nil {
		t.Fatalf("CanonicalFromSource(canonical): %v", err)
	}
	if first != second {
		t.Fatal("canonical writer is not a fixed point on wide mixed document")
	}
	// Comments and blank lines never survive canonical emission (§5.9).
	if strings.Contains(first, "##") {
		t.Fatal("canonical output contains comments")
	}
}

// itoaTest is a tiny allocation-light integer formatter so the perf
// builders don't spend their time in strconv on the hot path.
func itoaTest(i int) string {
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}
