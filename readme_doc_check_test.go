// The README's examples, executed.
//
// rust puts its README snippets under test (#265) and this does the same
// for Go. Both examples here were WRONG when first written — the error's
// spec_section was guessed as §5.5 when the envelope reports §6.2, and
// the formatting example claimed a respelling the formatter does not
// perform. Prose that is never run drifts from the code silently; this
// is what stops it.
package ktav_test

import (
	"errors"
	"strings"
	"testing"

	ktav "github.com/ktav-lang/golang"
)

func TestReadmeStructuredErrorExample(t *testing.T) {
	_, err := ktav.Loads("a: 1\na: 2\n")
	var kerr *ktav.Error
	if !errors.As(err, &kerr) {
		t.Fatalf("expected *ktav.Error, got %T (%v)", err, err)
	}
	for _, c := range []struct{ name, got, want string }{
		{"Class", kerr.Class, "DuplicateKey"},
		{"SpecSection", kerr.SpecSection, "§6.2"},
	} {
		if c.got != c.want {
			t.Errorf("%s = %q, README says %q", c.name, c.got, c.want)
		}
	}
	if kerr.Line != 2 {
		t.Errorf("Line = %d, README says 2", kerr.Line)
	}
	if kerr.Msg == "" {
		t.Error("Msg is empty; the README says it is never empty")
	}
}

// Task #303: Msg must be the core's own rendering, taken verbatim from
// the envelope's `message` field — not a locally reconstructed
// sentence (the README's own claim, which the code did not yet honour
// until this task). Distinguishes the two by their actual, different
// wording rather than just asserting non-empty: reconstructMessage's
// format was "ktav: <Class>[ <Reason>]...: <Body>", which always starts
// with the literal prefix "ktav: ".
func TestReadmeMsgIsTheCoresOwnRenderingNotAReconstructedSentence(t *testing.T) {
	_, err := ktav.LoadsStrict("version: 1.10\n")
	var kerr *ktav.Error
	if !errors.As(err, &kerr) {
		t.Fatalf("expected *ktav.Error, got %T (%v)", err, err)
	}
	const want = "would be inferred as a number and silently canonicalised"
	if !strings.Contains(kerr.Msg, want) {
		t.Errorf("Msg = %q, want it to contain the core's own wording %q", kerr.Msg, want)
	}
	if strings.HasPrefix(kerr.Msg, "ktav: ") {
		t.Errorf("Msg = %q, must not be the old reconstructed format", kerr.Msg)
	}
}

func TestReadmeFormatterExamples(t *testing.T) {
	cases := []struct{ in, want string }{
		{"## why\na:   {x: 1}\n", "## why\na: {\n    x: 1\n}\n"},
		{"## keep me\n\n\nport: 8080\n", "## keep me\n\nport: 8080\n"},
	}
	for _, c := range cases {
		got, err := ktav.FormatSource(c.in)
		if err != nil {
			t.Fatalf("FormatSource(%q): %v", c.in, err)
		}
		if got != c.want {
			t.Errorf("FormatSource(%q) = %q, README says %q", c.in, got, c.want)
		}
		again, err := ktav.FormatSource(got)
		if err != nil || again != got {
			t.Errorf("formatting is not a fixed point for %q: %q (%v)", c.in, again, err)
		}
	}
}
