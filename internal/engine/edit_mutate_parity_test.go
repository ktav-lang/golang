package engine

import "testing"

// Independent expected trees accompany exact source-byte assertions so a
// source-preserving but semantically incorrect splice cannot pass.
func TestMutateUTF8QuotedKeyAndArraySemantic(t *testing.T) {
	for _, tc := range []struct {
		name, source, path, want string
		semantic                 *pvalue
	}{
		{"quoted-unicode-key", "a: 1\n", "ключ,да", "a: 1\n\"ключ,да\": ok\n", func() *pvalue {
			m := newOmap()
			m.insertLeaf("a", pvInt("1"))
			m.insertLeaf("ключ,да", pvStr("ok"))
			return &pvalue{kind: kObj, obj: m}
		}()},
		{"array-removal", "items: [1, 2, 3]\n", "items.1", "items: [1, 3]\n", func() *pvalue {
			m := newOmap()
			m.insertLeaf("items", &pvalue{kind: kArr, arr: []*pvalue{pvInt("1"), pvInt("3")}})
			return &pvalue{kind: kObj, obj: m}
		}()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got string
			var err error
			if tc.name == "array-removal" {
				got, err = op().RemoveValue(tc.source, tc.path)
			} else {
				got, err = op().InsertValue(tc.source, tc.path, "ok")
			}
			if err != nil || got != tc.want {
				t.Fatalf("result = %q, %v; want %q", got, err, tc.want)
			}
			doc, _, parseErr := parseDoc(got, false, false)
			if parseErr != nil || !pvEqual(doc.(*pvalue), tc.semantic) {
				t.Fatalf("semantic tree mismatch: %v", parseErr)
			}
		})
	}
}

func TestMutateNilSegmentsAndInvalidUTF8Errors(t *testing.T) {
	for _, tc := range []struct {
		name string
		path []string
		want string
	}{
		{"nil-path", nil, "EditNotFound"},
		{"invalid-utf8", []string{string([]byte{0xff})}, "Message"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := op().RemoveValueSegments("a: 1\n", tc.path)
			if err == nil {
				t.Fatal("expected error")
			}
			ke, ok := err.(*Error)
			if !ok || ke.Class != tc.want {
				t.Fatalf("error = %#v, want class %s", err, tc.want)
			}
		})
	}
}
