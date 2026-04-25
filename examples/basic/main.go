// End-to-end demo: parse a Ktav document into a typed struct, walk
// the dynamic shape, then build a fresh document in Go and render it
// back to Ktav text.
//
// Run with the repo-built native library:
//
//	cargo build --release -p ktav-cabi
//	KTAV_LIB_PATH=$PWD/target/release/libktav_cabi.so \
//	    go run ./examples/basic
package main

import (
	"fmt"

	ktav "github.com/ktav-lang/golang"
)

const src = `
service: web
port:i 8080
ratio:f 0.75
tls: true
tags: [
    prod
    eu-west-1
]
db.host: primary.internal
db.timeout:i 30
`

// Config maps the document into typed Go fields. Decoding goes through
// encoding/json, so the usual `json:"…"` tags work as expected.
type Config struct {
	Service string   `json:"service"`
	Port    int64    `json:"port"`
	Ratio   float64  `json:"ratio"`
	TLS     bool     `json:"tls"`
	Tags    []string `json:"tags"`
	DB      struct {
		Host    string `json:"host"`
		Timeout int64  `json:"timeout"`
	} `json:"db"`
}

func main() {
	// ── 1. Decode straight into a struct. ─────────────────────────────
	var cfg Config
	if err := ktav.LoadsInto(src, &cfg); err != nil {
		panic(err)
	}
	fmt.Printf("service=%s port=%d tls=%v ratio=%.2f\n",
		cfg.Service, cfg.Port, cfg.TLS, cfg.Ratio)
	fmt.Printf("tags=%v\n", cfg.Tags)
	fmt.Printf("db: %s (timeout=%ds)\n\n", cfg.DB.Host, cfg.DB.Timeout)

	// ── 2. Or work with the dynamic shape, dispatching on type. ──────
	dyn, err := ktav.Loads(src)
	if err != nil {
		panic(err)
	}
	fmt.Println("shape:")
	for k, v := range dyn.(map[string]any) {
		fmt.Printf("  %-12s -> %s\n", k, describe(v))
	}

	// ── 3. Build a config in code, render it as Ktav text. ───────────
	doc := map[string]any{
		"name":  "frontend",
		"port":  int64(8443),
		"tls":   true,
		"ratio": 0.95,
		"upstreams": []any{
			upstream("a.example", 1080),
			upstream("b.example", 1080),
			upstream("c.example", 1080),
		},
		"notes": nil,
	}
	out, err := ktav.Dumps(doc)
	if err != nil {
		panic(err)
	}
	fmt.Println("\n--- rendered ---")
	fmt.Print(out)
}

func describe(v any) string {
	switch x := v.(type) {
	case nil:
		return "null"
	case bool:
		return fmt.Sprintf("bool=%v", x)
	case int64:
		return fmt.Sprintf("int=%d", x)
	case float64:
		return fmt.Sprintf("float=%g", x)
	case string:
		return fmt.Sprintf("str=%q", x)
	case []any:
		return fmt.Sprintf("array(%d)", len(x))
	case map[string]any:
		return fmt.Sprintf("object(%d)", len(x))
	default:
		return fmt.Sprintf("%T", v)
	}
}

func upstream(host string, port int64) map[string]any {
	return map[string]any{
		"host": host,
		"port": port,
	}
}
