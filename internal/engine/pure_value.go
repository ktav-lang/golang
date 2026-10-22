package engine

// Insertion-ordered value tree for the pure backend. Objects keep their
// pairs in insertion order (spec § 5.9.11) with a small hash index for
// O(1) duplicate-key detection (§ 5.5) and O(1) dotted-prefix descent
// (§ 5.3.2). Go's map[string]any cannot carry order, so the ordered
// tree is internal only; the public decode contract (plain maps) is
// built from it in pure.go.

// pvalue kinds.
const (
	kNull byte = iota
	kBool
	kInt   // s holds the canonical base-10 digits
	kFloat // s holds the ryu shortest-round-trip text
	kStr
	kArr
	kObj
)

// pvalue is the internal Value (spec § 5): scalars carry their
// canonical text, compounds carry ordered children.
type pvalue struct {
	kind byte
	b    bool
	s    string
	arr  []*pvalue
	obj  *omap
}

func pvStr(s string) *pvalue     { return &pvalue{kind: kStr, s: s} }
func pvInt(s string) *pvalue     { return &pvalue{kind: kInt, s: s} }
func pvFloat(s string) *pvalue   { return &pvalue{kind: kFloat, s: s} }
func pvBool(b bool) *pvalue      { return &pvalue{kind: kBool, b: b} }
func pvArr() *pvalue             { return &pvalue{kind: kArr, arr: []*pvalue{}} }
func pvObj() *pvalue             { return &pvalue{kind: kObj, obj: newOmap()} }
func (v *pvalue) isObject() bool { return v.kind == kObj }

// kindLabel matches the Rust renderer's diagnostic labels (used in
// KeyPathConflict messages and conflict classification).
func (v *pvalue) kindLabel() string {
	switch v.kind {
	case kNull:
		return "null"
	case kBool:
		return "bool"
	case kInt:
		return "integer"
	case kFloat:
		return "float"
	case kStr:
		return "string"
	case kArr:
		return "array"
	default:
		return "object"
	}
}

// omap is an insertion-ordered string-keyed map of *pvalue.
type omap struct {
	entries []oentry
	index   map[string]int32
}

type oentry struct {
	key string
	val *pvalue
}

func newOmap() *omap { return &omap{index: make(map[string]int32)} }

// insertLeaf inserts (key, val). If key is occupied it returns the
// existing value and occupied=true, leaving the table untouched.
func (m *omap) insertLeaf(key string, val *pvalue) (existing *pvalue, occupied bool) {
	if i, ok := m.index[key]; ok {
		return m.entries[i].val, true
	}
	m.index[key] = int32(len(m.entries))
	m.entries = append(m.entries, oentry{key: key, val: val})
	return nil, false
}

// descend returns the object stored at key, inserting an empty object
// when absent (spec § 5.3.2 synthetic prefix). blocked=true when a
// non-object value occupies the slot (§ 6.3 BlockedByValue).
func (m *omap) descend(key string) (child *omap, blocked bool) {
	if i, ok := m.index[key]; ok {
		e := &m.entries[i]
		if e.val.kind == kObj {
			probe("dotted_reopen", 1) // the reopen-merge point (§ 5.3.2)
			return e.val.obj, false
		}
		return nil, true
	}
	obj := pvObj()
	m.index[key] = int32(len(m.entries))
	m.entries = append(m.entries, oentry{key: key, val: obj})
	return obj.obj, false
}

// ─── trivia-carrying tree (formatter only) ───────────────────────────

// tline is one preserved trivia line: a blank line or a whole-line
// comment (trimmed source text, starting at the first `#`).
type tline struct {
	blank   bool
	comment string
}

// fcompound carries per-slot trivia for a compound fvalue.
type fcompound struct {
	pairs    []fpair // object pairs, insertion order
	items    []fitem // array items
	trailing []tline // trivia after the last pair/item, before the close
}

type fpair struct {
	key    string
	trivia []tline
	val    *fvalue
}

type fitem struct {
	trivia []tline
	val    *fvalue
}

// fvalue is a pvalue plus, for compounds, their trivia slots. Scalars
// carry no trivia of their own — only the pair/item slot holding them.
// A compound fvalue doubles as its insertion table (idx + comp.pairs).
type fvalue struct {
	v    *pvalue // always non-nil; kind decides the shape
	comp *fcompound
	idx  map[string]int32
}

// stamp wraps a plain pvalue tree into an fvalue tree with empty trivia
// throughout — sound because a comment owns a whole line and can never
// occur inside an inline compound or a dotted-key expansion.
//
// Iterative (explicit down-up frame stack): dotted keys (§ 5.3.2) build
// trees of unbounded depth, so the conversion must not recurse.
func stamp(v *pvalue) *fvalue {
	type sframe struct {
		in  *pvalue
		out *fvalue
		i   int // next child to link
	}
	root := &fvalue{v: v}
	if v.kind != kObj && v.kind != kArr {
		return root
	}
	root.comp = &fcompound{}
	stack := []sframe{{in: v, out: root}}
	for len(stack) > 0 {
		f := &stack[len(stack)-1]
		if f.in.kind == kObj {
			if f.i < len(f.in.obj.entries) {
				e := f.in.obj.entries[f.i]
				f.i++
				child := &fvalue{v: e.val}
				f.out.comp.pairs = append(f.out.comp.pairs, fpair{key: e.key, val: child})
				if e.val.kind == kObj || e.val.kind == kArr {
					child.comp = &fcompound{}
					stack = append(stack, sframe{in: e.val, out: child})
				}
				continue
			}
		} else { // kArr
			if f.i < len(f.in.arr) {
				item := f.in.arr[f.i]
				f.i++
				child := &fvalue{v: item}
				f.out.comp.items = append(f.out.comp.items, fitem{val: child})
				if item.kind == kObj || item.kind == kArr {
					child.comp = &fcompound{}
					stack = append(stack, sframe{in: item, out: child})
				}
				continue
			}
		}
		stack = stack[:len(stack)-1]
	}
	return root
}

// toPlain drops all trivia, reconstructing the plain pvalue tree.
// Iterative (explicit down-up frame stack), like stamp: unbounded-depth
// fvalue trees from dotted keys must not recurse.
func (f *fvalue) toPlain() *pvalue {
	type tframe struct {
		in  *fvalue
		out *pvalue
		i   int
	}
	root := &pvalue{kind: f.v.kind, b: f.v.b, s: f.v.s}
	if f.v.kind != kObj && f.v.kind != kArr {
		return root
	}
	if f.v.kind == kObj {
		root.obj = newOmap()
	} else {
		root.arr = []*pvalue{}
	}
	stack := []tframe{{in: f, out: root}}
	for len(stack) > 0 {
		fr := &stack[len(stack)-1]
		if fr.in.v.kind == kObj {
			if fr.i < len(fr.in.comp.pairs) {
				p := fr.in.comp.pairs[fr.i]
				fr.i++
				child := &pvalue{kind: p.val.v.kind, b: p.val.v.b, s: p.val.v.s}
				fr.out.obj.insertLeaf(p.key, child)
				if p.val.v.kind == kObj {
					child.obj = newOmap()
					stack = append(stack, tframe{in: p.val, out: child})
				} else if p.val.v.kind == kArr {
					child.arr = []*pvalue{}
					stack = append(stack, tframe{in: p.val, out: child})
				}
				continue
			}
		} else { // kArr
			if fr.i < len(fr.in.comp.items) {
				it := fr.in.comp.items[fr.i]
				fr.i++
				child := &pvalue{kind: it.val.v.kind, b: it.val.v.b, s: it.val.v.s}
				fr.out.arr = append(fr.out.arr, child)
				if it.val.v.kind == kObj {
					child.obj = newOmap()
					stack = append(stack, tframe{in: it.val, out: child})
				} else if it.val.v.kind == kArr {
					child.arr = []*pvalue{}
					stack = append(stack, tframe{in: it.val, out: child})
				}
				continue
			}
		}
		stack = stack[:len(stack)-1]
	}
	return root
}

func (f *fvalue) isObject() bool    { return f.v.kind == kObj }
func (f *fvalue) kindLabel() string { return f.v.kindLabel() }

// trimTrailingBlanks drops trailing blank entries from a trivia run —
// used whenever a run becomes a trailing bucket (before a close or at
// EOF): padding with nothing after it carries no meaning.
func trimTrailingBlanks(t []tline) []tline {
	for len(t) > 0 && t[len(t)-1].blank {
		t = t[:len(t)-1]
	}
	return t
}
