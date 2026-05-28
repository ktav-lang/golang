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
port: 8080
ratio: 0.75
tls: true
tags: [
    prod
    eu-west-1
]
db.host: primary
db.timeout: 30
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
	// Under spec 0.5, integers that overflow i64 fall back to String.
	src := `value: 99999999999999999999`
	got, err := ktav.Loads(src)
	if err != nil {
		t.Fatal(err)
	}
	m := got.(map[string]any)
	s, ok := m["value"].(string)
	if !ok {
		t.Fatalf("value is %T, want string (overflow → String per spec 0.5)", m["value"])
	}
	if s != "99999999999999999999" {
		t.Fatalf("bigint string = %q", s)
	}

	// Round-trip via Dumps with a *big.Int still encodes as an integer
	// scalar on the wire (the renderer accepts Value::Integer).
	bi := new(big.Int)
	bi.SetString("99999999999999999999", 10)
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

func TestEmitCanonical(t *testing.T) {
	requireCabi(t)
	// Round-trip: parse → EmitCanonical → parse → compare.
	src := `name: demo
count: 42
ratio: 0.5
flag: true
nothing: null
`
	got, err := ktav.Loads(src)
	if err != nil {
		t.Fatalf("Loads: %v", err)
	}
	canonical, err := ktav.EmitCanonical(got)
	if err != nil {
		t.Fatalf("EmitCanonical: %v", err)
	}
	back, err := ktav.Loads(canonical)
	if err != nil {
		t.Fatalf("Loads canonical: %v\n---\n%s", err, canonical)
	}
	m1 := got.(map[string]any)
	m2 := back.(map[string]any)
	if m1["name"] != m2["name"] || m1["count"] != m2["count"] ||
		m1["ratio"] != m2["ratio"] || m1["flag"] != m2["flag"] {
		t.Fatalf("round-trip mismatch\ngot=%#v\nback=%#v", m1, m2)
	}
	// Calling EmitCanonical twice must be byte-identical.
	canonical2, err := ktav.EmitCanonical(got)
	if err != nil {
		t.Fatalf("EmitCanonical 2: %v", err)
	}
	if canonical != canonical2 {
		t.Fatalf("not deterministic:\nfirst=%q\nsecond=%q", canonical, canonical2)
	}
}

func floatNaN() float64 {
	zero := 0.0
	return zero / zero
}

func TestDumpsRejectsScalarRoot(t *testing.T) {
	requireCabi(t)
	// Top-level scalars (Bool/String/Int/Float/Null) are still rejected
	// — only Object and Array are valid roots per spec § 5.0.1.
	_, err := ktav.Dumps("bare-string")
	if err == nil {
		t.Fatal("expected error for top-level scalar string")
	}
	_, err = ktav.Dumps(int64(42))
	if err == nil {
		t.Fatal("expected error for top-level scalar int")
	}
	_, err = ktav.Dumps(true)
	if err == nil {
		t.Fatal("expected error for top-level scalar bool")
	}
	_, err = ktav.Dumps(nil)
	if err == nil {
		t.Fatal("expected error for top-level null")
	}
}

func TestDumpsTopLevelArray(t *testing.T) {
	requireCabi(t)
	out, err := ktav.Dumps([]any{"foo", "bar", "baz"})
	if err != nil {
		t.Fatalf("Dumps top-level array: %v", err)
	}
	back, err := ktav.Loads(out)
	if err != nil {
		t.Fatalf("Loads back: %v\n---\n%s", err, out)
	}
	arr, ok := back.([]any)
	if !ok {
		t.Fatalf("back is %T, want []any", back)
	}
	if !reflect.DeepEqual(arr, []any{"foo", "bar", "baz"}) {
		t.Fatalf("round-trip mismatch: %#v", arr)
	}
}

func TestLoadsTopLevelArray(t *testing.T) {
	requireCabi(t)
	src := "foo\nbar\nbaz\n"
	got, err := ktav.Loads(src)
	if err != nil {
		t.Fatalf("Loads: %v", err)
	}
	arr, ok := got.([]any)
	if !ok {
		t.Fatalf("top is %T, want []any", got)
	}
	if !reflect.DeepEqual(arr, []any{"foo", "bar", "baz"}) {
		t.Fatalf("got %#v", arr)
	}
}

func TestDumpsForceStrings(t *testing.T) {
	requireCabi(t)
	doc := map[string]any{
		"port":  int64(8080),
		"ratio": 0.5,
		"flag":  true,
		"empty": nil,
		"name":  "demo",
	}
	out, err := ktav.DumpsForceStrings(doc)
	if err != nil {
		t.Fatalf("DumpsForceStrings: %v", err)
	}
	// All scalars should round-trip as Strings.
	back, err := ktav.Loads(out)
	if err != nil {
		t.Fatalf("Loads back: %v\n---\n%s", err, out)
	}
	m, ok := back.(map[string]any)
	if !ok {
		t.Fatalf("back is %T", back)
	}
	want := map[string]any{
		"port":  "8080",
		"ratio": "0.5",
		"flag":  "true",
		"empty": "null",
		"name":  "demo",
	}
	for k, v := range want {
		if m[k] != v {
			t.Fatalf("%s: got %#v (%T), want %q", k, m[k], m[k], v)
		}
	}
}

func TestDumpsForceStringsTopLevelArray(t *testing.T) {
	requireCabi(t)
	out, err := ktav.DumpsForceStrings([]any{int64(1), 2.5, true, nil, "x"})
	if err != nil {
		t.Fatalf("DumpsForceStrings: %v", err)
	}
	back, err := ktav.Loads(out)
	if err != nil {
		t.Fatalf("Loads back: %v\n---\n%s", err, out)
	}
	arr, ok := back.([]any)
	if !ok {
		t.Fatalf("back is %T, want []any", back)
	}
	want := []any{"1", "2.5", "true", "null", "x"}
	if !reflect.DeepEqual(arr, want) {
		t.Fatalf("got %#v, want %#v", arr, want)
	}
}
