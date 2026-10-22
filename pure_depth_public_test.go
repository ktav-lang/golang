package ktav_test

// Public depth-safety coverage at honest output sizes. Complexity
// claim (see golang-plan.md §13): every walk here is O(input + output)
// — the dotted/canonical writers and the tagged-JSON bridge are
// work-stack iterative, so tests only ever assert bounded outputs.
// The former depth-5000 Dumps/EmitCanonical test (~100 MB of
// indentation, quadratic-in-output) was removed as output-sensitivity
// abuse; deep values are covered by parse-only assertions instead.

import (
	"strings"
	"testing"
)

// TestPureDeepDottedKeysParseAndCanonical: depth ~200 via dotted keys
// ("a.a.a…" — §5.3.2 accepts this; neither spec §5.8.4 (an INLINE-only
// cap) nor the Rust reference (MAX_INLINE_DEPTH = 128, inline compounds
// only) bounds dotted depth). The PUBLIC Operations layer must parse
// and canonicalise without recursion-depth failure. Output stays
// bounded (~200·4-byte indents); the oracle rule is hand derived from
// `a.b: 1` → "a: {\n    b: 1\n}\n": the leaf line sits at indent 4·N
// (path depth N+1) and EXACTLY N closer lines follow it.
func TestPureDeepDottedKeysParseAndCanonical(t *testing.T) {
	ops := pureOps(t)
	const depth = 200
	var b strings.Builder
	b.WriteString("a")
	for i := 0; i < depth; i++ {
		b.WriteString(".a")
	}
	b.WriteString(": 1\n")
	src := b.String()

	v, err := ops.Loads(src)
	if err != nil {
		t.Fatalf("Loads(deep dotted): %v", err)
	}
	if _, ok := v.(map[string]any); !ok {
		t.Fatalf("Loads(deep dotted) root = %T, want map", v)
	}
	out, err := ops.CanonicalFromSource(src)
	if err != nil {
		t.Fatalf("CanonicalFromSource(deep dotted): %v", err)
	}
	leaf := strings.Repeat("    ", depth) + "a: 1\n"
	if !strings.Contains(out, leaf) {
		t.Fatalf("canonical output missing depth-%d leaf (len %d)", depth, len(out))
	}
	closers := 0
	for _, line := range strings.Split(out, "\n") {
		if strings.TrimLeft(line, " ") == "}" {
			closers++
		}
	}
	if closers != depth {
		t.Fatalf("canonical closers = %d, want %d (one per dotted segment)", closers, depth)
	}

	// The Loads/Dumps bridge must survive the same depth through the
	// public layer: tagged marshal and wire ingest are frame-stack
	// iterative on both sides.
	if _, err := ops.Dumps(v); err != nil {
		t.Fatalf("Dumps(deep dotted): %v", err)
	}
}

// TestBridgeDeepSourceLoad: deep multi-line nesting through the PUBLIC
// Loads path — decodeJSON is frame-stack iterative, so 3000 levels of
// wire nesting decode without recursion failure. PARSE ONLY: no
// canonical emission of the whole tree (that output would be
// quadratic in depth); the assertion is that the walk completes and
// the innermost array is reachable.
func TestBridgeDeepSourceLoad(t *testing.T) {
	ops := pureOps(t)
	const depth = 3000
	var b strings.Builder
	for i := 0; i < depth; i++ {
		b.WriteString("[\n")
	}
	for i := 0; i < depth; i++ {
		b.WriteString("]\n")
	}
	v, err := ops.Loads(b.String())
	if err != nil {
		t.Fatalf("Loads(depth %d): %v", depth, err)
	}
	arr, ok := v.([]any)
	if !ok {
		t.Fatalf("Loads(deep) root = %T, want []any", v)
	}
	for i := 1; i < depth; i++ {
		a, ok := arr[0].([]any)
		if !ok {
			t.Fatalf("Loads(deep) level %d = %T, want []any", i, arr[0])
		}
		arr = a
	}
	if len(arr) != 0 {
		t.Fatalf("innermost array = %v, want empty", arr)
	}
}

// TestBridgeDeepNestedMapExactEmission: a MODEST-depth (64) nested map
// through EmitCanonical must produce EXACT bytes. Hand-traced oracle:
// the canonical writer opens each single-key map with "<key>: {\n" and
// closes it with a "}" line at the key's indent (rule verified on the
// small shape `a.b: 1` → "a: {\n    b: 1\n}\n"), so 64 nested maps
// {"n": …} around {"leaf": int64(1)} emit:
//
//	n: {\n  (indent 0)
//	  n: {\n  (indent 4·i for i = 1..63)
//	    leaf: 1\n  (indent 4·64)
//	  }\n  … one closer per level, indents 4·63 down to 0
//
// Output is ~4.6 KB — bounded, and asserted exactly.
func TestBridgeDeepNestedMapExactEmission(t *testing.T) {
	ops := pureOps(t)
	const depth = 64
	var v any = map[string]any{"leaf": int64(1)}
	for i := 0; i < depth; i++ {
		v = map[string]any{"n": v}
	}
	out, err := ops.EmitCanonical(v)
	if err != nil {
		t.Fatalf("EmitCanonical(depth %d): %v", depth, err)
	}
	var want strings.Builder
	for i := 0; i < depth; i++ {
		want.WriteString(strings.Repeat("    ", i) + "n: {\n")
	}
	want.WriteString(strings.Repeat("    ", depth) + "leaf: 1\n")
	for i := depth - 1; i >= 0; i-- {
		want.WriteString(strings.Repeat("    ", i) + "}\n")
	}
	if out != want.String() {
		t.Fatalf("EmitCanonical(depth %d) mismatch\n got len %d\nwant len %d\ngot head %.200q",
			depth, len(out), want.Len(), out)
	}
}
