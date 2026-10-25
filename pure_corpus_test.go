package ktav_test

// Phase 3: the full spec-0.9 conformance corpus executed SEPARATELY
// through BOTH backends. The pure legs run unconditionally — pure mode
// must work without the native library ever being present — and the
// native legs are exercised only when the cabi artifact is available.
// Every expectation is the immutable corpus oracle from disk; a pure
// result that disagrees with a pinned native expectation fails here.
// Each backend leg uses its own ktav.Operations instance, so no leg can
// contaminate the package-global default backend.

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	ktav "github.com/ktav-lang/golang"
	"github.com/ktav-lang/golang/internal/native"
	"github.com/ktav-lang/golang/pure"
)

// pureForced reports whether the suite runs in pure-only mode (the CI
// "pure-backend" job sets KTAV_FORCE_PURE=1 before any test runs).
func pureForced() bool { return os.Getenv("KTAV_FORCE_PURE") == "1" }

// pureOps returns a fresh Operations bound to BackendPure. Instances
// never read or write the package-global backend, so tests (including
// future parallel ones) cannot contaminate each other.
func pureOps(t *testing.T) *pure.Operations {
	t.Helper()
	return pure.New()
}

// nativeOps returns a fresh Operations bound to BackendNative. Only
// call it behind hasCabi().
func nativeOps(t *testing.T) *ktav.Operations {
	t.Helper()
	ops, err := ktav.New(ktav.BackendNative)
	if err != nil {
		t.Fatalf("ktav.New(BackendNative): %v", err)
	}
	return ops
}

// hasCabi reports whether the native library artifact exists, without
// failing the test. Pure legs must never depend on it.
func hasCabi() bool {
	p := filepath.Join(cabiBuildDir, cabiName())
	if dir := os.Getenv("CARGO_TARGET_DIR"); dir != "" {
		p = filepath.Join(dir, "release", cabiName())
	}
	if override := os.Getenv("KTAV_LIB_PATH"); override != "" {
		p = override
	}
	_, err := os.Stat(p)
	return err == nil
}

// withBackend switches the package-GLOBAL backend and restores the
// suite-wide default afterwards (native normally, pure when
// KTAV_FORCE_PURE=1). Only the global-API test uses this; every corpus
// and pure test runs through its own instance instead.
func withBackend(t *testing.T, b ktav.Backend) {
	t.Helper()
	if err := ktav.UseBackend(b); err != nil {
		t.Fatalf("UseBackend(%d): %v", int(b), err)
	}
	prev := ktav.BackendNative
	if pureForced() {
		prev = ktav.BackendPure
	}
	t.Cleanup(func() {
		if err := ktav.UseBackend(prev); err != nil {
			t.Errorf("restore backend: %v", err)
		}
	})
}

// corpusFixturePaths walks one category directory and returns the
// sorted paths of the primary (.ktav) fixtures. Missing or empty
// categories fail (the corpus must never be silently skipped).
func corpusFixturePaths(t *testing.T, category string) []string {
	t.Helper()
	root := filepath.Join(requireSpec(t), category)
	var cases []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
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
		t.Fatalf("walk %s corpus: %v", category, err)
	}
	if len(cases) == 0 {
		t.Fatalf("category %s: no fixtures found (missing corpus must fail, not skip)", category)
	}
	if len(cases) != spec09Counts[category] {
		t.Fatalf("category %s: found %d fixtures, manifest pins %d", category, len(cases), spec09Counts[category])
	}
	return cases
}

// corpusRawFlags loads the manifest's fixture_flags and fails closed on
// any flag other than raw_bytes or on entries the manifest guard has
// not already vetted. Keyed EXACTLY as corpus_manifest_test.go builds
// its flag keys: "<category>/<fixture>", where <fixture> is the stem
// relative to the category directory (no category prefix).
func corpusRawFlags(t *testing.T) map[string]bool {
	t.Helper()
	manifest, err := loadCorpusManifest(requireSpec(t))
	if err != nil {
		t.Fatalf("load corpus manifest: %v", err)
	}
	raw := make(map[string]bool)
	for _, flag := range manifest.FixtureFlags {
		for _, name := range flag.Flags {
			if name != "raw_bytes" {
				t.Fatalf("unknown fixture flag %q on %s/%s (fail closed)", name, flag.Category, flag.Fixture)
			}
		}
		raw[flag.Category+"/"+filepath.ToSlash(flag.Fixture)] = true
	}
	return raw
}

// requireRawBytes asserts that raw_bytes-flagged fixtures really ARE
// invalid UTF-8 (the manifest guard's own contract, §6.15): reading one
// as valid text would be a corpus bug, and a flag on a valid file would
// mean the raw-byte preservation path is never actually exercised.
func requireRawBytes(t *testing.T, rawFlags map[string]bool, category, path string, src []byte) {
	t.Helper()
	rel, err := filepath.Rel(filepath.Join(requireSpec(t), category), path)
	if err != nil {
		t.Fatalf("rel %s under %s: %v", path, category, err)
	}
	stem := strings.TrimSuffix(filepath.ToSlash(rel), ".ktav")
	if !rawFlags[category+"/"+stem] {
		return
	}
	if utf8.Valid(src) {
		t.Fatalf("raw_bytes fixture %s/%s is valid UTF-8; corpus flag contract violated", category, stem)
	}
}

// TestPureCorpusValid runs every valid fixture through Loads on BOTH
// backends and compares against the .json oracle. The pure leg runs
// unconditionally; the native leg is a cross-check when the cabi
// artifact is present.
func TestPureCorpusValid(t *testing.T) {
	pure := pureOps(t)
	rawFlags := corpusRawFlags(t)

	for _, ktavPath := range corpusFixturePaths(t, "valid") {
		name := strings.TrimPrefix(ktavPath, requireSpec(t)+string(filepath.Separator))
		t.Run(name, func(t *testing.T) {
			srcBytes, err := readFixtureSource(ktavPath)
			if err != nil {
				t.Fatal(err)
			}
			requireRawBytes(t, rawFlags, "valid", ktavPath, srcBytes)
			oracleBytes, err := os.ReadFile(strings.TrimSuffix(ktavPath, ".ktav") + ".json")
			if err != nil {
				t.Fatalf("oracle missing: %v", err)
			}
			want, err := decodeOracle(oracleBytes)
			if err != nil {
				t.Fatalf("oracle decode: %v", err)
			}

			got, err := pure.Loads(string(srcBytes))
			if err != nil {
				t.Fatalf("pure Loads: %v\n--- input ---\n%s", err, srcBytes)
			}
			if !structEqual(got, want) {
				t.Fatalf("pure mismatch\nktav src:\n%s\npure got:  %#v\noracle:    %#v", srcBytes, got, want)
			}

			if hasCabi() {
				nat, err := nativeOps(t).Loads(string(srcBytes))
				if err != nil {
					t.Fatalf("native Loads: %v\n--- input ---\n%s", err, srcBytes)
				}
				if !structEqual(nat, want) {
					t.Fatalf("native disagrees with corpus oracle\nsrc:\n%s\nnative:  %#v\noracle:  %#v", srcBytes, nat, want)
				}
			}
		})
	}
}

// TestPureCorpusCanonical compares CanonicalFromSource byte-for-byte
// against every valid fixture's `.canonical.ktav` companion oracle, on
// the pure backend unconditionally and on the native backend when
// available. The companion file — never the PRIMARY source, never the
// other backend — is the expectation; a missing companion fails closed.
func TestPureCorpusCanonical(t *testing.T) {
	pure := pureOps(t)

	for _, ktavPath := range corpusFixturePaths(t, "valid") {
		canonicalPath := strings.TrimSuffix(ktavPath, ".ktav") + ".canonical.ktav"
		if _, err := os.Stat(canonicalPath); err != nil {
			t.Fatalf("canonical companion missing for %s: %v", ktavPath, err)
		}
		name := strings.TrimPrefix(ktavPath, requireSpec(t)+string(filepath.Separator))
		t.Run(name, func(t *testing.T) {
			src, err := readFixtureSource(ktavPath)
			if err != nil {
				t.Fatal(err)
			}
			oracle, err := os.ReadFile(canonicalPath)
			if err != nil {
				t.Fatal(err)
			}

			canonical, err := pure.CanonicalFromSource(string(src))
			if err != nil {
				t.Fatalf("pure CanonicalFromSource: %v\n--- input ---\n%s", err, src)
			}
			if canonical != string(oracle) {
				t.Fatalf("pure canonical mismatch\nwant:\n%s\ngot:\n%s", oracle, canonical)
			}

			if hasCabi() {
				nat, err := nativeOps(t).CanonicalFromSource(string(src))
				if err != nil {
					t.Fatalf("native CanonicalFromSource: %v\n--- input ---\n%s", err, src)
				}
				if nat != string(oracle) {
					t.Fatalf("native canonical disagrees with file oracle\nwant:\n%s\ngot:\n%s", oracle, nat)
				}
			}
		})
	}
}

// TestPureCorpusInvalid runs every invalid fixture through pure Loads
// and asserts the exact error envelope (class/reason/body/canonical/
// spec_section/line/line_text/path/span) against the .json oracle via
// the shared assertInvalidOracleError helper — the same pinned
// expectations the native conformance suite uses.
func TestPureCorpusInvalid(t *testing.T) {
	ops := pureOps(t)
	rawFlags := corpusRawFlags(t)

	for _, p := range corpusFixturePaths(t, "invalid") {
		name := strings.TrimPrefix(p, requireSpec(t)+string(filepath.Separator))
		t.Run(name, func(t *testing.T) {
			src, err := readFixtureSource(p)
			if err != nil {
				t.Fatal(err)
			}
			requireRawBytes(t, rawFlags, "invalid", p, src)
			oracleRaw, err := os.ReadFile(strings.TrimSuffix(p, ".ktav") + ".json")
			if err != nil {
				t.Fatalf("oracle missing: %v", err)
			}
			// Keep the PARSE error in its own variable: an `else if
			// err := ...` would shadow it, and the next check would
			// receive the previous assertion's nil instead of the
			// parse error.
			_, parseErr := ops.Loads(string(src))
			if parseErr == nil {
				t.Fatalf("pure Loads: expected parse error, got ok\n---\n%s", src)
			}
			if err := assertInvalidOracleError(parseErr, oracleRaw); err != nil {
				t.Fatalf("pure error envelope mismatch: %v\n--- input ---\n%s", err, src)
			}
			if err := assertExpectedLocation(parseErr, src, oracleRaw); err != nil {
				t.Fatalf("pure expected_location mismatch: %v\n--- input ---\n%s", err, src)
			}
		})
	}
}

// TestPureCorpusUnrepresentable checks that the pure writer refuses
// every unrepresentable JSON value with the pinned reason code.
func TestPureCorpusUnrepresentable(t *testing.T) {
	ops := pureOps(t)

	root := filepath.Join(requireSpec(t), "unrepresentable")
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("read unrepresentable corpus: %v", err)
	}
	var cases []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			cases = append(cases, filepath.Join(root, entry.Name()))
		}
	}
	if len(cases) == 0 {
		t.Fatal("no unrepresentable fixtures found")
	}
	if len(cases) != spec09Counts["unrepresentable"] {
		t.Fatalf("unrepresentable: found %d fixtures, manifest pins %d", len(cases), spec09Counts["unrepresentable"])
	}

	for _, p := range cases {
		name := strings.TrimPrefix(p, requireSpec(t)+string(filepath.Separator))
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
				{"Dumps", func() (string, error) { return ops.Dumps(value) }},
				{"EmitCanonical", func() (string, error) { return ops.EmitCanonical(value) }},
			} {
				out, err := call.fn()
				if err == nil {
					t.Fatalf("pure %s: expected writer to refuse value (reason %s), got ok:\n%s", call.label, reason, out)
				}
				if err := assertUnrepresentableError(err, reason, "pure "+call.label); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

// TestPureCorpusParseableUnrepresentable checks pure Loads accepts the
// document with the oracle value while every writer entry point refuses
// it with the pinned reason.
func TestPureCorpusParseableUnrepresentable(t *testing.T) {
	ops := pureOps(t)

	for _, p := range corpusFixturePaths(t, "parseable-unrepresentable") {
		name := strings.TrimPrefix(p, requireSpec(t)+string(filepath.Separator))
		t.Run(name, func(t *testing.T) {
			src, err := readFixtureSource(p)
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

			got, err := ops.Loads(string(src))
			if err != nil {
				t.Fatalf("pure Loads: %v\n--- input ---\n%s", err, src)
			}
			if !structEqual(got, want) {
				t.Fatalf("pure mismatch\nsrc:\n%s\ngot:    %#v\noracle: %#v", src, got, want)
			}

			if _, err := ops.CanonicalFromSource(string(src)); err == nil {
				t.Fatalf("pure CanonicalFromSource: expected refusal (reason %s)\n--- input ---\n%s", reason, src)
			} else if err := assertUnrepresentableError(err, reason, "pure CanonicalFromSource"); err != nil {
				t.Fatal(err)
			}
			for _, call := range []struct {
				label string
				fn    func(any) (string, error)
			}{
				{"Dumps", ops.Dumps},
				{"EmitCanonical", ops.EmitCanonical},
			} {
				out, err := call.fn(got)
				if err == nil {
					t.Fatalf("pure %s: expected refusal (reason %s), got ok:\n%s", call.label, reason, out)
				}
				if err := assertUnrepresentableError(err, reason, "pure "+call.label); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

// TestPureCorpusStrictLossy checks lax acceptance with the oracle value
// plus the exact LossyScalar payload (Body + Canonical) from the
// strict-lossy oracle, on the pure backend.
func TestPureCorpusStrictLossy(t *testing.T) {
	ops := pureOps(t)

	for _, p := range corpusFixturePaths(t, "strict-lossy") {
		name := strings.TrimPrefix(p, requireSpec(t)+string(filepath.Separator))
		t.Run(name, func(t *testing.T) {
			src, err := readFixtureSource(p)
			if err != nil {
				t.Fatal(err)
			}
			oracleRaw, err := os.ReadFile(strings.TrimSuffix(p, ".ktav") + ".json")
			if err != nil {
				t.Fatalf("oracle missing: %v", err)
			}
			var oracle struct {
				LaxValue      json.RawMessage `json:"lax_value"`
				ExpectedError string          `json:"expected_error"`
				Body          string          `json:"body"`
				Canonical     string          `json:"canonical"`
			}
			if err := json.Unmarshal(oracleRaw, &oracle); err != nil {
				t.Fatalf("oracle decode: %v", err)
			}
			want, err := decodeOracle(oracle.LaxValue)
			if err != nil {
				t.Fatalf("lax_value decode: %v", err)
			}

			got, err := ops.Loads(string(src))
			if err != nil {
				t.Fatalf("pure Loads: %v\n--- input ---\n%s", err, src)
			}
			if !structEqual(got, want) {
				t.Fatalf("pure lax mismatch\nsrc:\n%s\ngot:    %#v\noracle: %#v", src, got, want)
			}

			_, err = ops.LoadsStrict(string(src))
			var ktavErr *ktav.Error
			if !errors.As(err, &ktavErr) {
				t.Fatalf("pure LoadsStrict: want *ktav.Error, got %T (%v)", err, err)
			}
			if ktavErr.Class != oracle.ExpectedError || ktavErr.Body != oracle.Body || ktavErr.Canonical != oracle.Canonical {
				t.Fatalf("pure LoadsStrict error = {%s %q %q}, want {%s %q %q}",
					ktavErr.Class, ktavErr.Body, ktavErr.Canonical,
					oracle.ExpectedError, oracle.Body, oracle.Canonical)
			}
		})
	}
}

// TestPureCorpusNeverLoadsNative
// pure-mode corpus runs never trigger the native loader.
func TestPureCorpusNeverLoadsNative(t *testing.T) {
	if !pureForced() {
		t.Skip("native artifact present in this job; enforced by the pure-backend CI job (KTAV_FORCE_PURE=1)")
	}
	if native.Loaded() {
		t.Fatal("pure-mode corpus run loaded the native library; pure mode must never touch the loader")
	}
}
