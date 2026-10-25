package engine

// Splice engine (the Rust edit/splice.rs port): plan and apply the
// replacement of exactly the target's byte range, reusing the source's
// marker zone and whitespace, then verify the patched document
// semantically (fail closed on any divergence).

import (
	"strings"
	"unicode/utf8"
)

type targetCtx struct {
	inline    bool
	rootFirst bool
}

type planCtx struct {
	slot           int // slotPair / slotItem / slotBlock
	roleItemBlock  bool
	existingRaw    bool
	rootFirst      bool
	indent         string
	eol            string
	inlineCompound bool
	blockCompound  bool
}

// spliceBody is the insertion body: borrowed source-scoped text or an
// owned rendered string.
type spliceBody struct {
	borrowed string
	owned    string
	isOwned  bool
}

func bodyBorrowed(s string) spliceBody { return spliceBody{borrowed: s} }
func bodyOwned(s string) spliceBody    { return spliceBody{owned: s, isOwned: true} }
func (b spliceBody) str() string {
	if b.isOwned {
		return b.owned
	}
	return b.borrowed
}
func (b spliceBody) len() int { return len(b.str()) }

type splicePlan struct {
	from   int
	prefix [2]string
	body   spliceBody
}

// applyEdit is the splice engine proper (the Rust splice::apply port).
func applyEdit(text string, original *pvalue, segments []string, repl *pvalue) (string, *Error) {
	target := editLookupValue(original, segments)
	if target == nil {
		return "", editNotFound()
	}
	if pvEqual(target, repl) {
		return text, nil // byte-identical no-op, even noncanonical spelling
	}
	root, rootExplicit, terr := buildEditTree(text)
	if terr != nil {
		return "", editUnsupported("source text does not parse: " + terr.Error())
	}

	node, ctx, derr := editDescend(root, segments, rootExplicit)
	if derr != nil {
		return "", derr
	}
	if node.slot == nil {
		return "", editUnsupported(
			"target is a noncontiguous value (dotted-key/reopen intermediate); " +
				"no lossless single-range splice exists")
	}
	slot := node.slot
	slotKind := slot.kind

	var bodyRange [2]int
	var anchor int
	var blockItemRole bool
	switch slotKind {
	case slotPair:
		bodyRange = [2]int{slot.b0, slot.b1}
		anchor = slot.colon
	case slotItem:
		bodyRange = [2]int{slot.b0, slot.b1}
		anchor = slot.b0
	default: // slotBlock
		bodyRange = [2]int{slot.b0, slot.b1}
		if slot.hasAnchor {
			anchor = slot.colon
		} else {
			anchor = slot.b0
			blockItemRole = true
		}
	}
	b0, b1 := bodyRange[0], bodyRange[1]
	a := anchor
	if !(0 <= a && a <= b1 && b1 <= len(text)) || b0 > b1 {
		return "", editUnsupported("internal: invalid marker/body offsets")
	}

	var existingRaw bool
	var wsRaw, wsPlain string
	inlineItem := ctx.inline && slotKind != slotBlock && slotKind == slotItem
	switch {
	case slotKind == slotBlock:
		// no marker zone
	case inlineItem:
		// physically inline array item: `::` is literal data
	default:
		if a+1 < len(text) && text[a+1] == ':' {
			existingRaw = true
			if a+2 > b0 {
				return "", editUnsupported("internal: invalid raw marker range")
			}
			wsRaw = text[a+2 : b0]
		}
		if slotKind == slotPair {
			if a+1 > b0 {
				return "", editUnsupported("internal: invalid plain marker range")
			}
			wsPlain = text[a+1 : b0]
		}
	}

	// Structural rewrite of a container: losing interior comments is a
	// typed refusal BEFORE the splice. Multiline scalar blocks are data.
	if node.kind == kObj || node.kind == kArr {
		ranges := collectMultilineScalarRanges(node)
		if hasCommentLine(text, b0, b1, ranges) {
			return "", editUnsupported(
				"target container contains comment lines; structural " +
					"replacement cannot preserve them losslessly")
		}
	}

	indent := editLineIndent(text, a)
	eol := detectEOL(text, b1)
	pc := planCtx{
		slot:           slotKind,
		roleItemBlock:  slotKind == slotBlock && blockItemRole,
		existingRaw:    existingRaw,
		rootFirst:      ctx.rootFirst,
		indent:         indent,
		eol:            eol,
		inlineCompound: slotKind != slotBlock && (node.kind == kObj || node.kind == kArr),
		blockCompound:  slotKind == slotBlock && (node.kind == kObj || node.kind == kArr),
	}
	wsPairBlock := ""
	if slotKind == slotBlock && !blockItemRole {
		if a+1 > b0 {
			return "", editUnsupported("internal: invalid block pair marker range")
		}
		wsPairBlock = text[a+1 : b0]
	}

	var plan splicePlan
	var perr *Error
	if ctx.inline {
		plan, perr = planInlineSlot(slotKind, existingRaw, repl, a, b0, wsRaw)
	} else if slotKind == slotBlock {
		plan, perr = blockReplacement(repl, pc, a, b0, wsPairBlock)
	} else {
		plan, perr = planLinearSlot(repl, pc, a, b0, b1, wsRaw, wsPlain)
	}
	if perr != nil {
		return "", perr
	}
	from := plan.from
	if !(0 <= from && from <= b1 && b1 <= len(text)) {
		return "", editUnsupported("internal: splice range out of bounds")
	}
	insertionLen := plan.body.len() + len(plan.prefix[0]) + len(plan.prefix[1])
	var result strings.Builder
	result.Grow(len(text) - (b1 - from) + insertionLen)
	result.WriteString(text[:from])
	result.WriteString(plan.prefix[0])
	result.WriteString(plan.prefix[1])
	result.WriteString(plan.body.str())
	result.WriteString(text[b1:])
	candidate := result.String()

	adjusted, lerr := layoutAdjust(text, candidate, root, segments, repl, from, b1, insertionLen)
	if lerr != nil {
		return "", lerr
	}

	// Mandatory post-patch verification: re-parse and compare trees with
	// null slots (exactly the target replaced).
	newDoc, _, vperr := parseDoc(adjusted, false, false)
	if vperr != nil {
		return "", editUnsupported("patch produced an invalid document: " + vperr.Msg)
	}
	newTree := newDoc.(*pvalue)
	actual, takeErr := editTakeTarget(newTree, segments)
	if takeErr != nil {
		return "", editUnsupported("semantic mismatch after splice: patched target not addressable")
	}
	targetMatches := pvEqual(actual, repl)
	expected := original
	if _, terr := editTakeTarget(expected, segments); terr != nil {
		return "", editUnsupported("semantic mismatch after splice: source target not addressable")
	}
	if !targetMatches {
		return "", editUnsupported("semantic mismatch after splice: patched target differs from replacement")
	}
	if !pvEqual(newTree, expected) {
		return "", editUnsupported("semantic mismatch after splice: rest of the tree changed")
	}
	return adjusted, nil
}

// editDescend walks the positional tree by decoded segments.
func editDescend(root *editNode, segments []string, rootExplicit bool) (*editNode, targetCtx, *Error) {
	containerIsRoot := true
	cur := root
	rootFirst := false
	for _, seg := range segments {
		switch {
		case cur.isObject():
			rootFirst = false
			cur = cur.child(seg)
		case cur.isArray():
			idx, err := parseEditIndex(seg)
			if err != nil {
				return nil, targetCtx{}, err
			}
			rootFirst = containerIsRoot && !rootExplicit && idx == 0
			cur = cur.item(idx)
		default:
			return nil, targetCtx{}, editNotFound()
		}
		if cur == nil {
			return nil, targetCtx{}, editNotFound()
		}
		containerIsRoot = false
	}
	return cur, targetCtx{inline: cur.inline, rootFirst: rootFirst}, nil
}

func collectMultilineScalarRanges(node *editNode) [][2]int {
	var out [][2]int
	var walk func(n *editNode)
	walk = func(n *editNode) {
		if n.kind == kStr && n.slot != nil && n.slot.kind == slotBlock {
			out = append(out, [2]int{n.slot.b0, n.slot.b1})
			return
		}
		if n.kind == kObj {
			for _, e := range n.table.entries {
				walk(e.node)
			}
		} else if n.kind == kArr {
			for _, item := range n.items {
				walk(item)
			}
		}
	}
	walk(node)
	return out
}

// hasCommentLine reports whether any line in [from, to) starts (after
// § 3.3 whitespace) with `##`, outside the given multiline-data ranges.
func hasCommentLine(text string, from, to int, skip [][2]int) bool {
	sorted := make([][2]int, len(skip))
	copy(sorted, skip)
	for i := 1; i < len(sorted); i++ {
		for j := i; j > 0 && sorted[j][0] < sorted[j-1][0]; j-- {
			sorted[j], sorted[j-1] = sorted[j-1], sorted[j]
		}
	}
	skipI := 0
	i := from
	for i < to {
		lineStart := i
		j := lineStart
		for j < to && text[j] != '\n' && text[j] != '\r' {
			j++
		}
		line := text[lineStart:j]
		trimmed := trimLeadingInline(line)
		k := lineStart + (len(line) - len(trimmed))
		for skipI < len(sorted) && sorted[skipI][1] <= k {
			skipI++
		}
		inData := skipI < len(sorted) && sorted[skipI][0] <= k && k < sorted[skipI][1]
		if !inData && strings.HasPrefix(trimmed, "##") {
			return true
		}
		i = j
		for i < to && (text[i] == '\n' || text[i] == '\r') {
			i++
		}
	}
	return false
}

// ─── inline slot planning ─────────────────────────────────────────────

func planInlineSlot(slotKind int, existingRaw bool, repl *pvalue, a, b0 int, wsRaw string) (splicePlan, *Error) {
	switch {
	case slotKind == slotItem && repl.kind == kStr && repl.s == "":
		return splicePlan{from: b0, body: bodyBorrowed("()")}, nil
	case slotKind == slotItem:
		b, err := inlineBody(repl, true)
		if err != nil {
			return splicePlan{}, err
		}
		return splicePlan{from: b0, body: b}, nil
	case slotKind == slotPair && existingRaw && repl.kind != kStr:
		b, err := inlineBody(repl, false)
		if err != nil {
			return splicePlan{}, err
		}
		return splicePlan{from: a + 1, prefix: [2]string{wsRaw, ""}, body: b}, nil
	default:
		b, err := inlineBody(repl, !existingRaw)
		if err != nil {
			return splicePlan{}, err
		}
		return splicePlan{from: b0, body: b}, nil
	}
}

// ─── linear slot planning ─────────────────────────────────────────────

func planLinearSlot(repl *pvalue, pc planCtx, a, b0, b1 int, wsRaw, wsPlain string) (splicePlan, *Error) {
	existingRaw := pc.existingRaw
	body, desiredRaw, err := planValue(repl, pc)
	if err != nil {
		return splicePlan{}, err
	}
	switch {
	case existingRaw == desiredRaw:
		prefix := ""
		if b0 == b1 && wsPlain == "" && wsRaw == "" {
			prefix = " "
		}
		return splicePlan{from: b0, prefix: [2]string{prefix, ""}, body: body}, nil
	case existingRaw && !desiredRaw && pc.slot == slotPair:
		// raw → plain pair: drop only the second ':'.
		return splicePlan{from: a + 1, prefix: [2]string{sepWS(b0, b1, wsRaw), ""}, body: body}, nil
	case existingRaw && !desiredRaw:
		// raw → plain item: drop '::', keep the original WS.
		return splicePlan{from: a, prefix: [2]string{wsRaw, ""}, body: body}, nil
	case !existingRaw && desiredRaw && pc.slot == slotPair:
		// plain → raw pair: add only the second ':'.
		return splicePlan{from: a + 1, prefix: [2]string{":", sepWS(b0, b1, wsPlain)}, body: body}, nil
	default: // plain → raw item
		return splicePlan{from: a, prefix: [2]string{":: ", ""}, body: body}, nil
	}
}

func sepWS(b0, b1 int, ws string) string {
	if b0 == b1 && ws == "" {
		return " "
	}
	return ws
}

// ─── replacement encoding ─────────────────────────────────────────────

func scalarBody(v *pvalue) (spliceBody, *Error) {
	switch v.kind {
	case kNull:
		return bodyBorrowed("null"), nil
	case kBool:
		if v.b {
			return bodyBorrowed("true"), nil
		}
		return bodyBorrowed("false"), nil
	case kInt, kFloat:
		return bodyBorrowed(v.s), nil
	default:
		return spliceBody{}, editUnsupported("internal: scalar_body called on a compound value")
	}
}

// inlineBody renders a physically-inline value; forceString gives a
// numeric/keyword-like String body String-provenance via a leading
// \uXXXX escape.
func inlineBody(repl *pvalue, forceString bool) (spliceBody, *Error) {
	switch repl.kind {
	case kStr:
		force := forceString && needsRawMarker(repl.s)
		return bodyOwned(encodeInlineString(repl.s, force)), nil
	case kObj, kArr:
		s, err := inlineCompoundBody(repl, 0)
		if err != nil {
			return spliceBody{}, err
		}
		return bodyOwned(s), nil
	default:
		return scalarBody(repl)
	}
}

func encodeInlineString(s string, forceString bool) string {
	var out strings.Builder
	out.Grow(len(s) + 8)
	lastIdx := len(s)
	if r, size := utf8.DecodeLastRuneInString(s); r != utf8.RuneError || size > 1 {
		lastIdx = len(s) - size
	}
	if s == "" {
		lastIdx = 0
	}
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if forceString && i == 0 {
			writeUnicodeEscape(&out, r)
		} else {
			encodeInlineChar(&out, r, i, lastIdx)
		}
		i += size
	}
	return out.String()
}

func encodeInlineChar(out *strings.Builder, c rune, i, last int) {
	switch c {
	case '\\':
		out.WriteString("\\\\")
	case ',':
		out.WriteString("\\,")
	case '}':
		out.WriteString("\\}")
	case ']':
		out.WriteString("\\]")
	case '{':
		out.WriteString("\\{")
	case '[':
		out.WriteString("\\[")
	case '\n':
		out.WriteString("\\n")
	case '\r':
		out.WriteString("\\r")
	default:
		if (i == 0 || i == last) && isKtavWhitespaceR(c) {
			writeUnicodeEscape(out, c)
		} else {
			out.WriteRune(c)
		}
	}
}

func writeUnicodeEscape(out *strings.Builder, r rune) {
	out.WriteString("\\u")
	const hex = "0123456789ABCDEF"
	cp := uint32(r)
	if cp > 0xFFFF {
		for shift := 28; shift >= 0; shift -= 4 {
			out.WriteByte(hex[(cp>>uint(shift))&0xF])
		}
		return
	}
	for _, shift := range [4]uint{12, 8, 4, 0} {
		out.WriteByte(hex[(cp>>shift)&0xF])
	}
}

func inlineCompoundBody(v *pvalue, depth int) (string, *Error) {
	var out strings.Builder
	if err := pushInlineValue(v, depth, &out); err != nil {
		return "", err
	}
	return out.String(), nil
}

func pushInlineValue(v *pvalue, depth int, out *strings.Builder) *Error {
	if depth > maxInlineDepth {
		return editUnsupported("replacement nesting exceeds inline depth limit (128)")
	}
	switch v.kind {
	case kObj:
		out.WriteByte('{')
		for i, e := range v.obj.entries {
			if i != 0 {
				out.WriteString(", ")
			}
			pushEscapedKeySegment(e.key, false, out)
			out.WriteString(": ")
			if e.val.kind == kStr {
				pushInlineString(e.val.s, needsRawMarker(e.val.s), out)
			} else if err := pushInlineValue(e.val, depth+1, out); err != nil {
				return err
			}
		}
		out.WriteByte('}')
	case kArr:
		out.WriteByte('[')
		for i, item := range v.arr {
			if i != 0 {
				out.WriteString(", ")
			}
			switch {
			case item.kind == kStr && item.s == "":
				out.WriteString("()")
			case item.kind == kStr:
				pushInlineString(item.s, needsRawMarker(item.s), out)
			default:
				if err := pushInlineValue(item, depth+1, out); err != nil {
					return err
				}
			}
		}
		out.WriteByte(']')
	case kStr:
		pushInlineString(v.s, needsRawMarker(v.s), out)
	case kNull:
		out.WriteString("null")
	case kBool:
		if v.b {
			out.WriteString("true")
		} else {
			out.WriteString("false")
		}
	case kInt, kFloat:
		out.WriteString(v.s)
	}
	return nil
}

func pushInlineString(s string, forceString bool, out *strings.Builder) {
	lastIdx := len(s)
	if s != "" {
		if _, size := utf8.DecodeLastRuneInString(s); size <= len(s) {
			lastIdx = len(s) - size
		}
	}
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if forceString && i == 0 {
			writeUnicodeEscape(out, r)
		} else {
			encodeInlineChar(out, r, i, lastIdx)
		}
		i += size
	}
}

// ─── value planning (linear/block slots) ──────────────────────────────

func planValue(repl *pvalue, pc planCtx) (spliceBody, bool, *Error) {
	switch repl.kind {
	case kStr:
		return planString(repl.s, pc)
	case kNull, kBool, kInt, kFloat:
		b, err := scalarBody(repl)
		return b, false, err
	case kObj, kArr:
		if pc.inlineCompound {
			s, err := inlineCompoundBody(repl, 0)
			return bodyOwned(s), false, err
		}
		empty := (repl.kind == kObj && len(repl.obj.entries) == 0) ||
			(repl.kind == kArr && len(repl.arr) == 0)
		if empty {
			if pc.blockCompound {
				if repl.kind == kObj {
					return bodyOwned("{" + pc.eol + pc.indent + "}"), false, nil
				}
				return bodyOwned("[" + pc.eol + pc.indent + "]"), false, nil
			}
			if repl.kind == kObj {
				return bodyBorrowed("{}"), false, nil
			}
			return bodyBorrowed("[]"), false, nil
		}
		doc, err := renderCompoundBody(repl)
		if err != nil {
			return spliceBody{}, false, err
		}
		return bodyOwned(reindentDoc(doc, pc.indent, pc.eol)), false, nil
	}
	return spliceBody{}, false, editUnsupported("internal: unhandled replacement kind")
}

func planString(s string, pc planCtx) (spliceBody, bool, *Error) {
	if strings.IndexByte(s, '\r') >= 0 {
		return spliceBody{}, false, editUnsupported("replacement string contains CR (U+000D)")
	}
	if multilineReplacement(s) {
		b, err := verbatimBlockBody(s, pc.indent, pc.eol)
		if err != nil {
			return spliceBody{}, false, err
		}
		return b, false, nil
	}
	desiredRaw := false
	if pc.slot == slotItem {
		desiredRaw = itemNeedsRawMarker(s) || s == ""
	} else {
		desiredRaw = needsRawMarker(s)
	}
	if pc.slot == slotItem && pc.rootFirst {
		desiredRaw = desiredRaw || isPairShape(s) || strings.HasPrefix(s, "\uFEFF")
	}
	if pc.existingRaw {
		desiredRaw = true
	}
	return bodyBorrowed(s), desiredRaw, nil
}

// multilineReplacement mirrors the Rust plan_string / block_replacement
// multiline test: contains '\n', or edge § 3.3 whitespace.
func multilineReplacement(s string) bool {
	if strings.IndexByte(s, '\n') >= 0 {
		return true
	}
	if s == "" {
		return false
	}
	if r, _ := utf8.DecodeRuneInString(s); isKtavWhitespaceR(r) {
		return true
	}
	if r, ok := lastRune(s); ok && isKtavWhitespaceR(r) {
		return true
	}
	return false
}

// verbatimBlockBody renders `(( ... ))`: content verbatim, '\n' → the
// document EOL, a final LF becomes an empty line before `))`.
func verbatimBlockBody(s, indent, eol string) (spliceBody, *Error) {
	for _, line := range splitLF(s) {
		if trimMultilineWS(line) == "))" {
			return spliceBody{}, editUnsupported(
				"multiline string contains a line equal to '))'; " +
					"no lossless verbatim form exists")
		}
	}
	var out strings.Builder
	out.WriteString("((")
	out.WriteString(eol)
	first := true
	for _, line := range splitLF(s) {
		if !first {
			out.WriteString(eol)
		}
		first = false
		out.WriteString(line)
	}
	out.WriteString(eol)
	out.WriteString(indent)
	out.WriteString("))")
	return bodyOwned(out.String()), nil
}

func trimMultilineWS(line string) string {
	start := 0
	for start < len(line) && (line[start] == ' ' || line[start] == '\t') {
		start++
	}
	end := len(line)
	for end > start && (line[end-1] == ' ' || line[end-1] == '\t') {
		end--
	}
	return line[start:end]
}

// blockReplacement plans a replacement inside a multiline block slot.
func blockReplacement(repl *pvalue, pc planCtx, a, b0 int, wsPair string) (splicePlan, *Error) {
	if repl.kind == kStr {
		s := repl.s
		if strings.IndexByte(s, '\r') >= 0 {
			return splicePlan{}, editUnsupported("replacement string contains CR (U+000D)")
		}
		if multilineReplacement(s) {
			b, err := verbatimBlockBody(s, pc.indent, pc.eol)
			if err != nil {
				return splicePlan{}, err
			}
			return splicePlan{from: b0, body: b}, nil
		}
		if pc.roleItemBlock && s == "" {
			return splicePlan{from: b0, body: bodyBorrowed("()")}, nil
		}
		itemRaw := pc.roleItemBlock &&
			(itemNeedsRawMarker(s) ||
				(pc.rootFirst && (isPairShape(s) || strings.HasPrefix(s, "\uFEFF"))))
		pairRaw := !pc.roleItemBlock && needsRawMarker(s)
		switch {
		case itemRaw:
			return splicePlan{from: b0, prefix: [2]string{":: ", ""}, body: bodyBorrowed(s)}, nil
		case pairRaw:
			return splicePlan{from: a + 1, prefix: [2]string{":", wsPair}, body: bodyBorrowed(s)}, nil
		default:
			return splicePlan{from: b0, body: bodyBorrowed(s)}, nil
		}
	}
	pc.slot = slotBlock
	body, _, err := planValue(repl, pc)
	if err != nil {
		return splicePlan{}, err
	}
	return splicePlan{from: b0, body: body}, nil
}

// renderCompoundBody pretty-renders a non-empty compound the way the
// reference render_compound_item does: an array-item render at indent
// level 1, so one reindent level converts it to the target's context.
func renderCompoundBody(v *pvalue) (string, *Error) {
	wrapped := pvArr()
	wrapped.arr = []*pvalue{v}
	w := &writer{mode: modePretty}
	w.renderRoot(wrapped)
	if w.reason != "" {
		return "", editUnsupported("replacement is not representable: " + w.reason)
	}
	out := w.b.String()
	// Strip the wrapper's "[\n" and the trailing "]\n" (pretty root
	// array whose first item is a compound always takes this shape).
	if !strings.HasPrefix(out, "[\n") || !strings.HasSuffix(out, "]\n") {
		return "", editUnsupported("internal: unexpected compound render shape")
	}
	return out[2 : len(out)-2], nil
}

// reindentDoc re-indents a pretty render under the target's line indent.
// Verbatim-block interior lines are not re-indented (their content is
// data). Any heuristic miss is caught by the mandatory semantic
// verification (fail closed).
func reindentDoc(doc, indent, eol string) string {
	lines := splitLF(doc)
	// splitLF adds a trailing "" for a final newline; drop it.
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	var out strings.Builder
	out.Grow(len(doc))
	inVerbatim := false
	for i, line := range lines {
		if strings.HasPrefix(line, indentUnit) {
			line = line[len(indentUnit):]
		}
		verbatimOpen := strings.HasSuffix(line, "((")
		verbatimClose := trimMultilineWS(line) == "))"
		if i > 0 {
			out.WriteString(eol)
			skipPrefix := inVerbatim || line == ""
			if !skipPrefix {
				out.WriteString(indent)
			}
		}
		out.WriteString(line)
		if verbatimOpen {
			inVerbatim = true
		} else if verbatimClose {
			inVerbatim = false
		}
	}
	return out.String()
}

// ─── source-geometry helpers (shared by splice + layout) ──────────────

func editPhysicalLine(text string, pos int) string {
	start := 0
	if n := strings.LastIndexAny(text[:pos], "\n\r"); n >= 0 {
		start = n + 1
	}
	end := len(text)
	if n := strings.IndexAny(text[pos:], "\n\r"); n >= 0 {
		end = pos + n
	}
	return text[start:end]
}

func isCompoundBody(body string) bool {
	return (strings.HasPrefix(body, "{") && strings.HasSuffix(body, "}")) ||
		(strings.HasPrefix(body, "[") && strings.HasSuffix(body, "]"))
}

func editSlotBody(slot *editSlot) (int, int) {
	return slot.b0, slot.b1
}

func editLineIndent(text string, pos int) string {
	start := 0
	if n := strings.LastIndexAny(text[:pos], "\n\r"); n >= 0 {
		start = n + 1
	}
	end := start
	for end < len(text) && (text[end] == ' ' || text[end] == '\t') {
		end++
	}
	return text[start:end]
}

func detectEOL(text string, pos int) string {
	for i := pos; i < len(text); i++ {
		switch text[i] {
		case '\n':
			return "\n"
		case '\r':
			if i+1 < len(text) && text[i+1] == '\n' {
				return "\r\n"
			}
			return "\r"
		}
	}
	// Fall back to the last terminator before pos.
	for i := pos - 1; i >= 0; i-- {
		switch text[i] {
		case '\r':
			if i+1 < len(text) && text[i+1] == '\n' {
				return "\r\n"
			}
			return "\r"
		case '\n':
			return "\n"
		}
	}
	return "\n"
}
