// Command pureconsumer is a standalone smoke consumer for the packaged
// pure-Go backend. CI copies this file into an independent temp module,
// installs github.com/ktav-lang/golang@v0.8.1 THROUGH a file:// module
// proxy built from the checkout (scripts/module_proxy.py), and runs
// `go run .` with NO ktav_cabi artifact built — proving the PACKAGED
// module works as a real downloaded dependency in pure mode.
// It exercises the FULL public API through a per-instance Operations
// value AND the package-level global-default functions after UsePure.
package main

import (
	"fmt"
	"math"
	"os"

	ktav "github.com/ktav-lang/golang"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "pureconsumer:", err)
		os.Exit(1)
	}
}

func run() error {
	ops, err := ktav.New(ktav.BackendPure)
	if err != nil {
		return err
	}

	src := "server: {\n    host: 127.0.0.1\n    port: 8080\n}\ntags: [prod, edge]\nratio: 0.5\n"

	// ── per-instance operations (no global state touched) ──
	v, err := ops.Loads(src)
	if err != nil {
		return fmt.Errorf("instance Loads: %w", err)
	}
	m, ok := v.(map[string]any)
	if !ok || m["ratio"] != 0.5 {
		return fmt.Errorf("instance Loads value = %#v", v)
	}

	sv, err := ops.LoadsStrict(src)
	if err != nil {
		return fmt.Errorf("instance LoadsStrict: %w", err)
	}
	if _, ok := sv.(map[string]any); !ok {
		return fmt.Errorf("instance LoadsStrict type = %T", sv)
	}

	var cfg struct {
		Server struct {
			Host string
			Port int
		}
		Tags []string
	}
	if err := ops.LoadsInto(src, &cfg); err != nil {
		return fmt.Errorf("instance LoadsInto: %w", err)
	}
	if cfg.Server.Host != "127.0.0.1" || cfg.Server.Port != 8080 || len(cfg.Tags) != 2 {
		return fmt.Errorf("instance LoadsInto target = %#v", cfg)
	}

	dumped, err := ops.Dumps(map[string]any{"a": int64(1), "b": "x"})
	if err != nil {
		return fmt.Errorf("instance Dumps: %w", err)
	}
	if back, err := ops.Loads(dumped); err != nil || back.(map[string]any)["a"] != int64(1) {
		return fmt.Errorf("instance Dumps round trip = %q, %v", dumped, err)
	}

	forced, err := ops.DumpsForceStrings(map[string]any{"n": int64(7)})
	if err != nil {
		return fmt.Errorf("instance DumpsForceStrings: %w", err)
	}
	back, err := ops.Loads(forced)
	if err != nil || back.(map[string]any)["n"] != "7" {
		return fmt.Errorf("instance DumpsForceStrings round trip = %q, %v", forced, err)
	}

	canonical, err := ops.EmitCanonical(map[string]any{"z": math.Copysign(0, -1)})
	if err != nil {
		return fmt.Errorf("instance EmitCanonical: %w", err)
	}
	if canonical != "z: -0.0\n" {
		return fmt.Errorf("instance EmitCanonical = %q, want %q", canonical, "z: -0.0\n")
	}

	formatted, err := ops.FormatSource("a:   1\n\n\n## keep\nb: 2\n")
	if err != nil {
		return fmt.Errorf("instance FormatSource: %w", err)
	}
	if again, err := ops.FormatSource(formatted); err != nil || again != formatted {
		return fmt.Errorf("instance FormatSource not a fixed point: %q -> %q (%v)", formatted, again, err)
	}

	fromSrc, err := ops.CanonicalFromSource("k: [1, 2]\n")
	if err != nil {
		return fmt.Errorf("instance CanonicalFromSource: %w", err)
	}
	if fromSrc != "k: [\n    1\n    2\n]\n" {
		return fmt.Errorf("instance CanonicalFromSource = %q", fromSrc)
	}

	// ── package-level functions after the global UsePure switch ──
	if err := ktav.UsePure(); err != nil {
		return fmt.Errorf("UsePure: %w", err)
	}
	if _, err := ktav.Loads(src); err != nil {
		return fmt.Errorf("package Loads: %w", err)
	}
	if _, err := ktav.LoadsStrict(src); err != nil {
		return fmt.Errorf("package LoadsStrict: %w", err)
	}
	if err := ktav.LoadsInto(src, &cfg); err != nil {
		return fmt.Errorf("package LoadsInto: %w", err)
	}
	if _, err := ktav.Dumps(map[string]any{"g": []any{int64(1)}}); err != nil {
		return fmt.Errorf("package Dumps: %w", err)
	}
	if _, err := ktav.DumpsForceStrings(map[string]any{"g": true}); err != nil {
		return fmt.Errorf("package DumpsForceStrings: %w", err)
	}
	if _, err := ktav.EmitCanonical(map[string]any{"g": int64(1)}); err != nil {
		return fmt.Errorf("package EmitCanonical: %w", err)
	}
	if _, err := ktav.FormatSource(src); err != nil {
		return fmt.Errorf("package FormatSource: %w", err)
	}
	if _, err := ktav.CanonicalFromSource(src); err != nil {
		return fmt.Errorf("package CanonicalFromSource: %w", err)
	}

	fmt.Println("pureconsumer: all pure public operations OK (no native artifact)")
	return nil
}
