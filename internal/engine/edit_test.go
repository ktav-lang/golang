package engine

// Ports of the Rust reference edit suites (tests/suite/api/text/
// edit_value.rs + layout.rs): exact-output oracles for the splice
// engine, the 22 layout regressions, and the typed error table. Both
// the textual-path and segments APIs are exercised by the shared check
// helpers, so an ASCII-only shortcut in either entry point fails here.

import (
	"strings"
	"testing"
)

func op() *Operations { return New() }

func editCheck(t *testing.T, source string, path []string, replacement any, expected string) {
	t.Helper()
	textual := ""
	for i, seg := range path {
		esc := ""
		for j := 0; j < len(seg); j++ {
			switch seg[j] {
			case '.':
				esc += `\.`
			case '\\':
				esc += `\\`
			default:
				// Copy the raw byte: multi-byte UTF-8 runes must pass
				// through verbatim (string(byte) would re-encode the
				// byte value as a code point and corrupt Cyrillic etc.).
				esc += seg[j : j+1]
			}
		}
		if i > 0 {
			textual += "."
		}
		textual += esc
	}
	// Expected value tree: parse the source, replace the target.
	wantDoc, _, err := parseDoc(source, false, false)
	if err != nil {
		t.Fatalf("test bug: source does not parse: %v", err)
	}
	want := wantDoc.(*pvalue)
	wireRepl, werr := encodeEditReplacementTagged(replacement)
	if werr != nil {
		t.Fatalf("test bug: replacement not encodable: %v", werr)
	}
	repl, werr2 := ingestWireValue(wireRepl)
	if werr2 != nil {
		t.Fatalf("test bug: replacement wire ingest: %v", werr2)
	}
	if _, terr := editTakeTarget(want, path); terr != nil {
		t.Fatalf("test bug: path %v not addressable: %v", path, terr)
	}
	setTarget(want, path, repl)

	for name, fn := range map[string]func() (string, error){
		"textual": func() (string, error) { return op().EditValue(source, textual, replacement) },
		"segments": func() (string, error) {
			return op().EditValueSegments(source, path, replacement)
		},
	} {
		got, err := fn()
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if got != expected {
			t.Fatalf("%s: output mismatch\nwant: %q\ngot:  %q", name, expected, got)
		}
		gotDoc, _, perr := parseDoc(got, false, false)
		if perr != nil {
			t.Fatalf("%s: patch does not parse: %v\n%s", name, perr, got)
		}
		if !pvEqual(gotDoc.(*pvalue), want) {
			t.Fatalf("%s: patched tree diverges from expected\n%s", name, got)
		}
	}
}

// setTarget writes replacement at path (test-bug fatal on miss).
func setTarget(root *pvalue, path []string, value *pvalue) {
	cur := root
	for _, seg := range path[:len(path)-1] {
		if cur.kind == kObj {
			cur = cur.obj.get(seg)
		} else {
			cur = cur.arr[mustIndex(seg)]
		}
	}
	last := path[len(path)-1]
	if cur.kind == kObj {
		cur.obj.set(last, value)
	} else {
		cur.arr[mustIndex(last)] = value
	}
}

func mustIndex(seg string) int {
	n := 0
	for i := 0; i < len(seg); i++ {
		n = n*10 + int(seg[i]-'0')
	}
	return n
}

func editCheckUnsupported(t *testing.T, source, textualPath string, replacement any) {
	t.Helper()
	_, err := op().EditValue(source, textualPath, replacement)
	if err == nil {
		t.Fatal("expected EditUnsupported, got ok")
	}
	e, ok := err.(*Error)
	if !ok || e.Class != "EditUnsupported" {
		t.Fatalf("want EditUnsupported, got %#v", err)
	}
	if e.Reason != "" || e.Line != 0 || e.LineText != "" || e.Span != nil ||
		e.Path != nil || e.Canonical != "" || e.SpecSection != "" {
		t.Fatalf("edit error must null every other envelope field: %#v", e)
	}
}

func editCheckClass(t *testing.T, source, textualPath string, replacement any, class, body string) {
	t.Helper()
	_, err := op().EditValue(source, textualPath, replacement)
	if err == nil {
		t.Fatalf("expected %s, got ok", class)
	}
	e, ok := err.(*Error)
	if !ok || e.Class != class {
		t.Fatalf("want %s, got %#v", class, err)
	}
	if body != "" && e.Body != body {
		t.Fatalf("%s body = %q, want %q", class, e.Body, body)
	}
}

// ─── scalars, all types ───────────────────────────────────────────────

func TestEditStringReplacementPlain(t *testing.T) {
	editCheck(t, "port: 8080\nhost: db\n", []string{"port"}, "db.internal",
		"port: db.internal\nhost: db\n")
}

func TestEditIntReplacementSwitchesRawMarkerOff(t *testing.T) {
	editCheck(t, "port:: 8080\n", []string{"port"}, int64(9090), "port: 9090\n")
}

func TestEditNumericStringSwitchesRawMarkerOn(t *testing.T) {
	editCheck(t, "port: 8080\n", []string{"port"}, "42", "port:: 42\n")
}

func TestEditKeywordLikeStringNeedsRaw(t *testing.T) {
	editCheck(t, "mode: auto\n", []string{"mode"}, "true", "mode:: true\n")
}

func TestEditRawMarkerPreservedForString(t *testing.T) {
	editCheck(t, "k:: v1\n", []string{"k"}, "v2", "k:: v2\n")
}

func TestEditEmptyStringReplacementKeepsRaw(t *testing.T) {
	editCheck(t, "k:: x\n", []string{"k"}, "", "k:: \n")
}

func TestEditNullAndBoolReplacements(t *testing.T) {
	editCheck(t, "a: x\nb: y\nc: z\n", []string{"a"}, nil,
		"a: null\nb: y\nc: z\n")
	editCheck(t, "a: x\nb: y\nc: z\n", []string{"b"}, false,
		"a: x\nb: false\nc: z\n")
	editCheck(t, "a: x\nb: y\nc: z\n", []string{"c"}, true,
		"a: x\nb: y\nc: true\n")
}

func TestEditFloatReplacement(t *testing.T) {
	editCheck(t, "ratio: 1\n", []string{"ratio"}, 0.5, "ratio: 0.5\n")
}

// ─── nesting, arrays, keys ────────────────────────────────────────────

func TestEditNestedDottedPath(t *testing.T) {
	editCheck(t, "srv: {\n  host: a\n  port: 1\n}\n",
		[]string{"srv", "host"}, "b.example",
		"srv: {\n  host: b.example\n  port: 1\n}\n")
}

func TestEditDottedSourceKeyLeaf(t *testing.T) {
	editCheck(t, "a.b: 1\nc: 2\n", []string{"a", "b"}, int64(3), "a.b: 3\nc: 2\n")
}

func TestEditArrayIndexReplacement(t *testing.T) {
	editCheck(t, "list: [\n  one\n  two\n]\n",
		[]string{"list", "1"}, "TWO", "list: [\n  one\n  TWO\n]\n")
}

func TestEditNumericSegmentIsStringKeyInObject(t *testing.T) {
	editCheck(t, "0: first\nlist: [\n  x\n]\n", []string{"0"}, "zero",
		"0: zero\nlist: [\n  x\n]\n")
	editCheck(t, "0: first\nlist: [\n  x\n]\n", []string{"list", "0"}, "y",
		"0: first\nlist: [\n  y\n]\n")
}

func TestEditEscapedDotInPathVsRealKey(t *testing.T) {
	editCheck(t, "\"a.b\": 1\na: {\n  b: 2\n}\n", []string{"a.b"}, int64(9),
		"\"a.b\": 9\na: {\n  b: 2\n}\n")
	editCheck(t, "\"a.b\": 1\na: {\n  b: 2\n}\n", []string{"a", "b"}, int64(9),
		"\"a.b\": 1\na: {\n  b: 9\n}\n")
}

func TestEditEscapedBackslashInPath(t *testing.T) {
	editCheck(t, "\"a\\\\b\": 1\n", []string{"a\\b"}, int64(2), "\"a\\\\b\": 2\n")
}

// ─── inline contexts ──────────────────────────────────────────────────

func TestEditInlinePairByteExact(t *testing.T) {
	editCheck(t, "a: {x: 1, y: 2}\n", []string{"a", "y"}, int64(3),
		"a: {x: 1, y: 3}\n")
}

func TestEditInlineRawPair(t *testing.T) {
	editCheck(t, "a: {k:: v, j: w}\n", []string{"a", "k"}, "z",
		"a: {k:: z, j: w}\n")
}

func TestEditInlineCompoundReplacementInlineExact(t *testing.T) {
	editCheck(t, "a: {x:1}\n", []string{"a", "x"},
		map[string]any{"n": int64(1)}, "a: {x:{n: 1}}\n")
}

func TestEditInlineNestedCompoundReplacement(t *testing.T) {
	editCheck(t, "a: {x: 1}\n", []string{"a", "x"},
		map[string]any{"n": []any{map[string]any{"m": int64(2)}}},
		"a: {x: {n: [{m: 2}]}}\n")
}

// ─── containers ───────────────────────────────────────────────────────

func TestEditScalarToCompound(t *testing.T) {
	editCheck(t, "a: 1\n", []string{"a"}, map[string]any{"x": int64(1)},
		"a: {\n    x: 1\n}\n")
}

func TestEditEmptyCompoundToFilled(t *testing.T) {
	editCheck(t, "a: {}\nb: 2\n", []string{"a"}, map[string]any{"x": int64(1)},
		"a: {x: 1}\nb: 2\n")
}

func TestEditCompoundToScalar(t *testing.T) {
	editCheck(t, "a: {\n  x: 1\n}\n", []string{"a"}, int64(5), "a: 5\n")
}

// ─── reopen / noncontiguous ───────────────────────────────────────────

func TestEditReopenLeafEditableWholeTargetUnsupported(t *testing.T) {
	editCheck(t, "a: {}\na.y: 2\n", []string{"a", "y"}, int64(3),
		"a: {}\na.y: 3\n")
	editCheckUnsupported(t, "a: {}\na.y: 2\n", "a", int64(9))
}

// ─── multiline strings ────────────────────────────────────────────────

func TestEditMultilineStringReplacement(t *testing.T) {
	editCheck(t, "s: (\n hi\n)\n", []string{"s"}, "a\nb", "s: ((\na\nb\n))\n")
}

func TestEditMultilineTerminatorLineUnsupported(t *testing.T) {
	editCheckUnsupported(t, "s: (\n hi\n)\n", "s", "x\n))\ny")
}

// ─── byte-exact environment preservation ──────────────────────────────

func TestEditCommentsBlanksAndTrailingBytesPreserved(t *testing.T) {
	editCheck(t, "## top\n\na: 1\n\n## end\n", []string{"a"}, int64(2),
		"## top\n\na: 2\n\n## end\n")
}

func TestEditCRLFDocumentPreserved(t *testing.T) {
	editCheck(t, "a: 1\r\nb: two\r\n", []string{"b"}, "three",
		"a: 1\r\nb: three\r\n")
}

func TestEditCROnlyDocumentPreserved(t *testing.T) {
	editCheck(t, "a: 1\rb: two\r", []string{"b"}, "three", "a: 1\rb: three\r")
}

func TestEditBOMAndNoFinalNewlinePreserved(t *testing.T) {
	editCheck(t, "\uFEFFa: 1", []string{"a"}, int64(2), "\uFEFFa: 2")
}

func TestEditUTF8ValuesAndKeys(t *testing.T) {
	editCheck(t, "ключ: значение\n", []string{"ключ"}, "число", "ключ: число\n")
}

// ─── errors ───────────────────────────────────────────────────────────

func TestEditNotFoundCases(t *testing.T) {
	src := "a: {\n  b: 1\n}\nlist: [\n  x\n]\n"
	for _, path := range []string{"missing", "a.b.c", "list.5", "a.b.c.d"} {
		editCheckClass(t, src, path, "v", "EditNotFound", "")
	}
	editCheckClass(t, "a: scalar\n", "a.b", "v", "EditNotFound", "")
	// Empty segments list.
	if _, err := op().EditValueSegments(src, nil, "v"); err == nil {
		t.Fatal("empty segments must be EditNotFound")
	}
}

func TestEditInvalidPathCases(t *testing.T) {
	for _, tc := range []struct {
		path   string
		offset string
	}{
		{`a\nb`, "1"},
		{`a\`, "1"},
		{"", "0"},
		{"a..b", "2"},
		{".a", "0"},
	} {
		editCheckClass(t, "a: 1\n", tc.path, "v", "EditInvalidPath", tc.offset)
	}
}

func TestEditNonNumericIndexIsNotFound(t *testing.T) {
	editCheckClass(t, "list: [\n  x\n]\n", "list.abc", "v", "EditNotFound", "")
}

func TestEditMalformedSourceIsUnsupportedNotPanic(t *testing.T) {
	// Source validation comes FIRST: a malformed document surfaces the
	// normal parser envelope (UnclosedCompound with opener line), on
	// both backends — never a panic, never an invented EditUnsupported.
	for _, source := range []string{"a: {\n  x: 1\n", "a: {\n"} {
		_, err := op().EditValue(source, "missing", nil)
		if err == nil {
			t.Fatalf("expected the source parser error for %q, got ok", source)
		}
		e, ok := err.(*Error)
		if !ok || e.Class != "UnclosedCompound" {
			t.Fatalf("want the parser envelope (UnclosedCompound), got %#v", err)
		}
		if e.Line < 1 {
			t.Fatalf("UnclosedCompound must carry the opener line, got %#v", e)
		}
	}
}

func TestEditLeadingZeroIndexAndSigns(t *testing.T) {
	editCheck(t, "list: [\n  one\n  two\n]\n", []string{"list", "01"}, "TWO",
		"list: [\n  one\n  TWO\n]\n")
	editCheckClass(t, "list: [\n  one\n  two\n]\n", "list.+1", "x", "EditNotFound", "")
}

// ─── marker / whitespace byte exactness ───────────────────────────────

func TestEditPairSpacingAndEOLPreserved(t *testing.T) {
	editCheck(t, "  a:\t   old  \r\n", []string{"a"}, "new", "  a:\t   new  \r\n")
}

func TestEditIntAfterRawDropsOnlyExtraColon(t *testing.T) {
	editCheck(t, "k::  8080\n", []string{"k"}, int64(9090), "k:  9090\n")
}

func TestEditSameValueNoOpByteIdentical(t *testing.T) {
	editCheck(t, "a: x\n", []string{"a"}, "x", "a: x\n")
	editCheck(t, "n: 1_000\n", []string{"n"}, int64(1000), "n: 1_000\n")
}

func TestEditEmptyBodyPairReplacement(t *testing.T) {
	editCheck(t, "k: \n", []string{"k"}, "x", "k: x \n")
	editCheck(t, "k: \n", []string{"k"}, "", "k: \n")
}

func TestEditNoOpBypassesCommentAndNoncontiguousRefusals(t *testing.T) {
	src := "a: {\n  ## c\n  x: 1\n}\n"
	editCheck(t, src, []string{"a"}, map[string]any{"x": int64(1)}, src)
	src2 := "a: {}\na.y: 2\n"
	editCheck(t, src2, []string{"a"}, map[string]any{"y": int64(2)}, src2)
}

// ─── comments ─────────────────────────────────────────────────────────

func TestEditCommentedContainerStructuralReplacementRejected(t *testing.T) {
	editCheckUnsupported(t, "a: {\n  ## inner\n  x: 1\n}\n", "a",
		map[string]any{"y": int64(2)})
	editCheckUnsupported(t, "a: [\n ## keep\n 1\n]\n", "a", nil)
}

func TestEditLeafInsideCommentedContainerKeepsComments(t *testing.T) {
	editCheck(t, "a: {\n  ## inner\n  x: 1\n}\n", []string{"a", "x"}, int64(2),
		"a: {\n  ## inner\n  x: 2\n}\n")
}

func TestEditHashHashInsideMultilineStringIsData(t *testing.T) {
	editCheck(t, "a: {\n  s: (\n    ## not comment\n  )\n}\n", []string{"a"},
		int64(7), "a: 7\n")
}

func TestEditUnicodeWhitespaceCommentDetected(t *testing.T) {
	src := "a: {\n  \u00A0## c\n  x: 1\n}\n"
	editCheckUnsupported(t, src, "a", map[string]any{"y": int64(2)})
	editCheck(t, src, []string{"a", "x"}, int64(2),
		"a: {\n  \u00A0## c\n  x: 2\n}\n")
}

// ─── root contexts ────────────────────────────────────────────────────

func TestEditImplicitRootArrayFirstItemPairShapedGetsRaw(t *testing.T) {
	editCheck(t, ":: old\nsecond\n", []string{"0"}, "a: b", ":: a: b\nsecond\n")
}

func TestEditImplicitRootArraySecondItemBareOK(t *testing.T) {
	editCheck(t, ":: old\nsecond\n", []string{"1"}, "a: b", ":: old\na: b\n")
}

func TestEditExplicitRootArrayFirstItemRawPreserved(t *testing.T) {
	editCheck(t, "[\n  :: old\n  x\n]\n", []string{"0"}, "k: v",
		"[\n  :: k: v\n  x\n]\n")
}

func TestEditRootInlineChildren(t *testing.T) {
	editCheck(t, "{x: 1}\n", []string{"x"}, int64(2), "{x: 2}\n")
	editCheck(t, "[1, 2]\n", []string{"0"}, int64(7), "[7, 2]\n")
}

func TestEditInlineScalarEscapes(t *testing.T) {
	editCheck(t, "a: {x: 1}\n", []string{"a", "x"}, "y,z", "a: {x: y\\,z}\n")
	editCheck(t, "a: {x: 1}\n", []string{"a", "x"}, "y\\z", "a: {x: y\\\\z}\n")
}

// ─── multiline exact bytes ────────────────────────────────────────────

func TestEditMultilineEdgeWhitespaceStringsLossless(t *testing.T) {
	editCheck(t, "s: (\n hi\n)\n", []string{"s"}, " a ", "s: ((\n a \n))\n")
	editCheck(t, "s: (\n hi\n)\n", []string{"s"}, "\na", "s: ((\n\na\n))\n")
	editCheck(t, "s: (\n hi\n)\n", []string{"s"}, "a\n", "s: ((\na\n\n))\n")
	editCheck(t, "s: (\n hi\n)\n", []string{"s"}, "a\n\n", "s: ((\na\n\n\n))\n")
}

func TestEditCRReplacementRejected(t *testing.T) {
	editCheckUnsupported(t, "s: (\n hi\n)\n", "s", "a\r\nb")
}

func TestEditCompoundWithMultilineStringRoundTrips(t *testing.T) {
	editCheck(t, "a: 1\n", []string{"a"},
		map[string]any{"s": "a\nb"},
		"a: {\n    s: (\n        a\n        b\n    )\n}\n")
}

// ─── physical inline context lives on the leaf ────────────────────────

func TestEditDottedKeyInsideInlineEscapes(t *testing.T) {
	// Inline pair keys are dotted-expanded by the parser (a → b → c),
	// and the leaf keeps a real slot: the splice replaces exactly the
	// body bytes inside the inline braces (Rust: edit_value "a.b.c").
	editCheck(t, "a: {b.c: old}\n", []string{"a", "b", "c"}, "y,z",
		"a: {b.c: y\\,z}\n")
}

func TestEditReopenedFormerInlineChildIsPhysicalLine(t *testing.T) {
	editCheck(t, "a: {x:1}\na.y: y,z\n", []string{"a", "y"}, "q,w",
		"a: {x:1}\na.y: q,w\n")
}

func TestEditInlineArrayItemColonColonIsLiteralData(t *testing.T) {
	editCheck(t, "[:: old]\n", []string{"0"}, "new", "[new]\n")
	editCheck(t, "[x]\n", []string{"0"}, ":: new", "[:: new]\n")
}

// ─── multiline block roles ────────────────────────────────────────────

func TestEditBlockItemEmptyStringBecomesUnitForm(t *testing.T) {
	editCheck(t, "a: [\n(\nx\n)\n]\n", []string{"a", "0"}, "", "a: [\n()\n]\n")
}

func TestEditBlockItemCommentLikeGetsItemRaw(t *testing.T) {
	editCheck(t, "a: [\n(\nx\n)\n]\n", []string{"a", "0"}, "## text",
		"a: [\n:: ## text\n]\n")
	editCheck(t, "a: [\n(\nx\n)\n]\n", []string{"a", "0"}, "]",
		"a: [\n:: ]\n]\n")
}

func TestEditBlockPairEmptyStringStaysValid(t *testing.T) {
	editCheck(t, "s: (\nx\n)\n", []string{"s"}, "", "s: \n")
}

func TestEditBlockPairStringReplacement(t *testing.T) {
	editCheck(t, "s: (\nx\n)\n", []string{"s"}, "new", "s: new\n")
	editCheck(t, "s:\t (\nx\n)\n", []string{"s"}, "42", "s::\t 42\n")
}

// ─── inline writer provenance ─────────────────────────────────────────

func TestEditInlineItemProvenance(t *testing.T) {
	editCheck(t, "[old]\n", []string{"0"}, "true", "[\\u0074rue]\n")
	editCheck(t, "[old]\n", []string{"0"}, "42", "[\\u00342]\n")
	editCheck(t, "[old]\n", []string{"0"}, "", "[()]\n")
	editCheck(t, "a: {x: old}\n", []string{"a", "x"}, " café\u00A0",
		"a: {x: \\u0020café\\u00A0}\n")
}

func TestEditInlineRawPairTransitions(t *testing.T) {
	editCheck(t, "a: {x::  old}\n", []string{"a", "x"}, int64(9), "a: {x:  9}\n")
	editCheck(t, "a: {x:: old}\n", []string{"a", "x"}, "y}z", "a: {x:: y\\}z}\n")
	editCheck(t, "a: {x:: old}\n", []string{"a", "x"},
		map[string]any{"v": "true"}, "a: {x: {v: \\u0074rue}}\n")
	editCheck(t, "a: {x:: old}\n", []string{"a", "x"},
		map[string]any{"v": []any{""}}, "a: {x: {v: [()]}}\n")
}

func TestEditWholeInlineContainerTargetIsLineBare(t *testing.T) {
	editCheck(t, "a: {x:1}\n", []string{"a"}, "y,z", "a: y,z\n")
}

func TestEditStreamingInlineWriterPreservesMixedValues(t *testing.T) {
	editCheck(t, "a: {x: old}\n", []string{"a", "x"},
		map[string]any{
			"flag":  "true",
			"items": []any{"", false, nil, int64(2)},
		},
		"a: {x: {flag: \\u0074rue, items: [(), false, null, 2]}}\n")
}

// ─── marker transitions with exact bytes ──────────────────────────────

func TestEditPlainToRawKeepsWideSpacing(t *testing.T) {
	editCheck(t, "n:  5\n", []string{"n"}, "42", "n::  42\n")
	editCheck(t, "k: \tv\n", []string{"k"}, "42", "k:: \t42\n")
	editCheck(t, "k::  \tv\n", []string{"k"}, int64(9), "k:  \t9\n")
}

func TestEditRawToMultilineSwitchesToPlainBlock(t *testing.T) {
	editCheck(t, "k:: x\n", []string{"k"}, "a\nb", "k: ((\na\nb\n))\n")
}

func TestEditItemRawToPlainDropsMarkerKeepsIndent(t *testing.T) {
	editCheck(t, ":: 8080\nsecond\n", []string{"0"}, int64(9090),
		" 9090\nsecond\n")
	editCheck(t, "  ::\t 9\nsecond\n", []string{"0"}, int64(10),
		"  \t 10\nsecond\n")
}

func TestEditPlainToRawItemInsertsExactMarker(t *testing.T) {
	editCheck(t, "  5\nsecond\n", []string{"0"}, "## c", "  :: ## c\nsecond\n")
}

func TestEditScalarToEmptyArrayAndBack(t *testing.T) {
	editCheck(t, "a: 1\n", []string{"a"}, []any{}, "a: []\n")
	editCheck(t, "a: []\n", []string{"a"}, int64(1), "a: 1\n")
}

func TestEditVerificationEmptyObjectReplacement(t *testing.T) {
	// map[string]any{} is a VALID empty Object replacement (Rust:
	// plan_value renders it as "{}" in a non-block pair slot) — the
	// verification guard must accept it and pin the exact bytes.
	editCheck(t, "a: 1\n", []string{"a"}, map[string]any{}, "a: {}\n")
	// Empty Object into a BLOCK target keeps the block open/closer form.
	editCheck(t, "a: {\n  x: 1\n}\n", []string{"a"}, map[string]any{}, "a: {\n}\n")
}

func TestEditVerificationPreservesNullSiblings(t *testing.T) {
	editCheck(t, "a: [1, null]\nb: keep\n", []string{"a", "0"}, nil,
		"a: [null, null]\nb: keep\n")
}

func TestEditSpanOnlyMultilineStatePreservesData(t *testing.T) {
	for _, pair := range [][2]string{{"(", ")"}, {"((", "))"}} {
		source := "blob: " + pair[0] + "\n## data\n[not a bracket\n" + pair[1] + "\nouter: {x: 1}\n"
		editCheck(t, source, []string{"outer", "x"}, int64(2),
			"blob: "+pair[0]+"\n## data\n[not a bracket\n"+pair[1]+"\nouter: {x: 2}\n")
	}
}

func TestEditInlineRootPreservesQuotedKeysAndCRLF(t *testing.T) {
	editCheck(t, "{\"a.b\": [1, {x: old}]}\r\n", []string{"a.b", "1", "x"}, "new",
		"{\"a.b\": [1, {x: new}]}\r\n")
}

func TestEditSegmentsLiteralNoExtraUnescaping(t *testing.T) {
	// Segment "a\\b" is the literal two-backslash key, NOT `a\b`.
	editCheck(t, "\"a\\\\\\\\b\": 1\n", []string{"a\\\\b"}, int64(2),
		"\"a\\\\\\\\b\": 2\n")
}

func TestEditMixedPathSegments(t *testing.T) {
	editCheck(t, "\"a.b\": {\n  c: {\n    ключ: 1\n  }\n  ключ: 2\n}\n",
		[]string{"a.b", "c", "ключ"}, int64(9),
		"\"a.b\": {\n  c: {\n    ключ: 9\n  }\n  ключ: 2\n}\n")
	editCheck(t, "\"x\\\\y\": {\n  tail: 1\n}\n", []string{"x\\y", "tail"}, int64(2),
		"\"x\\\\y\": {\n  tail: 2\n}\n")
}

func TestEditMultilineUsesDocumentCRLF(t *testing.T) {
	editCheck(t, "s: (\r\n hi\r\n)\r\n", []string{"s"}, "a\nb",
		"s: ((\r\na\r\nb\r\n))\r\n")
	editCheck(t, "s: (\r\n hi\r\n)\r\n", []string{"s"}, "a\n",
		"s: ((\r\na\r\n\r\n))\r\n")
}

func TestEditMultilineUnicodeEdgeWhitespaceVerbatim(t *testing.T) {
	editCheck(t, "s: (\nx\n)\n", []string{"s"}, "\u00A0café ",
		"s: ((\n\u00A0café \n))\n")
}

func TestEditMultilineIndentedCloserLineRejected(t *testing.T) {
	editCheckUnsupported(t, "s: (\nx\n)\n", "s", "x\n  ))")
	editCheckUnsupported(t, "s: (\nx\n)\n", "s", "x\n\t))")
	editCheck(t, "s: (\nx\n)\n", []string{"s"}, "x\n))x", "s: ((\nx\n))x\n))\n")
}

// ─── layout regressions (the 22) ──────────────────────────────────────

func TestLayoutDeepLeafPreservesEachSourceForm(t *testing.T) {
	for _, source := range []string{
		"targets.storagehost.address: old\ntargets.storagehost.user: mcp-agent\n",
		"targets: {storagehost: {address: old, user: mcp-agent}}\n",
		"targets: {\n  storagehost: {\n    address: old\n    user: mcp-agent\n  }\n}\n",
	} {
		editCheck(t, source, []string{"targets", "storagehost", "address"},
			"192.168.0.88:22",
			replaceOnceAll(source, "old", "192.168.0.88:22"))
	}
}

func replaceOnceAll(s, old, new string) string {
	return strings.ReplaceAll(s, old, new)
}

func TestLayoutDeepArrayLeafPreservesMixedForms(t *testing.T) {
	for _, source := range []string{
		"targets: [{storagehost: [old, keep]}, other]\n",
		"targets: [\n  {storagehost: [old, keep]}\n  other\n]\n",
		"targets: [\n  {\n    storagehost: [\n      old\n      keep\n    ]\n  }\n  other\n]\n",
	} {
		editCheck(t, source, []string{"targets", "0", "storagehost", "0"}, "new",
			replaceOnceAll(source, "old", "new"))
	}
}

func TestLayoutWholeInlineContainersRemainInline(t *testing.T) {
	editCheck(t, "a: {x: 1}\n", []string{"a"}, map[string]any{"y": 2}, "a: {y: 2}\n")
	editCheck(t, "a: [1, 2]\n", []string{"a"}, []any{3, 4}, "a: [3, 4]\n")
	editCheck(t, "a: {}\n", []string{"a"}, map[string]any{"y": 2}, "a: {y: 2}\n")
	editCheck(t, "a: []\n", []string{"a"}, []any{3, 4}, "a: [3, 4]\n")
	editCheck(t, "a: {}\n", []string{"a"}, []any{3, 4}, "a: [3, 4]\n")
}

func TestLayoutEmptyingBlockContainersDoesNotCollapse(t *testing.T) {
	editCheck(t, "a: {\n  x: 1\n}\n", []string{"a"}, map[string]any{}, "a: {\n}\n")
	editCheck(t, "a: [\n  1\n]\n", []string{"a"}, []any{}, "a: [\n]\n")
}

func TestLayoutNewlineUnfoldsOnlyInlineAncestorChain(t *testing.T) {
	editCheck(t, "## top\na: [{x: old, untouched: [1, 2]}, {keep: yes}]\nz: keep\n",
		[]string{"a", "0", "x"}, "one\ntwo",
		"## top\na: [\n    {\n        x: ((\none\ntwo\n        ))\n        untouched: [1, 2]\n    }\n    {keep: yes}\n]\nz: keep\n")
}

func TestLayoutNewlineUnderDottedKeyDoesNotExpandDottedAncestors(t *testing.T) {
	editCheck(t, "a.b: [old, keep]\na.c: other\n",
		[]string{"a", "b", "0"}, "one\ntwo",
		"a.b: [\n    ((\none\ntwo\n    ))\n    keep\n]\na.c: other\n")
}

func TestLayoutDenseEscapesUnfoldButSparseDoNot(t *testing.T) {
	editCheck(t, "a: [old, keep]\n", []string{"a", "0"}, "a,b,c,d,e",
		"a: [\n    a,b,c,d,e\n    keep\n]\n")
	editCheck(t, "a: [old]\n", []string{"a", "0"}, "a,b,c,d",
		"a: [a\\,b\\,c\\,d]\n")
	sparse := "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxx" + "a,b,c,d,e"
	editCheck(t, "a: [old]\n", []string{"a", "0"}, sparse,
		"a: ["+replaceOnceAll(sparse, ",", "\\,")+"]\n")
}

func TestLayoutWidthBoundaryIsInCharactersNotBytes(t *testing.T) {
	value95 := repeatRune("я", 95)
	editCheck(t, "a: [old]\n", []string{"a", "0"}, value95, "a: ["+value95+"]\n")
	value96 := repeatRune("я", 96)
	editCheck(t, "a: [old]\n", []string{"a", "0"}, value96,
		"a: [\n    "+value96+"\n]\n")
}

func repeatRune(s string, n int) string {
	out := ""
	for i := 0; i < n; i++ {
		out += s
	}
	return out
}

func TestLayoutAlreadyLongLineOnlyUnfoldsWhenItGrows(t *testing.T) {
	x120 := repeatRune("x", 120)
	y120 := repeatRune("y", 120)
	source := "a: [" + x120 + ", keep]\n"
	editCheck(t, source, []string{"a", "0"}, y120, "a: ["+y120+", keep]\n")
	y121 := repeatRune("y", 121)
	editCheck(t, source, []string{"a", "0"}, y121,
		"a: [\n    "+y121+"\n    keep\n]\n")
}

func TestLayoutWidthExpansionStopsWhenInnerLineFits(t *testing.T) {
	value := repeatRune("x", 91)
	editCheck(t, "a: [{x: old}]\n", []string{"a", "0", "x"}, value,
		"a: [\n    {x: "+value+"}\n]\n")
}

func TestLayoutEscapeOverheadBoundaryIsExact(t *testing.T) {
	dense := repeatRune("x", 7) + "a,b,c,d,e"
	sparse := repeatRune("x", 8) + "a,b,c,d,e"
	editCheck(t, "a: [old]\n", []string{"a", "0"}, dense,
		"a: [\n    "+dense+"\n]\n")
	editCheck(t, "a: [old]\n", []string{"a", "0"}, sparse,
		"a: ["+replaceOnceAll(sparse, ",", "\\,")+"]\n")
}

func TestLayoutLongChangedLeafDoesNotUnfoldUnrelatedBranches(t *testing.T) {
	value := repeatRune("x", 100)
	editCheck(t, "a: [old, {untouched: [1, 2]}]\n", []string{"a", "0"}, value,
		"a: [\n    "+value+"\n    {untouched: [1, 2]}\n]\n")
}

func TestLayoutPromotedSiblingsKeepValuesWhenEscapesLoseMeaning(t *testing.T) {
	editCheck(t, "a: [old, a\\,b, \\u00342, :: literal]\n",
		[]string{"a", "0"}, "x\ny",
		"a: [\n    ((\nx\ny\n    ))\n    a,b\n    :: 42\n    :: :: literal\n]\n")
}

func TestLayoutPromotedPairsKeepKeySpellingAndRawMarkers(t *testing.T) {
	editCheck(t, "a: {\"x.y\": old, nested.key::  42, n: 0007}\n",
		[]string{"a", "x.y"}, "x\ny",
		"a: {\n    \"x.y\": ((\nx\ny\n    ))\n    nested.key::  42\n    n: 0007\n}\n")
}

func TestLayoutRootInlinePromotionPreservesBOMAndEOF(t *testing.T) {
	editCheck(t, "\uFEFF[old, keep]", []string{"0"}, "x\ny",
		"\uFEFF[\n    ((\nx\ny\n    ))\n    keep\n]")
}

func TestLayoutPromotionPreservesEOLCommentsAndNoFinalTerminator(t *testing.T) {
	for _, eol := range []string{"\n", "\r", "\r\n"} {
		source := "## header" + eol + "a: [old, keep]"
		expected := "## header" + eol + "a: [" + eol + "    ((" + eol + "x" + eol + "y" + eol + "    ))" + eol + "    keep" + eol + "]"
		editCheck(t, source, []string{"a", "0"}, "x\ny", expected)
	}
}

func TestLayoutCommentsOutsidePromotedValueRemainByteExact(t *testing.T) {
	editCheck(t, "list: [\n  ## before\n  {x: old}\n  ## after\n]\n",
		[]string{"list", "0", "x"}, "x\ny",
		"list: [\n  ## before\n  {\n      x: ((\nx\ny\n      ))\n  }\n  ## after\n]\n")
}

func TestLayoutNoopIgnoresAllReadabilityThresholds(t *testing.T) {
	x140 := repeatRune("x", 140)
	source := "a: [" + x140 + ", a\\,b\\,c\\,d\\,e]\n"
	editCheck(t, source, []string{"a", "0"}, x140, source)
	editCheck(t, source, []string{"a", "1"}, "a,b,c,d,e", source)
}

func TestLayoutTypeChangeDoesNotForceUnrelatedLayoutChanges(t *testing.T) {
	editCheck(t, "a: {x: [1], keep: [2]}\n", []string{"a", "x"}, true,
		"a: {x: true, keep: [2]}\n")
	editCheck(t, "a: [1]\n", []string{"a"}, "42", "a:: 42\n")
}

func TestLayoutReopenedInlineParentPromotesOnlyItsPhysicalDefinition(t *testing.T) {
	editCheck(t, "a: {x: old}\n## gap\na.y: keep\n", []string{"a", "x"}, "x\ny",
		"a: {\n    x: ((\nx\ny\n    ))\n}\n## gap\na.y: keep\n")
}

func TestLayoutImpossibleBlockEncodingKeepsValidInlineForm(t *testing.T) {
	editCheck(t, "a: [old]\n", []string{"a", "0"}, "a\rb", "a: [a\\rb]\n")
	editCheck(t, "a: [old]\n", []string{"a", "0"}, "a\n))\nb", "a: [a\\n))\\nb]\n")
	editCheck(t, "a: [old, a\\rb]\n", []string{"a", "0"}, "x\ny",
		"a: [x\\ny, a\\rb]\n")
}

func TestLayoutUnsafeCommentLossStillFailsClosed(t *testing.T) {
	for _, source := range []string{"a: {\n ## keep\n x: 1\n}\n", "a: [\n ## keep\n 1\n]\n"} {
		editCheckUnsupported(t, source, "a", nil)
	}
}

// TestEditNilAndEmptySegmentsAreNotFound: a nil []string IS the empty
// path by contract — EditNotFound after source validation, not a
// transport Message (json.Marshal(nil) would produce JSON null).
func TestEditNilAndEmptySegmentsAreNotFound(t *testing.T) {
	for _, tc := range []struct {
		name     string
		segments []string
	}{
		{"nil slice", nil},
		{"empty slice", []string{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := op().EditValueSegments("a: 1\n", tc.segments, "v")
			if err == nil {
				t.Fatal("expected EditNotFound, got ok")
			}
			e, ok := err.(*Error)
			if !ok || e.Class != "EditNotFound" {
				t.Fatalf("want EditNotFound, got %#v", err)
			}
		})
	}
}

// TestEditBrokenSourceBeforeEmptySegments: source validation comes
// FIRST — a nil (empty) path against a malformed document surfaces the
// real parser envelope, never EditNotFound or a transport Message.
func TestEditBrokenSourceBeforeEmptySegments(t *testing.T) {
	_, err := op().EditValueSegments("a: {\n", nil, "v")
	if err == nil {
		t.Fatal("expected the source parse error, got ok")
	}
	e, ok := err.(*Error)
	if !ok || e.Class != "UnclosedCompound" {
		t.Fatalf("want the source diagnostic (UnclosedCompound), got %#v", err)
	}
}

// TestEditLiteralUnicodeTextualPath exercises the TEXTUAL path form with
// multi-byte UTF-8 directly (not through the segments helper): escaped
// dot, plain Cyrillic neighbour, and the byte-offset rule for an empty
// segment after Unicode ('ключ' = 8 bytes → offset 9).
func TestEditLiteralUnicodeTextualPath(t *testing.T) {
	src := "\"ключ.точка\": 1\nключ: 2\n"
	got, err := op().EditValue(src, `ключ\.точка`, int64(9))
	if err != nil {
		t.Fatalf("EditValue: %v", err)
	}
	if got != "\"ключ.точка\": 9\nключ: 2\n" {
		t.Fatalf("output = %q", got)
	}
	got2, err := op().EditValue(src, "ключ", int64(9))
	if err != nil {
		t.Fatalf("EditValue: %v", err)
	}
	if got2 != "\"ключ.точка\": 1\nключ: 9\n" {
		t.Fatalf("output = %q", got2)
	}
	_, err = op().EditValue(src, "ключ..точка", "v")
	if err == nil {
		t.Fatal("expected EditInvalidPath")
	}
	he, ok := err.(*Error)
	if !ok || he.Class != "EditInvalidPath" {
		t.Fatalf("want EditInvalidPath, got %#v", err)
	}
	if he.Body != "9" {
		t.Fatalf("InvalidPath body = %q, want \"9\"", he.Body)
	}
}

// Scalar ROOTS through the edit replacement encoder: String/Bool/Null
// wire to plain values and Int/Float to single-entry tagged maps — none
// may trip the dumps-family ScalarRoot guard or panic in the wire walk
// (a scalar root has nothing to walk; prepareWriterValue must be total).
func TestEditScalarRootReplacementsAcrossTypes(t *testing.T) {
	cases := []struct {
		name string
		src  string
		path string
		repl any
		want string
	}{
		{"string", "port: 8080\nhost: db\n", "port", "db.internal", "port: db.internal\nhost: db\n"},
		{"int", "port:: 8080\n", "port", int64(9090), "port: 9090\n"},
		{"bool", "a: x\n", "a", true, "a: true\n"},
		{"null", "a: x\n", "a", nil, "a: null\n"},
		{"float", "ratio: 1\n", "ratio", 0.5, "ratio: 0.5\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := op().EditValue(tc.src, tc.path, tc.repl)
			if err != nil || got != tc.want {
				t.Fatalf("public EditValue = %q, %v; want %q", got, err, tc.want)
			}
			wire, werr := encodeEditReplacementTagged(tc.repl)
			if werr != nil {
				t.Fatalf("wire encode: %v", werr)
			}
			raw, err := PureEditValue([]byte(tc.src), []byte(tc.path), wire)
			if err != nil || string(raw) != tc.want {
				t.Fatalf("byte seam = %q, %v; want %q", raw, err, tc.want)
			}
		})
	}
}

// The packaged pure consumer's exact smoke case: an int replacement in
// a document with leading/trailing comments, both path forms.
func TestEditConsumerSmokeIntReplacement(t *testing.T) {
	src := "## top\n\na: 1\n\n## end\n"
	want := "## top\n\na: 2\n\n## end\n"
	got, err := op().EditValue(src, "a", int64(2))
	if err != nil || got != want {
		t.Fatalf("EditValue = %q, %v; want %q", got, err, want)
	}
	got, err = op().EditValueSegments(src, []string{"a"}, int64(2))
	if err != nil || got != want {
		t.Fatalf("EditValueSegments = %q, %v; want %q", got, err, want)
	}
}

// Извлечение значения из свежего дерева обнуляет только целевой слот.
func TestEditVerificationTakeOnFreshParse(t *testing.T) {
	doc, _, perr := parseDoc("k: v\nnested: {x: 1}\nlist: [one, two]\n", false, false)
	if perr != nil {
		t.Fatalf("parse: %v", perr)
	}
	root := doc.(*pvalue)

	got, terr := editTakeTarget(root, []string{"k"})
	if terr != nil {
		t.Fatalf("take k: %v", terr)
	}
	if got.kind != kStr || got.s != "v" {
		t.Fatalf("take k = %#v, want String \"v\"", got)
	}
	if slot := root.obj.get("k"); slot == nil || slot.kind != kNull {
		t.Fatalf("k slot after take = %#v, want null", slot)
	}

	got, terr = editTakeTarget(root, []string{"nested", "x"})
	if terr != nil {
		t.Fatalf("take nested.x: %v", terr)
	}
	if got.kind != kInt || got.s != "1" {
		t.Fatalf("take nested.x = %#v, want Integer 1", got)
	}

	got, terr = editTakeTarget(root, []string{"list", "1"})
	if terr != nil {
		t.Fatalf("take list.1: %v", terr)
	}
	if got.kind != kStr || got.s != "two" {
		t.Fatalf("take list.1 = %#v, want String \"two\"", got)
	}
}

// ─── transport strictness (source-less Message envelopes) ─────────────
// The byte seams must mirror the C ABI's transport layer: invalid UTF-8
// or surrogate-laden transports are Message-class errors, and
// encoding/json's silent normalisation of unpaired surrogate escapes
// must never produce a silently different pure result.

func editCheckMessage(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected transport Message error, got ok")
	}
	e, ok := err.(*Error)
	if !ok || e.Class != "Message" {
		t.Fatalf("want Message-class transport error, got %#v", err)
	}
	// Source-less: no invented locations.
	if e.Line != 0 || e.LineText != "" || e.Span != nil || e.Path != nil {
		t.Fatalf("transport error must carry no source location: %#v", e)
	}
}

func TestEditTransportPathInvalidUTF8(t *testing.T) {
	_, err := PureEditValue([]byte("a: 1\n"), []byte{'a', 0x80, 'b'}, []byte("null"))
	editCheckMessage(t, err)
}

func TestEditTransportPathEncodedSurrogate(t *testing.T) {
	// ED A0 80 is a surrogate encoded in UTF-8: both Rust's
	// std::str::from_utf8 and Go's utf8.Valid reject it (surrogates are
	// never valid UTF-8), so the seam rejects with a Message envelope.
	_, err := PureEditValue([]byte("a: 1\n"), []byte{'a', 0xED, 0xA0, 0x80}, []byte("null"))
	editCheckMessage(t, err)
}

func TestEditTransportSegmentsWire(t *testing.T) {
	src := []byte("a: 1\n")
	cases := []struct {
		name string
		path string
	}{
		{"non-string element", `["a", 1]`},
		{"object element", `["a", {}]`},
		{"unpaired high surrogate", `["a", "\ud800b"]`},
		{"lone low surrogate", `["a", "\udc00"]`},
		{"high followed by non-surrogate", `["\ud800x"]`},
		{"trailing data", `["a"] x`},
		{"invalid utf8 byte", "[" + string([]byte{0x80}) + "]"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := PureEditValueSegments(src, []byte(tc.path), []byte("2"))
			editCheckMessage(t, err)
		})
	}
}

func TestEditTransportSegmentsPairedSurrogateOK(t *testing.T) {
	// A properly paired escape decodes to the same rune on both sides:
	// it must be accepted and address the decoded key.
	_, err := PureEditValueSegments([]byte("\"к acima\": 1\n"),
		[]byte("[\"\\ud83d\\ude00\"]"), []byte("2"))
	if err == nil {
		t.Fatal("expected NotFound for an absent paired-surrogate key, got ok")
	}
	e, ok := err.(*Error)
	if !ok || e.Class != "EditNotFound" {
		t.Fatalf("paired surrogate must pass transport and fail lookup, got %#v", err)
	}
}

func TestEditTransportReplacementWire(t *testing.T) {
	// Go заменяет недопустимую руну валидным U+FFFD ещё до вызова API.
	repl := string(rune(0xD800)) + "x"
	got, err := op().EditValue("a: 1\n", "a", repl)
	if err != nil || got != "a: �x\n" {
		t.Fatalf("valid replacement character: output=%q error=%v", got, err)
	}
	_, err = PureEditValue([]byte("a: 1\n"), []byte("a"), []byte(`"\ud800x"`))
	editCheckMessage(t, err)
	_, err = PureEditValueSegments([]byte("a: 1\n"), []byte(`["a"]`), []byte(`"\ud800x"`))
	editCheckMessage(t, err)
	// Encoded surrogate bytes inside the wire JSON likewise.
	_, err = PureEditValue([]byte("a: 1\n"), []byte("a"), []byte{'"', 0xED, 0xA0, 0x80, '"'})
	editCheckMessage(t, err)
}

func TestEditTransportEscapeParityAcceptsLiteralBackslashU(t *testing.T) {
	// `\\uD800` in the wire JSON is a literal backslash followed by the
	// text uD800 — NOT a surrogate escape. It must be accepted (and the
	// unpaired-surrogate scanner must not fire on it), on BOTH spellings:
	// after an escaped backslash and after other escapes.
	_, err := PureEditValue([]byte("a: 1\n"), []byte("a"), []byte(`"\\uD800"`))
	if err != nil {
		t.Fatalf("literal backslash+uD800 rejected: %v", err)
	}
	out, err := op().EditValue("a: 1\n", "a", `\uD800`)
	if err != nil {
		t.Fatalf("literal backslash+uD800 rejected via public API: %v", err)
	}
	if out != "a: \\uD800\n" {
		t.Fatalf("output = %q, want %q", out, "a: \\uD800\n")
	}
	// An escaped backslash inside a longer string, with a real escape
	// later in the same buffer: parity must survive `\\\uXXXX`.
	_, err = PureEditValueSegments([]byte("a: 1\n"), []byte(`["a"]`),
		[]byte(`"x\\y\\uD800"`))
	if err != nil {
		t.Fatalf("literal backslash before uD800 rejected: %v", err)
	}
}

func TestEditTransportPairedSurrogateEscapeAccepted(t *testing.T) {
	// A correctly paired \ud83d\ude00 escape decodes to U+1F600 on both
	// sides: accepted, spliced, and re-parsed to the same String.
	out, err := PureEditValue([]byte("a: 1\n"), []byte("a"), []byte(`"\ud83d\ude00"`))
	if err != nil {
		t.Fatalf("paired surrogate escape rejected: %v", err)
	}
	doc, _, perr := parseDoc(string(out), false, false)
	if perr != nil {
		t.Fatalf("patch does not parse: %v\n%s", perr, out)
	}
	root := doc.(*pvalue)
	v := root.obj.get("a")
	if v == nil || v.kind != kStr || v.s != "\U0001F600" {
		t.Fatalf("paired escape decoded to %#v, want U+1F600", v)
	}
	_, err = PureEditValueSegments([]byte("a: 1\n"), []byte(`["a"]`), []byte(`"\ud83d\ude00 ok"`))
	if err != nil {
		t.Fatalf("paired surrogate escape rejected in segments wire: %v", err)
	}
}

func TestEditTransportSegmentsTrailingGarbage(t *testing.T) {
	// Malformed trailing garbage (not a clean EOF) must fail as a
	// Message, never pass as success.
	for _, tc := range []struct{ name, path string }{
		{"trailing brace", `["a"]}break`},
		{"trailing token", `["a"] oops`},
		{"truncated second value", `["a"] "unterminated`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := PureEditValueSegments([]byte("a: 1\n"), []byte(tc.path), []byte("2"))
			editCheckMessage(t, err)
		})
	}
}
