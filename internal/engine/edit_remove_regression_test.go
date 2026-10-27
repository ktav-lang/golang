package engine

import (
	"strings"
	"testing"
)

func objectWithEntries(entries ...oentry) *pvalue {
	m := newOmap()
	for _, e := range entries {
		m.insertLeaf(e.key, e.val)
	}
	return &pvalue{kind: kObj, obj: m}
}

func TestRemoveInlineWidthDoesNotExpandExactAndSemantic(t *testing.T) {
	source := "a: {x: 1, padding: " + strings.Repeat("p", 100) + ", y: 2}\n"
	got, err := op().RemoveValue(source, "a.y")
	want := "a: {x: 1, padding: " + strings.Repeat("p", 100) + "}\n"
	if err != nil || got != want {
		t.Fatalf("RemoveValue = %q, %v; want %q", got, err, want)
	}
	doc, _, parseErr := parseDoc(got, false, false)
	semantic := objectWithEntries(oentry{"a", objectWithEntries(oentry{"x", pvInt("1")}, oentry{"padding", pvStr(strings.Repeat("p", 100))})})
	if parseErr != nil || !pvEqual(doc.(*pvalue), semantic) {
		t.Fatalf("semantic tree mismatch: %v", parseErr)
	}
}
func TestRemoveInlineRootPreservesSurroundingWhitespace(t *testing.T) {
	source := "  {a: 1, b: 2}  \n"
	got, err := op().RemoveValue(source, "a")
	want := "  {b: 2}  \n"
	if err != nil || got != want {
		t.Fatalf("RemoveValue = %q, %v; want %q", got, err, want)
	}
}
func TestRemoveInlineRootArrayPreservesSurroundingWhitespaceAndBOM(t *testing.T) {
	for _, tc := range []struct{ name, source, want string }{
		{"whitespace", "  [1, 2]  \n", "  [2]  \n"},
		{"BOM", "\xef\xbb\xbf  [1, 2]  \n", "\xef\xbb\xbf  [2]  \n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := op().RemoveValue(tc.source, "0")
			if err != nil || got != tc.want {
				t.Fatalf("RemoveValue = %q, %v; want %q", got, err, tc.want)
			}
		})
	}
}

func TestRemoveDottedFirstLineBOMPlainCutPreservesSibling(t *testing.T) {
	source := "\xef\xbb\xbfa.b: 1\na.c: 2\n"
	got, err := op().RemoveValue(source, "a.b")
	want := "\xef\xbb\xbfa.c: 2\n"
	if err != nil || got != want {
		t.Fatalf("RemoveValue = %q, %v; want %q", got, err, want)
	}
}

func TestRemoveDottedFirstLinePreservesBOMAndMaterializesParent(t *testing.T) {
	source := "\xef\xbb\xbfa.b: 1\n"
	got, err := op().RemoveValue(source, "a.b")
	want := "\xef\xbb\xbfa: {}\n"
	if err != nil || got != want {
		t.Fatalf("RemoveValue = %q, %v; want %q", got, err, want)
	}
	doc, _, parseErr := parseDoc(got, false, false)
	semantic := objectWithEntries(oentry{"a", objectWithEntries()})
	if parseErr != nil || !pvEqual(doc.(*pvalue), semantic) {
		t.Fatalf("semantic tree mismatch: %v", parseErr)
	}
}

func TestRemoveInlineRootPreservesBOM(t *testing.T) {
	source := "\xef\xbb\xbf{a: 1, b: 2}\n"
	got, err := op().RemoveValue(source, "a")
	want := "\xef\xbb\xbf{b: 2}\n"
	if err != nil || got != want {
		t.Fatalf("RemoveValue = %q, %v; want %q", got, err, want)
	}
}

func TestRemoveInlineObjectMembersExactAndSemantic(t *testing.T) {
	cases := []struct {
		name, path, want string
		semantic         *pvalue
	}{
		{"first", "a.x", "a: {y: 2, z: 3}\n", objectWithEntries(oentry{"a", objectWithEntries(oentry{"y", pvInt("2")}, oentry{"z", pvInt("3")})})},
		{"middle", "a.y", "a: {x: 1, z: 3}\n", objectWithEntries(oentry{"a", objectWithEntries(oentry{"x", pvInt("1")}, oentry{"z", pvInt("3")})})},
		{"last", "a.z", "a: {x: 1, y: 2}\n", objectWithEntries(oentry{"a", objectWithEntries(oentry{"x", pvInt("1")}, oentry{"y", pvInt("2")})})},
		{"only", "a.x", "a: {}\n", objectWithEntries(oentry{"a", objectWithEntries()})},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := "a: {x: 1, y: 2, z: 3}\n"
			if tc.name == "only" {
				source = "a: {x: 1}\n"
			}
			got, err := op().RemoveValue(source, tc.path)
			if err != nil || got != tc.want {
				t.Fatalf("RemoveValue = %q, %v; want %q", got, err, tc.want)
			}
			doc, _, parseErr := parseDoc(got, false, false)
			if parseErr != nil || !pvEqual(doc.(*pvalue), tc.semantic) {
				t.Fatalf("semantic tree mismatch: %v", parseErr)
			}
		})
	}
}

func TestRemoveInlineArrayElementsExactAndSemantic(t *testing.T) {
	cases := []struct {
		name, path, want string
		semantic         *pvalue
	}{
		{"first", "a.0", "a: [2, 3]\n", objectWithEntries(oentry{"a", &pvalue{kind: kArr, arr: []*pvalue{pvInt("2"), pvInt("3")}}})},
		{"middle", "a.1", "a: [1, 3]\n", objectWithEntries(oentry{"a", &pvalue{kind: kArr, arr: []*pvalue{pvInt("1"), pvInt("3")}}})},
		{"last", "a.2", "a: [1, 2]\n", objectWithEntries(oentry{"a", &pvalue{kind: kArr, arr: []*pvalue{pvInt("1"), pvInt("2")}}})},
		{"only", "a.0", "a: []\n", objectWithEntries(oentry{"a", &pvalue{kind: kArr, arr: []*pvalue{}}})},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := "a: [1, 2, 3]\n"
			if tc.name == "only" {
				source = "a: [1]\n"
			}
			got, err := op().RemoveValue(source, tc.path)
			if err != nil || got != tc.want {
				t.Fatalf("RemoveValue = %q, %v; want %q", got, err, tc.want)
			}
			doc, _, parseErr := parseDoc(got, false, false)
			if parseErr != nil || !pvEqual(doc.(*pvalue), tc.semantic) {
				t.Fatalf("semantic tree mismatch: %v", parseErr)
			}
		})
	}
}

func TestRemoveInlineCompoundAndBlockCompound(t *testing.T) {
	for _, tc := range []struct{ name, source, path, want string }{
		{"nested inline compound", "a: {x: {p: 1}, y: 2}\n", "a.x", "a: {y: 2}\n"},
		{"inline compound whole target", "a: {x: 1}\n", "a", ""},
		{"block compound", "a: {\n  x: 1\n}\nb: keep\n", "a", "b: keep\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := op().RemoveValue(tc.source, tc.path)
			if err != nil || got != tc.want {
				t.Fatalf("RemoveValue = %q, %v; want %q", got, err, tc.want)
			}
		})
	}
}
