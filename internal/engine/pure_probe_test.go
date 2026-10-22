package engine

// Deterministic OPERATION-COUNT regression checks, driven by the
// package-internal nil scanProbe hook (pure_parser.go). HONEST COST
// (see golang-plan.md §13): when the hook is nil, production pays one
// predictable nil-test per coarse event (dotted descent/reopen, writer
// task, numeric-form scan) PLUS the byte-count arithmetic inside the
// two classification functions (needsRawMarker/needsRawMarkerContent
// and the grammar scanners) — that arithmetic is bounded by one pass
// over the scanned body and is kept unconditional on purpose: removing
// it would fork the classification's semantics between probed and
// unprobed builds. These tests install a counting closure and assert
// bounds on REAL operation counts across two input sizes (linear
// scaling: count(4x) <= 5*count(x) for a 4x size step), never
// wall-clock milliseconds and never a speedup. Depth-safety tests at
// the bottom pin the iterative-writer strategy: deep dotted keys
// (§5.3.2, unbounded by spec §5.8.4's inline-only cap and unbounded in
// the Rust reference too) must parse and canonicalise without
// recursion-depth failures.

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
// that LOOKS numeric triggers ONE bounded classification pass — the
// bytes examined (numeric_scan / rawmarker_scan, reported per BYTE at
// each scan's exit) never go superlinear in body size. Honest bounds:
// numeric_scan must scale linearly (4x size ⇒ ≤5x bytes) and
// rawmarker_scan must stay bounded by ONE pass over the body per
// emission (≤2x body bytes at the big step — no per-byte re-parse per
// emission). The body is an ORDINARY `big:` value, so §5.2
// classification actually runs on it (a `big::` raw-marker body would
// bypass the numeric grammar).
func TestProbeNumericLookingStringNoRescan(t *testing.T) {
	build := func(units int) string {
		return "big: " + strings.Repeat("1234567890123456789012345678901234567890", units) + "\n"
	}
	count := probeCounts(t)
	smallScans, smallBytes := 0, 0
	bigBody := 0
	for i, units := range []int{25000, 100000} { // ~1MB, ~4MB
		if i == 1 {
			smallScans, smallBytes = count["numeric_scan"], count["rawmarker_scan"]
			count["numeric_scan"], count["rawmarker_scan"] = 0, 0
		}
		src := build(units)
		if i == 1 {
			bigBody = len(src)
		}
		tagged, err := pureLoadsTagged([]byte(src))
		if err != nil {
			t.Fatalf("loads(%d units): %v", units, err)
		}
		if _, err := pureRenderTagged(tagged, modeCanonical, false); err != nil {
			t.Fatalf("render(%d units): %v", units, err)
		}
	}
	bigScans, bigBytes := count["numeric_scan"], count["rawmarker_scan"]
	if smallBytes == 0 || bigScans == 0 {
		t.Fatalf("numeric_scan reported no examined bytes: small %d, big %d", smallBytes, bigScans)
	}
	if bigScans > 5*smallScans {
		t.Fatalf("numeric_scan bytes not linear: 1MB→%d, 4MB→%d (want ≤ %d)", smallScans, bigScans, 5*smallScans)
	}
	if bigBytes > 2*bigBody {
		t.Fatalf("rawmarker_scan not bounded by one body pass: %d bytes for a %d-byte body", bigBytes, bigBody)
	}
}

// TestProbeLongDigitOverflowBounded: a 300-digit body matches the
// §3.6 integer grammar but overflows i64; classification must bail in
// ONE bounded pass (parseDecimalInt aborts at the first u64 overflow
// via mulAddU64 — no unbounded big.Int growth) and the value must land
// on the tagged wire as a PLAIN JSON String (no $i tag), not an error
// or a runaway. Control: a valid i64 literal MUST carry the $i tag.
func TestProbeLongDigitOverflowBounded(t *testing.T) {
	big := strings.Repeat("9", 300)
	wire, err := pureLoadsTagged([]byte("n: " + big + "\n"))
	if err != nil {
		t.Fatalf("loads(300 digits): %v", err)
	}
	got := string(wire)
	if !strings.Contains(got, `"`+big+`"`) {
		t.Fatalf("300-digit wire lost the String body: %.120s", got)
	}
	if strings.Contains(got, `"$i"`) {
		t.Fatalf("300-digit body wrongly tagged as integer: %.120s", got)
	}
	good, err := pureLoadsTagged([]byte("n: 1234567890123456789\n"))
	if err != nil {
		t.Fatalf("loads(in-i64 literal): %v", err)
	}
	if !strings.Contains(string(good), `{"$i":"1234567890123456789"}`) {
		t.Fatalf("valid i64 literal not tagged as integer: %s", good)
	}
}
