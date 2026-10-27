package ktav_test

import (
	"errors"
	ktav "github.com/ktav-lang/golang"
	"testing"
)

func TestSegmentsSourcePriorityAndNil(t *testing.T) {
	bad := string([]byte{0xff})
	calls := []func(string, []string) error{
		func(src string, s []string) error { _, e := ktav.InsertValueSegments(src, s, "v"); return e },
		func(src string, s []string) error { _, e := ktav.RemoveValueSegments(src, s); return e },
		func(src string, s []string) error { _, e := ktav.RenameKeySegments(src, s, "x"); return e },
	}
	for i, call := range calls {
		e := call("a: {\n", []string{bad})
		var ke *ktav.Error
		if !errors.As(e, &ke) || ke.Class != "UnclosedCompound" {
			t.Errorf("%d parse priority: %#v", i, e)
		}
		e = call("a: {}\n", []string{bad})
		ke = nil
		if !errors.As(e, &ke) || ke.Class != "Message" || ke.Line != 0 || ke.Span != nil {
			t.Errorf("%d UTF-8 Message: %#v", i, e)
		}
		ke = nil
		e = call("a: {}\n", nil)
		if !errors.As(e, &ke) || ke.Class != "EditNotFound" {
			t.Errorf("%d nil segments: %#v", i, e)
		}
	}
}
