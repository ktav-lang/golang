package ktav

// Deterministic OPERATION-COUNT regression checks, driven by the
// package-internal nil scanProbe hook (pure_parser.go). The hook is nil
// in production — its only cost is one predictable nil-test per coarse
// event (dotted descent/reopen, writer task, numeric-form scan). These
// tests install a counting closure and assert bounds on REAL operation
// counts across two input sizes (linear scaling: count(4x) <= 5*count(x)
// for a 4x size step), never wall-clock milliseconds and never a
// speedup. Depth-safety tests at the bottom pin the iterative-writer
// strategy: deep dotted keys (§5.3.2, unbounded by spec §5.8.4's
// inline-only cap and unbounded in the Rust reference too) must parse
// and canonicalise without recursion-depth failures.

import (
	"strings"
	"testing"
)

// probeCounts installs a counting scanProbe and restores the nil hook
// on cleanup.
func probeCounts(t *testing.T) map[string]int {
	t.Helper()
	counts := map[string]int{}
	prev := scanProbe
	scanProbe = func(kind string, n int) { counts[kind] += n }
	t.Cleanup(func() { scanProbe = prev })
	return counts
}

// wideDoc builds an n-pair flat object document.
func wideDoc(n int) string {
	var b strings.Builder
	for i := 0; i < n; i++ {
		b.WriteString("key")
		b.WriteString(itoa(i))
		b.WriteString(": ")
		b.WriteString(itoa(i))
		b.WriteByte('\n')
	}
	return b.String()
}

// dottedDoc builds pairs over p dotted prefixes reopened after
// intervening siblings — the reopen-merge hot path (§5.3.2).
func dottedDoc(p, leaves int) string {
	var b strings.Builder
	for i := 0; i < p; i++ {
		for l := 0; l < leaves; l++ {
			b.WriteString("grp")
			b.WriteString(itoa(i))
			b.WriteString(".leaf")
			b.WriteString(itoa(l))
			b.WriteString(": ")
			b.WriteString(itoa(i*leaves + l))
			b.WriteByte('\n')
			if l == 0 {
				b.WriteString("sib")
				b.WriteString(itoa(i))
				b.WriteString(": s\n")
			}
		}
	}
	return b.String()
}

// TestProbeLinearWideObjectWrite: writer work (emitted tasks) scales
// linearly with the number of pairs.
func TestProbeLinearWideObjectWrite(t *testing.T) {
	counts := probeCounts(t)
	_, err := pureCanonicalFromSource([]byte(wideDoc(2000)))
	if err != nil {
		t.Fatalf("canonical(2k): %v", err)
	}
	small := counts["write_task"]
	counts["write_task"] = 0
	_, err = pureCanonicalFromSource([]byte(wideDoc(8000)))
	if err != nil {
		t.Fatalf("canonical(8k): %v", err)
	}
	big := counts["write_task"]
	if big == 0 || big > 5*small {
		t.Fatalf("write_task not linear: 2k→%d, 8k→%d (want ≤ %d)", small, big, 5*small)
	}
}

// TestProbeLinearDottedReopenMerge: dotted descent and reopen-merge
// work scales linearly with the pair count.
func TestProbeLinearDottedReopenMerge(t *testing.T) {
	counts := probeCounts(t)
	_, err := pureCanonicalFromSource([]byte(dottedDoc(200, 10)))
	if err != nil {
		t.Fatalf("canonical(2k dotted): %v", err)
	}
	segSmall, reSmall := counts["dotted_segment"], counts["dotted_reopen"]
	counts["dotted_segment"], counts["dotted_reopen"] = 0, 0
	_, err = pureCanonicalFromSource([]byte(dottedDoc(800, 10)))
	if err != nil {
		t.Fatalf("canonical(8k dotted): %v", err)
	}
	segBig, reBig := counts["dotted_segment"], counts["dotted_reopen"]
	if segBig == 0 || segBig > 5*segSmall {
		t.Fatalf("dotted_segment not linear: %d → %d", segSmall, segBig)
	}
	if reBig == 0 || reBig > 5*reSmall {
		t.Fatalf("dotted_reopen not linear: %d → %d", reSmall, reBig)
	}
}

// TestProbeNumericLookingStringNoRescan: a megabyte-scale String body
// that LOOKS numeric triggers a constant number of numeric-form scans
// (one classification pass, one writer raw-marker pass) — never a
// per-byte re-parse. numeric_scan counts needsRawMarkerContent entries;
// rawmarker_scan counts scanned bytes and must stay proportional to
// input size, not quadratic.
func TestProbeNumericLookingStringNoRescan(t *testing.T) {
	build := func(units int) string {
		return "big:: " + strings.Repeat("1234567890123456789012345678901234567890.", units) + "\n"
	}
	count := probeCounts(t)
	smallScans, smallBytes := 0, 0
	for i, units := range []int{25000, 100000} { // ~1MB, ~4MB
		if i == 1 {
			smallScans, smallBytes = count["numeric_scan"], count["rawmarker_scan"]
			count["numeric_scan"], count["rawmarker_scan"] = 0, 0
		}
		src := build(units)
		tagged, err := pureLoadsTagged([]byte(src))
		if err != nil {
			t.Fatalf("loads(%d units): %v", units, err)
		}
		if _, err := pureRenderTagged(tagged, modeCanonical, false); err != nil {
			t.Fatalf("render(%d units): %v", units, err)
		}
	}
	bigScans, bigBytes := count["numeric_scan"], count["rawmarker_scan"]
	// The scan COUNT is a constant per emission (parse classification
	// + writer raw-marker pass) regardless of body size; the scanned
	// BYTE count stays linear in the body size.
	if smallScans == 0 || bigScans > 2*smallScans {
		t.Fatalf("numeric_scan not constant-ish: 1MB→%d, 4MB→%d", smallScans, bigScans)
	}
	if bigBytes == 0 || bigBytes > 5*smallBytes {
		t.Fatalf("rawmarker_scan bytes not linear: %d → %d", smallBytes, bigBytes)
	}
}

// TestPureDeepDottedKeysParseAndCanonical: depth ~2000 via dotted keys
// ("a.a.a…" — §5.3.2 accepts this; neither spec §5.8.4 (an INLINE-only
// cap) nor the Rust reference (MAX_INLINE_DEPTH = 128, inline compounds
// only) bounds dotted depth). Parse and CanonicalFromSource must
// complete without a stack overflow: the writer emission is iterative
// (explicit work stack), the dotted insertion is iterative, and the
// tagged-JSON bridge (marshal/ingest) is iterative as well. The output
// is compact-input canonical bytes (~8MB of indentation for depth 2000
// — memory-modest; never a 10^6-node Dumps).
func TestPureDeepDottedKeysParseAndCanonical(t *testing.T) {
	const depth = 2000
	var b strings.Builder
	b.WriteString("a")
	for i := 0; i < depth; i++ {
		b.WriteString(".a")
	}
	b.WriteString(": 1\n")
	src := b.String()

	root, _, perr := parseDoc(src, false, false)
	if perr != nil {
		t.Fatalf("parseDoc(deep dotted): %v", perr)
	}
	w := &writer{mode: modeCanonical}
	w.renderRoot(root.(*pvalue))
	if w.reason != "" {
		t.Fatalf("canonical(deep dotted) refused: %s", w.reason)
	}
	out := w.b.String()
	// The leaf line lands at nesting depth `depth` (indent 4·depth;
	// the path has depth+1 segments, root pair at indent 0).
	if !strings.HasSuffix(out, "\n"+strings.Repeat("    ", depth)+"a: 1\n") {
		t.Fatalf("deep canonical output missing depth-%d leaf (len %d)", depth, len(out))
	}

	// The Loads bridge must survive the same depth: tagged marshal and
	// wire ingest are both frame-stack iterative.
	tagged, err := pureLoadsTagged([]byte(src))
	if err != nil {
		t.Fatalf("pureLoadsTagged(deep dotted): %v", err)
	}
	if _, err := pureRenderTagged(tagged, modeCanonical, false); err != nil {
		t.Fatalf("pureRenderTagged(deep dotted): %v", err)
	}
}
