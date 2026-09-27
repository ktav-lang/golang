package ktav_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"

	ktav "github.com/ktav-lang/golang"
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
	"valid": 223, "invalid": 74, "unrepresentable": 5,
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

func validateSpec08Corpus(root string) error {
	manifest, err := loadCorpusManifest(root)
	if err != nil {
		return err
	}
	return validateCorpus(root, manifest, spec08Counts)
}

func validateCorpus(root string, manifest *corpusManifest, expected map[string]int) error {
	if manifest.SchemaVersion != 1 {
		return fmt.Errorf("unsupported manifest schema_version %d (want 1)", manifest.SchemaVersion)
	}
	if len(manifest.Categories) != len(expected) {
		return fmt.Errorf("manifest has %d categories; want exactly %d", len(manifest.Categories), len(expected))
	}
	for category, count := range expected {
		entry, ok := manifest.Categories[category]
		if !ok {
			return fmt.Errorf("manifest missing category %q", category)
		}
		if entry.Count != count {
			return fmt.Errorf("manifest category %q count is %d; want %d", category, entry.Count, count)
		}
	}
	for category := range manifest.Categories {
		if _, ok := expected[category]; !ok {
			return fmt.Errorf("unknown manifest category %q", category)
		}
	}

	dirs, err := os.ReadDir(root)
	if err != nil {
		return fmt.Errorf("read corpus directory: %w", err)
	}
	seenDirs := make(map[string]bool)
	for _, entry := range dirs {
		if entry.Name() == "manifest.json" || entry.Name() == "boundary-fixtures.json" {
			continue
		}
		if !entry.IsDir() {
			return fmt.Errorf("unexpected file in corpus root: %q", entry.Name())
		}
		if _, ok := expected[entry.Name()]; !ok {
			return fmt.Errorf("unknown corpus category directory %q", entry.Name())
		}
		seenDirs[entry.Name()] = true
	}
	for category := range expected {
		if !seenDirs[category] {
			return fmt.Errorf("missing corpus category directory %q", category)
		}
	}

	fixtures := make(map[string]map[string]map[string]bool)
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
			return fmt.Errorf("inspect %s: %w", category, err)
		}
		if len(fixtures[category]) != manifest.Categories[category].Count {
			return fmt.Errorf("category %q has %d fixtures; manifest declares %d", category, len(fixtures[category]), manifest.Categories[category].Count)
		}
		for stem, parts := range fixtures[category] {
			want := companionSet(category)
			if len(parts) != len(want) {
				return fmt.Errorf("fixture %s/%s has incomplete companions", category, stem)
			}
			for suffix := range want {
				if !parts[suffix] {
					return fmt.Errorf("fixture %s/%s missing .%s companion", category, stem, suffix)
				}
			}
		}
	}

	flagged := make(map[string]bool)
	for _, flag := range manifest.FixtureFlags {
		if len(flag.Flags) == 0 {
			return fmt.Errorf("fixture flag entry %s/%s has no flags", flag.Category, flag.Fixture)
		}
		if _, ok := expected[flag.Category]; !ok {
			return fmt.Errorf("fixture flag references unknown category %q", flag.Category)
		}
		parts, ok := fixtures[flag.Category][filepath.ToSlash(flag.Fixture)]
		if !ok {
			return fmt.Errorf("fixture flag references missing fixture %s/%s", flag.Category, flag.Fixture)
		}
		key := flag.Category + "/" + filepath.ToSlash(flag.Fixture)
		if flagged[key] {
			return fmt.Errorf("duplicate fixture flag entry for %s", key)
		}
		flagged[key] = true
		seenFlags := make(map[string]bool)
		for _, name := range flag.Flags {
			if name != "raw_bytes" {
				return fmt.Errorf("unknown fixture flag %q on %s", name, key)
			}
			if seenFlags[name] {
				return fmt.Errorf("duplicate fixture flag %q on %s", name, key)
			}
			seenFlags[name] = true
		}
		if flag.Category != "invalid" || !parts["ktav"] {
			return fmt.Errorf("raw_bytes flag requires an invalid .ktav fixture: %s", key)
		}
		raw, err := os.ReadFile(filepath.Join(root, flag.Category, filepath.FromSlash(flag.Fixture)+".ktav"))
		if err != nil {
			return fmt.Errorf("read raw_bytes fixture %s: %w", key, err)
		}
		if utf8.Valid(raw) {
			return fmt.Errorf("raw_bytes fixture %s is valid UTF-8", key)
		}
	}
	for category, categoryFixtures := range fixtures {
		for stem, parts := range categoryFixtures {
			if !parts["ktav"] || flagged[category+"/"+stem] {
				continue
			}
			raw, err := os.ReadFile(filepath.Join(root, category, filepath.FromSlash(stem)+".ktav"))
			if err != nil {
				return fmt.Errorf("read fixture %s/%s: %w", category, stem, err)
			}
			if !utf8.Valid(raw) {
				return fmt.Errorf("unflagged fixture %s/%s is not valid UTF-8", category, stem)
			}
		}
	}
	return nil
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

func readFixtureSource(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func assertInvalidOracleError(err error, raw []byte) error {
	var oracle struct {
		ExpectedError     string     `json:"expected_error"`
		ExpectedReason    *string    `json:"expected_reason"`
		Reason            *string    `json:"reason"`
		ExpectedBody      *string    `json:"expected_body"`
		Body              *string    `json:"body"`
		ExpectedCanonical *string    `json:"expected_canonical"`
		Canonical         *string    `json:"canonical"`
		ExpectedSection   *string    `json:"expected_spec_section"`
		SpecSection       *string    `json:"spec_section"`
		ExpectedLine      *int       `json:"expected_line"`
		Line              *int       `json:"line"`
		ExpectedLineText  *string    `json:"expected_line_text"`
		LineText          *string    `json:"line_text"`
		ExpectedPath      *[]string  `json:"expected_path"`
		Path              *[]string  `json:"path"`
		ExpectedSpan      *ktav.Span `json:"expected_span"`
		Span              *ktav.Span `json:"span"`
	}
	if err := json.Unmarshal(raw, &oracle); err != nil {
		return fmt.Errorf("decode invalid-fixture oracle: %w", err)
	}
	if oracle.ExpectedError == "" {
		return fmt.Errorf("invalid-fixture oracle has no expected_error class")
	}
	var got *ktav.Error
	if !errors.As(err, &got) {
		return fmt.Errorf("parse error is not *ktav.Error: %T (%v)", err, err)
	}
	if got.Class != oracle.ExpectedError {
		return fmt.Errorf("error class = %q, want %q (reason %q)", got.Class, oracle.ExpectedError, got.Reason)
	}
	if want := firstString(oracle.ExpectedReason, oracle.Reason); want != nil && got.Reason != *want {
		return fmt.Errorf("error reason = %q, want %q", got.Reason, *want)
	}
	if want := firstString(oracle.ExpectedBody, oracle.Body); want != nil && got.Body != *want {
		return fmt.Errorf("error body = %q, want %q", got.Body, *want)
	}
	if want := firstString(oracle.ExpectedCanonical, oracle.Canonical); want != nil && got.Canonical != *want {
		return fmt.Errorf("error canonical = %q, want %q", got.Canonical, *want)
	}
	if want := firstString(oracle.ExpectedSection, oracle.SpecSection); want != nil && got.SpecSection != *want {
		return fmt.Errorf("error spec_section = %q, want %q", got.SpecSection, *want)
	}
	if want := firstInt(oracle.ExpectedLine, oracle.Line); want != nil && got.Line != *want {
		return fmt.Errorf("error line = %d, want %d", got.Line, *want)
	}
	if want := firstString(oracle.ExpectedLineText, oracle.LineText); want != nil && got.LineText != *want {
		return fmt.Errorf("error line_text = %q, want %q", got.LineText, *want)
	}
	if want := firstPath(oracle.ExpectedPath, oracle.Path); want != nil && !reflect.DeepEqual(got.Path, *want) {
		return fmt.Errorf("error path = %#v, want %#v", got.Path, *want)
	}
	if want := firstSpan(oracle.ExpectedSpan, oracle.Span); want != nil && !reflect.DeepEqual(got.Span, want) {
		return fmt.Errorf("error span = %#v, want %#v", got.Span, want)
	}
	return nil
}

func assertUnrepresentableError(err error, reason, label string) error {
	var got *ktav.Error
	if !errors.As(err, &got) {
		return fmt.Errorf("%s: not *ktav.Error: %T (%v)", label, err, err)
	}
	section, ok := unrepresentableSpecSection(reason)
	if !ok {
		return fmt.Errorf("%s: unknown unrepresentable reason %q", label, reason)
	}
	if got.Class != "UnrepresentableAt" || got.Reason != reason || got.SpecSection != section {
		return fmt.Errorf("%s: structured error = {class:%q reason:%q section:%q}, want {UnrepresentableAt %q %s}",
			label, got.Class, got.Reason, got.SpecSection, reason, section)
	}
	return nil
}

func unrepresentableSpecSection(reason string) (string, bool) {
	switch reason {
	case "CRByte", "BothFormsRequired", "LeadingWhitespaceCollision", "TrailingWhitespaceCollision":
		return "§5.9.7", true
	case "EmptyKeyName", "NonFiniteFloat", "ScalarRoot":
		return "§5.9.0", true
	default:
		return "", false
	}
}

func firstString(a, b *string) *string {
	if a != nil {
		return a
	}
	return b
}

func firstInt(a, b *int) *int {
	if a != nil {
		return a
	}
	return b
}

func firstPath(a, b *[]string) *[]string {
	if a != nil {
		return a
	}
	return b
}

func firstSpan(a, b *ktav.Span) *ktav.Span {
	if a != nil {
		return a
	}
	return b
}

func TestCorpusManifestGuardRejectsMalformedCopies(t *testing.T) {
	tests := []struct {
		name string
		edit func(*corpusManifest, string)
	}{
		{"truncated category", func(_ *corpusManifest, root string) {
			for _, name := range []string{"one.ktav", "one.json", "one.canonical.ktav"} {
				_ = os.Remove(filepath.Join(root, "valid", name))
			}
		}},
		{"unknown category", func(m *corpusManifest, _ string) { m.Categories["extra"] = categoryEntry{Count: 1} }},
		{"missing category", func(m *corpusManifest, _ string) { delete(m.Categories, "strict-lossy") }},
		{"unknown category directory", func(_ *corpusManifest, root string) { _ = os.Mkdir(filepath.Join(root, "extra"), 0700) }},
		{"missing category directory", func(_ *corpusManifest, root string) { _ = os.RemoveAll(filepath.Join(root, "strict-lossy")) }},
		{"missing companion", func(_ *corpusManifest, root string) {
			_ = os.Remove(filepath.Join(root, "valid", "one.canonical.ktav"))
		}},
		{"unknown flag", func(m *corpusManifest, _ string) { m.FixtureFlags[0].Flags = []string{"mystery"} }},
		{"duplicate flag name", func(m *corpusManifest, _ string) { m.FixtureFlags[0].Flags = []string{"raw_bytes", "raw_bytes"} }},
		{"dangling flag fixture", func(m *corpusManifest, _ string) { m.FixtureFlags[0].Fixture = "missing" }},
		{"duplicate flag entry", func(m *corpusManifest, _ string) { m.FixtureFlags = append(m.FixtureFlags, m.FixtureFlags[0]) }},
		{"unexpected corpus root file", func(_ *corpusManifest, root string) { _ = os.WriteFile(filepath.Join(root, "extra.txt"), nil, 0600) }},
		{"raw_bytes must describe invalid UTF-8", func(_ *corpusManifest, root string) {
			_ = os.WriteFile(filepath.Join(root, "invalid", "one.ktav"), []byte("valid"), 0600)
		}},
		{"ordinary fixtures must be UTF-8", func(m *corpusManifest, root string) {
			m.FixtureFlags = nil
			_ = os.WriteFile(filepath.Join(root, "valid", "one.ktav"), []byte{0x80}, 0600)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, manifest := tinyCorpus(t)
			tt.edit(manifest, root)
			if err := validateCorpus(root, manifest, tinyCounts()); err == nil {
				t.Fatal("validateCorpus succeeded; want rejection")
			}
		})
	}
}

func TestCorpusManifestGuardAcceptsTinyCorpus(t *testing.T) {
	root, manifest := tinyCorpus(t)
	if err := validateCorpus(root, manifest, tinyCounts()); err != nil {
		t.Fatalf("validateCorpus rejected well-formed tiny corpus: %v", err)
	}
}

func TestCorpusManifestGuardRejectsUnsupportedSchemaVersions(t *testing.T) {
	for _, version := range []int{0, 2} {
		t.Run(fmt.Sprint(version), func(t *testing.T) {
			root, manifest := tinyCorpus(t)
			manifest.SchemaVersion = version
			if err := validateCorpus(root, manifest, tinyCounts()); err == nil {
				t.Fatal("validateCorpus succeeded; want schema rejection")
			}
		})
	}
}

func TestLoadCorpusManifestRejectsMalformedFiles(t *testing.T) {
	tests := []struct {
		name         string
		wantNotExist bool
		edit         func(*testing.T, string, *corpusManifest)
	}{
		{name: "schema 0", edit: func(t *testing.T, root string, m *corpusManifest) { m.SchemaVersion = 0; writeTinyManifest(t, root, m) }},
		{name: "schema 2", edit: func(t *testing.T, root string, m *corpusManifest) { m.SchemaVersion = 2; writeTinyManifest(t, root, m) }},
		{name: "trailing data", edit: func(t *testing.T, root string, m *corpusManifest) {
			writeTinyManifest(t, root, m)
			path := filepath.Join(root, "manifest.json")
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, append(raw, []byte(" {}")...), 0600); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "unknown field", edit: func(t *testing.T, root string, m *corpusManifest) {
			writeTinyManifest(t, root, m)
			path := filepath.Join(root, "manifest.json")
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			body := strings.TrimSuffix(string(raw), "}") + `,"unexpected":true}`
			if err := os.WriteFile(path, []byte(body), 0600); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "missing manifest", wantNotExist: true, edit: func(t *testing.T, root string, m *corpusManifest) {
			writeTinyManifest(t, root, m)
			if err := os.Remove(filepath.Join(root, "manifest.json")); err != nil {
				t.Fatal(err)
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, manifest := tinyCorpus(t)
			tt.edit(t, root, manifest)
			_, err := loadCorpusManifest(root)
			if err == nil {
				t.Fatal("loadCorpusManifest succeeded; want rejection")
			}
			if tt.wantNotExist && !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("missing manifest error = %v, want os.ErrNotExist", err)
			}
		})
	}
}

func TestLoadCorpusManifestAcceptsValidFile(t *testing.T) {
	root, manifest := tinyCorpus(t)
	writeTinyManifest(t, root, manifest)
	got, err := loadCorpusManifest(root)
	if err != nil {
		t.Fatalf("loadCorpusManifest rejected valid manifest: %v", err)
	}
	if got.SchemaVersion != 1 || !reflect.DeepEqual(got.Categories, manifest.Categories) {
		t.Fatalf("loaded manifest = %#v, want schema 1 and tiny corpus categories", got)
	}
}

func writeTinyManifest(t *testing.T, root string, manifest *corpusManifest) {
	t.Helper()
	raw, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "manifest.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestReadFixtureSourcePreservesRawBytes(t *testing.T) {
	want := []byte{'a', 0x80, 'b'}
	path := filepath.Join(t.TempDir(), "raw.ktav")
	if err := os.WriteFile(path, want, 0600); err != nil {
		t.Fatal(err)
	}
	got, err := readFixtureSource(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("raw source changed: got %v, want %v", got, want)
	}
}

func tinyCounts() map[string]int {
	return map[string]int{"valid": 1, "invalid": 1, "unrepresentable": 1, "parseable-unrepresentable": 1, "strict-lossy": 1}
}

func tinyCorpus(t *testing.T) (string, *corpusManifest) {
	t.Helper()
	root := t.TempDir()
	manifest := &corpusManifest{SchemaVersion: 1, Categories: make(map[string]categoryEntry), FixtureFlags: []fixtureFlag{{Category: "invalid", Fixture: "one", Flags: []string{"raw_bytes"}}}}
	files := map[string][]string{
		"valid":                     {"one.ktav", "one.json", "one.canonical.ktav"},
		"invalid":                   {"one.ktav", "one.json"},
		"unrepresentable":           {"one.json"},
		"parseable-unrepresentable": {"one.ktav", "one.json"},
		"strict-lossy":              {"one.ktav", "one.json"},
	}
	for category, names := range files {
		manifest.Categories[category] = categoryEntry{Count: 1}
		for _, name := range names {
			data := []byte("fixture")
			if category == "invalid" && name == "one.ktav" {
				data = []byte{0x80}
			}
			path := filepath.Join(root, category, name)
			if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, data, 0600); err != nil {
				t.Fatal(err)
			}
		}
	}
	return root, manifest
}
