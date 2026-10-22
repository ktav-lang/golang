package native

// Unit tests for the extracted artifact-version comparison
// (checkArtifactVersion): the match path, the exact mismatch message,
// and the empty/null-string rejection. No library is dlopened — the
// function is pure, which is the point of the extraction.

import (
	"strings"
	"testing"
)

func TestCheckArtifactVersionMatches(t *testing.T) {
	if err := checkArtifactVersion(LibVersion); err != nil {
		t.Fatalf("checkArtifactVersion(%q) = %v, want nil", LibVersion, err)
	}
}

func TestCheckArtifactVersionMismatch(t *testing.T) {
	err := checkArtifactVersion("0.8.0")
	if err == nil {
		t.Fatal("expected mismatch error, got nil")
	}
	want := "ktav_cabi version 0.8.0 does not match expected " + LibVersion
	if err.Error() != want {
		t.Fatalf("error = %q, want %q", err.Error(), want)
	}
}

func TestCheckArtifactVersionEmptyAndNull(t *testing.T) {
	// The loader maps a null C string to an early error; an empty
	// reported string must still be rejected here, never accepted.
	for _, got := range []string{"", " ", "0.8.1\n"} {
		if err := checkArtifactVersion(got); err == nil {
			t.Fatalf("checkArtifactVersion(%q) = nil, want mismatch", got)
		} else if !strings.Contains(err.Error(), "does not match expected") {
			t.Fatalf("checkArtifactVersion(%q) = %v, want mismatch message", got, err)
		}
	}
}
