package engine

import (
	"math"
	"testing"
)

func TestParsedExpectedTreeOwnershipMutationRegressions(t *testing.T) {
	arr := func(v ...*pvalue) *pvalue { return &pvalue{kind: kArr, arr: v} }
	cases := []struct {
		name, source, want string
		mutate             func(string) (string, error)
		semantic           *pvalue
	}{
		{"nested-insert-i64-max", "root: {keep: [1, {x: old}], target: {before: yes}}\n", "root: {keep: [1, {x: old}], target: {before: yes, after: 9223372036854775807}}\n", func(s string) (string, error) {
			return op().InsertValueSegments(s, []string{"root", "target", "after"}, int64(math.MaxInt64))
		}, objectWithEntries(oentry{"root", objectWithEntries(oentry{"keep", arr(pvInt("1"), objectWithEntries(oentry{"x", pvStr("old")}))}, oentry{"target", objectWithEntries(oentry{"before", pvStr("yes")}, oentry{"after", pvInt("9223372036854775807")})})})},
		{"dotted-remove-materialization-crlf-comment", "## lead\r\na.b: 1\r\n", "## lead\r\na: {}\r\n", func(s string) (string, error) { return op().RemoveValue(s, "a.b") }, objectWithEntries(oentry{"a", objectWithEntries()})},
		{"remove-nested-array", "root: {items: [{drop: 1}, {keep: [2, 3]}], sibling: yes}\n", "root: {items: [{keep: [2, 3]}], sibling: yes}\n", func(s string) (string, error) { return op().RemoveValue(s, "root.items.0") }, objectWithEntries(oentry{"root", objectWithEntries(oentry{"items", arr(objectWithEntries(oentry{"keep", arr(pvInt("2"), pvInt("3"))}))}, oentry{"sibling", pvStr("yes")})})},
		{"rename-nested", "root: {old: {x: 1}, sibling: [2]}\n", "root: {new: {x: 1}, sibling: [2]}\n", func(s string) (string, error) { return op().RenameKey(s, "root.old", "new") }, objectWithEntries(oentry{"root", objectWithEntries(oentry{"new", objectWithEntries(oentry{"x", pvInt("1")})}, oentry{"sibling", arr(pvInt("2"))})})},
		{"rename-nested-segments", "root: {old: {x: 1}, sibling: [2]}\n", "root: {new: {x: 1}, sibling: [2]}\n", func(s string) (string, error) { return op().RenameKeySegments(s, []string{"root", "old"}, "new") }, objectWithEntries(oentry{"root", objectWithEntries(oentry{"new", objectWithEntries(oentry{"x", pvInt("1")})}, oentry{"sibling", arr(pvInt("2"))})})},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.mutate(tc.source)
			if err != nil || got != tc.want {
				t.Fatalf("got %q, %v; want %q", got, err, tc.want)
			}
			gotDoc, _, parseErr := parseDoc(got, false, false)
			if parseErr != nil || !pvEqual(gotDoc.(*pvalue), tc.semantic) {
				t.Fatalf("semantic mismatch: %v", parseErr)
			}
			if len(tc.name) >= len("rename-nested") && tc.name[:len("rename-nested")] == "rename-nested" && !pvEqualKeyOrder(gotDoc.(*pvalue), tc.semantic) {
				t.Fatal("rename changed semantic key order")
			}
		})
	}
}

func TestParsedExpectedTreeOwnershipSourceErrorsFirst(t *testing.T) {
	for _, tc := range []struct {
		name string
		run  func() (string, error)
	}{
		{"insert", func() (string, error) { return op().InsertValue("broken: {\n", "", int64(1)) }},
		{"remove", func() (string, error) { return op().RemoveValue("broken: {\n", "") }},
		{"rename", func() (string, error) { return op().RenameKey("broken: {\n", "", "new") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.run()
			if err == nil {
				t.Fatal("expected source parse error")
			}
			if e, ok := err.(*Error); !ok || e.Class != "UnclosedCompound" {
				t.Fatalf("source error did not take precedence: %#v", err)
			}
		})
	}
}
