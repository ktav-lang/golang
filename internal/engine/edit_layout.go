package engine

// Layout adjustment (the Rust edit/splice/layout.rs port): expand the
// MINIMUM chain of containing inline wrappers when the patched line
// would exceed the readability thresholds — multiline replacement,
// growth beyond max(100 Unicode scalar values, the old physical line
// length), or dense inline escapes with >=25% overhead. Untouched
// nested branches stay inline; ok=false keeps the whole candidate
// inline instead of erroring (a valid inline candidate beats a
// formatting failure).

import (
	"strings"
	"unicode/utf8"
)

// layoutAdjust expands the minimum chain of containing inline wrappers
// when the patched line would exceed the readability thresholds.
func layoutAdjust(source, candidate string, root *editNode, segments []string, repl *pvalue, from, to, inserted int) (string, *Error) {
	ownerStart, ownerEnd, ok, lerr := inlineOwner(source, root, segments, from)
	if lerr != nil {
		return "", lerr
	}
	if !ok {
		return candidate, nil
	}
	if from < ownerStart {
		return candidate, nil
	}
	newEnd := ownerEnd - (to - from) + inserted
	if newEnd < ownerStart || newEnd > len(candidate) {
		return "", editUnsupported("invalid layout range")
	}
	body := candidate[ownerStart:newEnd]
	if !isCompoundBody(body) {
		return candidate, nil
	}
	oldWidth := utf8.RuneCountInString(editPhysicalLine(source, from))
	limit := oldWidth
	if limit < 100 {
		limit = 100
	}
	force, ferr := requiresBlock(repl, 0)
	if ferr != nil {
		return "", ferr
	}
	if !force && utf8.RuneCountInString(editPhysicalLine(candidate, from)) <= limit {
		return candidate, nil
	}
	indent := editLineIndent(source, ownerStart)
	eol := detectEOL(source, ownerEnd)
	expanded, ok2, eerr := expandInline(body, indent, eol, limit,
		from-ownerStart, from-ownerStart+inserted, force, 0)
	if eerr != nil {
		return "", eerr
	}
	if !ok2 {
		return candidate, nil
	}
	var result strings.Builder
	result.Grow(len(candidate))
	result.WriteString(candidate[:ownerStart])
	result.WriteString(expanded)
	result.WriteString(candidate[newEnd:])
	return result.String(), nil
}

// layoutAdjustInsert keeps insertion's inline candidate when standalone block
// encoding is impossible. mutateInsert performs the mandatory semantic check.
func layoutAdjustInsert(source, candidate string, root *editNode, segments []string, repl *pvalue, from, to, inserted int) (string, *Error) {
	adjusted, err := layoutAdjust(source, candidate, root, segments, repl, from, to, inserted)
	if !blockEncodable(repl) {
		return candidate, nil
	}
	return adjusted, err
}

// inlineOwner finds the smallest inline-compound range containing the
// splice position, if any (the Rust layout::inline_owner port).
func inlineOwner(source string, root *editNode, segments []string, pos int) (int, int, bool, *Error) {
	if root.inline {
		lineStart := 0
		if n := strings.LastIndexAny(source[:pos], "\n\r"); n >= 0 {
			lineStart = n + 1
		}
		line := source[lineStart:editPhysicalLineEnd(source, pos)]
		body := trimLeadingInline(trimInline(line))
		body = strings.TrimPrefix(body, "\uFEFF")
		abs := strings.LastIndex(line, body)
		if abs < 0 {
			return 0, 0, false, nil
		}
		start := lineStart + abs
		return start, start + len(body), true, nil
	}
	cur := root
	for _, segment := range segments {
		switch {
		case cur.isObject():
			cur = cur.child(segment)
		case cur.isArray():
			idx, err := parseEditIndex(segment)
			if err != nil {
				return 0, 0, false, err
			}
			cur = cur.item(idx)
		default:
			cur = nil
		}
		if cur == nil {
			return 0, 0, false, editNotFound()
		}
		if cur.slot != nil {
			start, end := editSlotBody(cur.slot)
			if end <= len(source) && start <= end {
				body := source[start:end]
				if (cur.kind == kObj || cur.kind == kArr) &&
					isCompoundBody(body) &&
					!strings.ContainsAny(body, "\n\r") {
					return start, end, true, nil
				}
			}
		}
	}
	if cur != nil && cur.inline {
		lineStart := 0
		if n := strings.LastIndexAny(source[:pos], "\n\r"); n >= 0 {
			lineStart = n + 1
		}
		line := source[lineStart:editPhysicalLineEnd(source, pos)]
		trimmed := trimLeadingInline(trimInline(line))
		trimmed = strings.TrimPrefix(trimmed, "\uFEFF")
		var body string
		if isCompoundBody(trimmed) {
			body = trimmed
		} else if colon := indexUnescapedColon(trimmed); colon >= 0 {
			body = trimLeadingInline(trimmed[colon+1:])
		} else {
			return 0, 0, false, nil
		}
		if isCompoundBody(body) {
			abs := strings.LastIndex(line, body)
			if abs < 0 {
				return 0, 0, false, nil
			}
			start := lineStart + abs
			return start, start + len(body), true, nil
		}
	}
	return 0, 0, false, nil
}

// editPhysicalLineEnd returns the end offset (exclusive) of the physical
// line containing pos (terminator excluded).
func editPhysicalLineEnd(text string, pos int) int {
	if n := strings.IndexAny(text[pos:], "\n\r"); n >= 0 {
		return pos + n
	}
	return len(text)
}

// requiresBlock reports whether the replacement contains a string that
// cannot stay inline: multiline, or dense escapes with >=25% overhead.
func requiresBlock(value *pvalue, depth int) (bool, *Error) {
	if depth > maxInlineDepth {
		return false, editUnsupported("replacement nesting exceeds inline depth limit (128)")
	}
	switch value.kind {
	case kStr:
		s := value.s
		if strings.ContainsAny(s, "\n\r") {
			return true, nil
		}
		encoded := encodeInlineString(s, false)
		escapes := 0
		for i := 0; i < len(encoded); i++ {
			if encoded[i] == '\\' {
				escapes++
				i++
			}
		}
		plain := utf8.RuneCountInString(s)
		overhead := utf8.RuneCountInString(encoded) - plain
		if overhead < 0 {
			overhead = 0
		}
		ceil := plain / 4
		if plain%4 != 0 {
			ceil++
		}
		return escapes >= 4 && overhead >= ceil, nil
	case kObj:
		for _, e := range value.obj.entries {
			if ok, err := requiresBlock(e.val, depth+1); ok || err != nil {
				return ok, err
			}
		}
		return false, nil
	case kArr:
		for _, item := range value.arr {
			if ok, err := requiresBlock(item, depth+1); ok || err != nil {
				return ok, err
			}
		}
		return false, nil
	default:
		return false, nil
	}
}

// blockEncodable mirrors insertion's standalone-block constraints: CR and a
// line consisting of `))` cannot be represented losslessly in block form.
func blockEncodable(value *pvalue) bool {
	switch value.kind {
	case kStr:
		if strings.ContainsRune(value.s, '\r') {
			return false
		}
		for _, line := range splitLF(value.s) {
			if trimMultilineWS(line) == "))" {
				return false
			}
		}
	case kObj:
		for _, entry := range value.obj.entries {
			if !blockEncodable(entry.val) {
				return false
			}
		}
	case kArr:
		for _, item := range value.arr {
			if !blockEncodable(item) {
				return false
			}
		}
	}
	return true
}

// expandInline renders the expanded block form of an inline compound.
// ok=false means no lossless expansion exists → keep the inline form.
func expandInline(body, indent, eol string, limit int, changedFrom, changedTo int, force bool, depth int) (string, bool, *Error) {
	if depth > maxInlineDepth {
		return "", false, editUnsupported("layout nesting exceeds inline depth limit")
	}
	isObject := body[0] == '{'
	inner := body[1 : len(body)-1]
	segments, serr := splitTopLevel(inner, 1, kspan{}, isObject)
	if serr != nil {
		return "", false, editUnsupported("cannot split inline layout")
	}
	childIndent := indent + indentUnit
	var out strings.Builder
	out.Grow(len(body))
	if isObject {
		out.WriteByte('{')
	} else {
		out.WriteByte('[')
	}
	cursor := 1 // search base inside body: parts never move backwards
	for _, part := range segments {
		part = trimInline(part)
		if part == "" {
			continue
		}
		offset := strings.Index(body[cursor:], part)
		if offset < 0 {
			return "", false, editUnsupported("cannot split inline layout")
		}
		offset += cursor
		cursor = offset + 1
		touched := offset < changedTo && offset+len(part) > changedFrom
		var prefix, value string
		raw := false
		if isObject {
			colon := indexUnescapedColon(part)
			if colon < 0 {
				return "", false, editUnsupported("inline pair has no separator")
			}
			after := part[colon+1:]
			if rest, isRaw := strings.CutPrefix(after, ":"); isRaw {
				raw = true
				after = rest
			}
			value = trimLeadingInline(after)
			prefix = part[:len(part)-len(value)]
		} else {
			prefix = ""
			value = part
		}
		out.WriteString(eol)
		out.WriteString(childIndent)
		if !raw && isCompoundBody(value) && touched &&
			(force || utf8.RuneCountInString(childIndent+part) > limit) {
			valueOffset := offset + strings.LastIndex(part, value)
			if valueOffset < offset {
				return "", false, editUnsupported("cannot split inline layout")
			}
			localFrom := changedFrom - valueOffset
			if localFrom < 0 {
				localFrom = 0
			}
			localTo := changedTo - valueOffset
			if localTo < 0 {
				localTo = 0
			}
			out.WriteString(prefix)
			child, ok, eerr := expandInline(value, childIndent, eol, limit, localFrom, localTo, force, depth+1)
			if eerr != nil {
				return "", false, eerr
			}
			if !ok {
				return "", false, nil
			}
			out.WriteString(child)
		} else {
			child, ok, eerr := standaloneInlinePart(part, prefix, value, raw, isObject, childIndent, eol)
			if eerr != nil {
				return "", false, eerr
			}
			if !ok {
				return "", false, nil
			}
			out.WriteString(child)
		}
	}
	out.WriteString(eol)
	out.WriteString(indent)
	if isObject {
		out.WriteByte('}')
	} else {
		out.WriteByte(']')
	}
	return out.String(), true, nil
}

// standaloneInlinePart moves one inline part to its own line, decoding
// and re-encoding when the inline escapes would lose their meaning.
// ok=false keeps the whole candidate inline.
func standaloneInlinePart(part, prefix, body string, raw, isObject bool, indent, eol string) (string, bool, *Error) {
	// Moving out of inline context stops escape decoding: check the
	// value's semantics first.
	var inlineWrap string
	if isObject {
		inlineWrap = "{" + part + "}"
	} else {
		inlineWrap = "[" + part + "]"
	}
	expectedDoc, _, perr := parseDoc(inlineWrap, false, false)
	if perr != nil {
		return "", false, editUnsupported("invalid inline layout")
	}
	expected := expectedDoc.(*pvalue)
	var lineText string
	if isObject {
		lineText = part
	} else {
		lineText = "[\n" + part + "\n]"
	}
	lineDoc, _, lerr := parseDoc(lineText, false, false)
	if lerr == nil && pvEqual(lineDoc.(*pvalue), expected) {
		return part, true, nil
	}

	var value *pvalue
	if raw {
		decoded, _, bad := processEscapes(body)
		if bad != "" {
			return "", false, editUnsupported("invalid inline string")
		}
		value = pvStr(decoded)
	} else {
		wrapped := "[" + body + "]"
		parsed, _, perr2 := parseDoc(wrapped, false, false)
		if perr2 != nil {
			return "", false, editUnsupported("cannot decode moved inline value")
		}
		arr := parsed.(*pvalue)
		if arr.kind != kArr || len(arr.arr) != 1 {
			return "", false, editUnsupported("moved inline value is not a single value")
		}
		value = arr.arr[0]
	}
	if value.kind == kStr {
		s := value.s
		if strings.IndexByte(s, '\r') >= 0 {
			return "", false, nil
		}
		for _, line := range splitLF(s) {
			if trimMultilineWS(line) == "))" {
				return "", false, nil
			}
		}
	}
	pc := planCtx{
		slot:           slotPair,
		existingRaw:    raw,
		indent:         indent,
		eol:            eol,
		inlineCompound: true,
	}
	if !isObject {
		pc.slot = slotItem
	}
	newBody, desiredRaw, perr3 := planValue(value, pc)
	if perr3 != nil {
		return "", false, perr3
	}
	var out strings.Builder
	if isObject {
		colon := indexUnescapedColon(part)
		if colon < 0 {
			return "", false, editUnsupported("moved pair has no separator")
		}
		out.WriteString(part[:colon+1])
		if desiredRaw {
			out.WriteByte(':')
		}
		var ws string
		if raw {
			if colon+2 <= len(prefix) {
				ws = prefix[colon+2:]
			}
		} else {
			if colon+1 <= len(prefix) {
				ws = prefix[colon+1:]
			}
		}
		if ws == "" {
			out.WriteByte(' ')
		} else {
			out.WriteString(ws)
		}
	} else if desiredRaw {
		out.WriteString(":: ")
	}
	out.WriteString(newBody.str())
	return out.String(), true, nil
}
