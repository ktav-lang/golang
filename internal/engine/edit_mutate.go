// Mutation entry points for source-preserving insert/remove/rename.
package engine

import (
	"strings"
	"unicode/utf8"
)

func pureInsertValueBytes(src, path, replacement []byte) ([]byte, error) {
	return pureMutateBytes(src, path, replacement, "insert", false)
}
func pureInsertValueSegmentsBytes(src, path, replacement []byte) ([]byte, error) {
	return pureMutateBytes(src, path, replacement, "insert", true)
}
func pureRemoveValueBytes(src, path []byte) ([]byte, error) {
	return pureMutateBytes(src, path, nil, "remove", false)
}
func pureRemoveValueSegmentsBytes(src, path []byte) ([]byte, error) {
	return pureMutateBytes(src, path, nil, "remove", true)
}
func pureRenameKeyBytes(src, path, newKey []byte) ([]byte, error) {
	return pureMutateBytes(src, path, newKey, "rename", false)
}
func pureRenameKeySegmentsBytes(src, path, newKey []byte) ([]byte, error) {
	return pureMutateBytes(src, path, newKey, "rename", true)
}

func pureMutateBytes(src, path, payload []byte, op string, literal bool) ([]byte, error) {
	text := string(src)
	if e := invalidUTF8Error(text); e != nil {
		return nil, e
	}
	parsed, _, e := parseDoc(text, false, false)
	if e != nil {
		enrichLineText(text, e)
		return nil, e
	}
	if uint64(len(src)) > uint64(^uint32(0)) {
		return nil, editUnsupported("source exceeds the u32 byte-size limit")
	}
	if e = transportUTF8Error(path); e != nil {
		return nil, e
	}
	var segs []string
	if literal {
		if e = jsonSurrogateEscapeError(path); e != nil {
			return nil, e
		}
		segs, e = decodeSegmentsWire(path)
	} else {
		segs, e = parseEditPath(string(path))
	}
	if e != nil {
		return nil, e
	}
	if len(segs) == 0 {
		return nil, editNotFound()
	}
	var replacement *pvalue
	if op == "rename" && !utf8.Valid(payload) {
		return nil, transportUTF8Error(payload)
	}
	if op == "rename" && len(payload) == 0 {
		return nil, editUnsupported("empty key name is not representable")
	}
	if op == "insert" {
		if e = transportUTF8Error(payload); e != nil {
			return nil, e
		}
		if e = jsonSurrogateEscapeError(payload); e != nil {
			return nil, e
		}
		replacement, e = ingestWireValue(payload)
		if e != nil {
			return nil, e
		}
		if e = validateEditIntegerDomain(replacement); e != nil {
			return nil, e
		}
	}
	switch op {
	case "rename":
		return mutateRename(text, parsed.(*pvalue), segs, string(payload))
	case "remove":
		return mutateRemove(text, parsed.(*pvalue), segs)
	default:
		return mutateInsert(text, parsed.(*pvalue), segs, replacement)
	}
}
func mutateRename(text string, orig *pvalue, segs []string, newKey string) ([]byte, error) {
	if len(segs) == 0 {
		return nil, editNotFound()
	}
	parent := editLookupValue(orig, segs[:len(segs)-1])
	if parent == nil {
		return nil, editNotFound()
	}
	key := segs[len(segs)-1]
	if parent.kind == kArr {
		return nil, editUnsupported("rename of an Array element is not supported; only Object keys can be renamed")
	}
	if parent.kind != kObj {
		return nil, editNotFound()
	}
	if parent.obj.get(key) == nil {
		return nil, editNotFound()
	}
	if key == newKey {
		return []byte(text), nil
	}
	if parent.obj.get(newKey) != nil {
		return nil, &Error{Msg: "edit target already exists; insertion never overwrites", Class: "EditAlreadyExists"}
	}
	root, explicit, e := buildEditTree(text)
	if e != nil {
		return nil, editUnsupported("source text does not parse: " + e.Error())
	}
	pn, _, descendErr := editDescend(root, segs[:len(segs)-1], explicit)
	if descendErr != nil {
		return nil, descendErr
	}
	n := pn.child(key)
	if n == nil || n.slot == nil {
		return nil, editUnsupported("target is a noncontiguous value (dotted-key/reopen intermediate); no lossless rename range exists")
	}
	if n.slot.kind == slotItem {
		return nil, editUnsupported("rename of an Array element is not supported; only Object keys can be renamed")
	}
	a, b := editPairKeyRange(text, n.slot.colon)
	keyText := text[a:b]
	keyParts := splitKeyPath(keyText)
	if len(keyParts) == 0 {
		return nil, editUnsupported("internal: key without segments")
	}
	last := keyParts[len(keyParts)-1]
	trimmedLast := trimInline(last)
	leadingBytes := strings.Index(last, trimmedLast)
	segStart := len(keyText) - len(last) + leadingBytes
	segEnd := segStart + len(trimmedLast)
	if segStart < 0 {
		return nil, editUnsupported("internal: invalid key segment range")
	}
	var enc strings.Builder
	first := len(pn.table.entries) > 0 && pn.table.entries[0].key == key
	pushEscapedKeySegment(newKey, len(segs) == 1 && first, &enc)
	out := text[:a+segStart] + enc.String() + text[a+segEnd:]
	expected := clonePV(orig)
	renamePVObject(editLookupValue(expected, segs[:len(segs)-1]).obj, key, newKey)
	if verifyErr := verifyMutatedKeyOrder(out, expected); verifyErr != nil {
		return nil, verifyErr
	}
	return []byte(out), nil
}
func mutateRemove(text string, orig *pvalue, segs []string) ([]byte, error) {
	par := editLookupValue(orig, segs[:len(segs)-1])
	if par == nil {
		return nil, editNotFound()
	}
	key := segs[len(segs)-1]
	root, explicit, e := buildEditTree(text)
	if e != nil {
		return nil, editUnsupported("source text does not parse: " + e.Error())
	}
	n, target, descendErr := editDescend(root, segs, explicit)
	if descendErr != nil {
		return nil, descendErr
	}
	expected := clonePV(orig)
	p := editLookupValue(expected, segs[:len(segs)-1])
	if p == nil {
		return nil, editNotFound()
	}
	if p.kind == kObj {
		if p.obj.get(key) == nil {
			return nil, editNotFound()
		}
		removePVObject(p.obj, key)
	} else if p.kind == kArr {
		i, pe := parseEditIndex(key)
		if pe != nil || i < 0 || i >= len(p.arr) {
			return nil, editNotFound()
		}
		p.arr = append(p.arr[:i], p.arr[i+1:]...)
	} else {
		return nil, editNotFound()
	}
	if n.slot == nil || editHasSyntheticParent(root, segs) {
		// Dotted leaves can be spliced only when their physical key resolves
		// relative to the nearest contiguous positional ancestor.
		if n.kind != kObj && n.kind != kArr {
			if from, to, materialized, ok := editDottedLeafRemoval(text, root, n, segs, expected); ok {
				out := text[:from] + materialized + text[to:]
				if verifyErr := verifyMutated(out, expected); verifyErr != nil {
					return nil, verifyErr
				}
				return []byte(out), nil
			}
		}
		return nil, editUnsupported("target is a noncontiguous value; no lossless removal range exists")
	}
	if n.kind == kObj || n.kind == kArr {
		if hasCommentLine(text, n.slot.b0, n.slot.b1, collectMultilineScalarRanges(n)) {
			return nil, editUnsupported("target contains comment lines; removal cannot preserve them losslessly")
		}
	}
	var from, to int
	if target.inline {
		var ok bool
		from, to, ok = inlineRemovalRange(text, root, segs, n.slot)
		if !ok {
			return nil, editUnsupported("cannot determine positional inline removal range")
		}
	} else {
		anchor, end := n.slot.colon, n.slot.b1
		if n.slot.kind == slotItem {
			anchor = n.slot.colon
		}
		if n.slot.kind == slotBlock {
			anchor = n.slot.b0
			end = n.slot.b1
		}
		from = strings.LastIndexAny(text[:anchor], "\r\n") + 1
		if from == 0 && strings.HasPrefix(text, "\xef\xbb\xbf") {
			from = 3
		}
		to = editLineEnd(text, end)
		if to == len(text) && !strings.HasSuffix(text[from:to], "\n") && !strings.HasSuffix(text[from:to], "\r") {
			if from > 0 && (text[from-1] == '\n' || text[from-1] == '\r') {
				from--
				if from > 0 && text[from] == '\n' && text[from-1] == '\r' {
					from--
				}
			}
		}
	}
	out := text[:from] + text[to:]
	if verifyErr := verifyMutated(out, expected); verifyErr != nil {
		return nil, verifyErr
	}
	return []byte(out), nil
}
func editHasSyntheticParent(root *editNode, segs []string) bool {
	if root == nil || len(segs) == 0 {
		return false
	}
	cur := root
	for _, s := range segs[:len(segs)-1] {
		if cur == nil {
			return false
		}
		switch cur.kind {
		case kObj:
			cur = cur.child(s)
		case kArr:
			i, err := parseEditIndex(s)
			if err != nil || i < 0 || i >= len(cur.items) {
				return false
			}
			cur = cur.items[i]
		default:
			return false
		}
		if cur == nil {
			return false
		}
		if cur.slot == nil {
			return true
		}
	}
	return false
}

func editDottedLeafRemoval(text string, root, target *editNode, segs []string, expected *pvalue) (int, int, string, bool) {
	if len(segs) < 2 || target.slot == nil || target.slot.kind != slotPair {
		return 0, 0, "", false
	}
	colon := target.slot.colon
	lineStart := strings.LastIndexAny(text[:colon], "\r\n") + 1
	lineEnd := editLineEnd(text, colon)
	keyStart := lineStart
	if keyStart == 0 && strings.HasPrefix(text, "\xef\xbb\xbf") {
		keyStart += 3
	}
	parts := splitKeyPath(strings.TrimSpace(text[keyStart:colon]))
	if len(parts) < 2 || len(parts) > len(segs) {
		return 0, 0, "", false
	}
	base := len(segs) - len(parts)
	for i, part := range parts {
		decoded, bad := decodeKeySegment(trimInline(part))
		if bad != "" || decoded != segs[base+i] {
			return 0, 0, "", false
		}
	}
	// Resolve the nearest physical positional ancestor. Dotted keys are
	// interpreted relative to that container (which may itself be in an
	// array), not relative to the document root.
	containerDepth := 0
	cur := root
	for i, segment := range segs[:len(segs)-1] {
		if cur == nil {
			return 0, 0, "", false
		}
		switch cur.kind {
		case kObj:
			cur = cur.child(segment)
		case kArr:
			index, err := parseEditIndex(segment)
			if err != nil || index < 0 || index >= len(cur.items) {
				return 0, 0, "", false
			}
			cur = cur.items[index]
		default:
			return 0, 0, "", false
		}
		if cur == nil {
			return 0, 0, "", false
		}
		if cur.slot != nil && cur.kind == kObj {
			containerDepth = i + 1
		}
	}
	if base < containerDepth || base >= len(segs) {
		return 0, 0, "", false
	}
	for i := 0; i < len(parts); i++ {
		if segs[base+i] == "" {
			return 0, 0, "", false
		}
	}
	// If the source after a plain cut already has the expected semantic tree,
	// the synthetic parent still has siblings and must not be materialized.
	if out := text[:keyStart] + text[lineEnd:]; verifyMutated(out, expected) == nil {
		return keyStart, lineEnd, "", true
	}
	var replacement strings.Builder
	indentEnd := keyStart
	for indentEnd < colon && (text[indentEnd] == ' ' || text[indentEnd] == '\t') {
		indentEnd++
	}
	replacement.WriteString(text[keyStart:indentEnd])
	for i, segment := range segs[base : len(segs)-1] {
		if i > 0 {
			replacement.WriteByte('.')
		}
		pushEscapedKeySegment(segment, false, &replacement)
	}
	replacement.WriteString(": {}")
	if lineEnd > colon {
		eolStart := lineEnd
		if text[lineEnd-1] == '\n' {
			eolStart--
			if eolStart > colon && text[eolStart-1] == '\r' {
				eolStart--
			}
		} else if text[lineEnd-1] == '\r' {
			eolStart--
		}
		replacement.WriteString(text[eolStart:lineEnd])
	}
	return keyStart, lineEnd, replacement.String(), true
}

func editLineEnd(s string, pos int) int {
	if pos < 0 || pos > len(s) {
		return len(s)
	}
	if i := strings.IndexAny(s[pos:], "\r\n"); i >= 0 {
		p := pos + i
		if s[p] == '\r' && p+1 < len(s) && s[p+1] == '\n' {
			return p + 2
		}
		return p + 1
	}
	return len(s)
}
func inlineRemovalRange(text string, root *editNode, segs []string, slot *editSlot) (int, int, bool) {
	parent := root
	for _, s := range segs[:len(segs)-1] {
		if parent.kind == kObj {
			parent = parent.child(s)
		} else if parent.kind == kArr {
			i, e := parseEditIndex(s)
			if e != nil || i < 0 || i >= len(parent.items) {
				return 0, 0, false
			}
			parent = parent.items[i]
		} else {
			return 0, 0, false
		}
		if parent == nil {
			return 0, 0, false
		}
	}
	var a, b int
	if parent.slot != nil {
		a, b = parent.slot.b0, parent.slot.b1
	} else if parent.inline {
		line := editPhysicalLine(text, slot.colon)
		lineStart := strings.LastIndexAny(text[:slot.colon], "\r\n") + 1
		lineEnd := lineStart + len(line)
		if lineStart == 0 && strings.HasPrefix(text, "\xef\xbb\xbf") {
			lineStart += 3
		}
		trimmed := trimInline(text[lineStart:lineEnd])
		a = lineStart + strings.Index(text[lineStart:lineEnd], trimmed)
		if trimmed == "" || a < lineStart || a >= lineEnd || (text[a] != '{' && text[a] != '[') {
			return 0, 0, false
		}
		b = lineEnd
		close := byte('}')
		if text[a] == '[' {
			close = ']'
		}
		if rel := strings.LastIndexByte(text[a:b], close); rel >= 0 {
			b = a + rel + 1
		} else {
			return 0, 0, false
		}
	} else {
		return 0, 0, false
	}
	if a >= b || (text[a] != '{' && text[a] != '[') {
		return 0, 0, false
	}
	lo, hi := a+1, b-1
	inner := text[lo:hi]
	parts, _ := splitTopLevel(inner, 1, kspan{}, text[a] == '{')
	if len(parts) <= 1 {
		return lo, hi, true
	}
	spans := [][2]int{}
	cursor := 0
	for _, part := range parts {
		idx := strings.Index(inner[cursor:], part)
		if idx < 0 {
			return 0, 0, false
		}
		x := cursor + idx
		trim := trimInline(part)
		if trim == "" {
			cursor = x + len(part)
			continue
		}
		lead := strings.Index(part, trim)
		spans = append(spans, [2]int{lo + x + lead, lo + x + lead + len(trim)})
		cursor = x + len(part)
	}
	if len(spans) == 0 {
		return 0, 0, false
	}
	for i := range spans {
		if slot.b0 >= spans[i][0] && slot.b1 <= spans[i][1] {
			if len(spans) == 1 {
				return lo, hi, true
			}
			if i+1 < len(spans) {
				return spans[i][0], spans[i+1][0], true
			}
			return spans[i-1][1], spans[i][1], true
		}
	}
	return 0, 0, false
}
func editSiblingIndent(text string, node *editNode) string {
	if node == nil || node.table == nil || len(node.table.entries) == 0 {
		return ""
	}
	last := node.table.entries[len(node.table.entries)-1].node
	if last == nil || last.slot == nil {
		return ""
	}
	pos := last.slot.b0
	if last.slot.kind == slotPair {
		pos = last.slot.colon
	}
	return editLineIndent(text, pos)
}

func mutateInsert(text string, orig *pvalue, segs []string, repl *pvalue) ([]byte, error) {
	if len(segs) == 0 {
		return nil, editNotFound()
	}
	parents := segs[:len(segs)-1]
	pv := editLookupValue(orig, parents)
	if pv == nil {
		return nil, editNotFound()
	}
	key := segs[len(segs)-1]
	if pv.kind == kArr {
		return nil, editUnsupported("insertion into an Array is not supported; only Object members can be created")
	}
	if pv.kind != kObj {
		return nil, editNotFound()
	}
	if pv.obj.get(key) != nil {
		return nil, &Error{Msg: "edit target already exists; insertion never overwrites", Class: "EditAlreadyExists"}
	}
	root, explicit, e := buildEditTree(text)
	if e != nil {
		return nil, editUnsupported("source text does not parse: " + e.Error())
	}
	parent, _, descendErr := editDescend(root, parents, explicit)
	if descendErr != nil {
		return nil, descendErr
	}
	if parent.kind != kObj {
		return nil, editNotFound()
	}
	expected := clonePV(orig)
	editLookupValue(expected, parents).obj.insertLeaf(key, clonePV(repl))
	var from, to int
	ins := ""
	var keyBuilder strings.Builder
	pushEscapedKeySegment(key, false, &keyBuilder)
	blockPair := func(indent, eol string) (string, *Error) {
		pc := planCtx{slot: slotPair, indent: indent, eol: eol, blockCompound: true}
		body, raw, err := planValue(repl, pc)
		if err != nil {
			return "", err
		}
		separator := ": "
		if raw {
			separator = ":: "
		}
		pair := keyBuilder.String() + separator + body.str()
		return pair, nil
	}
	inlineValue, inlineErr := renderInlineValue(repl)
	if inlineErr != nil {
		return nil, inlineErr
	}
	pair := keyBuilder.String() + ": " + inlineValue
	if parent.inline && (parent.slot != nil || len(parents) == 0) {
		bodyStart := from
		if parent.slot != nil {
			from = parent.slot.b1 - 1
			bodyStart = parent.slot.b0 + 1
		} else {
			// Root positional range comes from the parser-aligned tree; do not
			// infer delimiters by scanning source (comments may contain braces).
			from = root.close
			bodyStart = root.open + 1
			if root.open < 0 || from <= root.open {
				return nil, editUnsupported("invalid inline root range")
			}
		}
		to = from
		body := trimInline(text[bodyStart:from])
		if body != "" {
			trimmed := strings.TrimRightFunc(text[bodyStart:from], isKtavWhitespaceR)
			commaIsStructural := strings.HasSuffix(trimmed, ",")
			if commaIsStructural {
				slashes := 0
				for i := len(trimmed) - 2; i >= 0 && trimmed[i] == '\\'; i-- {
					slashes++
				}
				commaIsStructural = slashes%2 == 0
			}
			if commaIsStructural {
				from = bodyStart + len(trimmed)
				to = from
				ins = " " + pair + ","
			} else {
				ins = ", " + pair
			}
		} else {
			from = bodyStart
			ins = pair
		}
	} else if len(parents) == 0 && explicit {
		from = rootCloserOffset(text)
		to = from
		if strings.TrimSpace(text[from:editLineEnd(text, from)]) != "}" {
			// Empty explicit root may close on the opener line; insert after it
			// with the source EOL rather than placing a member before `{`.
			openEnd := editLineEnd(text, 0)
			if openEnd == len(text) {
				return nil, editUnsupported("explicit root has no separate closer line")
			}
			from = editLineEnd(text, 0)
			to = from
			pair, pairErr := blockPair("    ", detectEOL(text, from))
			if pairErr != nil {
				return nil, pairErr
			}
			ins = detectEOL(text, from) + "    " + pair + detectEOL(text, from)
		} else {
			indent := editSiblingIndent(text, parent)
			if indent == "" {
				indent = "    "
			}
			pair, pairErr := blockPair(indent, detectEOL(text, from))
			if pairErr != nil {
				return nil, pairErr
			}
			ins = indent + pair + detectEOL(text, from)
		}
	} else if len(parents) == 0 {
		from = len(text)
		to = from
		eol := detectEOL(text, from)
		pair, pairErr := blockPair("", eol)
		if pairErr != nil {
			return nil, pairErr
		}
		if from > 0 && !strings.HasSuffix(text, "\n") && !strings.HasSuffix(text, "\r") {
			ins = eol
		}
		ins += pair
		if strings.HasSuffix(text, "\n") || strings.HasSuffix(text, "\r") {
			ins += eol
		}
	} else if parent.slot != nil && parent.slot.kind == slotBlock {
		from = strings.LastIndexAny(text[:parent.slot.b1], "\r\n") + 1
		to = from
		indent := editSiblingIndent(text, parent)
		if indent == "" {
			indent = editLineIndent(text, parent.slot.b0) + "    "
		}
		eol := detectEOL(text, from)
		pair, pairErr := blockPair(indent, eol)
		if pairErr != nil {
			return nil, pairErr
		}
		ins = indent + pair + eol
	} else {
		return nil, editUnsupported("parent has no contiguous insertion point")
	}
	out := text[:from] + ins + text[to:]
	if parent.inline {
		adjusted, lerr := layoutAdjustInsert(text, out, root, parents, repl, from, to, len(ins))
		if lerr != nil {
			return nil, lerr
		}
		out = adjusted
	}
	if verifyErr := verifyMutated(out, expected); verifyErr != nil {
		return nil, verifyErr
	}
	return []byte(out), nil
}
func rootCloserOffset(s string) int {
	end := len(s)
	for end > 0 {
		start := strings.LastIndexAny(s[:end], "\r\n") + 1
		lineEnd := editLineEnd(s, start)
		if strings.TrimSpace(s[start:lineEnd]) == "}" {
			return start
		}
		if start == 0 {
			break
		}
		end = start - 1
	}
	return len(s)
}

func editPairKeyRange(text string, colon int) (int, int) {
	start := strings.LastIndexAny(text[:colon], "\r\n") + 1
	if start == 0 && strings.HasPrefix(text, "\xef\xbb\xbf") {
		start = 3
	}
	i := start
	boundary := start
	segStart := true
	b := text
	for i < colon {
		if segStart {
			for i < colon && (b[i] == ' ' || b[i] == '\t') {
				i++
			}
			if i < colon && (b[i] == '\'' || b[i] == '"' || b[i] == '`') {
				q := b[i]
				i++
				for i < colon {
					if b[i] == '\\' {
						i += 2
						continue
					}
					if b[i] == q {
						i++
						break
					}
					i++
				}
				segStart = false
				continue
			}
			segStart = false
		}
		switch b[i] {
		case '\\':
			i += 2
			continue
		case '.':
			boundary = i + 1
			segStart = true
		case ',', '{', '[':
			boundary = i + 1
			segStart = true
		}
		i++
	}
	end := colon
	for end > boundary && (b[end-1] == ' ' || b[end-1] == '\t') {
		end--
	}
	for boundary < end && (b[boundary] == ' ' || b[boundary] == '\t') {
		boundary++
	}
	return boundary, end
}
