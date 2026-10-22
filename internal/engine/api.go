package engine

// Operations provides the shared pure-Go implementation. Its zero value is ready to use.
type Operations struct{}

func New() *Operations { return &Operations{} }

func (o *Operations) Loads(src string) (any, error)                  { return loadsPublic(src, false) }
func (o *Operations) LoadsStrict(src string) (any, error)            { return loadsPublic(src, true) }
func (o *Operations) LoadsInto(src string, target any) error         { return loadsIntoPublic(src, target) }
func (o *Operations) Dumps(v any) (string, error)                    { return dumpsPublic(v, false, false) }
func (o *Operations) DumpsForceStrings(v any) (string, error)        { return dumpsPublic(v, true, false) }
func (o *Operations) EmitCanonical(v any) (string, error)            { return dumpsPublic(v, false, true) }
func (o *Operations) FormatSource(src string) (string, error)        { return formatPublic(src) }
func (o *Operations) CanonicalFromSource(src string) (string, error) { return canonicalPublic(src) }

func PureLoads(b []byte) ([]byte, error)               { return pureLoadsTagged(b) }
func PureLoadsStrict(b []byte) ([]byte, error)         { return pureLoadsStrictTagged(b) }
func PureDumps(b []byte) ([]byte, error)               { return pureDumpsTagged(b) }
func PureDumpsForceStrings(b []byte) ([]byte, error)   { return pureDumpsForceStringsTagged(b) }
func PureEmitCanonical(b []byte) ([]byte, error)       { return pureEmitCanonicalTagged(b) }
func PureFormatSource(b []byte) ([]byte, error)        { return pureFormatSource(b) }
func PureCanonicalFromSource(b []byte) ([]byte, error) { return pureCanonicalFromSource(b) }

// Shared conversion and wire seams used by the native-backed root package.
func DecodeJSON(raw []byte) (any, error)       { return decodeJSON(raw) }
func FlattenTagged(raw []byte) ([]byte, error) { return flattenTagged(raw) }
func EncodeWriterTagged(v any, forceStrings bool) ([]byte, error) {
	return encodeWriterTagged(v, forceStrings)
}
func PrepareWriterValue(v any, path []string, forceStrings bool) (any, error) {
	return prepareWriterValue(v, path, forceStrings)
}
func InvalidUTF8Error(src string) *Error  { return invalidUTF8Error(src) }
func ErrorFromEnvelope(raw []byte) *Error { return errorFromEnvelope(raw) }
func UnrepresentableError(reason string, path []string) *Error {
	return unrepresentableError(reason, path)
}
func NewErrorf(format string, args ...any) *Error { return newErrorf(format, args...) }
func FirstInvalidUTF8Span(src string) (int, int, bool) {
	return firstInvalidUTF8Span(src)
}
func LineOfOffset(src string, offset int) int { return lineOfOffset(src, offset) }
