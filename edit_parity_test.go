package ktav_test

// Native/pure parity for the source-edit entry points: on every case the
// two backends must agree byte-for-byte on success and on the error
// class otherwise. A native-only shortcut (or a hidden pure fallback
// behind the native path) fails here.

import (
	"testing"

	ktav "github.com/ktav-lang/golang"
	"github.com/ktav-lang/golang/pure"
)

func TestEditValueNativePureParity(t *testing.T) {
	requireCabi(t)
	native, err := ktav.New(ktav.BackendNative)
	if err != nil {
		t.Fatalf("native Operations: %v", err)
	}
	pureOps := pure.New()

	type editCase struct {
		name        string
		source      string
		path        []string
		textualPath string
		replacement any
		wantOutput  string // empty: expect an EditUnsupported-class error
	}
	cases := []editCase{
		{"plain", "port: 8080\nhost: db\n", []string{"port"}, "port", "db.internal",
			"port: db.internal\nhost: db\n"},
		{"raw marker on", "port: 8080\n", []string{"port"}, "port", "42", "port:: 42\n"},
		{"raw marker off", "port:: 8080\n", []string{"port"}, "port", int64(9090), "port: 9090\n"},
		{"nested block", "srv: {\n  host: a\n  port: 1\n}\n",
			[]string{"srv", "host"}, "srv.host", "b.example",
			"srv: {\n  host: b.example\n  port: 1\n}\n"},
		{"inline leaf", "a: {x: 1, y: 2}\n", []string{"a", "y"}, "a.y", int64(3),
			"a: {x: 1, y: 3}\n"},
		{"crlf verbatim", "s: (\r\n hi\r\n)\r\n", []string{"s"}, "s", "a\nb",
			"s: ((\r\na\r\nb\r\n))\r\n"},
		{"comment preserved", "## top\n\na: 1\n\n## end\n",
			[]string{"a"}, "a", int64(2), "## top\n\na: 2\n\n## end\n"},
		{"layout unfold", "## top\na: [{x: old, untouched: [1, 2]}, {keep: yes}]\nz: keep\n",
			[]string{"a", "0", "x"}, "a.0.x", "one\ntwo",
			"## top\na: [\n    {\n        x: ((\none\ntwo\n        ))\n        untouched: [1, 2]\n    }\n    {keep: yes}\n]\nz: keep\n"},
	}
	for i := range cases {
		c := cases[i]
		t.Run("ok/"+c.name, func(t *testing.T) {
			gotNative, nerr := native.EditValueSegments(c.source, c.path, c.replacement)
			gotPure, perr := pureOps.EditValueSegments(c.source, c.path, c.replacement)
			if nerr != nil || perr != nil {
				t.Fatalf("unexpected errors: native=%v pure=%v", nerr, perr)
			}
			if gotNative != c.wantOutput || gotPure != c.wantOutput {
				t.Fatalf("output divergence\nwant:   %q\nnative: %q\npure:   %q", c.wantOutput, gotNative, gotPure)
			}
			// The textual-path form must produce the identical bytes on
			// the active backend too.
			gotTextual, terr := ktav.EditValue(c.source, c.textualPath, c.replacement)
			if terr != nil || gotTextual != c.wantOutput {
				t.Fatalf("textual form = %q, %v; want %q", gotTextual, terr, c.wantOutput)
			}
		})
	}

	errorCases := []struct {
		name        string
		source      string
		textualPath string
		segments    []string // nil: skip the segments leg (path-only error)
		replacement any
		wantClass   string
	}{
		{"not found", "a: {\n  b: 1\n}\n", "missing", []string{"missing"}, "v", "EditNotFound"},
		{"bad escape", "a: 1\n", `a\nb`, nil, "v", "EditInvalidPath"},
		{"empty path", "a: 1\n", "", nil, "v", "EditInvalidPath"},
		{"trailing escape", "a: 1\n", `a\`, nil, "v", "EditInvalidPath"},
		{"empty segment", "a: 1\n", "a..b", nil, "v", "EditInvalidPath"},
		{"noncontiguous", "a: {}\na.y: 2\n", "a", []string{"a"}, int64(9), "EditUnsupported"},
		{"comment loss", "a: {\n  ## inner\n  x: 1\n}\n", "a", []string{"a"},
			map[string]any{"y": int64(2)}, "EditUnsupported"},
		{"CR replacement", "s: (\n hi\n)\n", "s", []string{"s"}, "a\r\nb", "EditUnsupported"},
		{"terminator line", "s: (\n hi\n)\n", "s", []string{"s"}, "x\n))\ny", "EditUnsupported"},
		{"malformed source", "a: {\n  x: 1\n", "a.x", []string{"a", "x"}, "v", "UnclosedCompound"},
	}
	for _, c := range errorCases {
		t.Run("err/"+c.name, func(t *testing.T) {
			var nerr, perr error
			if c.segments != nil {
				_, nerr = native.EditValueSegments(c.source, c.segments, c.replacement)
				_, perr = pureOps.EditValueSegments(c.source, c.segments, c.replacement)
				if nerr == nil || perr == nil {
					t.Fatalf("expected errors, got native=%v pure=%v", nerr, perr)
				}
				var seN, seP *ktav.Error
				if !asKtavError(nerr, &seN) || !asKtavError(perr, &seP) {
					t.Fatalf("segment errors are not *ktav.Error: %T / %T", nerr, perr)
				}
				if seN.Class != c.wantClass || seP.Class != c.wantClass {
					t.Fatalf("segment class divergence: native=%s pure=%s, want %s", seN.Class, seP.Class, c.wantClass)
				}
			}
			_, nerrT := native.EditValue(c.source, c.textualPath, c.replacement)
			_, perrT := pureOps.EditValue(c.source, c.textualPath, c.replacement)
			if nerrT == nil || perrT == nil {
				t.Fatalf("expected errors, got native=%v pure=%v", nerrT, perrT)
			}
			var ne, pe *ktav.Error
			if !asKtavError(nerrT, &ne) || !asKtavError(perrT, &pe) {
				t.Fatalf("errors are not *ktav.Error: %T / %T", nerrT, perrT)
			}
			if ne.Class != c.wantClass || pe.Class != c.wantClass {
				t.Fatalf("class divergence: native=%s pure=%s, want %s", ne.Class, pe.Class, c.wantClass)
			}
			// InvalidPath must expose the same UTF-8 byte offset on both
			// backends.
			if ne.Class == "EditInvalidPath" && ne.Body != pe.Body {
				t.Fatalf("InvalidPath offset divergence: native=%q pure=%q", ne.Body, pe.Body)
			}
		})
	}

	// Transport escape-parity parity: a Go string with a literal
	// backslash before "uD800" marshals to `\\uD800` — literal text,
	// NOT a surrogate escape — and BOTH backends must accept it and
	// splice it verbatim (regression for the surrogate scanner firing
	// on escaped backslashes).
	t.Run("ok/literal-backslash-uD800", func(t *testing.T) {
		repl := `\uD800` // Go string: backslash + uD800 (no surrogate)
		gotNative, nerr := native.EditValue("a: 1\n", "a", repl)
		gotPure, perr := pureOps.EditValue("a: 1\n", "a", repl)
		if nerr != nil || perr != nil {
			t.Fatalf("unexpected errors: native=%v pure=%v", nerr, perr)
		}
		want := "a: \\uD800\n"
		if gotNative != want || gotPure != want {
			t.Fatalf("output divergence\nwant:   %q\nnative: %q\npure:   %q", want, gotNative, gotPure)
		}
	})

	// A nil []string IS the empty path by contract: BOTH backends must
	// return EditNotFound on a valid source (not a transport Message —
	// the nil slice is normalised to `[]`, never JSON null) and the
	// real source diagnostic first when the source is malformed. This
	// subtest explicitly CALLS the method with nil — the shared error
	// table treats nil segments as "skip the segments leg".
	t.Run("err/nil-segments-empty-path", func(t *testing.T) {
		_, nerr := native.EditValueSegments("a: 1\n", nil, "v")
		_, perr := pureOps.EditValueSegments("a: 1\n", nil, "v")
		if nerr == nil || perr == nil {
			t.Fatalf("expected errors, got native=%v pure=%v", nerr, perr)
		}
		var ne, pe *ktav.Error
		if !asKtavError(nerr, &ne) || !asKtavError(perr, &pe) {
			t.Fatalf("errors are not *ktav.Error: %T / %T", nerr, perr)
		}
		if ne.Class != "EditNotFound" || pe.Class != "EditNotFound" {
			t.Fatalf("class divergence: native=%s pure=%s, want EditNotFound", ne.Class, pe.Class)
		}
		// Broken source: the source diagnostic wins over the empty path.
		_, nerr = native.EditValueSegments("a: {\n", nil, "v")
		_, perr = pureOps.EditValueSegments("a: {\n", nil, "v")
		if nerr == nil || perr == nil {
			t.Fatalf("expected source errors, got native=%v pure=%v", nerr, perr)
		}
		if !asKtavError(nerr, &ne) || !asKtavError(perr, &pe) {
			t.Fatalf("source errors are not *ktav.Error: %T / %T", nerr, perr)
		}
		if ne.Class != "UnclosedCompound" || pe.Class != "UnclosedCompound" {
			t.Fatalf("source-first divergence: native=%s pure=%s, want UnclosedCompound", ne.Class, pe.Class)
		}
	})

	// A CORRECT surrogate-pair replacement (U+1F600) must be accepted on
	// both backends and re-parse to the same String.
	t.Run("ok/surrogate-pair-replacement", func(t *testing.T) {
		repl := "\U0001F600"
		gotNative, nerr := native.EditValueSegments("a: 1\n", []string{"a"}, repl)
		gotPure, perr := pureOps.EditValueSegments("a: 1\n", []string{"a"}, repl)
		if nerr != nil || perr != nil {
			t.Fatalf("unexpected errors: native=%v pure=%v", nerr, perr)
		}
		want := "a: \U0001F600\n"
		if gotNative != want || gotPure != want {
			t.Fatalf("output divergence\nwant:   %q\nnative: %q\npure:   %q", want, gotNative, gotPure)
		}
	})

	// A Go string cannot hold an unpaired surrogate: string(rune(0xD800))
	// yields the VALID U+FFFD replacement character before the API is
	// called, so the public API must ACCEPT it on both backends and
	// produce identical bytes. (Raw invalid-surrogate wire JSON remains
	// a Message — that path is covered by the byte-seam tests.)
	t.Run("ok/replacement-character", func(t *testing.T) {
		repl := string(rune(0xD800)) + "x" // U+FFFD + "x"
		gotNative, nerr := native.EditValue("a: 1\n", "a", repl)
		gotPure, perr := pureOps.EditValue("a: 1\n", "a", repl)
		if nerr != nil || perr != nil {
			t.Fatalf("unexpected errors: native=%v pure=%v", nerr, perr)
		}
		want := "a: \uFFFDx\n"
		if gotNative != want || gotPure != want {
			t.Fatalf("output divergence\nwant:   %q\nnative: %q\npure:   %q", want, gotNative, gotPure)
		}
	})
}
