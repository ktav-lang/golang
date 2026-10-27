package ktav_test

import (
	"errors"
	"testing"

	ktav "github.com/ktav-lang/golang"
	"github.com/ktav-lang/golang/pure"
)

func TestInsertValuePathPriorityOverReplacementConversion(t *testing.T) {
	requireCabi(t)
	native, err := ktav.New(ktav.BackendNative)
	if err != nil {
		t.Fatalf("native Operations: %v", err)
	}
	type textualCall struct {
		name string
		call func(string, string, any) error
	}
	textual := []textualCall{
		{"native package", func(src, path string, v any) error { _, e := ktav.InsertValue(src, path, v); return e }},
		{"native Operations", func(src, path string, v any) error { _, e := native.InsertValue(src, path, v); return e }},
		{"pure package", func(src, path string, v any) error { _, e := pure.InsertValue(src, path, v); return e }},
		{"pure Operations", func(src, path string, v any) error { _, e := pure.New().InsertValue(src, path, v); return e }},
	}
	for _, tc := range textual {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.call("a: {}\n", ".", make(chan int))
			var ke *ktav.Error
			if !errors.As(err, &ke) || ke == nil || ke.Class != "EditInvalidPath" {
				t.Fatalf("error = %#v, want EditInvalidPath", err)
			}
		})
	}

	type segmentsCall struct {
		name string
		call func(string, []string, any) error
	}
	segments := []segmentsCall{
		{"native package", func(src string, p []string, v any) error { _, e := ktav.InsertValueSegments(src, p, v); return e }},
		{"native Operations", func(src string, p []string, v any) error { _, e := native.InsertValueSegments(src, p, v); return e }},
		{"pure package", func(src string, p []string, v any) error { _, e := pure.InsertValueSegments(src, p, v); return e }},
		{"pure Operations", func(src string, p []string, v any) error { _, e := pure.New().InsertValueSegments(src, p, v); return e }},
	}
	for _, tc := range segments {
		t.Run(tc.name, func(t *testing.T) {
			for _, path := range [][]string{nil, {}} {
				err := tc.call("a: {}\n", path, make(chan int))
				var ke *ktav.Error
				if !errors.As(err, &ke) || ke == nil || ke.Class != "EditNotFound" {
					t.Fatalf("path %#v: error = %#v, want EditNotFound", path, err)
				}
			}
		})
	}
}
