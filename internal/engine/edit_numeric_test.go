package engine

import (
	"math/big"
	"testing"
)

func TestInsertNumericDomainFocused(t *testing.T) {
	wide := new(big.Int)
	if _, ok := wide.SetString("9007199254740993", 10); !ok {
		t.Fatal("invalid wide integer fixture")
	}
	cases := []struct {
		name, source string
		wire         []byte
		want         *pvalue
		unsupported  bool
	}{
		{"overflow", "base: 1\n", []byte(`{"$i":"9223372036854775808"}`), nil, true},
		{"precise integer", "base: 1\n", mustEditIntegerWire(t, wide), pvInt("9007199254740993"), false},
		{"source plus spelling", "base: +1\n", []byte(`{"$i":"2"}`), pvInt("2"), false},
		{"source leading zero spelling", "base: 0007\n", []byte(`{"$i":"2"}`), pvInt("2"), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := pureInsertValueBytes([]byte(tc.source), []byte("added"), tc.wire)
			if tc.unsupported {
				e, ok := err.(*Error)
				if !ok || e.Class != "EditUnsupported" {
					t.Fatalf("error = %v; want EditUnsupported", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("insert error: %v", err)
			}
			gotDoc, _, parseErr := parseDoc(string(got), false, false)
			if parseErr != nil {
				t.Fatalf("parse output: %v", parseErr)
			}
			// Independent semantic oracle: expected source value plus explicit
			// inserted value; do not derive it by parsing the source under test.
			wantBase := pvInt("1")
			if tc.name == "source plus spelling" {
				wantBase = pvInt("1")
			} else if tc.name == "source leading zero spelling" {
				wantBase = pvStr("0007")
			}
			wantTree := pvObj()
			wantTree.obj.insertLeaf("base", wantBase)
			wantTree.obj.insertLeaf("added", tc.want)
			if !pvEqual(gotDoc.(*pvalue), wantTree) {
				t.Fatalf("semantic output differs from independent expectation")
			}
			if tc.name == "source plus spelling" || tc.name == "source leading zero spelling" {
				if len(got) < len(tc.source) || string(got[:len(tc.source)]) != tc.source {
					t.Fatalf("insert did not preserve original source spelling: %q", got)
				}
			}
		})
	}
}

func mustEditIntegerWire(t *testing.T, n *big.Int) []byte {
	t.Helper()
	wire, err := encodeEditReplacementTagged(n)
	if err != nil {
		t.Fatalf("encode big.Int replacement: %v", err)
	}
	return wire
}

func TestPvEqualDistinguishesNegativeZero(t *testing.T) {
	if pvEqual(pvFloat("-0.0"), pvFloat("0.0")) {
		t.Fatal("negative zero must differ from positive zero")
	}
}
