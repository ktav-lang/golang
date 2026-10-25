package ktav_test

// Spec 0.9 diagnostics regression runner: the 72 new invalid/diagnostics09
// fixtures (30 EOF + 42 malformed UTF-8) must produce the pinned error
// class AND the pinned expected_location (1-based line, raw-byte UTF-8
// start, exactly one of exact end / minimum_end) on BOTH backends. The
// native envelope comes through the C ABI, the pure one from the
// in-process engine — a divergence between the two, an ignored
// minimum_end, or a decoded line_text under InvalidUtf8 fails here.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	ktav "github.com/ktav-lang/golang"
	"github.com/ktav-lang/golang/pure"
)

func TestSpec09DiagnosticsLocationsNativeAndPure(t *testing.T) {
	requireCabi(t)
	dir := filepath.Join(requireSpec(t), "invalid", "diagnostics09")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("diagnostics09 corpus missing — the spec gitlink must be pinned to the 0.9 tests tree: %v", err)
	}
	var cases []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".ktav") {
			cases = append(cases, filepath.Join(dir, entry.Name()))
		}
	}
	if len(cases) != diagnostics09Count {
		t.Fatalf("diagnostics09: found %d fixtures, spec 0.9 pins %d (gitlink or corpus drift)", len(cases), diagnostics09Count)
	}

	pureOps := pure.New()
	for _, p := range cases {
		name := strings.TrimSuffix(filepath.Base(p), ".ktav")
		t.Run(name, func(t *testing.T) {
			src, err := readFixtureSource(p)
			if err != nil {
				t.Fatal(err)
			}
			oracleRaw, err := os.ReadFile(strings.TrimSuffix(p, ".ktav") + ".json")
			if err != nil {
				t.Fatalf("oracle missing: %v", err)
			}

			// Native leg (through the C ABI envelope).
			_, nerr := ktav.Loads(string(src))
			if nerr == nil {
				t.Fatalf("native Loads: expected parse error, got ok\n---\n%s", src)
			}
			if err := assertInvalidOracleError(nerr, oracleRaw); err != nil {
				t.Fatalf("native envelope: %v\n--- input ---\n%s", err, src)
			}
			if err := assertExpectedLocation(nerr, src, oracleRaw); err != nil {
				t.Fatalf("native expected_location: %v\n--- input ---\n%s", err, src)
			}

			// Pure leg (same pinned oracles, in-process engine).
			_, perr := pureOps.Loads(string(src))
			if perr == nil {
				t.Fatalf("pure Loads: expected parse error, got ok\n---\n%s", src)
			}
			if err := assertInvalidOracleError(perr, oracleRaw); err != nil {
				t.Fatalf("pure envelope: %v\n--- input ---\n%s", err, src)
			}
			if err := assertExpectedLocation(perr, src, oracleRaw); err != nil {
				t.Fatalf("pure expected_location: %v\n--- input ---\n%s", err, src)
			}

			// Both backends must agree on the class at minimum: a
			// malformed-UTF-8 tail must not lose to an earlier grammar
			// error on exactly one side.
			var ne, pe *ktav.Error
			if !asKtavError(nerr, &ne) || !asKtavError(perr, &pe) {
				t.Fatalf("errors are not *ktav.Error: %T / %T", nerr, perr)
			}
			if ne.Class != pe.Class {
				t.Fatalf("backend class divergence: native=%s pure=%s\n--- input ---\n%s", ne.Class, pe.Class, src)
			}
		})
	}
}

func asKtavError(err error, target **ktav.Error) bool {
	e, ok := err.(*ktav.Error)
	if ok {
		*target = e
	}
	return ok
}
