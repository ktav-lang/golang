package engine

import (
	"strings"
	"testing"
)

func regressionSegments(path string) []string { return strings.Split(path, ".") }

func TestInsertStandaloneInlineBodyOwnersExactAndSemantic(t *testing.T) {
	long := strings.Repeat("x", 120)
	arr := func(v *pvalue) *pvalue { return &pvalue{kind: kArr, arr: []*pvalue{v}} }
	obj := func(e ...oentry) *pvalue { return objectWithEntries(e...) }
	cases := []struct {
		name, source, path, want string
		value                    any
		semantic                 *pvalue
	}{
		{"pair-empty", "a: {  }\n", "a.b", "a: {b: 2}\n", int64(2), obj(oentry{"a", obj(oentry{"b", pvInt("2")})})},
		{"pair-spaces", "a: {x: 1   }\n", "a.b", "a: {x: 1   , b: 2}\n", int64(2), obj(oentry{"a", obj(oentry{"x", pvInt("1")}, oentry{"b", pvInt("2")})})},
		{"pair-fallback", "a: {x: 1   }\n", "a.b", "a: {x: 1   , b: line1\\n))\\nline2}\n", "line1\n))\nline2", obj(oentry{"a", obj(oentry{"x", pvInt("1")}, oentry{"b", pvStr("line1\n))\nline2")})})},
		{"pair-long", "a: {x: 1}\n", "a.b", "a: {\n    x: 1\n    b: " + long + "\n}\n", long, obj(oentry{"a", obj(oentry{"x", pvInt("1")}, oentry{"b", pvStr(long)})})},
		{"pair-multiline", "a: {x: 1}\n", "a.b", "a: {\n    x: 1\n    b: ((\nfirst\nsecond\n    ))\n}\n", "first\nsecond", obj(oentry{"a", obj(oentry{"x", pvInt("1")}, oentry{"b", pvStr("first\nsecond")})})},
		{"item-empty", "arr: [\n    {  }\n]\n", "arr.0.b", "arr: [\n    {b: 2}\n]\n", int64(2), obj(oentry{"arr", arr(obj(oentry{"b", pvInt("2")}))})},
		{"item-multiline", "arr: [\n    {x: 1}\n]\n", "arr.0.b", "arr: [\n    {\n        x: 1\n        b: ((\nfirst\nsecond\n        ))\n    }\n]\n", "first\nsecond", obj(oentry{"arr", arr(obj(oentry{"x", pvInt("1")}, oentry{"b", pvStr("first\nsecond")}))})},
		{"item-spaces", "arr: [\n    {x: 1   }\n]\n", "arr.0.b", "arr: [\n    {x: 1   , b: 2}\n]\n", int64(2), obj(oentry{"arr", arr(obj(oentry{"x", pvInt("1")}, oentry{"b", pvInt("2")}))})},
		{"item-fallback", "arr: [\n    {x: 1   }\n]\n", "arr.0.b", "arr: [\n    {x: 1   , b: line1\\n))\\nline2}\n]\n", "line1\n))\nline2", obj(oentry{"arr", arr(obj(oentry{"x", pvInt("1")}, oentry{"b", pvStr("line1\n))\nline2")}))})},
		{"item-long", "arr: [\n    {x: 1}\n]\n", "arr.0.b", "arr: [\n    {\n        x: 1\n        b: " + long + "\n    }\n]\n", long, obj(oentry{"arr", arr(obj(oentry{"x", pvInt("1")}, oentry{"b", pvStr(long)}))})},
	}
	for _, tc := range cases {
		for _, segments := range []bool{false, true} {
			tc, segments := tc, segments
			name := tc.name + "/textual"
			if segments {
				name = tc.name + "/segments"
			}
			t.Run(name, func(t *testing.T) {
				var got string
				var err error
				if segments {
					got, err = op().InsertValueSegments(tc.source, regressionSegments(tc.path), tc.value)
				} else {
					got, err = op().InsertValue(tc.source, tc.path, tc.value)
				}
				if err != nil || got != tc.want {
					t.Fatalf("Insert = %q, %v; want %q", got, err, tc.want)
				}
				doc, _, parseErr := parseDoc(got, false, false)
				if parseErr != nil || !pvEqual(doc.(*pvalue), tc.semantic) {
					t.Fatalf("semantic mismatch: %v", parseErr)
				}
			})
		}
	}
}
