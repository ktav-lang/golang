package engine

// The line-dispatch state machine shared by Loads/CanonicalFromSource
// (plain mode) and FormatSource (trivia mode): a stack of frames plus
// § 5.0.1 root detection, § 5.1 dispatch, and the § 5.3.2 dotted-key
// insertion tables. Compounds nest through the explicit frame stack —
// no recursion at this level (inline compounds recurse in pure_parse.go
// under the declared 128 cap).

// ─── trivia-tree tables ────────────────────────────────────────────────

// scanProbe is the package-internal performance-instrumentation hook:
// NIL by default, so production cost is a single predictable nil-test
// per coarse event (no locks, no atomics, no global mutable state
// beyond this var). Tests install a counting closure and assert bounds
// on REAL operation counts (scan/descent/write work), never wall-clock.
// See pure_perf_test.go and the plan's complexity notes.
var scanProbe func(kind string, n int)

// probe reports one instrumentation event when the hook is installed.
func probe(kind string, n int) {
	if scanProbe != nil {
		scanProbe(kind, n)
	}
}

// kvTable is the shared insertion-table interface over the two tree
// flavours: plainTable (plain pvalue/omap) and fvalueTable (trivia
// fvalue). Both use value receivers, so both value types satisfy it.
type kvTable interface {
	insertLeaf(key string, val any, trivia []tline) (any, bool)
	descend(key string) (kvTable, bool)
	isObj(val any) bool
	label(val any) string
}

// A compound *fvalue doubles as the insertion table for the trivia
// parser: pairs live in comp.pairs (insertion order) with a lazily
// built hash index for O(1) duplicate detection and O(1) dotted-prefix
// descent.

type fvalueTable struct{ f *fvalue }

func (t fvalueTable) ensureIndex() {
	if t.f.idx == nil {
		t.f.idx = make(map[string]int32, len(t.f.comp.pairs))
		for i, pr := range t.f.comp.pairs {
			t.f.idx[pr.key] = int32(i)
		}
	}
}

func (t fvalueTable) insertLeaf(key string, val any, trivia []tline) (any, bool) {
	t.ensureIndex()
	if i, ok := t.f.idx[key]; ok {
		return t.f.comp.pairs[i].val, true
	}
	t.f.idx[key] = int32(len(t.f.comp.pairs))
	t.f.comp.pairs = append(t.f.comp.pairs, fpair{key: key, trivia: trivia, val: val.(*fvalue)})
	return nil, false
}

func (t fvalueTable) descend(key string) (kvTable, bool) {
	t.ensureIndex()
	if i, ok := t.f.idx[key]; ok {
		child := t.f.comp.pairs[i].val
		if child.v.kind == kObj {
			return fvalueTable{child}, false
		}
		return nil, true
	}
	child := &fvalue{v: pvObj(), comp: &fcompound{}}
	t.f.idx[key] = int32(len(t.f.comp.pairs))
	t.f.comp.pairs = append(t.f.comp.pairs, fpair{key: key, val: child})
	return fvalueTable{child}, false
}

func (t fvalueTable) isObj(val any) bool   { return val.(*fvalue).isObject() }
func (t fvalueTable) label(val any) string { return val.(*fvalue).kindLabel() }

type plainTable struct{ m *omap }

func (t plainTable) insertLeaf(key string, val any, _ []tline) (any, bool) {
	return t.m.insertLeaf(key, val.(*pvalue))
}
func (t plainTable) descend(key string) (kvTable, bool) {
	child, blocked := t.m.descend(key)
	if blocked {
		return nil, true
	}
	return plainTable{child}, false
}
func (t plainTable) isObj(val any) bool   { return val.(*pvalue).isObject() }
func (t plainTable) label(val any) string { return val.(*pvalue).kindLabel() }

// insertValueAny inserts at a (possibly dotted) path into either table
// kind, validating each raw segment before decoding (§ 5.3.1) and
// applying the § 5.5 / § 6.3 outcome tables.
func insertValueAny(table kvTable, path string, value any, trivia []tline, line int, span kspan) *Error {
	if keyIsSingleSegment(path) {
		trimmed := trimInline(path)
		if trimmed == "" {
			return errEmptyKey(line, span)
		}
		if st := checkKey(trimmed); st != keyValid {
			if st == keyEmpty {
				return errEmptyKey(line, span)
			}
			return errInvalidKey(line, path, span)
		}
		decoded, bad := decodeKeySegment(trimmed)
		if bad != "" {
			return errBadEscapeSequence(line, span, bad)
		}
		existing, occupied := table.insertLeaf(decoded, value, trivia)
		if !occupied {
			return nil
		}
		if table.isObj(existing) != table.isObj(value) {
			return errKeyPathConflictOverwrite(line, path, table.label(existing), table.label(value), span)
		}
		return errDuplicateKey(line, path, span)
	}
	segments := splitKeyPath(path)
	probe("dotted_path", len(segments))
	cur := table
	for idx, seg := range segments {
		trimmed := trimInline(seg)
		if trimmed == "" {
			return errEmptyKey(line, span)
		}
		if st := checkKey(trimmed); st != keyValid {
			if st == keyEmpty {
				return errEmptyKey(line, span)
			}
			return errInvalidKey(line, path, span)
		}
		decoded, bad := decodeKeySegment(trimmed)
		if bad != "" {
			return errBadEscapeSequence(line, span, bad)
		}
		if idx+1 == len(segments) {
			// The line's pending trivia attaches to the TERMINAL pair
			// (a comment owns the whole line, §3.4, so it documents the
			// pair this line defines — even when reached through a
			// dotted path). descend() never copies trivia: reopening an
			// existing prefix (`a.c: 2` after `a.b: 1`) finds the
			// synthetic object slot and leaves its stored trivia
			// untouched, so a comment is neither duplicated nor lost.
			if _, occupied := cur.insertLeaf(decoded, value, trivia); occupied {
				return errDuplicateKey(line, path, span)
			}
			return nil
		}
		next, blocked := cur.descend(decoded)
		if blocked {
			return errKeyPathConflictBlocked(line, path, span)
		}
		probe("dotted_segment", 1)
		cur = next
	}
	return nil
}

// ─── the parser (plain + trivia modes) ────────────────────────────────

type pframe struct {
	object           bool
	plain            *omap      // plain mode, object frame
	plainArr         []*pvalue  // plain mode, array frame
	fmtObj           *fvalue    // trivia mode, object frame (table root)
	fmtArr           []fitem    // trivia mode, array frame items
	fmtComp          *fcompound // trivia mode: trailing trivia bucket
	pendingKey       string
	hasPendingKey    bool
	pendingKeySpan   kspan
	pendingKeyTrivia []tline
	pendingItemTriv  []tline
	atStart          bool
}

type multilineCollector struct {
	verbatim bool
	lines    []string
}

type parser struct {
	src                    string
	strict                 bool
	trivia                 bool
	stack                  []*pframe
	collecting             *multilineCollector
	openerOffsets          []int
	multilineOpener        int
	hasMultilineOpener     bool
	rootInitialized        bool
	rootConsumed           bool
	rootInlinePlain        *pvalue
	rootInlineFmt          *fvalue
	rootIsExplicitCompound bool
	pendingTrivia          []tline
	docLeading             []tline
}

func newParser(src string, strict, trivia bool) *parser {
	return &parser{src: src, strict: strict, trivia: trivia, multilineOpener: -1}
}

func (p *parser) newFrame(object bool) *pframe {
	f := &pframe{object: object, atStart: true}
	if p.trivia {
		f.fmtComp = &fcompound{}
		if object {
			f.fmtObj = &fvalue{v: pvObj(), comp: f.fmtComp}
		}
	} else if object {
		f.plain = newOmap()
	}
	return f
}

func (p *parser) top() *pframe { return p.stack[len(p.stack)-1] }

// makeValue converts a classify result into the mode's value type.
func (p *parser) makeValue(vs valueStart) any {
	var v *pvalue
	switch vs.kind {
	case vsNull:
		v = &pvalue{kind: kNull}
	case vsBool:
		v = pvBool(vs.b)
	case vsInt:
		v = pvInt(vs.s)
	case vsFloat:
		v = pvFloat(vs.s)
	case vsEmptyObject:
		v = pvObj()
	case vsEmptyArray:
		v = pvArr()
	default: // vsScalar, vsInline
		v = vs.inl
		if v == nil {
			v = pvStr(vs.s)
		}
	}
	if p.trivia {
		return stamp(v)
	}
	return v
}

// finish closes the document: unclosed-compound detection (§ 6.1) and
// end-of-document trivia attachment.
func (p *parser) finish(eofOffset int) (any, *Error) {
	if p.collecting != nil {
		start := eofOffset
		if p.hasMultilineOpener {
			start = p.multilineOpener
		}
		return nil, errUnclosedCompound("multiline", kspan{start, eofOffset})
	}
	if len(p.stack) > 1 || (p.rootIsExplicitCompound && !p.rootConsumed) {
		kind := "object"
		if !p.top().object {
			kind = "array"
		}
		start := p.openerOffsets[len(p.openerOffsets)-1]
		return nil, errUnclosedCompound(kind, kspan{start, eofOffset})
	}
	if p.trivia {
		eofTrivia := trimTrailingBlanks(p.pendingTrivia)
		p.pendingTrivia = nil
		if p.rootInlineFmt != nil {
			p.rootInlineFmt.comp.trailing = append(p.rootInlineFmt.comp.trailing, eofTrivia...)
			return p.rootInlineFmt, nil
		}
		if len(p.stack) == 0 {
			p.docLeading = append(p.docLeading, eofTrivia...)
			return &fvalue{v: pvObj(), comp: &fcompound{}}, nil
		}
		root := p.popFmtRoot()
		root.comp.trailing = append(root.comp.trailing, eofTrivia...)
		return root, nil
	}
	if p.rootInlinePlain != nil {
		return p.rootInlinePlain, nil
	}
	if len(p.stack) == 0 {
		return pvObj(), nil
	}
	f := p.top()
	if f.object {
		return &pvalue{kind: kObj, obj: f.plain}, nil
	}
	return &pvalue{kind: kArr, arr: f.plainArr}, nil
}

func (p *parser) popFmtRoot() *fvalue {
	f := p.top()
	if f.object {
		return f.fmtObj
	}
	return &fvalue{v: &pvalue{kind: kArr}, comp: &fcompound{items: f.fmtArr, trailing: f.fmtComp.trailing}}
}

// handleLine dispatches one source line (pre-split on LF/CR/CRLF).
func (p *parser) handleLine(raw string, lineNum int, lineStart int) *Error {
	if p.collecting != nil {
		trimmed := trimInline(raw)
		term := ")"
		if p.collecting.verbatim {
			term = "))"
		}
		if trimmed == term {
			text := multilineFinish(p.collecting.verbatim, p.collecting.lines)
			p.collecting = nil
			p.hasMultilineOpener = false
			return p.attachScalar(pvStr(text), lineNum, lineStart)
		}
		p.collecting.lines = append(p.collecting.lines, raw)
		return nil
	}

	trimmed := trimInline(raw)
	if trimmed == "" || (len(trimmed) >= 2 && trimmed[:2] == "##") {
		if p.trivia {
			if trimmed == "" {
				p.pushBlankTrivia()
			} else {
				p.pendingTrivia = append(p.pendingTrivia, tline{comment: trimmed})
			}
		}
		return nil
	}

	trimmedSpan := trimmedSpanIn(raw, trimmed, lineStart)

	if p.rootConsumed {
		return errOrphanLine(lineNum, trimmedSpan)
	}

	if !p.rootInitialized {
		p.rootInitialized = true
		if trimmed != "}" && trimmed != "]" {
			kind, inl, err := classifyRootKind(trimmed, lineNum, trimmedSpan, p.strict)
			if err != nil {
				return err
			}
			switch kind {
			case 0, 1: // closed inline compound root
				p.rootConsumed = true
				if p.trivia {
					p.docLeading = p.pendingTrivia
					p.pendingTrivia = nil
					p.rootInlineFmt = stamp(inl)
				} else {
					p.rootInlinePlain = inl
				}
				return nil
			case 2, 3: // lone `{` / `[` explicit root
				if p.trivia {
					p.docLeading = p.pendingTrivia
					p.pendingTrivia = nil
				}
				p.rootIsExplicitCompound = true
				p.stack = append(p.stack, p.newFrame(kind == 2))
				p.openerOffsets = append(p.openerOffsets, trimmedSpan.start)
				return nil
			case 4, 5: // implicit object / array root
				p.stack = append(p.stack, p.newFrame(kind == 4))
				p.openerOffsets = append(p.openerOffsets, 0)
			}
		}
	}

	if trimmed == "}" {
		return p.closeFrame('}', lineNum, trimmedSpan)
	}
	if trimmed == "]" {
		return p.closeFrame(']', lineNum, trimmedSpan)
	}

	var trivia []tline
	if p.trivia {
		trivia = p.pendingTrivia
		p.pendingTrivia = nil
	}
	p.top().atStart = false
	if p.top().object {
		return p.handleObjectPair(trimmed, lineNum, trimmedSpan, trivia)
	}
	return p.handleArrayItem(trimmed, lineNum, trimmedSpan, trivia)
}

func (p *parser) pushBlankTrivia() {
	if len(p.pendingTrivia) == 0 {
		if len(p.stack) == 0 && !p.rootInitialized {
			return // leading padding before any content: dropped
		}
		if len(p.stack) > 0 && p.top().atStart {
			return // blank padding at a compound's start: dropped
		}
	}
	if len(p.pendingTrivia) > 0 && p.pendingTrivia[len(p.pendingTrivia)-1].blank {
		return // a run of two or more collapses to exactly one
	}
	p.pendingTrivia = append(p.pendingTrivia, tline{blank: true})
}

func trimmedSpanIn(raw, trimmed string, lineStart int) kspan {
	if trimmed == "" {
		return kspan{lineStart, lineStart}
	}
	start := 0
	for start < len(raw) && inlineWSAt(raw, start) > 0 {
		start += inlineWSAt(raw, start)
	}
	return kspan{lineStart + start, lineStart + start + len(trimmed)}
}

func (p *parser) requireSepEnd(rest string, line int, bodyOff int, trimmedSpan kspan) *Error {
	if rest == "" || inlineWSAt(rest, 0) > 0 {
		return nil
	}
	return errMissingSeparatorSpace(line, kspan{bodyOff, trimmedSpan.end})
}

func (p *parser) attachScalar(v *pvalue, lineNum int, lineStart int) *Error {
	f := p.top()
	if f.object {
		if !f.hasPendingKey {
			return structErr("Other", lineNum, kspan{lineStart, lineStart}, "",
				"internal error — multi-line string closed without pending key", "", "", nil)
		}
		key, keySpan, trivia := f.pendingKey, f.pendingKeySpan, f.pendingKeyTrivia
		f.hasPendingKey = false
		f.pendingKeyTrivia = nil
		return p.insertPair(key, v, lineNum, keySpan, trivia)
	}
	if p.trivia {
		f.fmtArr = append(f.fmtArr, fitem{trivia: f.pendingItemTriv, val: stamp(v)})
		f.pendingItemTriv = nil
	} else {
		f.plainArr = append(f.plainArr, v)
	}
	return nil
}

func (p *parser) insertPair(key string, v *pvalue, line int, keySpan kspan, trivia []tline) *Error {
	if p.trivia {
		return insertValueAny(fvalueTable{p.top().fmtObj}, key, stamp(v), trivia, line, keySpan)
	}
	return insertValueAny(plainTable{p.top().plain}, key, v, trivia, line, keySpan)
}

func (p *parser) insertAny(key string, v any, trivia []tline, line int, keySpan kspan) *Error {
	if p.trivia {
		return insertValueAny(fvalueTable{p.top().fmtObj}, key, v, trivia, line, keySpan)
	}
	return insertValueAny(plainTable{p.top().plain}, key, v, trivia, line, keySpan)
}

func (p *parser) handleObjectPair(trimmed string, lineNum int, trimmedSpan kspan, trivia []tline) *Error {
	switch scanUnescapedColon(trimmed) {
	case colonUnterminatedQuote:
		return errUnterminatedQuotedKey(lineNum, trimmedSpan)
	case colonAbsent:
		return errMissingSeparator(lineNum, trimmedSpan)
	}
	colonPos := indexUnescapedColon(trimmed)
	key := trimTrailingInline(trimmed[:colonPos])
	keyStart := trimmedSpan.start
	keyEnd := keyStart + len(key)
	if key == "" {
		return errEmptyKey(lineNum, kspan{keyStart, keyStart + 1})
	}

	afterColon := trimmed[colonPos+1:]
	afterColonOff := trimmedSpan.start + colonPos + 1
	keySpan := kspan{keyStart, keyEnd}

	isRaw := len(afterColon) > 0 && afterColon[0] == ':'
	if isRaw {
		rest := afterColon[1:]
		if err := p.requireSepEnd(rest, lineNum, afterColonOff+1, trimmedSpan); err != nil {
			return err
		}
		v := pvStr(trimInline(rest))
		return p.insertPair(key, v, lineNum, keySpan, trivia)
	}
	if err := p.requireSepEnd(afterColon, lineNum, afterColonOff, trimmedSpan); err != nil {
		return err
	}
	vs, err := classifyValueStart(afterColon, lineNum, trimmedSpan, p.strict)
	if err != nil {
		return err
	}
	switch vs.kind {
	case vsOpenObject, vsOpenArray:
		f := p.top()
		f.pendingKey, f.hasPendingKey, f.pendingKeySpan = key, true, keySpan
		if p.trivia {
			f.pendingKeyTrivia = trivia
		}
		p.stack = append(p.stack, p.newFrame(vs.kind == vsOpenObject))
		p.openerOffsets = append(p.openerOffsets, trimmedSpan.end-1)
		return nil
	case vsOpenMultilineStripped, vsOpenMultilineVerbatim:
		f := p.top()
		f.pendingKey, f.hasPendingKey, f.pendingKeySpan = key, true, keySpan
		if p.trivia {
			f.pendingKeyTrivia = trivia
		}
		p.collecting = &multilineCollector{verbatim: vs.kind == vsOpenMultilineVerbatim}
		if vs.kind == vsOpenMultilineVerbatim {
			p.multilineOpener = trimmedSpan.end - 2
		} else {
			p.multilineOpener = trimmedSpan.end - 1
		}
		p.hasMultilineOpener = true
		return nil
	}
	return p.insertAny(key, p.makeValue(vs), trivia, lineNum, keySpan)
}

func (p *parser) handleArrayItem(trimmed string, lineNum int, trimmedSpan kspan, trivia []tline) *Error {
	if len(trimmed) >= 2 && trimmed[:2] == "::" {
		rest := trimmed[2:]
		if err := p.requireSepEnd(rest, lineNum, trimmedSpan.start+2, trimmedSpan); err != nil {
			return err
		}
		return p.pushItem(pvStr(trimLeadingInline(rest)), trivia)
	}
	vs, err := classifyValueStart(trimmed, lineNum, trimmedSpan, p.strict)
	if err != nil {
		return err
	}
	switch vs.kind {
	case vsOpenObject, vsOpenArray:
		if p.trivia {
			p.top().pendingItemTriv = trivia
		}
		p.stack = append(p.stack, p.newFrame(vs.kind == vsOpenObject))
		p.openerOffsets = append(p.openerOffsets, trimmedSpan.end-1)
		return nil
	case vsOpenMultilineStripped, vsOpenMultilineVerbatim:
		if p.trivia {
			p.top().pendingItemTriv = trivia
		}
		p.collecting = &multilineCollector{verbatim: vs.kind == vsOpenMultilineVerbatim}
		if vs.kind == vsOpenMultilineVerbatim {
			p.multilineOpener = trimmedSpan.end - 2
		} else {
			p.multilineOpener = trimmedSpan.end - 1
		}
		p.hasMultilineOpener = true
		return nil
	}
	if p.trivia {
		return p.pushItemF(p.makeValue(vs).(*fvalue), trivia)
	}
	return p.pushItem(p.makeValue(vs).(*pvalue), trivia)
}

func trimLeadingInline(s string) string {
	i := 0
	for i < len(s) {
		n := inlineWSAt(s, i)
		if n == 0 {
			break
		}
		i += n
	}
	return s[i:]
}

func (p *parser) pushItem(v *pvalue, trivia []tline) *Error {
	f := p.top()
	if p.trivia {
		f.fmtArr = append(f.fmtArr, fitem{trivia: trivia, val: stamp(v)})
	} else {
		f.plainArr = append(f.plainArr, v)
	}
	return nil
}

func (p *parser) pushItemF(v *fvalue, trivia []tline) *Error {
	f := p.top()
	f.fmtArr = append(f.fmtArr, fitem{trivia: trivia, val: v})
	return nil
}

func (p *parser) closeFrame(expected byte, lineNum int, trimmedSpan kspan) *Error {
	var trailing []tline
	if p.trivia {
		trailing = trimTrailingBlanks(p.pendingTrivia)
		p.pendingTrivia = nil
	}
	if len(p.stack) <= 1 {
		if len(p.stack) == 1 && p.rootIsExplicitCompound {
			f := p.top()
			if f.object != (expected == '}') {
				return errUnbalancedBracket(lineNum, trimmedSpan, p.frameOpener(f), expected)
			}
			if p.trivia {
				f.fmtComp.trailing = append(f.fmtComp.trailing, trailing...)
			}
			p.rootConsumed = true
			return nil
		}
		return errUnbalancedBracket(lineNum, trimmedSpan, expected, expected)
	}
	f := p.stack[len(p.stack)-1]
	p.stack = p.stack[:len(p.stack)-1]
	p.openerOffsets = p.openerOffsets[:len(p.openerOffsets)-1]
	if f.object != (expected == '}') {
		return errUnbalancedBracket(lineNum, trimmedSpan, p.frameOpener(f), expected)
	}
	if p.trivia {
		f.fmtComp.trailing = append(f.fmtComp.trailing, trailing...)
	}
	var v any
	if p.trivia {
		if f.object {
			v = f.fmtObj
		} else {
			v = &fvalue{v: &pvalue{kind: kArr},
				comp: &fcompound{items: f.fmtArr, trailing: f.fmtComp.trailing}}
		}
	} else {
		if f.object {
			v = &pvalue{kind: kObj, obj: f.plain}
		} else {
			v = &pvalue{kind: kArr, arr: f.plainArr}
		}
	}
	return p.attachChild(v, lineNum, trimmedSpan)
}

func (p *parser) frameOpener(f *pframe) byte {
	if f.object {
		return '{'
	}
	return '['
}

func (p *parser) attachChild(v any, lineNum int, trimmedSpan kspan) *Error {
	f := p.top()
	if f.object {
		if !f.hasPendingKey {
			return structErr("Other", lineNum, trimmedSpan, "",
				"internal error — closed compound without pending key", "", "", nil)
		}
		key, keySpan, trivia := f.pendingKey, f.pendingKeySpan, f.pendingKeyTrivia
		f.hasPendingKey = false
		f.pendingKeyTrivia = nil
		return p.insertAny(key, v, trivia, lineNum, keySpan)
	}
	if p.trivia {
		f.fmtArr = append(f.fmtArr, fitem{trivia: f.pendingItemTriv, val: v.(*fvalue)})
		f.pendingItemTriv = nil
	} else {
		f.plainArr = append(f.plainArr, v.(*pvalue))
	}
	return nil
}
