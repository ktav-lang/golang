package ktav

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/big"
	"reflect"
	"sort"
	"strconv"
)

// ─── tagged-JSON <-> Go conversion ────────────────────────────────────

// decodeJSON walks the Rust-emitted JSON (with $i / $f tags) into native
// Go values using an EXPLICIT frame stack, so wire nesting depth can
// never overflow the goroutine stack. Object key order is not preserved
// by the returned map.
func decodeJSON(raw []byte) (any, error) {
	if len(raw) == 0 {
		return nil, newError("ktav: empty decode input")
	}
	dec := json.NewDecoder(bytesReader(raw))
	dec.UseNumber()
	type djFrame struct {
		obj     map[string]any
		arr     []any
		curKey  string // last accepted object key (valid when gotKey)
		gotKey  bool   // curKey holds a real key ("" may be a JSON key)
		wantKey bool   // next string token is a key
	}
	attach := func(f *djFrame, v any) error {
		if f.obj != nil {
			if f.wantKey {
				key, ok := v.(string)
				if !ok {
					return newErrorf("non-string object key: %v", v)
				}
				f.curKey, f.gotKey, f.wantKey = key, true, false
				return nil
			}
			if !f.gotKey {
				return newError("decode: value without object key")
			}
			f.obj[f.curKey] = v
			f.gotKey, f.wantKey = false, true
			return nil
		}
		f.arr = append(f.arr, v)
		return nil
	}
	var stack []*djFrame
	var root any
	haveRoot := false
	tok, err := dec.Token()
	for {
		if err != nil {
			if err == io.EOF && len(stack) == 0 {
				return root, nil
			}
			return nil, newErrorf("decode: %s", err)
		}
		if haveRoot {
			return nil, newError("decode: trailing data after top-level value")
		}
		if d, ok := tok.(json.Delim); ok {
			switch d {
			case '{':
				stack = append(stack, &djFrame{obj: map[string]any{}, wantKey: true})
			case '[':
				stack = append(stack, &djFrame{arr: []any{}})
			case '}', ']':
				if len(stack) == 0 {
					return nil, newErrorf("decode: unexpected %q", d)
				}
				top := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				var closed any = any(top.arr)
				if top.obj != nil {
					closed = any(top.obj)
					if len(top.obj) == 1 {
						if s, ok := top.obj["$i"].(string); ok {
							if closed, err = parseIntegerScalar(s); err != nil {
								return nil, err
							}
						} else if s, ok := top.obj["$f"].(string); ok {
							if closed, err = parseFloatScalar(s); err != nil {
								return nil, err
							}
						}
					}
				}
				if len(stack) == 0 {
					root, haveRoot = closed, true
				} else if err = attach(stack[len(stack)-1], closed); err != nil {
					return nil, err
				}
			default:
				return nil, newErrorf("unexpected delim %q", d)
			}
		} else {
			v, serr := djScalar(tok)
			if serr != nil {
				return nil, serr
			}
			if len(stack) == 0 {
				root, haveRoot = v, true
			} else if err = attach(stack[len(stack)-1], v); err != nil {
				return nil, err
			}
		}
		tok, err = dec.Token()
	}
}

func djScalar(tok json.Token) (any, error) {
	switch t := tok.(type) {
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

// ─── tagged → untagged flattening (for LoadsInto) ─────────────────────

func flattenTagged(raw []byte) ([]byte, error) {
	var v any
	dec := json.NewDecoder(bytesReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	// haveRoot-style guard, mirroring decodeJSON: LoadsInto must
	// reject trailing data exactly like Loads/LoadsStrict — a second
	// top-level JSON value is an error, not a silent overwrite.
	if _, err := dec.Token(); err != io.EOF {
		return nil, newError("decode: trailing data after top-level value")
	}
	// dec.Decode is encoding/json (internally recursive with its own
	// ~10k nesting limit and an error, not a crash, beyond it); the
	// flattening walk below is iterative. json.Marshal on the way out
	// is encoding/json too — same character: bounded nesting, error
	// (not crash) beyond its limit.
	return json.Marshal(flattenAny(v))
}

// flattenAny untags tagged scalars iteratively: each fresh container is
// attached to its parent immediately (map/slice reference semantics),
// then filled by explicit work-stack tasks — no recursion on input
// depth. A single-entry {"$i": ...} / {"$f": ...} object collapses to
// the scalar; anything else keeps its keys verbatim.
func flattenAny(v any) any {
	type faTask struct {
		src    any
		key    string
		idx    int
		parent any // nil for the root
	}
	store := func(parent any, key string, idx int, out any, root *any) {
		switch p := parent.(type) {
		case nil:
			*root = out
		case map[string]any:
			p[key] = out
		case []any:
			p[idx] = out
		}
	}
	var root any
	stack := []faTask{{src: v}}
	for len(stack) > 0 {
		t := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		var out any = t.src
		if s, ok := t.src.(map[string]any); ok {
			untagged := false
			if len(s) == 1 {
				if str, ok := s["$i"].(string); ok {
					if n, err := parseIntegerScalar(str); err == nil {
						out = n
					} else {
						out = json.Number(str)
					}
					untagged = true
				} else if str, ok := s["$f"].(string); ok {
					out = json.Number(str)
					untagged = true
				}
			}
			if !untagged {
				m := make(map[string]any, len(s))
				out = m
				for k, x := range s {
					stack = append(stack, faTask{src: x, key: k, parent: m})
				}
			}
		} else if s, ok := t.src.([]any); ok {
			a := make([]any, len(s))
			out = a
			for i, x := range s {
				stack = append(stack, faTask{src: x, idx: i, parent: a})
			}
		}
		store(t.parent, t.key, t.idx, out, &root)
	}
	return root
}

// ─── Go → tagged JSON (for Dumps) ─────────────────────────────────────

func encodeWriterTagged(v any, forceStrings bool) ([]byte, error) {
	// NOTE (honest bounds): the final json.Marshal is encoding/json —
	// internally recursive, with its own nesting limit that returns an
	// error (not a crash) beyond it. The iterative, explicit-stack
	// guarantee of this package covers toTagged and prepareWriterValue
	// below; prepareWriterValue sorts each map's keys (alphabetical
	// wire order), which costs O(n log n) per map. This package does
	// not extend encoding/json's limits.
	value, err := toTagged(v)
	if err != nil {
		return nil, err
	}
	if !isCompoundRoot(value) {
		return nil, unrepresentableError("ScalarRoot", nil)
	}
	value, err = prepareWriterValue(value, nil, forceStrings)
	if err != nil {
		return nil, err
	}
	return json.Marshal(value)
}

func isCompoundRoot(v any) bool {
	switch t := v.(type) {
	case []any:
		return true
	case map[string]any:
		if len(t) == 1 {
			if _, ok := t["$i"]; ok {
				return false
			}
			if _, ok := t["$f"]; ok {
				return false
			}
		}
		return true
	default:
		return false
	}
}

// prepareWriterValue walks the tagged tree iteratively (explicit frame
// stack), coercing non-finite float payloads per forceStrings and
// rejecting empty key names with the full dotted path. A single shared
// path buffer is pushed/popped around each child's subtree (frames
// record the length to restore); a snapshot copy is taken ONLY when
// constructing an error, so path bookkeeping is O(1) per edge. The
// walk itself is linear in tree size EXCEPT the per-map key sort
// (alphabetical wire order): O(n log n) per map.
func prepareWriterValue(v any, _ []string, forceStrings bool) (any, error) {
	type pwFrame struct {
		dst     any      // map[string]any or []any, mutated in place
		keys    []string // sorted map keys (nil for a slice frame)
		idx     int      // next entry index
		pathLen int      // len(sharedPath) at frame entry; restored on pop
	}
	newFrame := func(src any) pwFrame {
		if m, ok := src.(map[string]any); ok {
			keys := make([]string, 0, len(m))
			for key := range m {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			return pwFrame{dst: m, keys: keys}
		}
		return pwFrame{dst: src}
	}
	pathErr := func(reason string, p []string) error {
		// cold path: snapshot the shared buffer so the caller owns it
		return unrepresentableError(reason, append([]string(nil), p...))
	}
	path := []string{}
	stack := []pwFrame{newFrame(v)}
	for len(stack) > 0 {
		f := &stack[len(stack)-1]
		var child any
		var mapDst map[string]any
		var sliceDst []any
		var key string
		if m, ok := f.dst.(map[string]any); ok {
			mapDst = m
			if f.idx >= len(f.keys) {
				stack = stack[:len(stack)-1]
				continue
			}
			key = f.keys[f.idx]
			// one push per edge: parent path + this key
			path = append(path[:f.pathLen], key)
			child = m[key]
			if key == "" {
				return nil, pathErr("EmptyKeyName", path)
			}
		} else {
			sliceDst = f.dst.([]any)
			if f.idx >= len(sliceDst) {
				stack = stack[:len(stack)-1]
				continue
			}
			// slice entries add no path segment (§5.9.5 dotted-key
			// paths name map edges only)
			path = path[:f.pathLen]
			child = sliceDst[f.idx]
		}
		f.idx++
		if fm, ok := child.(map[string]any); ok && len(fm) == 1 {
			if text, ok2 := fm["$f"].(goFloatPayload); ok2 && isNonFiniteFloatText(string(text)) {
				if !forceStrings {
					return nil, pathErr("NonFiniteFloat", path)
				}
				child = string(text)
			}
		}
		switch child.(type) {
		case map[string]any, []any:
			fr := newFrame(child)
			fr.pathLen = len(path)
			stack = append(stack, fr)
		default:
			if mapDst != nil {
				mapDst[key] = child
			} else {
				sliceDst[f.idx-1] = child
			}
		}
	}
	return v, nil
}

func isNonFiniteFloatText(s string) bool {
	return s == "NaN" || s == "+Inf" || s == "-Inf"
}

type goFloatPayload string

// toTagged converts a Go value into the tagged-JSON shape using an
// explicit work stack, so externally controlled depth can never
// overflow the goroutine stack. Cycle detection is ACTIVE-PATH based:
// when a container task pops it is marked active and an EXIT task is
// pushed BELOW its children, so the container stays in the active set
// until every descendant has completed. Revisiting an active container
// is a cycle (structured error); a shared DAG node, exited before the
// revisit, is converted again. Container identity is (kind, pointer,
// len, cap) — distinct slice views of one backing array keep distinct
// keys.
func toTagged(v any) (any, error) {
	type contKey struct {
		kind   reflect.Kind
		ptr    uintptr
		length int
		cap    int
	}
	keyOf := func(c any) contKey {
		rv := reflect.ValueOf(c)
		k := contKey{kind: rv.Kind(), ptr: rv.Pointer(), length: rv.Len()}
		if rv.Kind() == reflect.Slice {
			k.cap = rv.Cap()
		}
		return k
	}
	store := func(parent any, key string, idx int, out any, root *any) {
		switch p := parent.(type) {
		case nil:
			*root = out
		case map[string]any:
			p[key] = out
		case []any:
			p[idx] = out
		}
	}
	type ttTask struct {
		src    any
		key    string
		idx    int
		parent any     // destination container to store into (nil for root)
		isExit bool    // EXIT task: remove exitKey from the active set
		exit   contKey // container key to deactivate
	}
	// Active-path invariant: an EXIT task sits BELOW its children in the
	// LIFO stack, so it runs only after ALL descendants complete — the
	// exact window in which a revisit means a cycle. Sets are deleted on
	// exit, so nothing grows with the number of distinct containers.
	active := map[contKey]bool{}
	cyclic := func() *Error {
		return &Error{Class: "Message", Msg: "ktav: cyclic Go value cannot be encoded"}
	}
	ttScalar := func(x any) (any, bool, error) {
		switch t := x.(type) {
		case nil, bool, string:
			return t, true, nil
		case int:
			return taggedI(fmt.Sprintf("%d", t)), true, nil
		case int8:
			return taggedI(fmt.Sprintf("%d", t)), true, nil
		case int16:
			return taggedI(fmt.Sprintf("%d", t)), true, nil
		case int32:
			return taggedI(fmt.Sprintf("%d", t)), true, nil
		case int64:
			return taggedI(fmt.Sprintf("%d", t)), true, nil
		case uint:
			return taggedI(fmt.Sprintf("%d", t)), true, nil
		case uint8:
			return taggedI(fmt.Sprintf("%d", t)), true, nil
		case uint16:
			return taggedI(fmt.Sprintf("%d", t)), true, nil
		case uint32:
			return taggedI(fmt.Sprintf("%d", t)), true, nil
		case uint64:
			return taggedI(fmt.Sprintf("%d", t)), true, nil
		case *big.Int:
			if t == nil {
				return nil, true, nil
			}
			return taggedI(t.String()), true, nil
		case big.Int:
			return taggedI(t.String()), true, nil
		case float32:
			out, err := floatTag(float64(t))
			return out, true, err
		case float64:
			out, err := floatTag(t)
			return out, true, err
		case json.Number:
			s := string(t)
			if s == "" {
				return nil, true, newError("empty json.Number")
			}
			if isIntegerLiteral(s) {
				return taggedI(s), true, nil
			}
			return taggedF(s), true, nil
		default:
			return nil, false, nil
		}
	}
	var root any
	stack := []ttTask{{src: v}}
	for len(stack) > 0 {
		t := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if t.isExit {
			delete(active, t.exit)
			continue
		}
		switch s := t.src.(type) {
		case map[string]any, []any:
			k := keyOf(t.src)
			if active[k] {
				return nil, cyclic()
			}
			active[k] = true
			// EXIT FIRST, then children: it pops only after the whole
			// subtree has been converted.
			stack = append(stack, ttTask{isExit: true, exit: k})
			if m, ok := s.(map[string]any); ok {
				out := make(map[string]any, len(m))
				store(t.parent, t.key, t.idx, out, &root)
				for key, x := range m {
					stack = append(stack, ttTask{src: x, key: key, parent: out})
				}
			} else {
				a := s.([]any)
				out := make([]any, len(a))
				store(t.parent, t.key, t.idx, out, &root)
				for i, x := range a {
					stack = append(stack, ttTask{src: x, idx: i, parent: out})
				}
			}
		default:
			out, ok, err := ttScalar(t.src)
			if err != nil {
				return nil, err
			}
			if ok {
				store(t.parent, t.key, t.idx, out, &root)
			} else {
				// Fall back through encoding/json: lets callers pass
				// structs and custom maps via reflection. json.Marshal
				// is internally recursive; it enforces its own nesting
				// limit (~10k) and returns an error (not a crash)
				// beyond it — that depth behaviour is encoding/json's,
				// not something this package verifies or extends.
				raw, err := json.Marshal(t.src)
				if err != nil {
					return nil, err
				}
				var any1 any
				dec := json.NewDecoder(bytesReader(raw))
				dec.UseNumber()
				if err := dec.Decode(&any1); err != nil {
					return nil, err
				}
				stack = append(stack, ttTask{src: any1, key: t.key, idx: t.idx, parent: t.parent})
			}
		}
	}
	return root, nil
}

func taggedI(digits string) map[string]any {
	return map[string]any{"$i": digits}
}

func taggedF(text string) map[string]any {
	return map[string]any{"$f": text}
}

func floatTag(f float64) (any, error) {
	if isNaN(f) {
		return map[string]any{"$f": goFloatPayload("NaN")}, nil
	}
	if math.IsInf(f, 1) {
		return map[string]any{"$f": goFloatPayload("+Inf")}, nil
	}
	if math.IsInf(f, -1) {
		return map[string]any{"$f": goFloatPayload("-Inf")}, nil
	}
	s := formatFloat(f)
	return map[string]any{"$f": goFloatPayload(s)}, nil
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
