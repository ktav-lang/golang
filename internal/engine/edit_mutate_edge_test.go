package engine

import "testing"

func TestRemoveRootArrayLeadingTriviaExactAndSemantic(t *testing.T) {
	source := "\xef\xbb\xbf## [leading]\n\n\u00a0[1, 2]\n## ] trailing\n"
	want := "\xef\xbb\xbf## [leading]\n\n\u00a0[2]\n## ] trailing\n"
	wantTree := &pvalue{kind: kArr, arr: []*pvalue{pvInt("2")}}
	for _, tc := range []struct {
		name string
		call func() (string, error)
	}{
		{"textual", func() (string, error) { return op().RemoveValue(source, "0") }},
		{"segments", func() (string, error) { return op().RemoveValueSegments(source, []string{"0"}) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.call()
			if err != nil || got != want {
				t.Fatalf("Remove = %q, %v; want %q", got, err, want)
			}
			doc, _, parseErr := parseDoc(got, false, false)
			if parseErr != nil || !pvEqual(doc.(*pvalue), wantTree) {
				t.Fatalf("semantic tree mismatch: %v", parseErr)
			}
		})
	}
}

func TestInlineDottedLeafWithoutParentSlotUnsupported(t *testing.T) {
	source := "cfg: {a.b: 1, a.c: 2}\n"
	for _, tc := range []struct {
		name string
		call func() (string, error)
	}{
		{"remove/textual", func() (string, error) { return op().RemoveValue(source, "cfg.a.b") }},
		{"remove/segments", func() (string, error) { return op().RemoveValueSegments(source, []string{"cfg", "a", "b"}) }},
		{"insert/textual", func() (string, error) { return op().InsertValue(source, "cfg.a.d", int64(3)) }},
		{"insert/segments", func() (string, error) { return op().InsertValueSegments(source, []string{"cfg", "a", "d"}, int64(3)) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := tc.call(); err == nil {
				t.Fatal("expected EditUnsupported")
			} else if e, ok := err.(*Error); !ok || e.Class != "EditUnsupported" {
				t.Fatalf("error = %#v; want EditUnsupported", err)
			}
		})
	}
}
