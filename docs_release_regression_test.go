package ktav_test

import (
	"os"
	"strings"
	"testing"

	ktav "github.com/ktav-lang/golang"
)

func TestDocsReleaseQuickStartDumpOutput(t *testing.T) {
	requireCabi(t)
	doc := map[string]any{
		"name":  "frontend",
		"port":  int64(8443),
		"tls":   true,
		"ratio": 0.95,
		"upstreams": []any{
			map[string]any{"host": "a.example", "port": int64(1080)},
			map[string]any{"host": "b.example", "port": int64(1080)},
		},
		"notes": nil,
	}
	got, err := ktav.Dumps(doc)
	if err != nil {
		t.Fatal(err)
	}

	for _, lang := range []string{"en", "ru", "zh"} {
		t.Run(lang, func(t *testing.T) {
			section := docsReleaseLanguage(t, "root-docs/README/quick-start-build/body-1.md", lang)
			want := docsReleaseDumpCommentBlock(t, section)
			if got != want {
				t.Errorf("Dumps output differs from %s quick-start example:\n got: %q\nwant: %q", lang, got, want)
			}
		})
	}
}

func TestDocsReleaseKeyEscapingExamplesParse(t *testing.T) {
	requireCabi(t)
	for _, lang := range []string{"en", "ru", "zh"} {
		t.Run(lang, func(t *testing.T) {
			section := docsReleaseLanguage(t, "root-docs/README/key-escaping/body-1.md", lang)
			src := docsReleaseFence(t, section, "text")
			got, err := ktav.Loads(src)
			if err != nil {
				t.Fatalf("Loads of documented key examples: %v", err)
			}
			root, ok := got.(map[string]any)
			if !ok {
				t.Fatalf("top-level value is %T, want map[string]any", got)
			}
			if root["a.b"] != "v" || root["a:b"] != "v" {
				t.Errorf("escaped keys decoded incorrectly: %#v", root)
			}
			x, ok := root["x"].(map[string]any)
			if !ok || x["y.z"] != "v" {
				t.Errorf("dotted path with escaped segment decoded incorrectly: %#v", root["x"])
			}
		})
	}
}

func docsReleaseLanguage(t *testing.T, path, lang string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	marker := ">>>>> lang=" + lang
	lines := strings.Split(string(raw), "\n")
	for i, line := range lines {
		if line != marker {
			continue
		}
		start := i + 1
		for end := start; end < len(lines); end++ {
			if strings.HasPrefix(lines[end], ">>>>> lang=") {
				return strings.Join(lines[start:end], "\n")
			}
		}
		return strings.Join(lines[start:], "\n")
	}
	t.Fatalf("language section %q not found in %s", lang, path)
	return ""
}

func docsReleaseFence(t *testing.T, section, language string) string {
	t.Helper()
	open := "```" + language
	lines := strings.Split(section, "\n")
	for i, line := range lines {
		if line == open {
			for end := i + 1; end < len(lines); end++ {
				if lines[end] == "```" {
					return strings.Join(lines[i+1:end], "\n")
				}
			}
			t.Fatalf("unterminated %s fence", language)
		}
	}
	t.Fatalf("%s fence not found", language)
	return ""
}

func docsReleaseDumpCommentBlock(t *testing.T, section string) string {
	t.Helper()
	code := strings.Split(docsReleaseFence(t, section, "go"), "\n")
	for i, line := range code {
		if strings.TrimSpace(line) != "fmt.Print(out)" {
			continue
		}
		var output []string
		for _, comment := range code[i+1:] {
			comment = strings.TrimSpace(comment)
			if !strings.HasPrefix(comment, "//") {
				break
			}
			comment = strings.TrimPrefix(comment, "//")
			comment = strings.TrimPrefix(comment, " ")
			output = append(output, comment)
		}
		if len(output) == 0 {
			t.Fatal("quick-start code has no output comment block")
		}
		return strings.Join(output, "\n") + "\n"
	}
	t.Fatal("fmt.Print(out) not found in quick-start code")
	return ""
}
