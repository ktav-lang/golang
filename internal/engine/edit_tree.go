package engine

// Positional tree for the pure edit engine (the Rust edit/tree.rs port):
// a line walk mirroring the real parser's dispatch decisions that
// records, per value, the byte range a lossless splice may replace.

import (
	"fmt"
	"strings"
)

// editSlotKind mirrors the Rust Slot enum's three shapes.
const (
	slotPair = iota
	slotItem
	slotBlock
)

type editSlot struct {
	kind      int
	colon     int // Pair: byte of the first `:`
	b0, b1    int // Pair/Item: body bytes; Block: [b0, b1) range
	hasAnchor bool
}

type editNode struct {
	kind   byte // kStr for scalar, kObj, kArr
	table  *editTable
	items  []*editNode
	inline bool
	slot   *editSlot
}

func editScalarNode(slot *editSlot) *editNode {
	return &editNode{kind: kStr, slot: slot}
}

func editEmptyObjectNode(slot *editSlot) *editNode {
	return &editNode{kind: kObj, table: newEditTable(), slot: slot}
}

func editEmptyArrayNode(slot *editSlot) *editNode {
	return &editNode{kind: kArr, items: []*editNode{}, slot: slot}
}

func (n *editNode) isObject() bool { return n.kind == kObj }
func (n *editNode) isArray() bool  { return n.kind == kArr }

func (n *editNode) child(key string) *editNode {
	if !n.isObject() {
		return nil
	}
	return n.table.get(key)
}

func (n *editNode) item(i int) *editNode {
	if !n.isArray() || i < 0 || i >= len(n.items) {
		return nil
	}
	return n.items[i]
}

func (n *editNode) kindLabel() string {
	switch n.kind {
	case kObj:
		return "object"
	case kArr:
		return "array"
	default:
		return "string"
	}
}

type editEntry struct {
	key  string
	node *editNode
}

// editTable is the insertion-ordered object table with dotted-key /
// reopen semantics identical to the real parser's omap.
type editTable struct {
	entries []editEntry
	index   map[string]int
}

func newEditTable() *editTable { return &editTable{index: make(map[string]int)} }

func (t *editTable) get(key string) *editNode {
	if i, ok := t.index[key]; ok {
		return t.entries[i].node
	}
	return nil
}

func (t *editTable) insert(key string, node *editNode) bool {
	if _, ok := t.index[key]; ok {
		return false
	}
	t.index[key] = len(t.entries)
	t.entries = append(t.entries, editEntry{key: key, node: node})
	return true
}

// descend returns the object table at key, creating a synthetic object
// when absent; reopen (existing object) invalidates the child's slot —
// the value became noncontiguous. blocked mirrors § 6.3.
func (t *editTable) descend(key string) (*editTable, bool, bool) {
	if i, ok := t.index[key]; ok {
		e := &t.entries[i]
		if e.node.kind == kObj {
			e.node.slot = nil
			return e.node.table, true, false
		}
		return nil, false, true
	}
	child := editEmptyObjectNode(nil)
	t.index[key] = len(t.entries)
	t.entries = append(t.entries, editEntry{key: key, node: child})
	return child.table, false, false
}

// editInsertValue mirrors the parser's insertValueAny over editTable:
// dotted-key expansion, reopen, and duplicate/conflict outcomes. The
// source has already parsed, so any anomaly is an internal divergence —
// surfaced as an error the caller maps to EditUnsupported.
func editInsertValue(t *editTable, rawKey string, node *editNode) error {
	if keyIsSingleSegment(rawKey) {
		trimmed := trimInline(rawKey)
		if trimmed == "" {
			return fmt.Errorf("empty key")
		}
		decoded, bad := decodeKeySegment(trimmed)
		if bad != "" {
			return fmt.Errorf("bad key escape")
		}
		if !t.insert(decoded, node) {
			return fmt.Errorf("duplicate key")
		}
		return nil
	}
	segments := splitKeyPath(rawKey)
	cur := t
	for idx, seg := range segments {
		trimmed := trimInline(seg)
		if trimmed == "" {
			return fmt.Errorf("empty key")
		}
		decoded, bad := decodeKeySegment(trimmed)
		if bad != "" {
			return fmt.Errorf("bad key escape")
		}
		if idx+1 == len(segments) {
			if !cur.insert(decoded, node) {
				return fmt.Errorf("duplicate key")
			}
			return nil
		}
		next, _, blocked := cur.descend(decoded)
		if blocked {
			return fmt.Errorf("key path conflict")
		}
		cur = next
	}
	return nil
}

type editFrame struct {
	object        bool
	table         *editTable
	items         []*editNode
	pendingKey    string
	hasPendingKey bool
	pendingSpan   kspan
	pendingAnchor int
	hasAnchor     bool
	rangeStart    int
}

type editBuilder struct {
	stack          []*editFrame
	collecting     int // 0 none, 1 stripped `(`, 2 verbatim `((`
	collectingOpen int
	rootInit       bool
	rootConsumed   bool
	rootInline     *editNode
	rootIsExplicit bool
}

func (b *editBuilder) top() *editFrame { return b.stack[len(b.stack)-1] }

func (b *editBuilder) internalErr(msg string) error { return &editInternalError{msg: msg} }

// editInternalError marks an unreachable-after-parse divergence; the
// caller maps it to EditUnsupported (fail closed).
type editInternalError struct{ msg string }

func (e *editInternalError) Error() string { return e.msg }

func (b *editBuilder) handleLine(raw string, lineNum, lineStart int) *Error {
	if herr := b.handleLineInner(raw, lineNum, lineStart); herr != nil {
		return newErrorf("%s", herr.Error())
	}
	return nil
}

func (b *editBuilder) handleLineInner(raw string, lineNum, lineStart int) error {
	if b.collecting != 0 {
		trimmed := trimInline(raw)
		term := ")"
		if b.collecting == 2 {
			term = "))"
		}
		if trimmed == term {
			end := editOffInLine(raw, lineStart, trimmed) + len(trimmed)
			node := editScalarNode(&editSlot{kind: slotBlock, b0: b.collectingOpen, b1: end})
			b.collecting = 0
			return b.attachScalar(node, lineNum, lineStart)
		}
		return nil
	}

	trimmed := trimInline(raw)
	if trimmed == "" || strings.HasPrefix(trimmed, "##") {
		return nil
	}
	ts := trimmedSpanIn(raw, trimmed, lineStart)

	if b.rootConsumed {
		return b.internalErr("orphan line after top-level inline")
	}

	if !b.rootInit {
		b.rootInit = true
		if trimmed != "}" && trimmed != "]" {
			kind, _, err := classifyRootKind(trimmed, lineNum, ts, false)
			if err != nil {
				return b.internalErr("root classify: " + err.Msg)
			}
			switch kind {
			case 0, 1: // closed inline compound root
				b.rootConsumed = true
				isObj := trimmed[0] == '{'
				node, ierr := positionInlineTop(trimmed, ts.start, lineNum, ts, isObj)
				if ierr != nil {
					return ierr
				}
				b.rootInline = node
				return nil
			case 2: // explicit `{` root
				b.rootIsExplicit = true
				b.stack = append(b.stack, &editFrame{object: true, table: newEditTable(), rangeStart: ts.start})
				return nil
			case 3: // explicit `[` root
				b.rootIsExplicit = true
				b.stack = append(b.stack, &editFrame{items: []*editNode{}, rangeStart: ts.start})
				return nil
			case 4: // implicit object root
				b.stack = append(b.stack, &editFrame{object: true, table: newEditTable()})
			case 5: // implicit array root
				b.stack = append(b.stack, &editFrame{items: []*editNode{}})
			}
		}
	}

	if trimmed == "}" {
		return b.closeFrame(true, lineNum, ts)
	}
	if trimmed == "]" {
		return b.closeFrame(false, lineNum, ts)
	}

	if b.top().object {
		return b.handleObjectPair(trimmed, lineNum, ts)
	}
	return b.handleArrayItem(trimmed, lineNum, ts)
}

func editOffInLine(line string, lineStart int, slice string) int {
	// The slice is always a late (value-side) occurrence within line;
	// the last match is the intended one. A mismatched range is caught
	// by the mandatory post-splice verification (fail closed).
	idx := strings.LastIndex(line, slice)
	if idx < 0 {
		idx = 0
	}
	return lineStart + idx
}

func (b *editBuilder) requireSepEnd(rest string) error {
	if rest == "" || inlineWSAt(rest, 0) > 0 {
		return nil
	}
	return b.internalErr("missing separator space")
}

func (b *editBuilder) handleObjectPair(line string, lineNum int, ts kspan) error {
	switch scanUnescapedColon(line) {
	case colonUnterminatedQuote:
		return b.internalErr("unterminated quoted key")
	case colonAbsent:
		return b.internalErr("missing separator")
	}
	colonPos := indexUnescapedColon(line)
	key := trimInline(line[:colonPos])
	keySpan := kspan{ts.start, ts.start + len(key)}
	colonOff := ts.start + colonPos
	afterColon := line[colonPos+1:]

	if len(afterColon) > 0 && afterColon[0] == ':' {
		// raw marker
		rest := afterColon[1:]
		if err := b.requireSepEnd(rest); err != nil {
			return err
		}
		body := trimInline(rest)
		node := editScalarNode(&editSlot{kind: slotPair, colon: colonOff, b0: editOffInLine(line, ts.start, body), b1: ts.end})
		return b.insertPair(key, node, keySpan)
	}
	if err := b.requireSepEnd(afterColon); err != nil {
		return err
	}
	afterTrimmed := trimLeadingInline(afterColon)
	valueStart := editOffInLine(line, ts.start, afterTrimmed)
	vs, err := classifyValueStart(afterColon, lineNum, ts, false)
	if err != nil {
		return b.internalErr("value classify: " + err.Msg)
	}
	switch vs.kind {
	case vsScalar, vsNull, vsBool, vsInt, vsFloat:
		node := editScalarNode(&editSlot{kind: slotPair, colon: colonOff, b0: valueStart, b1: ts.end})
		return b.insertPair(key, node, keySpan)
	case vsEmptyObject:
		return b.insertPair(key, editEmptyObjectNode(&editSlot{kind: slotPair, colon: colonOff, b0: valueStart, b1: ts.end}), keySpan)
	case vsEmptyArray:
		return b.insertPair(key, editEmptyArrayNode(&editSlot{kind: slotPair, colon: colonOff, b0: valueStart, b1: ts.end}), keySpan)
	case vsOpenObject, vsOpenArray:
		b.setPending(key, keySpan, colonOff)
		isObj := vs.kind == vsOpenObject
		if isObj {
			b.stack = append(b.stack, &editFrame{object: true, table: newEditTable(), rangeStart: valueStart})
		} else {
			b.stack = append(b.stack, &editFrame{items: []*editNode{}, rangeStart: valueStart})
		}
		return nil
	case vsOpenMultilineStripped:
		b.setPending(key, keySpan, colonOff)
		b.collecting = 1
		b.collectingOpen = valueStart
		return nil
	case vsOpenMultilineVerbatim:
		b.setPending(key, keySpan, colonOff)
		b.collecting = 2
		b.collectingOpen = valueStart
		return nil
	default: // vsInline
		isObj := afterTrimmed[0] == '{'
		node, ierr := positionInlineTop(afterTrimmed, valueStart, lineNum, ts, isObj)
		if ierr != nil {
			return ierr
		}
		// The container is bound to the pair's line; children stay inline.
		node.inline = false
		node.slot = &editSlot{kind: slotPair, colon: colonOff, b0: valueStart, b1: ts.end}
		return b.insertPair(key, node, keySpan)
	}
}

func (b *editBuilder) handleArrayItem(line string, lineNum int, ts kspan) error {
	lineStart := ts.start

	if rest, ok := strings.CutPrefix(line, "::"); ok {
		if err := b.requireSepEnd(rest); err != nil {
			return err
		}
		body := trimLeadingInline(rest)
		// colon carries the MARKER start for items (Rust Slot::Item
		// start): the splice's anchor zone `::`/ws lives before b0.
		node := editScalarNode(&editSlot{kind: slotItem, colon: ts.start, b0: editOffInLine(line, lineStart, body), b1: ts.end})
		return b.pushItem(node)
	}

	vs, err := classifyValueStart(line, lineNum, ts, false)
	if err != nil {
		return b.internalErr("value classify: " + err.Msg)
	}
	bodyStart := editOffInLine(line, lineStart, trimLeadingInline(line))
	switch vs.kind {
	case vsScalar, vsNull, vsBool, vsInt, vsFloat:
		return b.pushItem(editScalarNode(&editSlot{kind: slotItem, colon: ts.start, b0: bodyStart, b1: ts.end}))
	case vsEmptyObject:
		return b.pushItem(editEmptyObjectNode(&editSlot{kind: slotItem, colon: ts.start, b0: bodyStart, b1: ts.end}))
	case vsEmptyArray:
		return b.pushItem(editEmptyArrayNode(&editSlot{kind: slotItem, colon: ts.start, b0: bodyStart, b1: ts.end}))
	case vsOpenObject:
		b.stack = append(b.stack, &editFrame{object: true, table: newEditTable(), rangeStart: bodyStart})
		return nil
	case vsOpenArray:
		b.stack = append(b.stack, &editFrame{items: []*editNode{}, rangeStart: bodyStart})
		return nil
	case vsOpenMultilineStripped:
		b.collecting = 1
		b.collectingOpen = bodyStart
		return nil
	case vsOpenMultilineVerbatim:
		b.collecting = 2
		b.collectingOpen = bodyStart
		return nil
	default: // vsInline
		trimmed := trimLeadingInline(line)
		isObj := trimmed[0] == '{'
		node, ierr := positionInlineTop(trimmed, bodyStart, lineNum, ts, isObj)
		if ierr != nil {
			return ierr
		}
		node.inline = false
		node.slot = &editSlot{kind: slotItem, colon: ts.start, b0: bodyStart, b1: ts.end}
		return b.pushItem(node)
	}
}

func (b *editBuilder) setPending(key string, keySpan kspan, anchor int) {
	f := b.top()
	f.pendingKey, f.hasPendingKey = key, true
	f.pendingSpan = keySpan
	f.pendingAnchor, f.hasAnchor = anchor, true
}

func (b *editBuilder) insertPair(key string, node *editNode, keySpan kspan) error {
	f := b.top()
	if !f.object {
		return b.internalErr("pair outside object frame")
	}
	if err := editInsertValue(f.table, key, node); err != nil {
		return b.internalErr(err.Error())
	}
	return nil
}

func (b *editBuilder) pushItem(node *editNode) error {
	f := b.top()
	if f.object {
		return b.internalErr("item outside array frame")
	}
	f.items = append(f.items, node)
	return nil
}

func (b *editBuilder) attachScalar(node *editNode, lineNum, lineStart int) error {
	f := b.top()
	if f.object {
		if !f.hasPendingKey {
			return b.internalErr("multiline closed without pending key")
		}
		key, keySpan, anchor, hasAnchor := f.pendingKey, f.pendingSpan, f.pendingAnchor, f.hasAnchor
		f.hasPendingKey = false
		if node.kind == kStr && node.slot != nil && node.slot.kind == slotBlock {
			node.slot.colon = anchor
			node.slot.hasAnchor = hasAnchor
		}
		return b.insertPair(key, node, keySpan)
	}
	f.items = append(f.items, node)
	return nil
}

func (b *editBuilder) closeFrame(object bool, lineNum int, ts kspan) error {
	if len(b.stack) == 1 && b.rootIsExplicit {
		b.rootConsumed = true
		return nil
	}
	if len(b.stack) <= 1 {
		return b.internalErr("unbalanced bracket")
	}
	frame := b.stack[len(b.stack)-1]
	b.stack = b.stack[:len(b.stack)-1]
	var node *editNode
	if frame.object {
		node = &editNode{kind: kObj, table: frame.table}
	} else {
		node = &editNode{kind: kArr, items: frame.items}
	}
	var anchor int
	var hasAnchor bool
	if parent := b.top(); parent.object && parent.hasAnchor {
		anchor, hasAnchor = parent.pendingAnchor, true
		parent.hasAnchor = false
	}
	node.slot = &editSlot{kind: slotBlock, colon: anchor, hasAnchor: hasAnchor, b0: frame.rangeStart, b1: ts.end}

	parent := b.top()
	if parent.object {
		if !parent.hasPendingKey {
			return b.internalErr("closed compound without pending key")
		}
		key, keySpan := parent.pendingKey, parent.pendingSpan
		parent.hasPendingKey = false
		return b.insertPair(key, node, keySpan)
	}
	parent.items = append(parent.items, node)
	return nil
}

func (b *editBuilder) finish(eofOffset int) (*editNode, bool, error) {
	if b.collecting != 0 {
		return nil, false, b.internalErr("unclosed multiline")
	}
	if len(b.stack) > 1 || (b.rootIsExplicit && !b.rootConsumed) {
		return nil, false, b.internalErr("unclosed compound")
	}
	if b.rootInline != nil {
		return b.rootInline, b.rootIsExplicit, nil
	}
	if len(b.stack) == 0 {
		return editEmptyObjectNode(nil), b.rootIsExplicit, nil
	}
	f := b.top()
	if f.object {
		return &editNode{kind: kObj, table: f.table}, b.rootIsExplicit, nil
	}
	return &editNode{kind: kArr, items: f.items}, b.rootIsExplicit, nil
}

// buildEditTree walks the document line by line, mirroring the parser's
// dispatch decisions, and records byte ranges per value.
func buildEditTree(text string) (root *editNode, rootExplicit bool, err error) {
	b := &editBuilder{}
	if herr := splitLines(text, bomSkip(text), b.handleLine); herr != nil {
		return nil, false, fmt.Errorf("%s", herr.Msg)
	}
	root, rootExplicit, ferr := b.finish(len(text))
	if ferr != nil {
		return nil, false, ferr
	}
	return root, rootExplicit, nil
}

// ─── inline positioning ───────────────────────────────────────────────

// positionInlineTop positions an inline-compound body (`{...}`/`[…]`)
// whose first byte sits at baseAbs in the source.
func positionInlineTop(body string, baseAbs, lineNum int, span kspan, isObject bool) (*editNode, error) {
	return positionInline(body, baseAbs, lineNum, span, isObject, 0)
}

func positionInline(body string, baseAbs, lineNum int, span kspan, isObject bool, depth int) (*editNode, error) {
	if depth > maxInlineDepth {
		return nil, fmt.Errorf("inline depth exceeded")
	}
	inner := body[1 : len(body)-1]
	segments, serr := splitTopLevel(inner, lineNum, span, isObject)
	if serr != nil {
		return nil, fmt.Errorf("inline split failed")
	}
	if isObject {
		table := newEditTable()
		for _, seg := range segments {
			trimmed := trimInline(seg)
			if trimmed == "" {
				continue // trailing comma
			}
			segBase := editOffInLine(body, baseAbs, trimmed)
			segEnd := segBase + len(trimmed)
			colon := indexUnescapedColon(trimmed)
			if colon < 0 {
				return nil, fmt.Errorf("inline pair without separator")
			}
			colonAbs := segBase + colon
			key := trimmed[:colon]
			after := trimmed[colon+1:]
			var node *editNode
			if rest, isRaw := strings.CutPrefix(after, ":"); isRaw {
				v := trimLeadingInline(rest)
				node = &editNode{kind: kStr, inline: true, slot: &editSlot{
					kind: slotPair, colon: colonAbs,
					b0: editOffInLine(trimmed, segBase, v), b1: segEnd,
				}}
			} else {
				v := trimLeadingInline(after)
				valueAbs := editOffInLine(trimmed, segBase, v)
				var ierr error
				node, ierr = positionInlineChild(v, valueAbs, lineNum, span, depth)
				if ierr != nil {
					return nil, ierr
				}
				node.slot = &editSlot{kind: slotPair, colon: colonAbs, b0: valueAbs, b1: segEnd}
			}
			rawKey := key
			// Inline keys go through the SAME dotted expansion and
			// decoding as the parser's insertValue, so the edit tree's
			// shape matches the parsed tree (`{b.c: old}` nests b→c
			// with the slot on the leaf, exactly like the reference).
			if ierr := editInsertValue(table, rawKey, node); ierr != nil {
				return nil, fmt.Errorf("inline insert: %v", ierr)
			}
		}
		return &editNode{kind: kObj, table: table, inline: true}, nil
	}
	items := []*editNode{}
	for _, seg := range segments {
		trimmed := trimInline(seg)
		if trimmed == "" {
			continue
		}
		segBase := editOffInLine(body, baseAbs, trimmed)
		segEnd := segBase + len(trimmed)
		v := trimLeadingInline(trimmed)
		valueAbs := editOffInLine(trimmed, segBase, v)
		node, ierr := positionInlineChild(v, valueAbs, lineNum, span, depth)
		if ierr != nil {
			return nil, ierr
		}
		node.slot = &editSlot{kind: slotItem, colon: segBase, b0: valueAbs, b1: segEnd}
		items = append(items, node)
	}
	return &editNode{kind: kArr, items: items, inline: true}, nil
}

func positionInlineChild(v string, vAbs, lineNum int, span kspan, depth int) (*editNode, error) {
	if len(v) == 0 {
		return &editNode{kind: kStr, inline: true}, nil
	}
	switch v[0] {
	case '{':
		return positionInline(v, vAbs, lineNum, span, true, depth+1)
	case '[':
		return positionInline(v, vAbs, lineNum, span, false, depth+1)
	default:
		return &editNode{kind: kStr, inline: true}, nil
	}
}

// ─── pvalue helpers ───────────────────────────────────────────────────

// pvEqual compares two plain trees structurally.
func pvEqual(a, b *pvalue) bool {
	if a.kind != b.kind || a.b != b.b || a.s != b.s {
		return false
	}
	switch a.kind {
	case kArr:
		if len(a.arr) != len(b.arr) {
			return false
		}
		for i := range a.arr {
			if !pvEqual(a.arr[i], b.arr[i]) {
				return false
			}
		}
		return true
	case kObj:
		// Order-independent like the reference IndexMap equality: a
		// host-provided replacement map may list keys in a different
		// order than the source.
		if len(a.obj.entries) != len(b.obj.entries) {
			return false
		}
		for _, e := range a.obj.entries {
			o := b.obj.get(e.key)
			if o == nil || !pvEqual(e.val, o) {
				return false
			}
		}
		return true
	default:
		return true
	}
}

// editLookupValue walks the parsed original to the target (no-op check).
func editLookupValue(v *pvalue, segments []string) *pvalue {
	cur := v
	for _, seg := range segments {
		switch cur.kind {
		case kObj:
			cur = cur.obj.get(seg)
		case kArr:
			idx, err := parseEditIndex(seg)
			if err != nil {
				return nil
			}
			if idx < 0 || idx >= len(cur.arr) {
				return nil
			}
			cur = cur.arr[idx]
		default:
			return nil
		}
		if cur == nil {
			return nil
		}
	}
	return cur
}

// editTakeTarget removes and returns the target from a temporary tree
// (replaced with null) for the post-splice semantic verification.
func editTakeTarget(v *pvalue, segments []string) (*pvalue, *Error) {
	if len(segments) == 0 {
		return nil, editNotFound()
	}
	cur := v
	for _, seg := range segments[:len(segments)-1] {
		switch cur.kind {
		case kObj:
			cur = cur.obj.get(seg)
		case kArr:
			idx, err := parseEditIndex(seg)
			if err != nil {
				return nil, err
			}
			if idx < 0 || idx >= len(cur.arr) {
				return nil, editNotFound()
			}
			cur = cur.arr[idx]
		default:
			return nil, editNotFound()
		}
		if cur == nil {
			return nil, editNotFound()
		}
	}
	last := segments[len(segments)-1]
	switch cur.kind {
	case kObj:
		e := cur.obj.get(last)
		if e == nil {
			return nil, editNotFound()
		}
		cur.obj.set(last, pvNullValue())
		return e, nil
	case kArr:
		idx, err := parseEditIndex(last)
		if err != nil {
			return nil, err
		}
		if idx < 0 || idx >= len(cur.arr) {
			return nil, editNotFound()
		}
		old := cur.arr[idx]
		cur.arr[idx] = pvNullValue()
		return old, nil
	default:
		return nil, editNotFound()
	}
}

func pvNullValue() *pvalue { return &pvalue{kind: kNull} }

func (m *omap) get(key string) *pvalue {
	if i, ok := m.index[key]; ok {
		return m.entries[i].val
	}
	return nil
}

func (m *omap) set(key string, v *pvalue) {
	if i, ok := m.index[key]; ok {
		m.entries[i].val = v
	}
}
