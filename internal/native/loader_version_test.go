package native

// Unit tests for the extracted artifact-ABI comparison
// (checkArtifactABI): the match path, the exact mismatch message, and
// the rejection of every wrong value. No library is dlopened — the
// function is pure, which is the point of the extraction. The ABI gate
// (expected 2 since the source-edit symbols joined the exported set) is
// the loader's sole provenance check; ktav_version is diagnostics only.

import (
	"strings"
	"testing"
)

func TestCheckArtifactABIMatches(t *testing.T) {
	if err := checkArtifactABI(ExpectedABIVersion); err != nil {
		t.Fatalf("checkArtifactABI(%d) = %v, want nil", ExpectedABIVersion, err)
	}
}

func TestCheckArtifactABIRejectsEveryWrongValue(t *testing.T) {
	// 1 is the pre-source-edit ABI: an old binary must be refused with
	// a clear version-mismatch error, never substituted with the pure
	// backend.
	for _, got := range []uint64{0, 1, 3, 99} {
		err := checkArtifactABI(got)
		if err == nil {
			t.Fatalf("checkArtifactABI(%d) = nil, want mismatch", got)
		}
		want := "does not match expected 2"
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("checkArtifactABI(%d) = %q, want message containing %q", got, err.Error(), want)
		}
	}
}

func TestExpectedABIVersionIsTwo(t *testing.T) {
	// The bump 1 → 2 accompanied the two source-edit symbols; a silent
	// revert of either would corrupt the load contract.
	if ExpectedABIVersion != 2 {
		t.Fatalf("ExpectedABIVersion = %d, want 2", ExpectedABIVersion)
	}
}
