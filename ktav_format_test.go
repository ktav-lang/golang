package ktav_test

import (
	"errors"
	"strconv"
	"strings"
	"testing"

	ktav "github.com/ktav-lang/golang"
)

func TestFormatSourceKeepsCommentsAndIsFixedPoint(t *testing.T) {
	requireCabi(t)
	src := `## top-level comment

outer:
    ## nested comment
    inner: 1
    list: [
        10
        20
    ]


`
	once, err := ktav.FormatSource(src)
	if err != nil {
		t.Fatalf("FormatSource: %v", err)
	}
	for _, comment := range []string{"## top-level comment", "## nested comment"} {
		if !strings.Contains(once, comment) {
			t.Errorf("output lost comment %q:\n%s", comment, once)
		}
	}
	if strings.Contains(once, "\n\n\n") {
		t.Errorf("blank run not collapsed:\n%s", once)
	}
	twice, err := ktav.FormatSource(once)
	if err != nil {
		t.Fatalf("FormatSource(fixed): %v", err)
	}
	if twice != once {
		t.Errorf("not a fixed point:\nonce:\n%s\ntwice:\n%s", once, twice)
	}
}

func TestFormatSourceMatchesCanonicalOnPlainDoc(t *testing.T) {
	requireCabi(t)
	src := "a: 1\nb:\n    c: [x y z]\nd: true\n"
	want, err := ktav.CanonicalFromSource(src)
	if err != nil {
		t.Fatalf("CanonicalFromSource: %v", err)
	}
	got, err := ktav.FormatSource(src)
	if err != nil {
		t.Fatalf("FormatSource: %v", err)
	}
	if got != want {
		t.Errorf("FormatSource != CanonicalFromSource:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestErrorSurfacesStructuredFields(t *testing.T) {
	requireCabi(t)
	_, err := ktav.Loads("a: [")
	if err == nil {
		t.Fatal("expected error")
	}
	var ke *ktav.Error
	if !errors.As(err, &ke) {
		t.Fatalf("errors.As(*ktav.Error) failed: %v", err)
	}
	// line = Error::line is the envelope's uniform rule; span-only
	// kinds (e.g. UnclosedCompound) carry null line but populated
	// line_text and span. Assert what IS carried.
	if ke.Class == "" {
		t.Error("Class is empty")
	}
	if ke.LineText == "" {
		t.Error("LineText is empty")
	}
	if ke.Span == nil {
		t.Error("Span is nil")
	}
	if ke.Line > 0 && !strings.Contains(ke.Error(), "at line "+strconv.Itoa(ke.Line)) {
		t.Errorf("Error() = %q, want it to mention \"at line %d\"", ke.Error(), ke.Line)
	}
	if strings.HasPrefix(ke.Error(), "{") {
		t.Errorf("Error() looks like raw JSON: %q", ke.Error())
	}
}

func TestErrorMessageEnvelope(t *testing.T) {
	requireCabi(t)
	_, err := ktav.Dumps("bare-string")
	if err == nil {
		t.Fatal("expected error")
	}
	var ke *ktav.Error
	if !errors.As(err, &ke) {
		t.Fatalf("errors.As(*ktav.Error) failed: %v", err)
	}
	if ke.Class != "Message" {
		t.Errorf("Class = %q, want \"Message\"", ke.Class)
	}
	if !strings.Contains(ke.Body, "object or array") {
		t.Errorf("Body = %q, want it to contain \"object or array\"", ke.Body)
	}
	if !strings.Contains(ke.Error(), "object or array") {
		t.Errorf("Error() = %q, want it to contain \"object or array\"", ke.Error())
	}
}

func TestErrorPathSegmentsExact(t *testing.T) {
	requireCabi(t)
	_, err := ktav.Dumps(map[string]any{"": int64(1)})
	if err == nil {
		t.Fatal("expected error")
	}
	var ke *ktav.Error
	if !errors.As(err, &ke) {
		t.Fatalf("errors.As(*ktav.Error) failed: %v", err)
	}
	if ke.Reason != "EmptyKeyName" {
		t.Errorf("Reason = %q, want \"EmptyKeyName\"", ke.Reason)
	}
	if ke.Class != "Unrepresentable" && ke.Class != "UnrepresentableAt" {
		t.Errorf("Class = %q, want \"Unrepresentable\" or \"UnrepresentableAt\"", ke.Class)
	}
	if ke.Path == nil {
		t.Fatal("Path is nil, want non-nil exact segments")
	}
	if len(ke.Path) != 1 || ke.Path[0] != "" {
		t.Errorf("Path = %#v, want []string{\"\"} (exact decoded segments, not joined)", ke.Path)
	}
}
