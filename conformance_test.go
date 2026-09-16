package ktav_test

import (
	"encoding/json"
	"errors"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	ktav "github.com/ktav-lang/golang"
)

func TestConformanceValid(t *testing.T) {
	requireCabi(t)
	specRoot := requireSpec(t)

	var cases []string
	err := filepath.Walk(filepath.Join(specRoot, "valid"), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".ktav") &&
			!strings.HasSuffix(path, ".canonical.ktav") {
			cases = append(cases, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	if len(cases) == 0 {
		t.Fatal("no fixtures found")
	}

	for _, ktavPath := range cases {
		jsonPath := strings.TrimSuffix(ktavPath, ".ktav") + ".json"
		name := strings.TrimPrefix(ktavPath, specRoot+string(filepath.Separator))
		t.Run(name, func(t *testing.T) {
			srcBytes, err := os.ReadFile(ktavPath)
			if err != nil {
				t.Fatal(err)
			}
			oracleBytes, err := os.ReadFile(jsonPath)
			if err != nil {
				t.Fatalf("oracle missing: %v", err)
			}

			got, err := ktav.Loads(string(srcBytes))
			if err != nil {
				t.Fatalf("Loads: %v\n--- input ---\n%s", err, srcBytes)
			}

			want, err := decodeOracle(oracleBytes)
			if err != nil {
				t.Fatalf("oracle decode: %v", err)
			}

			if !structEqual(got, want) {
				t.Fatalf("mismatch\nktav src:\n%s\nktav got: %#v\noracle:   %#v",
					srcBytes, got, want)
			}
		})
	}
}

// decodeOracle parses the reference JSON with json.Number, then lifts
// numeric literals into the same Go shapes `Loads` produces (int64,
// *big.Int, float64) so we can compare by structural equality.
func decodeOracle(raw []byte) (any, error) {
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	return liftOracle(v), nil
}

func liftOracle(v any) any {
	switch t := v.(type) {
	case map[string]any:
		for k, x := range t {
			t[k] = liftOracle(x)
		}
		return t
	case []any:
		for i, x := range t {
			t[i] = liftOracle(x)
		}
		return t
	case json.Number:
		s := string(t)
		if !strings.ContainsAny(s, ".eE") {
			// Integer literal
			if n, ok := new(big.Int).SetString(s, 10); ok {
				if n.IsInt64() {
					return n.Int64()
				}
				return n
			}
		}
		f, err := t.Float64()
		if err != nil {
			return t
		}
		return f
	default:
		return v
	}
}

// structEqual compares with one subtlety: *big.Int equality is by .Cmp,
// not by pointer.
func structEqual(a, b any) bool {
	switch av := a.(type) {
	case *big.Int:
		bv, ok := b.(*big.Int)
		return ok && av.Cmp(bv) == 0
	case float64:
		bv, ok := b.(float64)
		return ok && av == bv
	case map[string]any:
		bm, ok := b.(map[string]any)
		if !ok || len(av) != len(bm) {
			return false
		}
		for k, v := range av {
			if !structEqual(v, bm[k]) {
				return false
			}
		}
		return true
	case []any:
		bs, ok := b.([]any)
		if !ok || len(av) != len(bs) {
			return false
		}
		for i := range av {
			if !structEqual(av[i], bs[i]) {
				return false
			}
		}
		return true
	default:
		return reflect.DeepEqual(a, b)
	}
}

// TestConformanceCanonical verifies that EmitCanonical round-trips: parse the
// base `.ktav` fixture, emit canonical form, and compare byte-for-byte with
// the `.canonical.ktav` oracle.
func TestConformanceCanonical(t *testing.T) {
	requireCabi(t)
	specRoot := requireSpec(t)

	var cases []string
	err := filepath.Walk(filepath.Join(specRoot, "valid"), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".canonical.ktav") {
			cases = append(cases, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}

	for _, canonicalPath := range cases {
		basePath := strings.TrimSuffix(canonicalPath, ".canonical.ktav") + ".ktav"
		name := strings.TrimPrefix(canonicalPath, specRoot+string(filepath.Separator))
		t.Run(name, func(t *testing.T) {
			src, err := os.ReadFile(basePath)
			if err != nil {
				t.Fatal(err)
			}
			oracle, err := os.ReadFile(canonicalPath)
			if err != nil {
				t.Fatal(err)
			}

			canonical, err := ktav.CanonicalFromSource(string(src))
			if err != nil {
				t.Fatalf("CanonicalFromSource: %v\n--- input ---\n%s", err, src)
			}
			if canonical != string(oracle) {
				t.Fatalf("canonical mismatch\nwant:\n%s\ngot:\n%s", oracle, canonical)
			}
		})
	}
}

func TestConformanceInvalid(t *testing.T) {
	requireCabi(t)
	specRoot := requireSpec(t)

	var cases []string
	err := filepath.Walk(filepath.Join(specRoot, "invalid"), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".ktav") &&
			!strings.HasSuffix(path, ".canonical.ktav") {
			cases = append(cases, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}

	for _, p := range cases {
		name := strings.TrimPrefix(p, specRoot+string(filepath.Separator))
		t.Run(name, func(t *testing.T) {
			src, err := os.ReadFile(p)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := ktav.Loads(string(src)); err == nil {
				t.Fatalf("expected parse error, got ok\n---\n%s", src)
			}
		})
	}
}

// decodeUnrepJSON reads one of the unrepresentable-fixture oracles and
// returns (value, reason). The fixture-specific one-key {"$float":
// "NaN"|"Infinity"|"-Infinity"} marker becomes a non-finite Go float64.
func decodeUnrepJSON(raw []byte) (any, string, error) {
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.UseNumber()
	var doc struct {
		Value                 any    `json:"value"`
		UnrepresentableReason string `json:"unrepresentable_reason"`
	}
	if err := dec.Decode(&doc); err != nil {
		return nil, "", err
	}
	return liftMarker(doc.Value), doc.UnrepresentableReason, nil
}

// liftMarker substitutes the {"$float": ...} marker, then lifts the rest
// of the tree exactly like liftOracle.
func liftMarker(v any) any {
	if m, ok := v.(map[string]any); ok && len(m) == 1 {
		if s, ok := m["$float"].(string); ok {
			switch s {
			case "NaN":
				return math.NaN()
			case "Infinity":
				return math.Inf(1)
			case "-Infinity":
				return math.Inf(-1)
			}
		}
	}
	switch t := v.(type) {
	case map[string]any:
		for k, x := range t {
			t[k] = liftMarker(x)
		}
		return t
	case []any:
		for i, x := range t {
			t[i] = liftMarker(x)
		}
		return t
	case json.Number:
		return liftOracle(t)
	default:
		return v
	}
}

// TestConformanceUnrepresentable walks spec 0.7's unrepresentable/
// category: JSON values a conforming WRITER must refuse. The binding
// must reject them on both writer entry points (Dumps and
// EmitCanonical) with a *ktav.Error.
func TestConformanceUnrepresentable(t *testing.T) {
	requireCabi(t)
	specRoot := requireSpec(t)

	var cases []string
	err := filepath.Walk(filepath.Join(specRoot, "unrepresentable"), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".json") {
			cases = append(cases, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	if len(cases) == 0 {
		t.Fatal("no unrepresentable fixtures found")
	}

	for _, p := range cases {
		name := strings.TrimPrefix(p, specRoot+string(filepath.Separator))
		t.Run(name, func(t *testing.T) {
			raw, err := os.ReadFile(p)
			if err != nil {
				t.Fatal(err)
			}
			value, reason, err := decodeUnrepJSON(raw)
			if err != nil {
				t.Fatalf("oracle decode: %v", err)
			}

			for _, call := range []struct {
				label string
				fn    func() (string, error)
			}{
				{"Dumps", func() (string, error) { return ktav.Dumps(value) }},
				{"EmitCanonical", func() (string, error) { return ktav.EmitCanonical(value) }},
			} {
				out, err := call.fn()
				if err == nil {
					t.Fatalf("%s: expected writer to refuse value (reason %s), got ok:\n%s", call.label, reason, out)
				}
				var ktavErr *ktav.Error
				if !errors.As(err, &ktavErr) {
					t.Fatalf("%s: not *ktav.Error: %T (%v)", call.label, err, err)
				}
				t.Logf("%s refused (reason %s): %v", call.label, reason, err)
			}
		})
	}
}

// TestConformanceParseableUnrepresentable walks spec 0.7's
// parseable-unrepresentable/ category: documents the parser accepts but
// no canonical writer may emit. Loads must succeed and match the
// oracle value; CanonicalFromSource must refuse with a *ktav.Error.
func TestConformanceParseableUnrepresentable(t *testing.T) {
	requireCabi(t)
	specRoot := requireSpec(t)

	var cases []string
	err := filepath.Walk(filepath.Join(specRoot, "parseable-unrepresentable"), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".ktav") &&
			!strings.HasSuffix(path, ".canonical.ktav") {
			cases = append(cases, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	if len(cases) == 0 {
		t.Fatal("no parseable-unrepresentable fixtures found")
	}

	for _, p := range cases {
		name := strings.TrimPrefix(p, specRoot+string(filepath.Separator))
		t.Run(name, func(t *testing.T) {
			src, err := os.ReadFile(p)
			if err != nil {
				t.Fatal(err)
			}
			oracleRaw, err := os.ReadFile(strings.TrimSuffix(p, ".ktav") + ".json")
			if err != nil {
				t.Fatalf("oracle missing: %v", err)
			}
			want, reason, err := decodeUnrepJSON(oracleRaw)
			if err != nil {
				t.Fatalf("oracle decode: %v", err)
			}

			got, err := ktav.Loads(string(src))
			if err != nil {
				t.Fatalf("Loads: %v\n--- input ---\n%s", err, src)
			}
			if !structEqual(got, want) {
				t.Fatalf("mismatch\nktav src:\n%s\nktav got: %#v\noracle:   %#v", src, got, want)
			}

			if _, err := ktav.CanonicalFromSource(string(src)); err == nil {
				t.Fatalf("expected canonical emit to refuse (reason %s)\n--- input ---\n%s", reason, src)
			} else {
				var ktavErr *ktav.Error
				if !errors.As(err, &ktavErr) {
					t.Fatalf("not *ktav.Error: %T (%v)", err, err)
				}
				t.Logf("canonical emit refused (reason %s): %v", reason, err)
			}
		})
	}
}
