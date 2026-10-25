package main

// Independent JSON oracle decode + structural compare + error-envelope
// checks, ported from conformance_test.go (decodeOracle/liftOracle/
// structEqual/decodeUnrepJSON) and corpus_manifest_test.go
// (assertInvalidOracleError/assertUnrepresentableError), using *pure.Error.

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"reflect"
	"strings"

	"github.com/ktav-lang/golang/pure"
)

// decodeOracle parses the reference JSON with json.Number, then lifts
// numeric literals into the same Go shapes Loads produces (int64,
// *big.Int, float64).
func decodeOracle(raw []byte) (any, error) {
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	if err := dec.Decode(new(any)); err != io.EOF {
		return nil, fmt.Errorf("oracle JSON has trailing data: %v", err)
	}
	return liftOracle(v), nil
}

func liftOracle(v any) any {
	switch t := v.(type) {
	case map[string]any:
		for k, x := range t {
			t[k] = liftOracle(x)
		}
		return t
	case []any:
		for i, x := range t {
			t[i] = liftOracle(x)
		}
		return t
	case json.Number:
		s := string(t)
		if !strings.ContainsAny(s, ".eE") {
			if n, ok := new(big.Int).SetString(s, 10); ok {
				if n.IsInt64() {
					return n.Int64()
				}
				return n
			}
		}
		f, err := t.Float64()
		if err != nil {
			return t
		}
		return f
	default:
		return v
	}
}

// structEqual: *big.Int by numeric value, floats by exact equality
// (NaN never appears in valid oracles; -0.0 == 0.0 in Go, matching the
// in-repo reference), maps key-order-insensitive, slices in order.
func structEqual(a, b any) bool {
	switch av := a.(type) {
	case *big.Int:
		bv, ok := b.(*big.Int)
		return ok && av.Cmp(bv) == 0
	case float64:
		bv, ok := b.(float64)
		return ok && av == bv
	case map[string]any:
		bm, ok := b.(map[string]any)
		if !ok || len(av) != len(bm) {
			return false
		}
		for k, v := range av {
			bv, ok := bm[k]
			if !ok {
				return false
			}
			if !structEqual(v, bv) {
				return false
			}
		}
		return true
	case []any:
		bs, ok := b.([]any)
		if !ok || len(av) != len(bs) {
			return false
		}
		for i := range av {
			if !structEqual(av[i], bs[i]) {
				return false
			}
		}
		return true
	default:
		return reflect.DeepEqual(a, b)
	}
}

// decodeUnrepJSON reads one of the unrepresentable-fixture oracles:
// (value, reason), with the {"$float": ...} marker lifted to a
// non-finite float64.
func decodeUnrepJSON(raw []byte) (any, string, error) {
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.UseNumber()
	var doc struct {
		Value                 any    `json:"value"`
		UnrepresentableReason string `json:"unrepresentable_reason"`
	}
	if err := dec.Decode(&doc); err != nil {
		return nil, "", err
	}
	if err := dec.Decode(new(any)); err != io.EOF {
		return nil, "", fmt.Errorf("oracle JSON has trailing data: %v", err)
	}
	return liftMarker(doc.Value), doc.UnrepresentableReason, nil
}

func liftMarker(v any) any {
	if m, ok := v.(map[string]any); ok && len(m) == 1 {
		if s, ok := m["$float"].(string); ok {
			switch s {
			case "NaN":
				return math.NaN()
			case "Infinity":
				return math.Inf(1)
			case "-Infinity":
				return math.Inf(-1)
			}
		}
	}
	switch t := v.(type) {
	case map[string]any:
		for k, x := range t {
			t[k] = liftMarker(x)
		}
		return t
	case []any:
		for i, x := range t {
			t[i] = liftMarker(x)
		}
		return t
	case json.Number:
		return liftOracle(t)
	default:
		return v
	}
}

// assertInvalidOracleError compares the full *pure.Error envelope
// against the invalid-fixture .json oracle; optional fields are checked
// only when present in the oracle (expected_* takes precedence).
func assertInvalidOracleError(err error, raw []byte) error {
	var oracle struct {
		ExpectedError     string     `json:"expected_error"`
		ExpectedReason    *string    `json:"expected_reason"`
		Reason            *string    `json:"reason"`
		ExpectedBody      *string    `json:"expected_body"`
		Body              *string    `json:"body"`
		ExpectedCanonical *string    `json:"expected_canonical"`
		Canonical         *string    `json:"canonical"`
		ExpectedSection   *string    `json:"expected_spec_section"`
		SpecSection       *string    `json:"spec_section"`
		ExpectedLine      *int       `json:"expected_line"`
		Line              *int       `json:"line"`
		ExpectedLineText  *string    `json:"expected_line_text"`
		LineText          *string    `json:"line_text"`
		ExpectedPath      *[]string  `json:"expected_path"`
		Path              *[]string  `json:"path"`
		ExpectedSpan      *pure.Span `json:"expected_span"`
		Span              *pure.Span `json:"span"`
	}
	if err := json.Unmarshal(raw, &oracle); err != nil {
		return fmt.Errorf("decode invalid-fixture oracle: %w", err)
	}
	if oracle.ExpectedError == "" {
		return fmt.Errorf("invalid-fixture oracle has no expected_error class")
	}
	var got *pure.Error
	if !errors.As(err, &got) {
		return fmt.Errorf("parse error is not *pure.Error: %T (%v)", err, err)
	}
	if got.Class != oracle.ExpectedError {
		return fmt.Errorf("error class = %q, want %q (reason %q)", got.Class, oracle.ExpectedError, got.Reason)
	}
	if want := firstString(oracle.ExpectedReason, oracle.Reason); want != nil && got.Reason != *want {
		return fmt.Errorf("error reason = %q, want %q", got.Reason, *want)
	}
	if want := firstString(oracle.ExpectedBody, oracle.Body); want != nil && got.Body != *want {
		return fmt.Errorf("error body = %q, want %q", got.Body, *want)
	}
	if want := firstString(oracle.ExpectedCanonical, oracle.Canonical); want != nil && got.Canonical != *want {
		return fmt.Errorf("error canonical = %q, want %q", got.Canonical, *want)
	}
	if want := firstString(oracle.ExpectedSection, oracle.SpecSection); want != nil && got.SpecSection != *want {
		return fmt.Errorf("error spec_section = %q, want %q", got.SpecSection, *want)
	}
	if want := firstInt(oracle.ExpectedLine, oracle.Line); want != nil && got.Line != *want {
		return fmt.Errorf("error line = %d, want %d", got.Line, *want)
	}
	if want := firstString(oracle.ExpectedLineText, oracle.LineText); want != nil && got.LineText != *want {
		return fmt.Errorf("error line_text = %q, want %q", got.LineText, *want)
	}
	if want := firstPath(oracle.ExpectedPath, oracle.Path); want != nil && !reflect.DeepEqual(got.Path, *want) {
		return fmt.Errorf("error path = %#v, want %#v", got.Path, *want)
	}
	if want := firstSpan(oracle.ExpectedSpan, oracle.Span); want != nil && !reflect.DeepEqual(got.Span, want) {
		return fmt.Errorf("error span = %#v, want %#v", got.Span, want)
	}
	return nil
}

// assertUnrepresentableError mirrors the in-repo helper: every writer
// refusal must be UnrepresentableAt with the pinned reason and the
// spec section implied by that reason.
func assertUnrepresentableError(err error, reason, label string) error {
	var got *pure.Error
	if !errors.As(err, &got) {
		return fmt.Errorf("%s: not *pure.Error: %T (%v)", label, err, err)
	}
	section, ok := unrepresentableSpecSection(reason)
	if !ok {
		return fmt.Errorf("%s: unknown unrepresentable reason %q", label, reason)
	}
	if got.Class != "UnrepresentableAt" || got.Reason != reason || got.SpecSection != section {
		return fmt.Errorf("%s: structured error = {class:%q reason:%q section:%q}, want {UnrepresentableAt %q %s}",
			label, got.Class, got.Reason, got.SpecSection, reason, section)
	}
	return nil
}

func unrepresentableSpecSection(reason string) (string, bool) {
	switch reason {
	case "CRByte", "BothFormsRequired", "LeadingWhitespaceCollision", "TrailingWhitespaceCollision":
		return "§5.9.7", true
	case "EmptyKeyName", "NonFiniteFloat", "ScalarRoot":
		return "§5.9.0", true
	default:
		return "", false
	}
}

func firstString(a, b *string) *string {
	if a != nil {
		return a
	}
	return b
}

func firstInt(a, b *int) *int {
	if a != nil {
		return a
	}
	return b
}

func firstPath(a, b *[]string) *[]string {
	if a != nil {
		return a
	}
	return b
}

func firstSpan(a, b *pure.Span) *pure.Span {
	if a != nil {
		return a
	}
	return b
}

// expectedLocationShape is the 0.9 expected_location metadata: 1-based
// line, raw-byte start, and EXACTLY ONE of end / minimum_end. Both or
// neither is a malformed oracle — rejected, never ignored.
type expectedLocationShape struct {
	Line       *int `json:"line"`
	Start      *int `json:"start"`
	End        *int `json:"end"`
	MinimumEnd *int `json:"minimum_end"`
}

// assertExpectedLocation validates the 0.9 expected_location metadata
// against the structured error's line and span (line >= 1, nonempty
// covering span, exact end OR minimum_end respected). InvalidUtf8 must
// additionally leave line_text null.
func assertExpectedLocation(err error, src []byte, raw []byte) error {
	var oracle struct {
		ExpectedError    string                 `json:"expected_error"`
		ExpectedLocation *expectedLocationShape `json:"expected_location"`
	}
	if err := json.Unmarshal(raw, &oracle); err != nil {
		return fmt.Errorf("decode expected_location oracle: %w", err)
	}
	loc := oracle.ExpectedLocation
	if loc == nil {
		return nil
	}
	var got *pure.Error
	if !errors.As(err, &got) {
		return fmt.Errorf("expected_location check: error is not *pure.Error: %T (%v)", err, err)
	}
	if loc.Line == nil || loc.Start == nil {
		return fmt.Errorf("expected_location metadata needs line and start")
	}
	if loc.End == nil && loc.MinimumEnd == nil {
		return fmt.Errorf("expected_location metadata needs exactly one of end/minimum_end")
	}
	if loc.End != nil && loc.MinimumEnd != nil {
		return fmt.Errorf("expected_location metadata must not carry both end and minimum_end")
	}
	if *loc.Line < 1 {
		return fmt.Errorf("expected_location line must be >= 1, got %d", *loc.Line)
	}
	if *loc.Start < 0 || *loc.Start > len(src) {
		return fmt.Errorf("expected_location start %d out of range (source is %d bytes)", *loc.Start, len(src))
	}
	if got.Line != *loc.Line {
		return fmt.Errorf("error line = %d, expected_location line = %d", got.Line, *loc.Line)
	}
	if got.Span == nil {
		return fmt.Errorf("error has null span; expected_location pins start=%d", *loc.Start)
	}
	if got.Span.Start != *loc.Start {
		return fmt.Errorf("error span start = %d, expected_location start = %d", got.Span.Start, *loc.Start)
	}
	if got.Span.End <= got.Span.Start {
		return fmt.Errorf("error span [%d,%d) is empty; spans must be nonempty and covering", got.Span.Start, got.Span.End)
	}
	if loc.End != nil {
		if got.Span.End != *loc.End {
			return fmt.Errorf("error span end = %d, expected_location end = %d", got.Span.End, *loc.End)
		}
	} else if got.Span.End < *loc.MinimumEnd {
		return fmt.Errorf("error span end = %d is below expected_location minimum_end = %d", got.Span.End, *loc.MinimumEnd)
	}
	if oracle.ExpectedError == "InvalidUtf8" && got.LineText != "" {
		return fmt.Errorf("InvalidUtf8 must leave line_text null, got %q", got.LineText)
	}
	return nil
}
