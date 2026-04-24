package ktav

import (
	"bytes"
	"io"
	"math"
	"strconv"
)

func bytesReader(b []byte) io.Reader { return bytes.NewReader(b) }

func isNaN(f float64) bool { return math.IsNaN(f) }
func isInf(f float64) bool { return math.IsInf(f, 0) }

// formatFloat produces a decimal representation with a guaranteed '.' so
// the Rust renderer's grammar check ("floats need a decimal point or
// exponent") is satisfied.
func formatFloat(f float64) string {
	s := strconv.FormatFloat(f, 'g', -1, 64)
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '.' || c == 'e' || c == 'E' {
			return s
		}
	}
	return s + ".0"
}

// orderedMap is a small helper used during decoding; the final result
// is surfaced as a plain map[string]any (insertion order preserved by
// the key slice). Consumers that need order should use LoadsInto into
// a struct, or post-process.
type orderedMap struct {
	keys []string
	data map[string]any
}

func (m orderedMap) asMap() map[string]any {
	if m.data == nil {
		return map[string]any{}
	}
	return m.data
}
