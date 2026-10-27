package engine

// Helpers for semantic verification and temporary mutation trees.
func clonePV(v *pvalue) *pvalue {
	if v == nil {
		return nil
	}
	out := &pvalue{kind: v.kind, b: v.b, s: v.s}
	switch v.kind {
	case kArr:
		out.arr = make([]*pvalue, len(v.arr))
		for i, child := range v.arr {
			out.arr[i] = clonePV(child)
		}
	case kObj:
		out.obj = newOmap()
		for _, entry := range v.obj.entries {
			out.obj.insertLeaf(entry.key, clonePV(entry.val))
		}
	}
	return out
}

func renderInlineValue(v *pvalue) (string, *Error) {
	return inlineCompoundBody(v, 0)
}

func verifyMutated(src string, expected *pvalue) error {
	parsed, _, err := parseDoc(src, false, false)
	if err != nil {
		return editUnsupported("mutation produced invalid source: " + err.Error())
	}
	actual, ok := parsed.(*pvalue)
	if !ok || !pvEqual(actual, expected) {
		return editUnsupported("mutation changed the parsed value unexpectedly")
	}
	return nil
}

// verifyMutatedKeyOrder guards source edits against parsers that preserve values
// while changing the insertion order of object keys.
func verifyMutatedKeyOrder(src string, expected *pvalue) error {
	parsed, _, err := parseDoc(src, false, false)
	if err != nil {
		return editUnsupported("mutation produced invalid source: " + err.Error())
	}
	actual, ok := parsed.(*pvalue)
	if !ok || !pvEqualKeyOrder(actual, expected) {
		return editUnsupported("mutation changed parsed key order unexpectedly")
	}
	return nil
}

func pvEqualKeyOrder(a, b *pvalue) bool {
	if a == nil || b == nil || a.kind != b.kind || a.b != b.b || a.s != b.s {
		return a == b
	}
	switch a.kind {
	case kArr:
		if len(a.arr) != len(b.arr) {
			return false
		}
		for i := range a.arr {
			if !pvEqualKeyOrder(a.arr[i], b.arr[i]) {
				return false
			}
		}
	case kObj:
		if len(a.obj.entries) != len(b.obj.entries) {
			return false
		}
		for i, e := range a.obj.entries {
			if e.key != b.obj.entries[i].key || !pvEqualKeyOrder(e.val, b.obj.entries[i].val) {
				return false
			}
		}
	}
	return true
}

func removePVObject(obj *omap, key string) {
	idx, ok := obj.index[key]
	if !ok {
		return
	}
	delete(obj.index, key)
	copy(obj.entries[idx:], obj.entries[idx+1:])
	obj.entries = obj.entries[:len(obj.entries)-1]
	for i := int(idx); i < len(obj.entries); i++ {
		obj.index[obj.entries[i].key] = int32(i)
	}
}

func renamePVObject(obj *omap, oldKey, newKey string) {
	idx, ok := obj.index[oldKey]
	if !ok {
		return
	}
	if _, exists := obj.index[newKey]; exists {
		return
	}
	delete(obj.index, oldKey)
	obj.entries[idx].key = newKey
	obj.index[newKey] = idx
}
