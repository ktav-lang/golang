package main

// Entry point + preserved smoke checks. The packaged pure consumer may
// import ONLY the Go standard library and github.com/ktav-lang/golang/pure
// (which itself imports only internal/engine). No ktav root package, no
// internal/*, no native loader.

import (
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/ktav-lang/golang/pure"
)

func main() {
	spec := flag.String("spec", "", "path to the spec corpus tests dir (default: $KTAV_SPEC_DIR or spec/versions/0.8/tests)")
	flag.Parse()
	if err := run(*spec); err != nil {
		fmt.Fprintln(os.Stderr, "pureconsumer:", err)
		os.Exit(1)
	}
}

func run(specFlag string) error {
	if err := runSmoke(); err != nil {
		return err
	}
	if err := runNegativeControls(); err != nil {
		return err
	}
	specRoot := specFlag
	if specRoot == "" {
		specRoot = os.Getenv("KTAV_SPEC_DIR")
	}
	if specRoot == "" {
		specRoot = filepath.Join("spec", "versions", "0.8", "tests")
	}
	if err := runCorpus(specRoot); err != nil {
		return err
	}
	fmt.Println("pureconsumer: all package and instance operations OK; full 0.8 corpus OK")
	return nil
}

// runSmoke is the original packaged-consumer smoke check, preserved
// unchanged. It exercises every package-level and instance operation of
// the installed github.com/ktav-lang/golang/pure artifact.
func runSmoke() error {
	ops := pure.New()
	src := "server: {\n    host: 127.0.0.1\n    port: 8080\n}\ntags: [prod, edge]\nratio: 0.5\n"
	v, err := pure.Loads(src)
	if err != nil {
		return fmt.Errorf("package Loads: %w", err)
	}
	m, ok := v.(map[string]any)
	if !ok || m["ratio"] != 0.5 {
		return fmt.Errorf("package Loads value = %#v", v)
	}
	strict, err := pure.LoadsStrict(src)
	if err != nil {
		return fmt.Errorf("package LoadsStrict: %w", err)
	}
	if _, ok := strict.(map[string]any); !ok {
		return fmt.Errorf("strict type = %T", strict)
	}
	var cfg struct {
		Server struct {
			Host string
			Port int
		}
		Tags []string
	}
	if err := pure.LoadsInto(src, &cfg); err != nil {
		return fmt.Errorf("package LoadsInto: %w", err)
	}
	if cfg.Server.Host != "127.0.0.1" || cfg.Server.Port != 8080 || len(cfg.Tags) != 2 {
		return fmt.Errorf("typed value = %#v", cfg)
	}

	opsValue, err := ops.Loads(src)
	if err != nil || fmt.Sprintf("%#v", opsValue) != fmt.Sprintf("%#v", v) {
		return fmt.Errorf("instance Loads = %#v, %v", opsValue, err)
	}
	if _, err := ops.LoadsStrict(src); err != nil {
		return fmt.Errorf("instance LoadsStrict: %w", err)
	}
	if err := ops.LoadsInto(src, &cfg); err != nil {
		return fmt.Errorf("instance LoadsInto: %w", err)
	}
	for _, call := range []struct {
		name string
		fn   func() (string, error)
	}{
		{"Dumps", func() (string, error) { return pure.Dumps(map[string]any{"a": int64(1), "b": "x"}) }},
		{"DumpsForceStrings", func() (string, error) { return pure.DumpsForceStrings(map[string]any{"n": int64(7)}) }},
		{"EmitCanonical", func() (string, error) { return pure.EmitCanonical(map[string]any{"z": math.Copysign(0, -1)}) }},
		{"FormatSource", func() (string, error) { return pure.FormatSource("a:   1\n\n\n## keep\nb: 2\n") }},
		{"CanonicalFromSource", func() (string, error) { return pure.CanonicalFromSource("k: [1, 2]\n") }},
	} {
		out, err := call.fn()
		if err != nil {
			return fmt.Errorf("%s: %w", call.name, err)
		}
		switch call.name {
		case "Dumps":
			if out != "a: 1\nb: x\n" {
				return fmt.Errorf("Dumps = %q", out)
			}
		case "DumpsForceStrings":
			if out != "n:: 7\n" {
				return fmt.Errorf("DumpsForceStrings = %q", out)
			}
		case "EmitCanonical":
			if out != "z: -0.0\n" {
				return fmt.Errorf("EmitCanonical = %q", out)
			}
		case "CanonicalFromSource":
			if out != "k: [\n    1\n    2\n]\n" {
				return fmt.Errorf("CanonicalFromSource = %q", out)
			}
		}
	}
	if out, err := ops.Dumps(map[string]any{"a": int64(1)}); err != nil || out != "a: 1\n" {
		return fmt.Errorf("instance Dumps = %q, %v", out, err)
	}
	if out, err := ops.DumpsForceStrings(map[string]any{"n": int64(7)}); err != nil || out != "n:: 7\n" {
		return fmt.Errorf("instance force strings = %q, %v", out, err)
	}
	if out, err := ops.EmitCanonical(map[string]any{"a": int64(1)}); err != nil || out != "a: 1\n" {
		return fmt.Errorf("instance canonical = %q, %v", out, err)
	}
	formatted, err := pure.FormatSource("a:   1\n\n\n## keep\nb: 2\n")
	if err != nil {
		return err
	}
	if len(strings.TrimSpace(formatted)) == 0 || !strings.Contains(formatted, "## keep") {
		return fmt.Errorf("FormatSource lost comment or content: %q", formatted)
	}
	if again, err := ops.FormatSource(formatted); err != nil || again != formatted {
		return fmt.Errorf("FormatSource fixed point: %q %v", again, err)
	}
	if out, err := ops.CanonicalFromSource("k: [1, 2]\n"); err != nil || out != "k: [\n    1\n    2\n]\n" {
		return fmt.Errorf("instance CanonicalFromSource = %q, %v", out, err)
	}
	_, err = pure.Loads("anchor: ok\njust some text\n")
	var structured *pure.Error
	if !errors.As(err, &structured) || structured.Class != "MissingSeparator" || structured.Span == nil || structured.Span.Start >= structured.Span.End {
		return fmt.Errorf("structured error = %#v (%v)", structured, err)
	}
	fmt.Println("pureconsumer: all package and instance operations OK")
	return nil
}

// runNegativeControls is a consumer-internal sanity check of the oracle
// helpers (structEqual/decodeOracle/decodeUnrepJSON) at their known
// sharp edges: the structEqual map branch must reject equal-length maps
// with different keys (the zero-value presence trap), and both oracle
// decoders must fail closed on any trailing content after the JSON
// value while accepting pure whitespace. They run inside the packaged
// consumer so CI exercises the exact shipped helpers, not a copy.
func runNegativeControls() error {
	// structEqual map branch: different keys, same length → unequal.
	if structEqual(
		map[string]any{"a": nil},
		map[string]any{"b": nil},
	) {
		return fmt.Errorf("negative: structEqual accepted {a:nil} vs {b:nil}")
	}
	if structEqual(
		map[string]any{"a": int64(1)},
		map[string]any{"b": int64(1)},
	) {
		return fmt.Errorf("negative: structEqual accepted {a:1} vs {b:1}")
	}
	// Equal maps with nil values must still compare equal.
	if !structEqual(
		map[string]any{"a": nil},
		map[string]any{"a": nil},
	) {
		return fmt.Errorf("negative: structEqual rejected {a:nil} vs {a:nil}")
	}
	// Same trap nested one level deeper.
	if structEqual(
		map[string]any{"o": map[string]any{"a": nil}},
		map[string]any{"o": map[string]any{"b": nil}},
	) {
		return fmt.Errorf("negative: structEqual accepted nested {a:nil} vs {b:nil}")
	}
	// Slices stay order-sensitive: a known unequal pair must differ.
	if structEqual(
		[]any{int64(1), int64(2)},
		[]any{int64(2), int64(1)},
	) {
		return fmt.Errorf("negative: structEqual accepted [1,2] vs [2,1]")
	}
	// Decoders: trailing JSON value → error.
	if _, err := decodeOracle([]byte(`{"k": 1} 2`)); err == nil {
		return fmt.Errorf("negative: decodeOracle accepted trailing JSON")
	}
	if _, _, err := decodeUnrepJSON([]byte(`{"value": 1, "unrepresentable_reason": "NonFiniteFloat"} []`)); err == nil {
		return fmt.Errorf("negative: decodeUnrepJSON accepted trailing JSON")
	}
	// Trailing garbage → error.
	if _, err := decodeOracle([]byte(`{"k": 1} oops`)); err == nil {
		return fmt.Errorf("negative: decodeOracle accepted trailing garbage")
	}
	if _, _, err := decodeUnrepJSON([]byte(`{"value": 1, "unrepresentable_reason": "NonFiniteFloat"} x`)); err == nil {
		return fmt.Errorf("negative: decodeUnrepJSON accepted trailing garbage")
	}
	// Whitespace-only trailing content → OK.
	if v, err := decodeOracle([]byte("  {\"k\": 1} \n\t ")); err != nil || v == nil {
		return fmt.Errorf("negative: decodeOracle rejected whitespace-padded JSON: %v", err)
	}
	if v, _, err := decodeUnrepJSON([]byte("\n {\"value\": 1, \"unrepresentable_reason\": \"NonFiniteFloat\"}\t ")); err != nil || v == nil {
		return fmt.Errorf("negative: decodeUnrepJSON rejected whitespace-padded JSON: %v", err)
	}
	fmt.Println("pureconsumer: oracle helper negative controls OK")
	return nil
}
