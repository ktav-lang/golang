package engine

// Package-internal contract tests for the tagged-JSON wire layer
// (ktav_wire.go): decodeJSON's trailing-data rejection and empty-key
// handling, and prepareWriterValue's exact decoded error paths. The
// public-facing cycle/DAG/keyword coverage lives in ktav_bridge_test.go
// and pure_depth_public_test.go (package ktav_test).

import (
	"reflect"
	"strings"
	"testing"
)

func TestDecodeJSONTrailingDataRejected(t *testing.T) {
	cases := []struct {
		name string
		raw  string
	}{
		{"two objects", `{"a":1} {"b":2}`},
		{"two arrays", `[[1]][[2]]`},
		{"scalar then scalar", `1 2`},
		{"object then scalar", `{"a":1} 3`},
		{"scalar then object", `3 {"a":1}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			v, err := decodeJSON([]byte(c.raw))
			if err == nil {
				t.Fatalf("decodeJSON(%s) = %#v, want trailing-data error", c.raw, v)
			}
			if e, ok := err.(*Error); !ok || !strings.Contains(e.Error(), "trailing data") {
				t.Fatalf("decodeJSON(%s) err = %v, want trailing-data error", c.raw, err)
			}
		})
	}
	// Controls: single top-level values still decode.
	for _, ok := range []string{`{"a":1}`, `[1]`, `7`, `"s"`, `true`, `null`} {
		if _, err := decodeJSON([]byte(ok)); err != nil {
			t.Fatalf("decodeJSON(%s): %v", ok, err)
		}
	}
}

func TestFlattenTaggedTrailingDataRejected(t *testing.T) {
	// flattenTagged (the LoadsInto path) must mirror decodeJSON's
	// haveRoot guard: a second top-level JSON value is an error with
	// the SAME wording, so the documented LoadsInto guarantee is real.
	_, err := flattenTagged([]byte(`{"a":1} {"b":2}`))
	if err == nil {
		t.Fatal("flattenTagged(trailing data) = ok, want error")
	}
	if e, ok := err.(*Error); !ok || !strings.Contains(e.Error(), "trailing data after top-level value") {
		t.Fatalf("flattenTagged(trailing data) err = %v, want trailing-data error", err)
	}
	// Controls: single top-level values still flatten.
	for _, ok := range []string{`{"a":1}`, `[1,{"$i":"7"}]`, `7`, `"s"`, `null`} {
		if _, err := flattenTagged([]byte(ok)); err != nil {
			t.Fatalf("flattenTagged(%s): %v", ok, err)
		}
	}
}

func TestDecodeJSONEmptyKeySentinel(t *testing.T) {
	// "" is a LEGITIMATE JSON key in tagged wire form — the parser must
	// not use the empty string as an internal sentinel.
	v, err := decodeJSON([]byte(`{"": 1}`))
	if err != nil {
		t.Fatalf("decodeJSON({\"\": 1}): %v", err)
	}
	m, ok := v.(map[string]any)
	if !ok {
		t.Fatalf("root = %T, want map", v)
	}
	got, exists := m[""]
	if !exists {
		t.Fatalf("key \"\" missing from %#v", m)
	}
	if n, isInt := got.(int64); !isInt || n != 1 {
		t.Fatalf(`m[""] = %#v, want int64 1`, got)
	}
	// Negative control: a value with NO pending key is still an error.
	if _, err := decodeJSON([]byte(`{"a": 1, 2}`)); err == nil {
		t.Fatal("decodeJSON(non-string key position) = ok, want error")
	}
}

func TestPrepareWriterValueReportsExactPath(t *testing.T) {
	// root map a → {b: NaN payload}: the NonFiniteFloat error must
	// carry the exact decoded path ["a","b"] — independent of the
	// walk's internal path bookkeeping.
	payload := map[string]any{
		"a": map[string]any{"b": map[string]any{"$f": goFloatPayload("NaN")}},
	}
	_, err := prepareWriterValue(payload, nil, false)
	if err == nil {
		t.Fatal("prepareWriterValue(NaN payload) = ok, want NonFiniteFloat")
	}
	e, ok := err.(*Error)
	if !ok || e.Path == nil || !reflect.DeepEqual(e.Path, []string{"a", "b"}) {
		t.Fatalf("NonFiniteFloat path = %#v (err %v), want exactly [a b]", err, err)
	}
	// Deeper: a slice level adds no path segment.
	payload2 := map[string]any{
		"a": []any{map[string]any{"b": map[string]any{"$f": goFloatPayload("+Inf")}}},
	}
	_, err = prepareWriterValue(payload2, nil, false)
	e2, ok := err.(*Error)
	if !ok || !reflect.DeepEqual(e2.Path, []string{"a", "b"}) {
		t.Fatalf("slice-level path = %#v, want exactly [a b]", err)
	}
	// forceStrings coerces instead of failing.
	out, err := prepareWriterValue(payload, nil, true)
	if err != nil {
		t.Fatalf("prepareWriterValue(forceStrings): %v", err)
	}
	coerced := out.(map[string]any)["a"].(map[string]any)["b"]
	if s, ok := coerced.(string); !ok || s != "NaN" {
		t.Fatalf("forceStrings payload = %#v, want \"NaN\"", coerced)
	}
	// EmptyKeyName carries its exact path too.
	_, err = prepareWriterValue(map[string]any{"": int64(1)}, nil, false)
	e3, ok := err.(*Error)
	if !ok || !reflect.DeepEqual(e3.Path, []string{""}) {
		t.Fatalf("EmptyKeyName path = %#v, want exactly [\"\"]", err)
	}
}
