package ktav_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	ktav "github.com/ktav-lang/golang"
)

var mutationBenchmarkSink string

type mutationCase struct {
	name, operation, source, path, expected string
	segments                                []string
	semantic                                map[string]any
}

func mutationCases(rows int) []mutationCase {
	var source strings.Builder
	original := make(map[string]any, rows)
	for i := 0; i < rows; i++ {
		key := fmt.Sprintf("key%d", i)
		fmt.Fprintf(&source, "%s: old\n", key)
		original[key] = "old"
	}
	text := source.String()
	target := fmt.Sprintf("key%d", rows/2)
	line := target + ": old\n"
	copyMap := func() map[string]any {
		m := make(map[string]any, len(original)+1)
		for k, v := range original {
			m[k] = v
		}
		return m
	}
	inserted := copyMap()
	inserted["added"] = int64(1)
	removed := copyMap()
	delete(removed, target)
	renamed := copyMap()
	delete(renamed, target)
	renamed["renamed"] = "old"
	return []mutationCase{
		{fmt.Sprintf("insert_%d", rows), "insert", text, "added", text + "added: 1\n", []string{"added"}, inserted},
		{fmt.Sprintf("remove_%d", rows), "remove", text, target, strings.Replace(text, line, "", 1), []string{target}, removed},
		{fmt.Sprintf("rename_%d", rows), "rename", text, target, strings.Replace(text, line, "renamed: old\n", 1), []string{target}, renamed},
	}
}

func BenchmarkSourceMutations(b *testing.B) {
	cases := append(mutationCases(4), mutationCases(4096)...)
	cases = append(cases, mutationCases(32768)...)
	for _, backend := range []struct {
		name string
		kind ktav.Backend
	}{{"pure", ktav.BackendPure}, {"native", ktav.BackendNative}} {
		ops, err := ktav.New(backend.kind)
		if err != nil {
			b.Fatalf("%s backend: %v", backend.name, err)
		}
		for _, tc := range cases {
			for _, segments := range []bool{false, true} {
				pathMode := "text"
				if segments {
					pathMode = "segments"
				}
				b.Run(backend.name+"/"+tc.name+"/"+pathMode, func(b *testing.B) {
					run := func() (string, error) {
						switch tc.operation {
						case "insert":
							if segments {
								return ops.InsertValueSegments(tc.source, tc.segments, int64(1))
							}
							return ops.InsertValue(tc.source, tc.path, int64(1))
						case "remove":
							if segments {
								return ops.RemoveValueSegments(tc.source, tc.segments)
							}
							return ops.RemoveValue(tc.source, tc.path)
						default:
							if segments {
								return ops.RenameKeySegments(tc.source, tc.segments, "renamed")
							}
							return ops.RenameKey(tc.source, tc.path, "renamed")
						}
					}
					out, err := run()
					if err != nil || out != tc.expected {
						b.Fatalf("%s: unexpected output/error: %v", tc.name, err)
					}
					actual, err := ops.Loads(out)
					if err != nil || !reflect.DeepEqual(actual, tc.semantic) {
						b.Fatalf("%s: semantic mismatch: %v", tc.name, err)
					}
					b.ReportAllocs()
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						out, err = run()
						if err != nil {
							b.Fatal(err)
						}
						mutationBenchmarkSink = out
					}
				})
			}
		}
	}
}
