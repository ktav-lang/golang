package ktav_test

import (
	"errors"
	"reflect"
	"testing"

	ktav "github.com/ktav-lang/golang"
	"github.com/ktav-lang/golang/pure"
)

type mutationCall func() (string, error)

func checkMutationOutput(t *testing.T, fn mutationCall, want string, wantValue any) {
	t.Helper()
	got, err := fn()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != want {
		t.Fatalf("source = %q, want exact source %q", got, want)
	}
	gotValue, err := pure.Loads(got)
	if err != nil {
		t.Fatalf("mutated source does not parse: %v", err)
	}
	if !reflect.DeepEqual(gotValue, wantValue) {
		t.Fatalf("parsed value = %#v, want semantic oracle %#v", gotValue, wantValue)
	}
}

func checkMutationError(t *testing.T, fn mutationCall, class string) {
	t.Helper()
	_, err := fn()
	var e *ktav.Error
	if !errors.As(err, &e) || e.Class != class {
		t.Fatalf("error = %#v (%v), want class %s", e, err, class)
	}
}

func TestMutationNativePureParity(t *testing.T) {
	requireCabi(t)
	native, err := ktav.New(ktav.BackendNative)
	if err != nil {
		t.Fatalf("native Operations: %v", err)
	}
	pureOps := pure.New()
	// Expected source bytes are pinned independently of either implementation;
	// each successful result is also parsed to check semantic validity.
	checks := []struct {
		name      string
		want      string
		wantValue any
		calls     []mutationCall
	}{
		{"insert-text", "a: {x: 1, y: two}\n", map[string]any{"a": map[string]any{"x": int64(1), "y": "two"}}, []mutationCall{
			func() (string, error) { return ktav.InsertValue("a: {x: 1}\n", "a.y", "two") },
			func() (string, error) { return native.InsertValue("a: {x: 1}\n", "a.y", "two") },
			func() (string, error) { return pure.InsertValue("a: {x: 1}\n", "a.y", "two") },
			func() (string, error) { return pureOps.InsertValue("a: {x: 1}\n", "a.y", "two") },
		}},
		{"insert-segments", "a: {\"b.c\": v}\n", map[string]any{"a": map[string]any{"b.c": "v"}}, []mutationCall{
			func() (string, error) { return ktav.InsertValueSegments("a: {}\n", []string{"a", "b.c"}, "v") },
			func() (string, error) { return native.InsertValueSegments("a: {}\n", []string{"a", "b.c"}, "v") },
			func() (string, error) { return pure.InsertValueSegments("a: {}\n", []string{"a", "b.c"}, "v") },
			func() (string, error) { return pureOps.InsertValueSegments("a: {}\n", []string{"a", "b.c"}, "v") },
		}},
		{"remove-text", "b: 2\n", map[string]any{"b": int64(2)}, []mutationCall{
			func() (string, error) { return ktav.RemoveValue("a: 1\nb: 2\n", "a") },
			func() (string, error) { return native.RemoveValue("a: 1\nb: 2\n", "a") },
			func() (string, error) { return pure.RemoveValue("a: 1\nb: 2\n", "a") },
			func() (string, error) { return pureOps.RemoveValue("a: 1\nb: 2\n", "a") },
		}},
		{"remove-array-index", "a: [1, 3]\n", map[string]any{"a": []any{int64(1), int64(3)}}, []mutationCall{
			func() (string, error) { return ktav.RemoveValue("a: [1, 2, 3]\n", "a.1") },
			func() (string, error) { return native.RemoveValueSegments("a: [1, 2, 3]\n", []string{"a", "1"}) },
			func() (string, error) { return pure.RemoveValue("a: [1, 2, 3]\n", "a.1") },
			func() (string, error) { return pureOps.RemoveValueSegments("a: [1, 2, 3]\n", []string{"a", "1"}) },
		}},
		{"remove-segments", "a: {}\n", map[string]any{"a": map[string]any{}}, []mutationCall{
			func() (string, error) { return ktav.RemoveValueSegments("a: {b: 1}\n", []string{"a", "b"}) },
			func() (string, error) { return native.RemoveValueSegments("a: {b: 1}\n", []string{"a", "b"}) },
			func() (string, error) { return pure.RemoveValueSegments("a: {b: 1}\n", []string{"a", "b"}) },
			func() (string, error) { return pureOps.RemoveValueSegments("a: {b: 1}\n", []string{"a", "b"}) },
		}},
		{"rename-text-raw-key", "\"new.key\": 1\n", map[string]any{"new.key": int64(1)}, []mutationCall{
			func() (string, error) { return ktav.RenameKey("a: 1\n", "a", `new.key`) },
			func() (string, error) { return native.RenameKey("a: 1\n", "a", `new.key`) },
			func() (string, error) { return pure.RenameKey("a: 1\n", "a", `new.key`) },
			func() (string, error) { return pureOps.RenameKey("a: 1\n", "a", `new.key`) },
		}},
		{"rename-segments", "a: {new\\\\key: 1}\n", map[string]any{"a": map[string]any{"new\\key": int64(1)}}, []mutationCall{
			func() (string, error) { return ktav.RenameKeySegments("a: {b: 1}\n", []string{"a", "b"}, `new\key`) },
			func() (string, error) { return native.RenameKeySegments("a: {b: 1}\n", []string{"a", "b"}, `new\key`) },
			func() (string, error) { return pure.RenameKeySegments("a: {b: 1}\n", []string{"a", "b"}, `new\key`) },
			func() (string, error) {
				return pureOps.RenameKeySegments("a: {b: 1}\n", []string{"a", "b"}, `new\key`)
			},
		}},
	}
	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			for i, fn := range check.calls {
				t.Run(string(rune('a'+i)), func(t *testing.T) { checkMutationOutput(t, fn, check.want, check.wantValue) })
			}
		})
	}

	// Invalid source bytes must win over invalid segment bytes, independently
	// for each mutation and each public/native/pure entry point.
	badSource := "a: " + string([]byte{0xff}) + "\n"
	badSegment := string([]byte{0xff})
	for _, check := range []struct {
		name  string
		calls []mutationCall
	}{
		{"insert", []mutationCall{
			func() (string, error) { return ktav.InsertValueSegments(badSource, []string{badSegment}, "v") },
			func() (string, error) { return native.InsertValueSegments(badSource, []string{badSegment}, "v") },
			func() (string, error) { return pure.InsertValueSegments(badSource, []string{badSegment}, "v") },
			func() (string, error) { return pureOps.InsertValueSegments(badSource, []string{badSegment}, "v") },
		}},
		{"remove", []mutationCall{
			func() (string, error) { return ktav.RemoveValueSegments(badSource, []string{badSegment}) },
			func() (string, error) { return native.RemoveValueSegments(badSource, []string{badSegment}) },
			func() (string, error) { return pure.RemoveValueSegments(badSource, []string{badSegment}) },
			func() (string, error) { return pureOps.RemoveValueSegments(badSource, []string{badSegment}) },
		}},
		{"rename", []mutationCall{
			func() (string, error) { return ktav.RenameKeySegments(badSource, []string{badSegment}, "renamed") },
			func() (string, error) { return native.RenameKeySegments(badSource, []string{badSegment}, "renamed") },
			func() (string, error) { return pure.RenameKeySegments(badSource, []string{badSegment}, "renamed") },
			func() (string, error) { return pureOps.RenameKeySegments(badSource, []string{badSegment}, "renamed") },
		}},
	} {
		t.Run("source-first-invalid-utf8/"+check.name, func(t *testing.T) {
			for i, fn := range check.calls {
				t.Run(string(rune('a'+i)), func(t *testing.T) {
					_, err := fn()
					var e *ktav.Error
					if !errors.As(err, &e) || e.Class != "InvalidUtf8" || e.Span == nil {
						t.Fatalf("error = %#v (%v), want InvalidUtf8 with source span", e, err)
					}
				})
			}
		})
	}

	// Nil segments are the empty path, not a malformed transport.
	for _, check := range []struct {
		name  string
		calls []mutationCall
	}{
		{"insert", []mutationCall{
			func() (string, error) { return ktav.InsertValueSegments("a: 1\n", nil, "v") },
			func() (string, error) { return native.InsertValueSegments("a: 1\n", nil, "v") },
			func() (string, error) { return pure.InsertValueSegments("a: 1\n", nil, "v") },
			func() (string, error) { return pureOps.InsertValueSegments("a: 1\n", nil, "v") },
		}},
		{"remove", []mutationCall{
			func() (string, error) { return ktav.RemoveValueSegments("a: 1\n", nil) },
			func() (string, error) { return native.RemoveValueSegments("a: 1\n", nil) },
			func() (string, error) { return pure.RemoveValueSegments("a: 1\n", nil) },
			func() (string, error) { return pureOps.RemoveValueSegments("a: 1\n", nil) },
		}},
		{"rename", []mutationCall{
			func() (string, error) { return ktav.RenameKeySegments("a: 1\n", nil, "renamed") },
			func() (string, error) { return native.RenameKeySegments("a: 1\n", nil, "renamed") },
			func() (string, error) { return pure.RenameKeySegments("a: 1\n", nil, "renamed") },
			func() (string, error) { return pureOps.RenameKeySegments("a: 1\n", nil, "renamed") },
		}},
	} {
		t.Run("nil-segments/"+check.name, func(t *testing.T) {
			for i, fn := range check.calls {
				t.Run(string(rune('a'+i)), func(t *testing.T) { checkMutationError(t, fn, "EditNotFound") })
			}
		})
	}

	// A valid source with invalid UTF-8 segments is a source-less Message
	// error; it must not claim source positions.
	for _, check := range []struct {
		name  string
		calls []mutationCall
	}{
		{"insert", []mutationCall{
			func() (string, error) { return ktav.InsertValueSegments("a: 1\n", []string{badSegment}, "v") },
			func() (string, error) { return native.InsertValueSegments("a: 1\n", []string{badSegment}, "v") },
			func() (string, error) { return pure.InsertValueSegments("a: 1\n", []string{badSegment}, "v") },
			func() (string, error) { return pureOps.InsertValueSegments("a: 1\n", []string{badSegment}, "v") },
		}},
		{"remove", []mutationCall{
			func() (string, error) { return ktav.RemoveValueSegments("a: 1\n", []string{badSegment}) },
			func() (string, error) { return native.RemoveValueSegments("a: 1\n", []string{badSegment}) },
			func() (string, error) { return pure.RemoveValueSegments("a: 1\n", []string{badSegment}) },
			func() (string, error) { return pureOps.RemoveValueSegments("a: 1\n", []string{badSegment}) },
		}},
		{"rename", []mutationCall{
			func() (string, error) { return ktav.RenameKeySegments("a: 1\n", []string{badSegment}, "renamed") },
			func() (string, error) { return native.RenameKeySegments("a: 1\n", []string{badSegment}, "renamed") },
			func() (string, error) { return pure.RenameKeySegments("a: 1\n", []string{badSegment}, "renamed") },
			func() (string, error) { return pureOps.RenameKeySegments("a: 1\n", []string{badSegment}, "renamed") },
		}},
	} {
		t.Run("invalid-utf8-segments/"+check.name, func(t *testing.T) {
			for i, fn := range check.calls {
				t.Run(string(rune('a'+i)), func(t *testing.T) {
					_, err := fn()
					var e *ktav.Error
					if !errors.As(err, &e) || e.Class != "Message" || e.Line != 0 || e.Span != nil {
						t.Fatalf("error = %#v (%v), want source-less Message", e, err)
					}
				})
			}
		})
	}

	for _, check := range []struct {
		name  string
		class string
		calls []mutationCall
	}{
		{"missing", "EditNotFound", []mutationCall{
			func() (string, error) { return ktav.RemoveValue("a: 1\n", "missing") },
			func() (string, error) { return native.RemoveValueSegments("a: 1\n", []string{"missing"}) },
			func() (string, error) { return pure.RemoveValue("a: 1\n", "missing") },
			func() (string, error) { return pureOps.RemoveValueSegments("a: 1\n", []string{"missing"}) },
		}},
		{"source-first-insert", "UnclosedCompound", []mutationCall{
			func() (string, error) { return ktav.InsertValue("a: {\n", "a.b", make(chan int)) },
			func() (string, error) {
				return native.InsertValueSegments("a: {\n", []string{"a", "b"}, make(chan int))
			},
			func() (string, error) { return pure.InsertValue("a: {\n", "a.b", make(chan int)) },
			func() (string, error) {
				return pureOps.InsertValueSegments("a: {\n", []string{"a", "b"}, make(chan int))
			},
		}},
		{"existing-insert", "EditAlreadyExists", []mutationCall{
			func() (string, error) { return ktav.InsertValue("a: {x: 1}\n", "a.x", "v") },
			func() (string, error) { return native.InsertValueSegments("a: {x: 1}\n", []string{"a", "x"}, "v") },
			func() (string, error) { return pure.InsertValue("a: {x: 1}\n", "a.x", "v") },
			func() (string, error) { return pureOps.InsertValueSegments("a: {x: 1}\n", []string{"a", "x"}, "v") },
		}},
		{"array-insert", "EditUnsupported", []mutationCall{
			func() (string, error) { return ktav.InsertValue("a: [x]\n", "a.0", "v") },
			func() (string, error) { return native.InsertValueSegments("a: [x]\n", []string{"a", "0"}, "v") },
			func() (string, error) { return pure.InsertValue("a: [x]\n", "a.0", "v") },
			func() (string, error) { return pureOps.InsertValueSegments("a: [x]\n", []string{"a", "0"}, "v") },
		}},
	} {
		t.Run("error/"+check.name, func(t *testing.T) {
			for i, fn := range check.calls {
				t.Run(string(rune('a'+i)), func(t *testing.T) { checkMutationError(t, fn, check.class) })
			}
		})
	}

}
