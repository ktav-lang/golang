package ktav_test

import (
	"encoding/json"
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
		if !info.IsDir() && strings.HasSuffix(path, ".ktav") {
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

func TestConformanceInvalid(t *testing.T) {
	requireCabi(t)
	specRoot := requireSpec(t)

	var cases []string
	err := filepath.Walk(filepath.Join(specRoot, "invalid"), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".ktav") {
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
