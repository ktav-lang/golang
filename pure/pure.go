package pure

import "github.com/ktav-lang/golang/internal/engine"

type Error = engine.Error
type Span = engine.Span
type Operations = engine.Operations

func New() *Operations                               { return engine.New() }
func Loads(src string) (any, error)                  { return engine.New().Loads(src) }
func LoadsStrict(src string) (any, error)            { return engine.New().LoadsStrict(src) }
func LoadsInto(src string, target any) error         { return engine.New().LoadsInto(src, target) }
func Dumps(v any) (string, error)                    { return engine.New().Dumps(v) }
func DumpsForceStrings(v any) (string, error)        { return engine.New().DumpsForceStrings(v) }
func EmitCanonical(v any) (string, error)            { return engine.New().EmitCanonical(v) }
func FormatSource(src string) (string, error)        { return engine.New().FormatSource(src) }
func CanonicalFromSource(src string) (string, error) { return engine.New().CanonicalFromSource(src) }

// EditValue replaces the value addressed by the textual dotted path,
// splicing only the target's byte range. See the ktav package's
// EditValue for the full contract; this pure entry never touches the
// native library.
func EditValue(src, textualPath string, replacement any) (string, error) {
	return engine.New().EditValue(src, textualPath, replacement)
}

// EditValueSegments is EditValue with pre-decoded literal segments: no
// additional unescaping is performed.
func EditValueSegments(src string, segments []string, replacement any) (string, error) {
	return engine.New().EditValueSegments(src, segments, replacement)
}
