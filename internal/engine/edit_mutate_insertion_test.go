package engine

import "testing"

func TestInsertMutationInlineLayoutRegressionsExactAndSemantic(t *testing.T) {
	cases := []struct {
		name, source, path, value, want string
		semantic                        *pvalue
	}{
		{
			name:   "minimum-nested-inline-expansion-preserves-sibling",
			source: "a: {x: {p: 1}, y: {r: 2}}\n", path: "a.x.q", value: "line1\nline2",
			want:     "a: {\n    x: {\n        p: 1\n        q: ((\nline1\nline2\n        ))\n    }\n    y: {r: 2}\n}\n",
			semantic: objectWithEntries(oentry{"a", objectWithEntries(oentry{"x", objectWithEntries(oentry{"p", pvInt("1")}, oentry{"q", pvStr("line1\nline2")})}, oentry{"y", objectWithEntries(oentry{"r", pvInt("2")})})}),
		},
		{
			name:   "closing-marker-inline-fallback",
			source: "a: {x: 1}\n", path: "a.y", value: "line1\n))\nline2",
			want:     "a: {x: 1, y: line1\\n))\\nline2}\n",
			semantic: objectWithEntries(oentry{"a", objectWithEntries(oentry{"x", pvInt("1")}, oentry{"y", pvStr("line1\n))\nline2")})}),
		},
		{
			name:   "cr-and-closing-marker-inline-fallback",
			source: "a: {x: 1}\n", path: "a.y", value: "a\r\nb",
			want:     "a: {x: 1, y: a\\r\\nb}\n",
			semantic: objectWithEntries(oentry{"a", objectWithEntries(oentry{"x", pvInt("1")}, oentry{"y", pvStr("a\r\nb")})}),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := op().InsertValue(tc.source, tc.path, tc.value)
			if err != nil || got != tc.want {
				t.Fatalf("InsertValue = %q, %v; want %q", got, err, tc.want)
			}
			doc, _, parseErr := parseDoc(got, false, false)
			if parseErr != nil || !pvEqual(doc.(*pvalue), tc.semantic) {
				t.Fatalf("semantic tree mismatch: %v", parseErr)
			}
		})
	}
}
func TestInsertInlineEmptyExact(t *testing.T) {
	for _, tc := range []struct{ source, key, want string }{
		{"{  }", "x", "{x: 1}"},
		{"{a: {  }}", "a.x", "{a: {x: 1}}"},
	} {
		got, err := op().InsertValue(tc.source, tc.key, int64(1))
		if err != nil || got != tc.want {
			t.Fatalf("InsertValue(%q, %q) = %q, %v; want %q", tc.source, tc.key, got, err, tc.want)
		}
	}
}

func TestInsertInlineRootPreservesSurroundingWhitespaceAndTrailingComma(t *testing.T) {
	for _, tc := range []struct{ source, want string }{
		{"  {a: 1}  \n", "  {a: 1, b: 2}  \n"},
		{"{a: 1,}", "{a: 1, b: 2,}"},
		{"{a: value\\,}", "{a: value\\,, b: 2}"},
		{"{a: 1,\u00a0}", "{a: 1, b: 2,\u00a0}"},
	} {
		got, err := op().InsertValue(tc.source, "b", int64(2))
		if err != nil || got != tc.want {
			t.Fatalf("InsertValue(%q) = %q, %v; want %q", tc.source, got, err, tc.want)
		}
	}
}
func TestInsertInlineRootLeadingTriviaExactAndSemantic(t *testing.T) {
	cases := []struct{ name, source, want string }{
		{"comments", "## {leading}\n{x:1}\n## }tail\n", "## {leading}\n{x:1, b: 2}\n## }tail\n"},
		{"blank+BOM+Unicode whitespace", "\xef\xbb\xbf\n\n\u00a0{x:1}\n## }tail\n", "\xef\xbb\xbf\n\n\u00a0{x:1, b: 2}\n## }tail\n"},
	}
	for _, tc := range cases {
		for _, viaSegments := range []bool{false, true} {
			name := tc.name + "/textual"
			if viaSegments {
				name = tc.name + "/segments"
			}
			t.Run(name, func(t *testing.T) {
				var got string
				var err error
				if viaSegments {
					got, err = op().InsertValueSegments(tc.source, []string{"b"}, int64(2))
				} else {
					got, err = op().InsertValue(tc.source, "b", int64(2))
				}
				if err != nil || got != tc.want {
					t.Fatalf("Insert = %q, %v; want %q", got, err, tc.want)
				}
				doc, _, parseErr := parseDoc(got, false, false)
				wantTree := objectWithEntries(oentry{"x", pvInt("1")}, oentry{"b", pvInt("2")})
				if parseErr != nil || !pvEqual(doc.(*pvalue), wantTree) {
					t.Fatalf("semantic tree mismatch: %v", parseErr)
				}
			})
		}
	}
}

func TestInsertBlockAwarePairRenderingExact(t *testing.T) {
	for _, tc := range []struct{ source, key, value, want string }{
		{"a: 1\r\n", "001", "numeric", "a: 1\r\n001: numeric\r\n"},
		{"{\r\n    x: 1\r\n}\r\n", "y", "line1\nline2", "{\r\n    x: 1\r\n    y: ((\r\nline1\r\nline2\r\n    ))\r\n}\r\n"},
	} {
		got, err := op().InsertValue(tc.source, tc.key, tc.value)
		if err != nil || got != tc.want {
			t.Fatalf("InsertValue = %q, %v; want %q", got, err, tc.want)
		}
	}
}

func TestInsertBlockPairRawSeparatorAndEmptyRootIndentExact(t *testing.T) {
	for _, tc := range []struct{ source, key, want string }{
		{"a: 1\r\n", "n", "a: 1\r\nn:: 0007\r\n"},
		{"{\r\n}\r\n", "n", "{\r\n    n:: 0007\r\n}\r\n"},
	} {
		got, err := op().InsertValue(tc.source, tc.key, "0007")
		if err != nil || got != tc.want {
			t.Fatalf("InsertValue(%q) = %q, %v; want %q", tc.source, got, err, tc.want)
		}
	}
}

func TestInsertValuePreservesRootBOMExact(t *testing.T) {
	got, err := op().InsertValue("\xef\xbb\xbf{}", "n", int64(1))
	want := "\xef\xbb\xbf{n: 1}"
	if err != nil || got != want {
		t.Fatalf("InsertValue(BOM root) = %q, %v; want %q", got, err, want)
	}
}

func TestInsertExplicitEmptyBlockCommentOpenerExact(t *testing.T) {
	got, err := op().InsertValue("{\r\n## comment {\r\n}\r\n", "x", int64(1))
	want := "{\r\n## comment {\r\n    x: 1\r\n}\r\n"
	if err != nil || got != want {
		t.Fatalf("InsertValue = %q, %v; want %q", got, err, want)
	}
}

func TestInsertImplicitRootAfterTrailingCommentExact(t *testing.T) {
	got, err := op().InsertValue("a: 1\n## tail\n", "b", int64(2))
	want := "a: 1\n## tail\nb: 2\n"
	if err != nil || got != want {
		t.Fatalf("InsertValue(trailing comment) = %q, %v; want %q", got, err, want)
	}
	doc, _, parseErr := parseDoc(got, false, false)
	if parseErr != nil {
		t.Fatalf("parse inserted source: %v", parseErr)
	}
	wantTree := objectWithEntries(oentry{"a", pvInt("1")}, oentry{"b", pvInt("2")})
	if !pvEqual(doc.(*pvalue), wantTree) {
		t.Fatalf("semantic tree mismatch: got %#v", doc)
	}
}
func TestInsertRootBOMOnlyAndFinalEOLExact(t *testing.T) {
	for _, tc := range []struct{ source, want string }{
		{"\xef\xbb\xbf", "\xef\xbb\xbf\nkey: 1"},
		{"a: 1\r\n", "a: 1\r\nkey: 1\r\n"},
		{"a: 1", "a: 1\nkey: 1"},
	} {
		got, err := op().InsertValue(tc.source, "key", int64(1))
		if err != nil || got != tc.want {
			t.Fatalf("InsertValue(%q) = %q, %v; want %q", tc.source, got, err, tc.want)
		}
	}
}

func TestInsertCRAndDoubleCloserFallbackExact(t *testing.T) {
	for _, tc := range []struct{ value, want string }{
		{"a\rb", "a: {x: 1, y: a\\rb}\n"},
		{"a\n))\nb", "a: {x: 1, y: a\\n))\\nb}\n"},
	} {
		got, err := op().InsertValue("a: {x: 1}\n", "a.y", tc.value)
		if err != nil || got != tc.want {
			t.Fatalf("InsertValue(%q) = %q, %v; want %q", tc.value, got, err, tc.want)
		}
	}
}
