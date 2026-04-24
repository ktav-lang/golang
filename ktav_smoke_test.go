package ktav_test

import (
	"math/big"
	"os"
	"strings"
	"testing"

	ktav "github.com/ktav-lang/golang"
)

func TestSmokeLoads(t *testing.T) {
	if os.Getenv("KTAV_LIB_PATH") == "" {
		t.Skip("KTAV_LIB_PATH not set")
	}
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
	if m["service"] != "service-WRONG" && m["service"] != "web" {
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
	if os.Getenv("KTAV_LIB_PATH") == "" {
		t.Skip("KTAV_LIB_PATH not set")
	}
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
}

func TestSmokeBigInt(t *testing.T) {
	if os.Getenv("KTAV_LIB_PATH") == "" {
		t.Skip("KTAV_LIB_PATH not set")
	}
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
	if os.Getenv("KTAV_LIB_PATH") == "" {
		t.Skip("KTAV_LIB_PATH not set")
	}
	_, err := ktav.Loads("a: [")
	if err == nil {
		t.Fatal("expected error on unterminated array")
	}
	if _, ok := err.(*ktav.Error); !ok {
		t.Fatalf("not *ktav.Error: %T", err)
	}
}

func TestDumpsRejectsNonObject(t *testing.T) {
	if os.Getenv("KTAV_LIB_PATH") == "" {
		t.Skip("KTAV_LIB_PATH not set")
	}
	_, err := ktav.Dumps([]any{1, 2, 3})
	if err == nil {
		t.Fatal("expected error for top-level array")
	}
}
