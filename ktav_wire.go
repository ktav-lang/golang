package ktav

import "github.com/ktav-lang/golang/internal/engine"

func decodeJSON(raw []byte) (any, error)       { return engine.DecodeJSON(raw) }
func flattenTagged(raw []byte) ([]byte, error) { return engine.FlattenTagged(raw) }
func encodeWriterTagged(v any, force bool) ([]byte, error) {
	return engine.EncodeWriterTagged(v, force)
}
func prepareWriterValue(v any, path []string, force bool) (any, error) {
	return engine.PrepareWriterValue(v, path, force)
}
