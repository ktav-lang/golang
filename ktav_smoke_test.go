package ktav_test

import (
	"errors"
	"math/big"
	"reflect"
	"strings"
	"testing"

	ktav "github.com/ktav-lang/golang"
)

func TestSmokeLoads(t *testing.T) {
	requireCabi(t)
	src := `service: web
port:i 8080
ratio:f 0.75
tls: true
tags: [
    prod
    eu-west-1
]
db.host: primary
db.timeout:i 30
`
	got, err := ktav.Loads(src)
	if err != nil {
		t.Fatalf("Loads: %v", err)
	}
	m, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("top is %T", got)
	}
	if m["service"] != "web" {
		t.Fatalf("service: %v", m["service"])
	}
	if m["port"] != int64(8080) {
		t.Fatalf("port = %v (%T)", m["port"], m["port"])
	}
	if m["ratio"] != 0.75 {
		t.Fatalf("ratio = %v", m["ratio"])
	}
	if m["tls"] != true {
		t.Fatalf("tls = %v", m["tls"])
	}
	db, ok := m["db"].(map[string]any)
	if !ok {
		t.Fatalf("db = %v", m["db"])
	}
	if db["host"] != "primary" {
		t.Fatalf("db.host = %v", db["host"])
	}
	if db["timeout"] != int64(30) {
		t.Fatalf("db.timeout = %v", db["timeout"])
	}
}

func TestSmokeRoundTrip(t *testing.T) {
	requireCabi(t)
	input := map[string]any{
		"name":    "demo",
		"count":   int64(42),
		"ratio":   0.5,
		"flag":    true,
		"nothing": nil,
		"tags":    []any{"a", "b"},
		"nested":  map[string]any{"inner": int64(1)},
	}
	out, err := ktav.Dumps(input)
	if err != nil {
		t.Fatalf("Dumps: %v", err)
	}
	back, err := ktav.Loads(out)
	if err != nil {
		t.Fatalf("Loads back: %v\n---\n%s", err, out)
	}
	b, ok := back.(map[string]any)
	if !ok {
		t.Fatalf("back is %T", back)
	}
	if b["name"] != "demo" || b["count"] != int64(42) || b["ratio"] != 0.5 || b["flag"] != true {
		t.Fatalf("round-trip mismatch: %#v\n---\n%s", b, out)
	}
	if b["nothing"] != nil {
		t.Fatalf("nothing = %#v", b["nothing"])
	}
	if !reflect.DeepEqual(b["tags"], []any{"a", "b"}) {
		t.Fatalf("tags = %#v", b["tags"])
	}
	nested, ok := b["nested"].(map[string]any)
	if !ok {
		t.Fatalf("nested is %T", b["nested"])
	}
	if nested["inner"] != int64(1) {
		t.Fatalf("nested.inner = %v (%T)", nested["inner"], nested["inner"])
	}
}

func TestSmokeBigInt(t *testing.T) {
	requireCabi(t)
	src := `value:i 99999999999999999999`
	got, err := ktav.Loads(src)
	if err != nil {
		t.Fatal(err)
	}
	m := got.(map[string]any)
	bi, ok := m["value"].(*big.Int)
	if !ok {
		t.Fatalf("value is %T", m["value"])
	}
	if bi.String() != "99999999999999999999" {
		t.Fatalf("bigint = %s", bi.String())
	}

	// Re-encode
	out, err := ktav.Dumps(map[string]any{"v": bi})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "99999999999999999999") {
		t.Fatalf("dump missing bigint: %s", out)
	}
}

func TestSmokeErrorPath(t *testing.T) {
	requireCabi(t)
	_, err := ktav.Loads("a: [")
	if err == nil {
		t.Fatal("expected error on unterminated array")
	}
	var ktavErr *ktav.Error
	if !errors.As(err, &ktavErr) {
		t.Fatalf("not *ktav.Error: %T", err)
	}
}

func TestDumpsNaNRejected(t *testing.T) {
	requireCabi(t)
	// NaN is rejected on the Go side (before hitting Rust) — the
	// resulting error should still be *ktav.Error so callers can match
	// every binding-emitted failure with one errors.As.
	doc := map[string]any{"x": floatNaN()}
	_, err := ktav.Dumps(doc)
	if err == nil {
		t.Fatal("expected error for NaN")
	}
	var ktavErr *ktav.Error
	if !errors.As(err, &ktavErr) {
		t.Fatalf("Go-side error not *ktav.Error: %T (%v)", err, err)
	}
}

func floatNaN() float64 {
	zero := 0.0
	return zero / zero
}

func TestDumpsRejectsNonObject(t *testing.T) {
	requireCabi(t)
	_, err := ktav.Dumps([]any{1, 2, 3})
	if err == nil {
		t.Fatal("expected error for top-level array")
	}
}
