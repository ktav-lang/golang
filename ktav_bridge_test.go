package ktav_test

// Public-bridge safety through ktav.New(BackendPure) operations:
// toTagged's active-path cycle detection (structured error, no
// panic/hang), DAG sharing, decodeJSON trailing-data rejection, the
// needsRawMarker keyword/shortcut guards (§5.2 — a String body that
// would re-classify must be emitted with the `::` raw marker and
// re-parse as the SAME String), and depth at honest output sizes
// (see pure_depth_public_test.go).

import (
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"

	ktav "github.com/ktav-lang/golang"
	"github.com/ktav-lang/golang/pure"
)

func bridgeOps(t *testing.T) *pure.Operations {
	t.Helper()
	return pure.New()
}

func TestBridgeRootPureCompatibilityParityAndErrorIdentity(t *testing.T) {
	legacy, err := ktav.New(ktav.BackendPure)
	if err != nil {
		t.Fatal(err)
	}
	modern := pure.New()
	src := "a: 1\n"
	for _, call := range []struct {
		name           string
		legacy, modern func() (any, error)
	}{
		{"Loads", func() (any, error) { return legacy.Loads(src) }, func() (any, error) { return modern.Loads(src) }},
		{"LoadsStrict", func() (any, error) { return legacy.LoadsStrict(src) }, func() (any, error) { return modern.LoadsStrict(src) }},
		{"LoadsInto", func() (any, error) { target := map[string]any{}; return target, legacy.LoadsInto(src, &target) }, func() (any, error) { target := map[string]any{}; return target, modern.LoadsInto(src, &target) }},
	} {
		a, ea := call.legacy()
		b, eb := call.modern()
		if ea != nil || eb != nil || !structEqual(a, b) {
			t.Fatalf("%s parity: legacy=%#v/%v pure=%#v/%v", call.name, a, ea, b, eb)
		}
	}
	for _, call := range []struct {
		name           string
		legacy, modern func() (string, error)
	}{
		{"Dumps", func() (string, error) { return legacy.Dumps(map[string]any{"a": int64(1)}) }, func() (string, error) { return modern.Dumps(map[string]any{"a": int64(1)}) }},
		{"DumpsForceStrings", func() (string, error) { return legacy.DumpsForceStrings(map[string]any{"a": int64(1)}) }, func() (string, error) { return modern.DumpsForceStrings(map[string]any{"a": int64(1)}) }},
		{"EmitCanonical", func() (string, error) { return legacy.EmitCanonical(map[string]any{"a": int64(1)}) }, func() (string, error) { return modern.EmitCanonical(map[string]any{"a": int64(1)}) }},
		{"FormatSource", func() (string, error) { return legacy.FormatSource("a:  1\n") }, func() (string, error) { return modern.FormatSource("a:  1\n") }},
		{"CanonicalFromSource", func() (string, error) { return legacy.CanonicalFromSource(src) }, func() (string, error) { return modern.CanonicalFromSource(src) }},
	} {
		a, ea := call.legacy()
		b, eb := call.modern()
		if ea != nil || eb != nil || a != b {
			t.Fatalf("%s parity: %q/%v vs %q/%v", call.name, a, ea, b, eb)
		}
	}
	_, legacyErr := legacy.Loads("anchor: ok\njust text\n")
	_, pureErr := modern.Loads("anchor: ok\njust text\n")
	assertErrorParity(t, "malformed source", legacyErr, pureErr)
	assertErrorFields(t, "malformed source", legacyErr, "MissingSeparator", "", "", "", nil, true, false)
	_, legacyStrictErr := legacy.LoadsStrict("value: 1.10\n")
	_, pureStrictErr := modern.LoadsStrict("value: 1.10\n")
	assertErrorParity(t, "strict lossy payload", legacyStrictErr, pureStrictErr)
	assertErrorFields(t, "strict lossy payload", legacyStrictErr, "LossyScalar", "", "1.10", "1.1", nil, true, false)
	for _, test := range []struct {
		name      string
		value     any
		class     string
		reason    string
		path      []string
		pathIsNil bool
	}{
		{"empty-key", map[string]any{"": int64(1)}, "UnrepresentableAt", "EmptyKeyName", []string{""}, false},
		{"non-finite-float", map[string]any{"value": math.NaN()}, "UnrepresentableAt", "NonFiniteFloat", []string{"value"}, false},
		{"scalar-root", true, "UnrepresentableAt", "ScalarRoot", []string{}, false},
	} {
		_, le := legacy.Dumps(test.value)
		_, pe := modern.Dumps(test.value)
		assertErrorParity(t, test.name, le, pe)
		assertErrorFields(t, test.name, le, test.class, test.reason, "", "", test.path, test.pathIsNil, true)
	}
}

func assertErrorFields(t *testing.T, name string, err error, class, reason, body, canonical string, path []string, pathIsNil, spanIsNil bool) {
	t.Helper()
	var got *ktav.Error
	if err == nil || !errors.As(err, &got) || got == nil {
		t.Fatalf("%s: expected structured error, got %v", name, err)
	}
	if class != "" && got.Class != class || got.Reason != reason || got.Body != body || got.Canonical != canonical {
		t.Fatalf("%s: class/reason/body/canonical = %q/%q/%q/%q, want %q/%q/%q/%q", name, got.Class, got.Reason, got.Body, got.Canonical, class, reason, body, canonical)
	}
	if (got.Path == nil) != pathIsNil || !reflect.DeepEqual(got.Path, path) {
		t.Fatalf("%s: Path = %#v (nil=%t), want %#v (nil=%t)", name, got.Path, got.Path == nil, path, pathIsNil)
	}
	if (got.Span == nil) != spanIsNil {
		t.Fatalf("%s: Span = %#v (nil=%t), want nil=%t", name, got.Span, got.Span == nil, spanIsNil)
	}
	if got.Span != nil && got.Span.Start >= got.Span.End {
		t.Fatalf("%s: Span = %#v, want nonempty range", name, got.Span)
	}
}

func assertErrorParity(t *testing.T, name string, legacyErr, pureErr error) {
	t.Helper()
	if legacyErr == nil || pureErr == nil {
		t.Fatalf("%s: expected errors, got legacy=%v pure=%v", name, legacyErr, pureErr)
	}
	var legacyRoot *ktav.Error
	var legacyPure *pure.Error
	var modernRoot *ktav.Error
	var modernPure *pure.Error
	if !errors.As(legacyErr, &legacyRoot) || !errors.As(legacyErr, &legacyPure) || !errors.As(pureErr, &modernRoot) || !errors.As(pureErr, &modernPure) {
		t.Fatalf("%s: errors must resolve through both aliases: legacy=%T pure=%T", name, legacyErr, pureErr)
	}
	if legacyRoot != legacyPure || modernRoot != modernPure {
		t.Fatalf("%s: aliases did not preserve actual error identity", name)
	}
	if !reflect.DeepEqual(legacyRoot, modernRoot) {
		t.Fatalf("%s: full error fields differ: legacy=%#v pure=%#v", name, legacyRoot, modernRoot)
	}
}

// writer entry point.
func wantCyclic(t *testing.T, name string, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s(cyclic) = ok, want structured error", name)
	}
	var e *ktav.Error
	if !errors.As(err, &e) {
		t.Fatalf("%s(cyclic) err = %T (%v), want *ktav.Error", name, err, err)
	}
	if e.Class != "Message" || !strings.Contains(e.Error(), "cyclic") {
		t.Fatalf("%s(cyclic) = %q (class %q), want Message/cyclic", name, e.Error(), e.Class)
	}
}

// TestBridgeCyclicMapRejected: a self-referential map must surface as
// a structured *ktav.Error on every writer entry point.
func TestBridgeCyclicMapRejected(t *testing.T) {
	ops := bridgeOps(t)
	m := map[string]any{"a": int64(1)}
	m["self"] = m
	for _, call := range []struct {
		name string
		fn   func() (string, error)
	}{
		{"Dumps", func() (string, error) { return ops.Dumps(m) }},
		{"DumpsForceStrings", func() (string, error) { return ops.DumpsForceStrings(m) }},
		{"EmitCanonical", func() (string, error) { return ops.EmitCanonical(m) }},
	} {
		t.Run(call.name, func(t *testing.T) {
			out, err := call.fn()
			if err == nil {
				t.Fatalf("%s(cyclic map) = %q, want structured error", call.name, out)
			}
			wantCyclic(t, call.name, err)
		})
	}
}

// TestBridgeCyclicSliceRejected: same contract for a self-referential
// slice.
func TestBridgeCyclicSliceRejected(t *testing.T) {
	ops := bridgeOps(t)
	s := make([]any, 1)
	s[0] = s
	out, err := ops.Dumps(map[string]any{"k": s})
	if err == nil {
		t.Fatalf("Dumps(cyclic slice) = %q, want structured error", out)
	}
	wantCyclic(t, "Dumps", err)
}

// TestBridgeTwoContainerCycleRejected: THE regression the old
// pending-count scheme failed — starting b decremented a's counter, so
// a→b→a looped forever. The active-path ENTER/EXIT scheme must reject
// it on the first revisit.
func TestBridgeTwoContainerCycleRejected(t *testing.T) {
	ops := bridgeOps(t)
	a := map[string]any{}
	b := map[string]any{}
	a["b"] = b
	b["a"] = a
	out, err := ops.EmitCanonical(a)
	if err == nil {
		t.Fatalf("EmitCanonical(a→b→a) = %q, want structured error", out)
	}
	wantCyclic(t, "EmitCanonical", err)
	// same through the slice flavours
	s1 := make([]any, 1)
	s2 := make([]any, 1)
	s1[0] = s2
	s2[0] = s1
	_, err = ops.Dumps([]any{s1})
	if err == nil {
		t.Fatal("Dumps(slice→slice cycle) = ok, want structured error")
	}
	wantCyclic(t, "Dumps", err)
}

// TestBridgeThreeNodeCycleRejected: a→b→c→a.
func TestBridgeThreeNodeCycleRejected(t *testing.T) {
	ops := bridgeOps(t)
	a := map[string]any{}
	b := map[string]any{}
	c := map[string]any{}
	a["b"] = b
	b["c"] = c
	c["a"] = a
	_, err := ops.EmitCanonical(a)
	if err == nil {
		t.Fatal("EmitCanonical(3-cycle) = ok, want structured error")
	}
	wantCyclic(t, "EmitCanonical", err)
}

// TestBridgeDAGDiamondWorks: the SAME submap referenced from two
// sibling branches is not on any single active path twice — encoding
// must succeed and BOTH branches must carry the shared value EXACTLY
// (structEqual against the full expected value after a round trip
// through Loads, not a substring probe).
func TestBridgeDAGDiamondWorks(t *testing.T) {
	ops := bridgeOps(t)
	shared := map[string]any{"x": int64(1)}
	m := map[string]any{"a": shared, "b": shared}
	out, err := ops.EmitCanonical(m)
	if err != nil {
		t.Fatalf("EmitCanonical(DAG diamond): %v", err)
	}
	back, err := ops.Loads(out)
	if err != nil {
		t.Fatalf("Loads(EmitCanonical output %q): %v", out, err)
	}
	want := map[string]any{
		"a": map[string]any{"x": int64(1)},
		"b": map[string]any{"x": int64(1)},
	}
	if !structEqual(back, want) {
		t.Fatalf("DAG diamond round trip = %#v, want %#v (both branches exact)", back, want)
	}
	// and each branch carries the SAME shared content as the original
	// node (independent negative control against one-branch-only fix)
	root := back.(map[string]any)
	if !structEqual(root["a"], shared) || !structEqual(root["b"], shared) {
		t.Fatalf("branch mismatch: a=%#v b=%#v shared=%#v", root["a"], root["b"], shared)
	}
}

// TestBridgeDistinctEmptySlicesEncode: two SEPARATELY-ALLOCATED empty
// slices are distinct containers (different pointers) — the slice-view
// identity key (kind, ptr, len, cap) must not conflate or reject them.
func TestBridgeDistinctEmptySlicesEncode(t *testing.T) {
	ops := bridgeOps(t)
	e1 := []any{}
	e2 := []any{}
	if e1 == nil || e2 == nil {
		t.Fatal("test bug: slices must be non-nil")
	}
	m := map[string]any{"a": e1, "b": e2}
	out, err := ops.EmitCanonical(m)
	if err != nil {
		t.Fatalf("EmitCanonical(two empty slices): %v", err)
	}
	back, err := ops.Loads(out)
	if err != nil {
		t.Fatalf("Loads(%q): %v", out, err)
	}
	want := map[string]any{"a": []any{}, "b": []any{}}
	if !structEqual(back, want) {
		t.Fatalf("two empty slices round trip = %#v, want %#v", back, want)
	}
}

// TestBridgeOverlappingSliceViewsEncode: two OVERLAPPING windows of
// ONE backing array have different (len, cap) AND different data, yet
// the SAME base pointer — pointer-only identity would conflate them.
// The (kind, ptr, len, cap) identity key must encode each window
// independently, and both must round-trip to their exact window
// values (structEqual against hand-derived expectations, not
// substring probes). The cycles this must NOT relax stay covered by
// TestBridgeTwoContainerCycleRejected (map and slice flavours).
func TestBridgeOverlappingSliceViewsEncode(t *testing.T) {
	ops := bridgeOps(t)
	shared := map[string]any{"deep": []any{"leaf"}}
	back := []any{
		map[string]any{"n": int64(1), "sub": shared},
		map[string]any{"n": int64(2), "sub": shared},
		map[string]any{"n": int64(3)},
	}
	v1 := back[0:2]
	v2 := back[1:3]
	if structEqual(v1[0], v2[0]) {
		t.Fatal("test bug: windows must hold different data")
	}
	m := map[string]any{"v1": v1, "v2": v2}
	out, err := ops.EmitCanonical(m)
	if err != nil {
		t.Fatalf("EmitCanonical(overlapping views): %v", err)
	}
	got, err := ops.Loads(out)
	if err != nil {
		t.Fatalf("Loads(%q): %v", out, err)
	}
	want := map[string]any{
		"v1": []any{
			map[string]any{"n": int64(1), "sub": map[string]any{"deep": []any{"leaf"}}},
			map[string]any{"n": int64(2), "sub": map[string]any{"deep": []any{"leaf"}}},
		},
		"v2": []any{
			map[string]any{"n": int64(2), "sub": map[string]any{"deep": []any{"leaf"}}},
			map[string]any{"n": int64(3)},
		},
	}
	if !structEqual(got, want) {
		t.Fatalf("overlapping views round trip = %#v, want %#v (both windows exact)", got, want)
	}
}

// TestBridgeKeywordStringsForceRawMarker: §5.2 — a Go String whose body
// would re-classify as keyword/compound/multi-line-opener ("null",
// "true", "false", "(", "((", "()", "(())") MUST be emitted with the
// `::` raw marker so it RE-PARSES as the same String byte-content.
// Dumps (plain) likewise renders through needsRawMarker. The valid-i64
// control pins that an integer stays an Integer.
func TestBridgeKeywordStringsForceRawMarker(t *testing.T) {
	ops := bridgeOps(t)
	bodies := []string{"null", "true", "false", "(", "((", "()", "(())"}
	for _, body := range bodies {
		m := map[string]any{"k": body}
		for _, call := range []struct {
			name string
			fn   func() (string, error)
		}{
			{"Dumps", func() (string, error) { return ops.Dumps(m) }},
			{"DumpsForceStrings", func() (string, error) { return ops.DumpsForceStrings(m) }},
		} {
			t.Run(call.name+"/"+body, func(t *testing.T) {
				out, err := call.fn()
				if err != nil {
					t.Fatalf("%s(String %q): %v", call.name, body, err)
				}
				wantLine := "k:: " + body + "\n"
				if !strings.Contains(out, wantLine) {
					t.Fatalf("%s(String %q) = %q, want raw-marker line %q", call.name, body, out, wantLine)
				}
				back, err := ops.Loads(strings.TrimSuffix(out, "\n"))
				if err != nil {
					t.Fatalf("Loads(%q): %v", out, err)
				}
				root, ok := back.(map[string]any)
				if !ok {
					t.Fatalf("Loads(%q) = %T, want map", out, back)
				}
				s, ok := root["k"].(string)
				if !ok {
					t.Fatalf("String %q re-parsed as %T (%#v), want string", body, root["k"], root["k"])
				}
				if s != body {
					t.Fatalf("String %q re-parsed as %q", body, s)
				}
			})
		}
	}
	// Control: force-string of a BOOL must re-parse as String "true",
	// and a real integer stays an Integer.
	fs, err := ops.DumpsForceStrings(map[string]any{"k": true})
	if err != nil {
		t.Fatalf("DumpsForceStrings(bool): %v", err)
	}
	if !strings.Contains(fs, "k:: true\n") {
		t.Fatalf("DumpsForceStrings(bool) = %q, want `k:: true`", fs)
	}
	back, err := ops.Loads(strings.TrimSuffix(fs, "\n"))
	if err != nil {
		t.Fatalf("Loads(%q): %v", fs, err)
	}
	if s, ok := back.(map[string]any)["k"].(string); !ok || s != "true" {
		t.Fatalf("force-stringed bool re-parsed as %#v, want String \"true\"", back.(map[string]any)["k"])
	}
	n, err := ops.Loads("k: 12\n")
	if err != nil {
		t.Fatalf("Loads(int): %v", err)
	}
	if v, ok := n.(map[string]any)["k"].(int64); !ok || v != 12 {
		t.Fatalf("integer control = %#v, want int64 12", n.(map[string]any)["k"])
	}
}
