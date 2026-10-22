package main

// Corpus manifest guard + per-category execution. This is a direct port
// of the in-repo corpus_manifest_test.go guard (fail closed, never
// skip) plus pure_corpus_test.go category semantics, running everything
// through the INSTALLED github.com/ktav-lang/golang/pure package.

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/ktav-lang/golang/pure"
)

type corpusManifest struct {
	Comment       string                   `json:"$comment"`
	SchemaVersion int                      `json:"schema_version"`
	Categories    map[string]categoryEntry `json:"categories"`
	FixtureFlags  []fixtureFlag            `json:"fixture_flags"`
}

type categoryEntry struct {
	Count int `json:"count"`
}

type fixtureFlag struct {
	Category string   `json:"category"`
	Fixture  string   `json:"fixture"`
	Flags    []string `json:"flags"`
	Note     string   `json:"note"`
}

var spec08Counts = map[string]int{
	"valid": 228, "invalid": 90, "unrepresentable": 5,
	"parseable-unrepresentable": 4, "strict-lossy": 13,
}

func loadCorpusManifest(root string) (*corpusManifest, error) {
	raw, err := os.ReadFile(filepath.Join(root, "manifest.json"))
	if err != nil {
		return nil, fmt.Errorf("read manifest.json: %w", err)
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var manifest corpusManifest
	if err := dec.Decode(&manifest); err != nil {
		return nil, fmt.Errorf("decode manifest.json: %w", err)
	}
	if err := dec.Decode(new(any)); err != io.EOF {
		return nil, fmt.Errorf("decode manifest.json trailing data: %v", err)
	}
	if manifest.SchemaVersion != 1 {
		return nil, fmt.Errorf("unsupported manifest schema_version %d (want 1)", manifest.SchemaVersion)
	}
	return &manifest, nil
}

// validateAndCollect is the manifest guard from corpus_manifest_test.go.
// On success it also returns, per category, the sorted list of fixture
// stems discovered on disk (each with a complete companion set).
func validateAndCollect(root string, manifest *corpusManifest, expected map[string]int) (map[string][]string, error) {
	if len(manifest.Categories) != len(expected) {
		return nil, fmt.Errorf("manifest has %d categories; want exactly %d", len(manifest.Categories), len(expected))
	}
	for category, count := range expected {
		entry, ok := manifest.Categories[category]
		if !ok {
			return nil, fmt.Errorf("manifest missing category %q", category)
		}
		if entry.Count != count {
			return nil, fmt.Errorf("manifest category %q count is %d; want %d", category, entry.Count, count)
		}
	}
	for category := range manifest.Categories {
		if _, ok := expected[category]; !ok {
			return nil, fmt.Errorf("unknown manifest category %q", category)
		}
	}

	dirs, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("read corpus directory: %w", err)
	}
	seenDirs := make(map[string]bool)
	for _, entry := range dirs {
		if entry.Name() == "manifest.json" || entry.Name() == "boundary-fixtures.json" {
			continue
		}
		if !entry.IsDir() {
			return nil, fmt.Errorf("unexpected file in corpus root: %q", entry.Name())
		}
		if _, ok := expected[entry.Name()]; !ok {
			return nil, fmt.Errorf("unknown corpus category directory %q", entry.Name())
		}
		seenDirs[entry.Name()] = true
	}
	for category := range expected {
		if !seenDirs[category] {
			return nil, fmt.Errorf("missing corpus category directory %q", category)
		}
	}

	fixtures := make(map[string]map[string]map[string]bool)
	stems := make(map[string][]string)
	for category := range expected {
		fixtures[category] = make(map[string]map[string]bool)
		categoryRoot := filepath.Join(root, category)
		err := filepath.WalkDir(categoryRoot, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(categoryRoot, path)
			if err != nil {
				return err
			}
			stem, suffix, ok := fixturePart(filepath.ToSlash(rel))
			if !ok {
				return fmt.Errorf("unexpected file in %s: %q", category, filepath.ToSlash(rel))
			}
			parts := fixtures[category][stem]
			if parts == nil {
				parts = make(map[string]bool)
				fixtures[category][stem] = parts
			}
			if parts[suffix] {
				return fmt.Errorf("duplicate %s companion for %s/%s", suffix, category, stem)
			}
			parts[suffix] = true
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("inspect %s: %w", category, err)
		}
		if len(fixtures[category]) != manifest.Categories[category].Count {
			return nil, fmt.Errorf("category %q has %d fixtures; manifest declares %d", category, len(fixtures[category]), manifest.Categories[category].Count)
		}
		for stem, parts := range fixtures[category] {
			want := companionSet(category)
			if len(parts) != len(want) {
				return nil, fmt.Errorf("fixture %s/%s has incomplete companions", category, stem)
			}
			for suffix := range want {
				if !parts[suffix] {
					return nil, fmt.Errorf("fixture %s/%s missing .%s companion", category, stem, suffix)
				}
			}
			stems[category] = append(stems[category], stem)
		}
		sort.Strings(stems[category])
	}

	flagged := make(map[string]bool)
	for _, flag := range manifest.FixtureFlags {
		if len(flag.Flags) == 0 {
			return nil, fmt.Errorf("fixture flag entry %s/%s has no flags", flag.Category, flag.Fixture)
		}
		if _, ok := expected[flag.Category]; !ok {
			return nil, fmt.Errorf("fixture flag references unknown category %q", flag.Category)
		}
		parts, ok := fixtures[flag.Category][filepath.ToSlash(flag.Fixture)]
		if !ok {
			return nil, fmt.Errorf("fixture flag references missing fixture %s/%s", flag.Category, flag.Fixture)
		}
		key := flag.Category + "/" + filepath.ToSlash(flag.Fixture)
		if flagged[key] {
			return nil, fmt.Errorf("duplicate fixture flag entry for %s", key)
		}
		flagged[key] = true
		seenFlags := make(map[string]bool)
		for _, name := range flag.Flags {
			if name != "raw_bytes" {
				return nil, fmt.Errorf("unknown fixture flag %q on %s", name, key)
			}
			if seenFlags[name] {
				return nil, fmt.Errorf("duplicate fixture flag %q on %s", name, key)
			}
			seenFlags[name] = true
		}
		if flag.Category != "invalid" || !parts["ktav"] {
			return nil, fmt.Errorf("raw_bytes flag requires an invalid .ktav fixture: %s", key)
		}
		raw, err := os.ReadFile(filepath.Join(root, flag.Category, filepath.FromSlash(flag.Fixture)+".ktav"))
		if err != nil {
			return nil, fmt.Errorf("read raw_bytes fixture %s: %w", key, err)
		}
		if utf8.Valid(raw) {
			return nil, fmt.Errorf("raw_bytes fixture %s is valid UTF-8", key)
		}
	}
	for category, categoryFixtures := range fixtures {
		for stem, parts := range categoryFixtures {
			if !parts["ktav"] || flagged[category+"/"+stem] {
				continue
			}
			raw, err := os.ReadFile(filepath.Join(root, category, filepath.FromSlash(stem)+".ktav"))
			if err != nil {
				return nil, fmt.Errorf("read fixture %s/%s: %w", category, stem, err)
			}
			if !utf8.Valid(raw) {
				return nil, fmt.Errorf("unflagged fixture %s/%s is not valid UTF-8", category, stem)
			}
		}
	}
	return stems, nil
}

func fixturePart(path string) (stem, suffix string, ok bool) {
	for _, ext := range []string{".canonical.ktav", ".ktav", ".json"} {
		if strings.HasSuffix(path, ext) {
			return strings.TrimSuffix(path, ext), strings.TrimPrefix(ext, "."), true
		}
	}
	return "", "", false
}

func companionSet(category string) map[string]bool {
	switch category {
	case "valid":
		return map[string]bool{"ktav": true, "json": true, "canonical.ktav": true}
	case "invalid", "parseable-unrepresentable", "strict-lossy":
		return map[string]bool{"ktav": true, "json": true}
	default:
		return map[string]bool{"json": true}
	}
}

// runCorpus validates the manifest guard, then runs ALL five categories
// through the installed pure package. Any problem is a hard error.
func runCorpus(specRoot string) error {
	manifest, err := loadCorpusManifest(specRoot)
	if err != nil {
		return err
	}
	stems, err := validateAndCollect(specRoot, manifest, spec08Counts)
	if err != nil {
		return err
	}

	counts := map[string]int{}
	if counts["valid"], err = runValid(specRoot, stems["valid"]); err != nil {
		return err
	}
	if counts["invalid"], err = runInvalid(specRoot, stems["invalid"]); err != nil {
		return err
	}
	if counts["unrepresentable"], err = runUnrepresentable(specRoot); err != nil {
		return err
	}
	if counts["parseable-unrepresentable"], err = runParseableUnrepresentable(specRoot, stems["parseable-unrepresentable"]); err != nil {
		return err
	}
	if counts["strict-lossy"], err = runStrictLossy(specRoot, stems["strict-lossy"]); err != nil {
		return err
	}
	fmt.Printf("pureconsumer: corpus %s\n", func() string {
		parts := make([]string, 0, len(spec08Counts))
		for _, category := range []string{"valid", "invalid", "unrepresentable", "parseable-unrepresentable", "strict-lossy"} {
			parts = append(parts, fmt.Sprintf("%s=%d", category, counts[category]))
		}
		return strings.Join(parts, " ")
	}())
	return nil
}

// runValid mirrors TestPureCorpusValid + TestPureCorpusCanonical plus a
// GoValue→Ktav→GoValue round trip: parse vs JSON oracle, canonical
// bytes vs .canonical.ktav companion, and dump(parse(src)) must parse
// back to the same value.
func runValid(root string, stems []string) (int, error) {
	for _, stem := range stems {
		ktavPath := filepath.Join(root, "valid", filepath.FromSlash(stem)+".ktav")
		src, err := os.ReadFile(ktavPath)
		if err != nil {
			return 0, fmt.Errorf("read fixture %s: %w", stem, err)
		}
		if !utf8.Valid(src) {
			return 0, fmt.Errorf("valid fixture %s is not valid UTF-8", stem)
		}
		oracleBytes, err := os.ReadFile(filepath.Join(root, "valid", filepath.FromSlash(stem)+".json"))
		if err != nil {
			return 0, fmt.Errorf("oracle missing for %s: %w", stem, err)
		}
		want, err := decodeOracle(oracleBytes)
		if err != nil {
			return 0, fmt.Errorf("oracle decode %s: %w", stem, err)
		}
		canonicalBytes, err := os.ReadFile(filepath.Join(root, "valid", filepath.FromSlash(stem)+".canonical.ktav"))
		if err != nil {
			return 0, fmt.Errorf("canonical companion missing for %s: %w", stem, err)
		}

		got, err := pure.Loads(string(src))
		if err != nil {
			return 0, fmt.Errorf("valid %s: pure Loads: %w", stem, err)
		}
		if !structEqual(got, want) {
			return 0, fmt.Errorf("valid %s: value mismatch\ngot:    %#v\noracle: %#v", stem, got, want)
		}

		canonical, err := pure.CanonicalFromSource(string(src))
		if err != nil {
			return 0, fmt.Errorf("valid %s: pure CanonicalFromSource: %w", stem, err)
		}
		if canonical != string(canonicalBytes) {
			return 0, fmt.Errorf("valid %s: canonical mismatch\nwant:\n%s\ngot:\n%s", stem, canonicalBytes, canonical)
		}

		dumped, err := pure.Dumps(got)
		if err != nil {
			return 0, fmt.Errorf("valid %s: pure Dumps: %w", stem, err)
		}
		reparsed, err := pure.Loads(dumped)
		if err != nil {
			return 0, fmt.Errorf("valid %s: pure Loads(Dumps(v)): %w", stem, err)
		}
		if !structEqual(reparsed, got) {
			return 0, fmt.Errorf("valid %s: round trip mismatch\ndump:   %s\norig:   %#v\nreparsed: %#v", stem, dumped, got, reparsed)
		}
	}
	return len(stems), nil
}

// runInvalid mirrors TestPureCorpusInvalid: Loads must fail with the
// exact error envelope from the .json oracle.
func runInvalid(root string, stems []string) (int, error) {
	for _, stem := range stems {
		src, err := os.ReadFile(filepath.Join(root, "invalid", filepath.FromSlash(stem)+".ktav"))
		if err != nil {
			return 0, fmt.Errorf("read fixture %s: %w", stem, err)
		}
		oracleRaw, err := os.ReadFile(filepath.Join(root, "invalid", filepath.FromSlash(stem)+".json"))
		if err != nil {
			return 0, fmt.Errorf("oracle missing for %s: %w", stem, err)
		}
		_, err = pure.Loads(string(src))
		if err == nil {
			return 0, fmt.Errorf("invalid %s: expected parse error, got ok\n---\n%s", stem, src)
		}
		if err := assertInvalidOracleError(err, oracleRaw); err != nil {
			return 0, fmt.Errorf("invalid %s: %w", stem, err)
		}
	}
	return len(stems), nil
}

// runUnrepresentable mirrors TestPureCorpusUnrepresentable: both writer
// entry points must refuse with the pinned reason.
func runUnrepresentable(root string) (int, error) {
	categoryRoot := filepath.Join(root, "unrepresentable")
	entries, err := os.ReadDir(categoryRoot)
	if err != nil {
		return 0, fmt.Errorf("read unrepresentable corpus: %w", err)
	}
	var cases []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			cases = append(cases, entry.Name())
		}
	}
	if len(cases) != spec08Counts["unrepresentable"] {
		return 0, fmt.Errorf("unrepresentable: found %d fixtures, manifest pins %d", len(cases), spec08Counts["unrepresentable"])
	}
	sort.Strings(cases)
	for _, name := range cases {
		raw, err := os.ReadFile(filepath.Join(categoryRoot, name))
		if err != nil {
			return 0, err
		}
		value, reason, err := decodeUnrepJSON(raw)
		if err != nil {
			return 0, fmt.Errorf("unrepresentable %s: oracle decode: %w", name, err)
		}
		for _, call := range []struct {
			label string
			fn    func() (string, error)
		}{
			{"Dumps", func() (string, error) { return pure.Dumps(value) }},
			{"EmitCanonical", func() (string, error) { return pure.EmitCanonical(value) }},
		} {
			out, err := call.fn()
			if err == nil {
				return 0, fmt.Errorf("unrepresentable %s: pure %s: expected refusal (reason %s), got ok:\n%s", name, call.label, reason, out)
			}
			if err := assertUnrepresentableError(err, reason, "unrepresentable "+name+" pure "+call.label); err != nil {
				return 0, err
			}
		}
	}
	return len(cases), nil
}

// runParseableUnrepresentable mirrors TestPureCorpusParseableUnrepresentable.
func runParseableUnrepresentable(root string, stems []string) (int, error) {
	for _, stem := range stems {
		src, err := os.ReadFile(filepath.Join(root, "parseable-unrepresentable", filepath.FromSlash(stem)+".ktav"))
		if err != nil {
			return 0, fmt.Errorf("read fixture %s: %w", stem, err)
		}
		oracleRaw, err := os.ReadFile(filepath.Join(root, "parseable-unrepresentable", filepath.FromSlash(stem)+".json"))
		if err != nil {
			return 0, fmt.Errorf("oracle missing for %s: %w", stem, err)
		}
		want, reason, err := decodeUnrepJSON(oracleRaw)
		if err != nil {
			return 0, fmt.Errorf("parseable-unrepresentable %s: oracle decode: %w", stem, err)
		}

		got, err := pure.Loads(string(src))
		if err != nil {
			return 0, fmt.Errorf("parseable-unrepresentable %s: pure Loads: %w", stem, err)
		}
		if !structEqual(got, want) {
			return 0, fmt.Errorf("parseable-unrepresentable %s: value mismatch\ngot:    %#v\noracle: %#v", stem, got, want)
		}

		for _, call := range []struct {
			label string
			fn    func() (string, error)
		}{
			{"CanonicalFromSource", func() (string, error) { return pure.CanonicalFromSource(string(src)) }},
			{"Dumps", func() (string, error) { return pure.Dumps(got) }},
			{"EmitCanonical", func() (string, error) { return pure.EmitCanonical(got) }},
		} {
			out, err := call.fn()
			if err == nil {
				return 0, fmt.Errorf("parseable-unrepresentable %s: pure %s: expected refusal (reason %s), got ok:\n%s", stem, call.label, reason, out)
			}
			if err := assertUnrepresentableError(err, reason, "parseable-unrepresentable "+stem+" pure "+call.label); err != nil {
				return 0, err
			}
		}
	}
	return len(stems), nil
}

// runStrictLossy mirrors TestPureCorpusStrictLossy.
func runStrictLossy(root string, stems []string) (int, error) {
	for _, stem := range stems {
		src, err := os.ReadFile(filepath.Join(root, "strict-lossy", filepath.FromSlash(stem)+".ktav"))
		if err != nil {
			return 0, fmt.Errorf("read fixture %s: %w", stem, err)
		}
		oracleRaw, err := os.ReadFile(filepath.Join(root, "strict-lossy", filepath.FromSlash(stem)+".json"))
		if err != nil {
			return 0, fmt.Errorf("oracle missing for %s: %w", stem, err)
		}
		var oracle struct {
			LaxValue      json.RawMessage `json:"lax_value"`
			ExpectedError string          `json:"expected_error"`
			Body          string          `json:"body"`
			Canonical     string          `json:"canonical"`
		}
		if err := json.Unmarshal(oracleRaw, &oracle); err != nil {
			return 0, fmt.Errorf("strict-lossy %s: oracle decode: %w", stem, err)
		}
		want, err := decodeOracle(oracle.LaxValue)
		if err != nil {
			return 0, fmt.Errorf("strict-lossy %s: lax_value decode: %w", stem, err)
		}

		got, err := pure.Loads(string(src))
		if err != nil {
			return 0, fmt.Errorf("strict-lossy %s: pure Loads: %w", stem, err)
		}
		if !structEqual(got, want) {
			return 0, fmt.Errorf("strict-lossy %s: lax value mismatch\ngot:    %#v\noracle: %#v", stem, got, want)
		}

		_, err = pure.LoadsStrict(string(src))
		var ktavErr *pure.Error
		if err == nil {
			return 0, fmt.Errorf("strict-lossy %s: pure LoadsStrict: expected error, got ok", stem)
		}
		if !errors.As(err, &ktavErr) {
			return 0, fmt.Errorf("strict-lossy %s: pure LoadsStrict: want *pure.Error, got %T (%v)", stem, err, err)
		}
		if ktavErr.Class != oracle.ExpectedError || ktavErr.Body != oracle.Body || ktavErr.Canonical != oracle.Canonical {
			return 0, fmt.Errorf("strict-lossy %s: pure LoadsStrict error = {%s %q %q}, want {%s %q %q}",
				stem, ktavErr.Class, ktavErr.Body, ktavErr.Canonical,
				oracle.ExpectedError, oracle.Body, oracle.Canonical)
		}
	}
	return len(stems), nil
}
