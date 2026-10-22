package ktav

import "github.com/ktav-lang/golang/internal/engine"

type Error = engine.Error
type Span = engine.Span

func errorFromEnvelope(raw []byte) *Error { return engine.ErrorFromEnvelope(raw) }
func invalidUTF8Error(src string) *Error  { return engine.InvalidUTF8Error(src) }
func unrepresentableError(reason string, path []string) *Error {
	return engine.UnrepresentableError(reason, path)
}
func firstInvalidUTF8Span(src string) (int, int, bool) {
	return engine.FirstInvalidUTF8Span(src)
}
func lineOfOffset(src string, pos int) int { return engine.LineOfOffset(src, pos) }
