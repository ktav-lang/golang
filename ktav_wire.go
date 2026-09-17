package ktav

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
)

// ─── tagged-JSON <-> Go conversion ────────────────────────────────────

// decodeJSON walks the Rust-emitted JSON (with $i / $f tags) into native
// Go values. Object key order is preserved via orderedMap.
func decodeJSON(raw []byte) (any, error) {
	if len(raw) == 0 {
		return nil, newError("ktav: empty decode input")
	}
	dec := json.NewDecoder(bytesReader(raw))
	dec.UseNumber()
	tok, err := dec.Token()
	if err != nil {
		return nil, newErrorf("decode: %s", err)
	}
	return decodeValue(dec, tok)
}

func decodeValue(dec *json.Decoder, tok json.Token) (any, error) {
	switch t := tok.(type) {
	case json.Delim:
		switch t {
		case '{':
			return decodeObject(dec)
		case '[':
			return decodeArray(dec)
		default:
			return nil, newErrorf("unexpected delim %q", t)
		}
	case bool:
		return t, nil
	case json.Number:
		if i, err := t.Int64(); err == nil {
			return i, nil
		}
		f, err := t.Float64()
		if err != nil {
			return nil, newErrorf("bad number literal: %q", string(t))
		}
		return f, nil
	case string:
		return t, nil
	case nil:
		return nil, nil
	default:
		return nil, newErrorf("unexpected token %T", tok)
	}
}

func decodeArray(dec *json.Decoder) ([]any, error) {
	out := []any{}
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return nil, newErrorf("decode: %s", err)
		}
		v, err := decodeValue(dec, tok)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	if _, err := dec.Token(); err != nil { // consume ']'
		return nil, newErrorf("decode: %s", err)
	}
	return out, nil
}

func decodeObject(dec *json.Decoder) (any, error) {
	// Read the first key/value if any. The tagged-scalar shapes
	// (`{"$i": "..."}`, `{"$f": "..."}`) are single-entry objects with a
	// string payload — peek the first entry, special-case it, and only
	// allocate a map for the general case.
	if !dec.More() {
		if _, err := dec.Token(); err != nil { // consume '}'
			return nil, err
		}
		return map[string]any{}, nil
	}

	firstKeyTok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	firstKey, ok := firstKeyTok.(string)
	if !ok {
		return nil, newErrorf("non-string object key: %v", firstKeyTok)
	}
	firstValTok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	firstVal, err := decodeValue(dec, firstValTok)
	if err != nil {
		return nil, err
	}

	if !dec.More() {
		if _, err := dec.Token(); err != nil { // consume '}'
			return nil, err
		}
		if firstKey == "$i" || firstKey == "$f" {
			s, ok := firstVal.(string)
			if !ok {
				return nil, newErrorf("%s payload must be a string", firstKey)
			}
			if firstKey == "$i" {
				return parseIntegerScalar(s)
			}
			return parseFloatScalar(s)
		}
		return map[string]any{firstKey: firstVal}, nil
	}

	out := make(map[string]any)
	out[firstKey] = firstVal
	for dec.More() {
		ktok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		key, ok := ktok.(string)
		if !ok {
			return nil, newErrorf("non-string object key: %v", ktok)
		}
		vtok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		v, err := decodeValue(dec, vtok)
		if err != nil {
			return nil, err
		}
		out[key] = v
	}
	if _, err := dec.Token(); err != nil { // consume '}'
		return nil, err
	}
	return out, nil
}

// ─── tagged → untagged flattening (for LoadsInto) ─────────────────────

func flattenTagged(raw []byte) ([]byte, error) {
	var v any
	dec := json.NewDecoder(bytesReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	v = flattenAny(v)
	return json.Marshal(v)
}

func flattenAny(v any) any {
	switch t := v.(type) {
	case map[string]any:
		if len(t) == 1 {
			if s, ok := t["$i"].(string); ok {
				if n, err := parseIntegerScalar(s); err == nil {
					return n
				}
				return json.Number(s)
			}
			if s, ok := t["$f"].(string); ok {
				return json.Number(s)
			}
		}
		out := make(map[string]any, len(t))
		for k, val := range t {
			out[k] = flattenAny(val)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, x := range t {
			out[i] = flattenAny(x)
		}
		return out
	default:
		return v
	}
}

// ─── Go → tagged JSON (for Dumps) ─────────────────────────────────────

func encodeTagged(v any) ([]byte, error) {
	enc, err := toTagged(v)
	if err != nil {
		return nil, err
	}
	return json.Marshal(enc)
}

func toTagged(v any) (any, error) {
	switch t := v.(type) {
	case nil:
		return nil, nil
	case bool:
		return t, nil
	case string:
		return t, nil
	case int:
		return taggedI(fmt.Sprintf("%d", t)), nil
	case int8:
		return taggedI(fmt.Sprintf("%d", t)), nil
	case int16:
		return taggedI(fmt.Sprintf("%d", t)), nil
	case int32:
		return taggedI(fmt.Sprintf("%d", t)), nil
	case int64:
		return taggedI(fmt.Sprintf("%d", t)), nil
	case uint:
		return taggedI(fmt.Sprintf("%d", t)), nil
	case uint8:
		return taggedI(fmt.Sprintf("%d", t)), nil
	case uint16:
		return taggedI(fmt.Sprintf("%d", t)), nil
	case uint32:
		return taggedI(fmt.Sprintf("%d", t)), nil
	case uint64:
		return taggedI(fmt.Sprintf("%d", t)), nil
	case *big.Int:
		if t == nil {
			return nil, nil
		}
		return taggedI(t.String()), nil
	case big.Int:
		return taggedI(t.String()), nil
	case float32:
		return floatTag(float64(t))
	case float64:
		return floatTag(t)
	case json.Number:
		s := string(t)
		if s == "" {
			return nil, newError("empty json.Number")
		}
		if isIntegerLiteral(s) {
			return taggedI(s), nil
		}
		return taggedF(s), nil
	case []any:
		out := make([]any, len(t))
		for i, x := range t {
			y, err := toTagged(x)
			if err != nil {
				return nil, err
			}
			out[i] = y
		}
		return out, nil
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, x := range t {
			y, err := toTagged(x)
			if err != nil {
				return nil, err
			}
			out[k] = y
		}
		return out, nil
	default:
		// Fall back through encoding/json: lets callers pass structs
		// and custom maps via reflection.
		raw, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		var any1 any
		dec := json.NewDecoder(bytesReader(raw))
		dec.UseNumber()
		if err := dec.Decode(&any1); err != nil {
			return nil, err
		}
		return toTagged(any1)
	}
}

func taggedI(digits string) map[string]any {
	return map[string]any{"$i": digits}
}

func taggedF(text string) map[string]any {
	return map[string]any{"$f": text}
}

func floatTag(f float64) (any, error) {
	if isNaN(f) || isInf(f) {
		return nil, newError("ktav: NaN / Inf are not representable")
	}
	s := formatFloat(f)
	return taggedF(s), nil
}

// ─── misc small helpers ───────────────────────────────────────────────

func parseIntegerScalar(s string) (any, error) {
	if s == "" {
		return nil, newError("empty :i scalar")
	}
	// Try to fit into int64 first; otherwise return *big.Int so the
	// caller never silently loses precision.
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		return n, nil
	}
	bi := new(big.Int)
	if _, ok := bi.SetString(s, 10); !ok {
		return nil, newErrorf("bad integer literal: %q", s)
	}
	return bi, nil
}

func parseFloatScalar(s string) (float64, error) {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, newErrorf("bad float literal: %q", s)
	}
	return f, nil
}

func isIntegerLiteral(s string) bool {
	if s == "" {
		return false
	}
	i := 0
	if s[0] == '-' || s[0] == '+' {
		i++
	}
	if i == len(s) {
		return false
	}
	for ; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}
