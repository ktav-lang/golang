package ktav

import (
	"math"
	"testing"
)

func TestWriterRejectionsCarryStructuredFields(t *testing.T) {
	for name, write := range map[string]func() error{
		"Dumps":             func() error { _, err := Dumps(42); return err },
		"DumpsForceStrings": func() error { _, err := DumpsForceStrings("root"); return err },
		"EmitCanonical":     func() error { _, err := EmitCanonical(true); return err },
	} {
		t.Run(name, func(t *testing.T) {
			got := write()
			err, ok := got.(*Error)
			if !ok {
				t.Fatalf("expected *Error, got %T", got)
			}
			if err.Class != "UnrepresentableAt" || err.Reason != "ScalarRoot" || err.SpecSection != "§5.9.0" {
				t.Fatalf("unexpected scalar-root envelope: %#v", err)
			}
			if err.Path == nil || len(err.Path) != 0 || err.Span != nil || err.Line != 0 {
				t.Fatalf("unexpected scalar-root location: %#v", err)
			}
		})
	}

	for _, number := range []struct {
		name  string
		value float64
	}{{"NaN", math.NaN()}, {"PositiveInfinity", math.Inf(1)}, {"NegativeInfinity", math.Inf(-1)}} {
		t.Run(number.name, func(t *testing.T) {
			value := map[string]any{"配置": []any{map[string]any{"nested.key": number.value}}}
			for name, write := range map[string]func() error{
				"Dumps":         func() error { _, err := Dumps(value); return err },
				"EmitCanonical": func() error { _, err := EmitCanonical(value); return err },
			} {
				t.Run(name, func(t *testing.T) {
					got := write()
					err, ok := got.(*Error)
					if !ok {
						t.Fatalf("expected *Error, got %T", got)
					}
					if err.Class != "UnrepresentableAt" || err.Reason != "NonFiniteFloat" || err.SpecSection != "§5.9.0" {
						t.Fatalf("unexpected non-finite envelope: %#v", err)
					}
					if len(err.Path) != 2 || err.Path[0] != "配置" || err.Path[1] != "nested.key" || err.Span != nil || err.Line != 0 {
						t.Fatalf("unexpected non-finite path: %#v", err)
					}
				})
			}
		})
	}

	for _, test := range []struct {
		name  string
		value map[string]any
		want  string
	}{
		{"empty-key-before-float", map[string]any{"a": map[string]any{"": 1}, "z": math.NaN()}, "EmptyKeyName"},
		{"float-before-empty-key", map[string]any{"a": math.NaN(), "z": map[string]any{"": 1}}, "NonFiniteFloat"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := Dumps(test.value)
			ktavErr, ok := err.(*Error)
			if !ok {
				t.Fatalf("expected *Error, got %T", err)
			}
			if ktavErr.Class != "UnrepresentableAt" || ktavErr.Reason != test.want || ktavErr.SpecSection != "§5.9.0" {
				t.Fatalf("unexpected precedence envelope: %#v", ktavErr)
			}
			if test.want == "EmptyKeyName" && (len(ktavErr.Path) != 2 || ktavErr.Path[0] != "a" || ktavErr.Path[1] != "") {
				t.Fatalf("unexpected empty-key path: %#v", ktavErr.Path)
			}
		})
	}
}

func TestDumpsForceStringsPreservesNonFiniteFloatText(t *testing.T) {
	for _, test := range []struct {
		value float64
		want  string
	}{{math.NaN(), "NaN"}, {math.Inf(1), "+Inf"}, {math.Inf(-1), "-Inf"}} {
		out, err := DumpsForceStrings(map[string]any{"value": test.value})
		if err != nil {
			t.Fatalf("DumpsForceStrings(%v): %v", test.value, err)
		}
		got, err := Loads(out)
		if err != nil {
			t.Fatalf("Loads(%q): %v", out, err)
		}
		if actual := got.(map[string]any)["value"]; actual != test.want {
			t.Errorf("round-trip value = %#v, want %q", actual, test.want)
		}
	}
}

func TestInvalidUTF8EntryPointsCarryStructuredFields(t *testing.T) {
	src := "valid: 雪\ninvalid: \xff"
	checks := map[string]func() error{
		"Loads":               func() error { _, err := Loads(src); return err },
		"LoadsStrict":         func() error { _, err := LoadsStrict(src); return err },
		"LoadsInto":           func() error { return LoadsInto(src, new(any)) },
		"FormatSource":        func() error { _, err := FormatSource(src); return err },
		"CanonicalFromSource": func() error { _, err := CanonicalFromSource(src); return err },
	}
	for name, check := range checks {
		t.Run(name, func(t *testing.T) {
			got := check()
			err, ok := got.(*Error)
			if !ok {
				t.Fatalf("expected *Error, got %T", got)
			}
			if err.Class != "InvalidUtf8" || err.Reason != "" || err.SpecSection != "§6.15" {
				t.Fatalf("unexpected invalid-UTF-8 envelope: %#v", err)
			}
			if err.Span == nil || err.Span.Start != len("valid: 雪\ninvalid: ") || err.Span.End != err.Span.Start || err.Line != 0 {
				t.Fatalf("unexpected invalid-UTF-8 span: %#v", err)
			}
		})
	}
}
