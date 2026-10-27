# ktav — Go bindings

[![Go Reference](https://pkg.go.dev/badge/github.com/ktav-lang/golang.svg)](https://pkg.go.dev/github.com/ktav-lang/golang)
[![CI](https://img.shields.io/github/actions/workflow/status/ktav-lang/golang/ci.yml?style=flat-square&logo=github&label=CI)](https://github.com/ktav-lang/golang/actions)
![License: MIT OR Apache-2.0](https://img.shields.io/badge/license-MIT%20OR%20Apache--2.0-blue?style=flat-square)
[![Playground](https://img.shields.io/badge/playground-try%20online-7c3aed?style=flat-square&logo=rocket&logoColor=white)](https://ktav-lang.github.io/)

**Languages:** **English** · [Русский](docs/README.ru.md) · [简体中文](docs/README.zh.md)

**Playground:** convert JSON / YAML / TOML / INI ⇄ Ktav in your browser at **[ktav-lang.github.io](https://ktav-lang.github.io/)**.

Go bindings for the [Ktav configuration format](https://github.com/ktav-lang/spec).
Thin wrapper around the reference Rust parser, loaded at runtime through
[`purego`](https://github.com/ebitengine/purego) — so **no `cgo` on the
consumer side**, standard `go build` just works.

```bash
go get github.com/ktav-lang/golang
```

## Quick start

### Parse — decode straight into a typed struct

```go
package main

import (
    "fmt"

    ktav "github.com/ktav-lang/golang"
)

const src = `
service: web
port: 8080
ratio: 0.75
tls: true
tags: [
    prod
    eu-west-1
]
db.host: primary.internal
db.timeout: 30
`

type Config struct {
    Service string   `json:"service"`
    Port    int64    `json:"port"`
    Ratio   float64  `json:"ratio"`
    TLS     bool     `json:"tls"`
    Tags    []string `json:"tags"`
    DB      struct {
        Host    string `json:"host"`
        Timeout int64  `json:"timeout"`
    } `json:"db"`
}

func main() {
    var cfg Config
    if err := ktav.LoadsInto(src, &cfg); err != nil {
        panic(err)
    }
    fmt.Printf("port=%d host=%s timeout=%ds\n",
        cfg.Port, cfg.DB.Host, cfg.DB.Timeout)
}
```

### Walk — work with the dynamic shape, dispatch on type

```go
dyn, _ := ktav.Loads(src)
for k, v := range dyn.(map[string]any) {
    switch x := v.(type) {
    case bool:           fmt.Printf("%s is bool=%v\n", k, x)
    case int64:          fmt.Printf("%s is int=%d\n", k, x)
    case float64:        fmt.Printf("%s is float=%g\n", k, x)
    case string:         fmt.Printf("%s is str=%q\n", k, x)
    case []any:          fmt.Printf("%s is array(%d)\n", k, len(x))
    case map[string]any: fmt.Printf("%s is object(%d)\n", k, len(x))
    case nil:            fmt.Printf("%s is null\n", k)
    }
}
```

### Build & render — construct a document in code

```go
doc := map[string]any{
    "name":  "frontend",
    "port":  int64(8443),
    "tls":   true,
    "ratio": 0.95,
    "upstreams": []any{
        map[string]any{"host": "a.example", "port": int64(1080)},
        map[string]any{"host": "b.example", "port": int64(1080)},
    },
    "notes": nil,
}
out, _ := ktav.Dumps(doc)
fmt.Print(out)
// name: frontend
// notes: null
// port: 8443
// ratio: 0.95
// tls: true
// upstreams: [
//     {
//         host: a.example
//         port: 1080
//     }
//     {
//         host: b.example
//         port: 1080
//     }
// ]
```

`encoding/json` sorts Go map keys, so this output order is deterministic;
the `map[string]any` itself does not preserve insertion order.

A complete runnable version lives in [`examples/basic`](examples/basic/main.go).

## API

| Function | Purpose |
| --- | --- |
| `Loads(s string) (any, error)` | Parse a Ktav document into native Go values. Top-level may be an Object (`map[string]any`) or an Array (`[]any`) per spec § 5.0.1. |
| `LoadsStrict(s string) (any, error)` | Parse with strict numeric spelling checks, using the same Go type mapping as `Loads`. |
| `LoadsInto(s string, target any) error` | Parse into an arbitrary `target` (struct, map, …) via `encoding/json`. |
| `Dumps(v any) (string, error)` | Render a Go value as Ktav text. Top-level must encode to an object or array. |
| `DumpsForceStrings(v any) (string, error)` | Like `Dumps`, but coerces every leaf scalar (integer, float, bool, null) to a String via the raw `::` marker. Compounds preserve their structure. |
| `EmitCanonical(v any) (string, error)` | Render a Go value as canonical Ktav (spec § 5.9). Go maps are encoded through `encoding/json`, which sorts string keys lexicographically. |
| `CanonicalFromSource(src string) (string, error)` | Parse Ktav and immediately emit canonical form, preserving source key order. |
| `FormatSource(src string) (string, error)` | Format Ktav source into its normalised spelling, **keeping every comment**. See below. |
| `EditValue(src, path string, replacement any) (string, error)` | Replace the value addressed by the textual dotted path **in the source text** and return the patched source. See [Source editing](#source-editing). |
| `EditValueSegments(src string, segments []string, replacement any) (string, error)` | Same, with pre-decoded literal segments (no extra unescaping). |
| `InsertValue(src, path string, value any) (string, error)` | Insert a value at the textual dotted path in the source text. |
| `InsertValueSegments(src string, segments []string, value any) (string, error)` | Same, with pre-decoded literal segments (no extra unescaping). |
| `RemoveValue(src, path string) (string, error)` | Remove the value addressed by the textual dotted path from the source text. |
| `RemoveValueSegments(src string, segments []string) (string, error)` | Same, with pre-decoded literal segments (no extra unescaping). |
| `RenameKey(src, path, newKey string) (string, error)` | Rename the key addressed by the textual dotted path in the source text. |
| `RenameKeySegments(src string, segments []string, newKey string) (string, error)` | Same, with pre-decoded literal segments (no extra unescaping). |

## Formatting — canonical spelling, comments kept

`FormatSource` and `EmitCanonical` are different operations:

- **`EmitCanonical`** takes a Go value and writes canonical form. Comments
  are not part of a value and cannot survive.
- **`FormatSource`** takes source text and normalizes its spelling while
  preserving every whole-line comment (spec § 3.4). Both it and
  `CanonicalFromSource` preserve source key order.

The snippets below use these imports:

```go
import (
    "fmt"

    ktav "github.com/ktav-lang/golang"
)
```

```go
func formatExample() error {
	out, err := ktav.FormatSource("## why\na:   {x: 1}\n")
	if err != nil {
		return err
	}
	fmt.Print(out)
	return nil
}
```

Blank lines are grouping hints: a run of two or more becomes one, and
blank padding immediately inside a bracket is dropped.

```go
func formatBlankLinesExample() error {
	out, err := ktav.FormatSource("## keep me\n\n\nport: 8080\n")
	if err != nil {
		return err
	}
	fmt.Print(out)
	return nil
}
```

Formatting reaches a fixed point. Handle each `(string, error)` result
before passing the next string to the formatter:

```go
func isFixedPoint(src string) (bool, error) {
	once, err := ktav.FormatSource(src)
	if err != nil {
		return false, err
	}
	twice, err := ktav.FormatSource(once)
	if err != nil {
		return false, err
	}
	return once == twice, nil
}
```

Without comments or blank lines, the result equals `CanonicalFromSource`
for the same text.

> **Unreleased (0.9 preparation):** These editing APIs and updated diagnostics are not included in published 0.8.1 packages. Native backends require the pinned 0.9 core (C ABI 2 for C-FFI); pure backends remain standalone.

## Source editing

`EditValue` and `EditValueSegments` change one value **inside the source
text** and return the patched source. This is not a parse → mutate →
re-serialize round trip: only the target's byte range (plus, when
readability demands it, the inline wrappers containing it) is spliced, so
everything else survives byte-for-byte — comments, blank lines, key
spelling, dotted/inline/block forms, the document's CR / LF / CRLF bytes,
its BOM, and whether it ends with a newline.

```go
out, err := ktav.EditValue(src, "targets.storagehost.address", "10.0.0.9:22")
// out == src with only that scalar's bytes replaced

out, err := ktav.EditValueSegments(src, []string{"targets", "storagehost", "address"}, "10.0.0.9:22")
```

Path rules:

- `EditValue` takes a textual path (`key1.key2.0.key4`). Inside a
  segment `\.` is a literal dot and `\\` a literal backslash; any other
  escape, a trailing lone backslash, or an empty segment is an
  `*ktav.Error` with `Class == "EditInvalidPath"` whose `Body` is the
  UTF-8 byte offset of the problem inside the path argument.
- `EditValueSegments` takes pre-decoded literal segments — no additional
  unescaping. A numeric segment is a string key inside an Object and a
  zero-based ASCII-digit index inside an Array (leading zeros allowed,
  signs are not indexes). An empty segment list is `"EditNotFound"`.
- `EditValue` replaces an existing value in source text. For insertion, removal, or key renaming, see [Insert, remove, and rename](#insert-remove-and-rename). A missing key, an out-of-bounds index, or descent through a scalar is `"EditNotFound"`.

Replacement values use the ordinary Go type mapping (`Dumps` accepts
them, scalars included); a host integer never degrades to a float.

Guarantees:

- A no-op replacement (the new value equals the old one) returns the
  source byte-identical, however noncanonical its spelling.
- Marker transitions are minimal: switching a scalar to a numeric- or
  keyword-like String toggles only the second `:` of the `::` marker,
  and the original spacing around it is preserved.
- Dotted, inline, and block forms are preserved, including a whole
  inline container replacement and emptied-but-not-collapsed block
  containers.
- Inline ancestors expand to block form only when readability requires:
  a multiline replacement string, growth beyond max(100 Unicode scalar
  values, the old physical line length), or ≥ 4 inline escapes with ≥ 25%
  overhead. Untouched nested branches stay inline; new block levels use
  four spaces and the source EOL.
- Errors of class `"EditUnsupported"` name pairs with no lossless
  single-range patch: noncontiguous dotted-key/reopen targets, structural
  replacement that would drop interior comment lines (leaf edits inside
  such containers still work and keep the comments), or values without a
  lossless representation in the target context.
- Every successful patch is re-parsed and semantically verified before
  it is returned, so silent corruption is impossible.
- The native (C ABI) and pure-Go backends agree byte-for-byte; both are
  exercised against the same regression corpus.

## Insert, remove, and rename

The root `ktav` package and standalone `pure` package expose the same APIs:

```go
func InsertValue(src, textualPath string, replacement any) (string, error)
func InsertValueSegments(src string, segments []string, replacement any) (string, error)
func RemoveValue(src, textualPath string) (string, error)
func RemoveValueSegments(src string, segments []string) (string, error)
func RenameKey(src, textualPath, newKey string) (string, error)
func RenameKeySegments(src string, segments []string, newKey string) (string, error)
```

Textual paths use `.` separators; `\.` and `\\` encode a literal dot and backslash. Segment APIs take literal, pre-decoded segments, so `[]string{"a.b"}` addresses the single key `a.b`. For example:

```go
out, err := ktav.InsertValue(src, `list\.item.new`, "value")
out, err := ktav.RemoveValue(src, `list\.item.0`)
out, err := ktav.RenameKeySegments(src, []string{"list.item", "old"}, "new.name")
```

Insert's final segment is a new Object key, and the parent must exist and be an Object; it appends the pair. Remove accepts a leaf or subtree, and removes an array element together with its delimiter. Neither insertion nor removal prunes empty ancestors: for example, removing the last `b` from dotted source `a.b: 1\n` yields `a: {}\n` (the empty parent is materialized), rather than pruning `a`. Rename changes one Object key in place, preserving the pair's value and position; `newKey` is one literal key, not a path. Renaming to the same key is a byte-identical no-op.

Invalid textual paths return `*ktav.Error` with `Class == "EditInvalidPath"`. Missing targets/parents, invalid array indexes, or descent through scalars produce `EditNotFound`; inserting an existing key or renaming to a sibling key produces `EditAlreadyExists`. Array insertion and array-key rename, and structural cases that cannot be represented by a lossless single-range splice, produce `EditUnsupported`. Array indexes are zero-based ASCII digits (leading zeros allowed; signs are not indexes); numeric segments remain ordinary string keys in Objects. Insert values use the regular `Dumps` Go type mapping.

The C-FFI ABI 2 library must export all six symbols: `ktav_insert_value`, `ktav_insert_value_segments`, `ktav_remove_value`, `ktav_remove_value_segments`, `ktav_rename_key`, and `ktav_rename_key_segments`. An older ABI 2 library without these symbols is incompatible with this Go binding and cannot support these APIs. The pure-Go APIs are standalone and do not load or depend on the native library; native and pure implementations are independent backends.

### Source-edit cookbook

These examples describe the unreleased 0.9 API, not the published 0.8.1 package. They replace existing values only: keys are not renamed, missing paths are not created, and entries are not removed. Each snippet is independent; Rust `?` assumes an enclosing `Result`-returning function. Go examples panic only for these fixed, known-valid inputs; applications should handle the returned error.

The binding examples use the standalone pure backend, without a native binary. Native entry points accept the same paths and values: `ktav.EditValue` (Go), `Ktav.editValue` (Java), `KtavApi.EditValue(..., KtavBackend.Native)` (C#), `Ktav::editValue` (PHP), the package's native/FFI exports (JS), and `ktav.edit_value` (Python). C-ABI backends require ABI 2; PyO3/N-API/WASM require the matching preparing-0.9 core. Bun/Deno FFI calls return promises: use `await`; pure/Node/WASM examples shown here are synchronous.

```go
import "github.com/ktav-lang/golang/pure"
```

#### 1. Dotted deployment settings

Only the address changes; the comment, dotted form and neighbouring user entry remain byte-for-byte unchanged.

```go
source := "## deployment\ntargets.storagehost.address: old\ntargets.storagehost.user: mcp-agent\n"
updated, err := pure.EditValue(source, "targets.storagehost.address", "192.168.0.88:22")
if err != nil {
    panic(err)
}
if updated != "## deployment\ntargets.storagehost.address: 192.168.0.88:22\ntargets.storagehost.user: mcp-agent\n" { panic("unexpected edit") }
```

Result:

```ktav
## deployment
targets.storagehost.address: 192.168.0.88:22
targets.storagehost.user: mcp-agent
```

#### 2. Literal dots: textual path or segments

The quoted source key is literally `a.b`. The textual path escapes its dot; the segments call passes the decoded key directly. Both calls return the same result.

```go
source := "\"a.b\": {port: 80}\n"
updated, err := pure.EditValue(source, "a\\.b.port", int64(90))
if err != nil {
    panic(err)
}
same, err := pure.EditValueSegments(source, []string{"a.b", "port"}, int64(90))
if err != nil {
    panic(err)
}
if updated != "\"a.b\": {port: 90}\n" { panic("unexpected edit") }
if same != updated { panic("path APIs differ") }
```

Result:

```ktav
"a.b": {port: 90}
```

#### 3. A leaf inside a mixed-form array

`0` is an array index here. Inside an Object, the same segment would address the key named `0`; untouched containers remain inline.

```go
source := "servers: [{host: old, keep: yes}, standby]\n"
updated, err := pure.EditValueSegments(source, []string{"servers", "0", "host"}, "new")
if err != nil {
    panic(err)
}
if updated != "servers: [{host: new, keep: yes}, standby]\n" { panic("unexpected edit") }
```

Result:

```ktav
servers: [{host: new, keep: yes}, standby]
```

#### 4. A numeric-looking String stays a String

The replacement is a String, not an Integer. The raw marker preserves that distinction. Supply a typed Integer when a numeric value is intended.

```go
source := "port: 8080\n"
updated, err := pure.EditValue(source, "port", "9090")
if err != nil {
    panic(err)
}
if updated != "port:: 9090\n" { panic("unexpected edit") }
```

Result:

```ktav
port:: 9090
```

#### 5. Multiline replacement: expand only what is necessary

The changed array expands with four-space indentation. The untouched object stays inline, and verbatim string contents are not reindented.

```go
source := "a: [old, {keep: yes}]\n"
updated, err := pure.EditValue(source, "a.0", "x\ny")
if err != nil {
    panic(err)
}
if updated != "a: [\n    ((\nx\ny\n    ))\n    {keep: yes}\n]\n" { panic("unexpected edit") }
```

Result:

```ktav
a: [
    ((
x
y
    ))
    {keep: yes}
]
```

#### 6. No-op preserves noncanonical spelling

The Integer value is already 1. Equality returns the original bytes, including `+1` and the original separator spacing; readability thresholds are bypassed.

```go
source := "n  : +1\n"
updated, err := pure.EditValue(source, "n", int64(1))
if err != nil {
    panic(err)
}
if updated != source { panic("no-op changed source") }
```

Result:

```ktav
n  : +1
```

#### 7. BOM, CRLF and an unterminated final line

The returned string keeps the BOM and CRLF, and does not gain a final newline. Its escaped representation is shown to make the bytes visible.

```go
source := "\ufeffport: 8080\r\nhost: db"
updated, err := pure.EditValue(source, "port", int64(9090))
if err != nil {
    panic(err)
}
if updated != "\ufeffport: 9090\r\nhost: db" { panic("unexpected edit") }
```

Result:

```text
"\uFEFFport: 9090\r\nhost: db"
```

Expansion is deterministic: a multiline replacement, growth beyond `max(100, original line length)`, or at least four escapes with at least 25% encoding overhead for an individual replacement string. Width counts Unicode scalar values, not UTF-8 bytes or terminal columns. An already-long line without growth is kept. New levels use four spaces and the source EOL. If CR or a `))` data line prevents lossless standalone encoding, a valid inline candidate is retained; these characters do not automatically make every inline edit unsupported.

#### 8. Exact Integer replacement

The source Integer domain is signed 64-bit. This value exceeds JavaScript's safe-Number range but fits i64; use an exact host representation, not a rounded floating-point value. A typed Integer outside i64 cannot be preserved as an Integer in source and is refused instead of truncated. Use a String when a large identifier is text. Float identity distinguishes negative and positive zero; Object no-op equality ignores key order, while Array order remains significant.

```go
updated, err := pure.EditValue("n: 1\n", "n", int64(9007199254740993))
if err != nil { panic(err) }
if updated != "n: 9007199254740993\n" { panic("precision lost") }
```

```ktav
n: 9007199254740993
```

### Diagnostics and safe failures in 0.9

Source coordinates are UTF-8 byte offsets, including a BOM; spans are half-open `[start, end)`, and lines are 1-based. LF, lone CR and CRLF are handled, with CRLF counted as one terminator. A Unicode character may occupy several bytes.

An unfinished `a: [\n` reports `UnclosedCompound`, line 1 and span `3..5`: the span begins at the innermost unmatched opener, not the start of its key or line. For raw bytes `61 3A 20 FF` (`a: ` followed by invalid UTF-8), validation reports `InvalidUtf8`, line 1 and a covering span `3..4`; `line_text` is null rather than fabricated by lossy decoding. UTF-8 validation precedes grammar processing.

A malformed textual path, including an empty textual path or empty textual segment, is `EditInvalidPath`; its offset is measured in the path's UTF-8 bytes. An empty segments list, missing key, out-of-range index or scalar descent is `EditNotFound`. Object numeric segments remain keys; array indexes use ASCII digits, allow leading zeros and reject signs.

Whole dotted/reopened intermediate objects may occupy several physical ranges and are `EditUnsupported`; their existing leaves can still be edited. Replacing a container must not discard interior comments. No failed call returns a partial patch or changes the supplied source. The returned text is re-parsed and semantically checked: the target must equal the replacement and the rest of the value tree must remain equal.

For bindings, invalid source is reported with the ordinary parser envelope before path validation. Malformed wire payloads are `Message` errors without invented source coordinates. The wire envelope has `error`, `reason`, `line`, `line_text`, `span`, `path`, `body`, `canonical`, `spec_section` and `message`; host accessor names are documented in the errors section. `body` carries the decimal path-byte offset for `EditInvalidPath`, null for `EditNotFound`, and the refusal reason for `EditUnsupported`.

## Structured errors

Native parser and writer failures carry the structured envelope shared
by Ktav bindings, so tools can act on fields instead of parsing a
message. Other failures, such as native-library loading/download and Go
JSON conversion errors, are ordinary Go errors and need not be
`*ktav.Error`:

```go
if _, err := ktav.Loads("a: 1\na: 2\n"); err != nil {
    var kerr *ktav.Error
    if errors.As(err, &kerr) {
        fmt.Println(kerr.Class)       // "DuplicateKey"
        fmt.Println(kerr.Line)        // 2
        fmt.Println(kerr.SpecSection) // "§6.2"
    }
}
```

| Field | Meaning |
| --- | --- |
| `Msg` | Human-readable error text. Native-origin errors use the core's rendering; host-synthesized errors use binding wording. |
| `Class` | Structured error class — `DuplicateKey`, `Unrepresentable`, `UnrepresentableAt`, `InvalidUtf8`, `Message`. |
| `Reason` | Host writer reasons include `NonFiniteFloat`, `ScalarRoot`, and `EmptyKeyName`. `InvalidUtf8` is a source-error class with an empty reason. |
| `Line` | 1-based source line; `0` when the envelope carries null. |
| `LineText` | Text of the offending line. |
| `Span` | `*Span` — byte offsets into the UTF-8 source. |
| `Path` | Exact decoded key segments. |
| `Body` | The offending value as written. |
| `Canonical` | What the canonical form would have been. |
| `SpecSection` | The clause violated, e.g. `§3.6/§5.2`. |

Two details that are easy to get wrong:

- **`Span` is byte offsets into UTF-8**, not UTF-16 code units. An LSP
  consumer either converts or negotiates `positionEncoding: "utf-8"`.
- **`Path` is a slice of segments, never a joined string.** A key
  literally named `a.b` is one segment and cannot be confused with a
  two-segment path — there is no separator to be ambiguous about.

For native structured errors, `Msg` is taken from the core verbatim
rather than assembled here, so the same document produces the same error
text in every Ktav binding. Loader, Go conversion, and host-synthesized
validation errors are outside that cross-binding guarantee.

## Type mapping

| Ktav             | Go                                              |
| ---------------- | ----------------------------------------------- |
| `null`           | `nil`                                           |
| `true` / `false` | `bool`                                          |
| integer scalar   | `int64` if it fits; otherwise `string`           |
| float scalar     | `float64`                                       |
| bare scalar      | `string`                                        |
| `[ ... ]`        | `[]any`                                         |
| `{ ... }`        | `map[string]any` (Go map; no insertion-order guarantee) |

Under spec 0.8, integers and floats are inferred from the scalar body's
lexical form (no typed markers). An integer outside `int64` range parses
as a String. On encode, Go `int*` / `uint*` / `*big.Int` become integer
scalars, but an out-of-range `*big.Int` parses back as a String;
`float32` / `float64` become float scalars and `string` stays a bare
scalar. `Dumps` and `EmitCanonical` reject `NaN` and `±Inf`;
`DumpsForceStrings` converts them to String leaves. Structs are serialized
through `encoding/json` first, so `json:"..."` tags are honoured.

## Key escaping

Since spec 0.6.4 a literal `.` or `:` inside a key segment is written
with a backslash:

```text
## The key is the single segment "a.b".
a\.b: v
## The key is the single segment "a:b".
a\:b: v
## This is the path ["x", "y.z"].
x.y\.z: v
```

A literal backslash in a key is `\\`.

## How the native library is resolved

At first call the Go package resolves `ktav_cabi` in this order:

1. **`$KTAV_LIB_PATH`** — absolute path to a local build. Most useful
   for development.
2. **User cache** — `<os.UserCacheDir>/ktav-go/v<version>/…`, downloaded
   on a previous call.
3. **GitHub Release download** — the matching asset is fetched once
   from `github.com/ktav-lang/golang/releases/download/v<version>/<name>`
   and cached under (2). Requires network on first call after install.

# Pure-Go backend

Use `go get github.com/ktav-lang/golang/pure@v0.8.1`, then import:

```go
package main

import (
    "fmt"
    "log"

    "github.com/ktav-lang/golang/pure"
)

func main() {
    value, err := pure.Loads("server.port: 8080\n")
    if err != nil {
        log.Fatal(err)
    }
    text, err := pure.Dumps(value)
    if err != nil {
        log.Fatal(err)
    }
    restored, err := pure.Loads(text)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Print(text)
    fmt.Printf("%#v\n", restored)
}
```

Package functions and `pure.New()` expose all eight operations: `Loads`,
`LoadsStrict`, `LoadsInto`, `Dumps`, `DumpsForceStrings`, `EmitCanonical`,
`FormatSource`, `CanonicalFromSource`, plus full `pure.Error`/`pure.Span` aliases.
The package shares `internal/engine`; its runtime graph is stdlib-only.

Root `ktav.New(ktav.BackendPure)` remains compatible, but root imports require
`github.com/ebitengine/purego v0.8.4` and `golang.org/x/sys v0.28.0` even with
BackendPure. An offline packaged consumer uses a file-only proxy, readonly mode
and an import whitelist to verify no external runtime imports for `/pure`. Its
CI job runs the entire 0.8 conformance corpus through the installed `/pure`
artifact with `CGO_ENABLED=0` and with `rustc`/`cargo` trapped on `PATH`.

Inline compounds have a 128-level limit. Plain-map canonical keys sort
alphabetically (O(n log n)); source canonicalization and formatting preserve
source order. `encoding/json` bridge paths retain depth limits; unlimited depth
is not promised. Node docs and Go build/test commands are development tools,
not `/pure` runtime dependencies.

## Runtime support

- Go 1.21+.
- Prebuilt binaries for: `linux/amd64`, `linux/arm64`, `darwin/amd64`,
  `darwin/arm64`, `windows/amd64`, `windows/arm64`.
- Linux distros must use glibc 2.17+ (Rust's default target). Alpine
  (musl) support is planned.

## License

MIT OR Apache-2.0 — see [LICENSE-MIT](LICENSE-MIT) and
[LICENSE-APACHE](LICENSE-APACHE).

## Other Ktav implementations

- [`spec`](https://github.com/ktav-lang/spec) — specification + conformance suite
- [`rust`](https://github.com/ktav-lang/rust) — reference Rust crate (`cargo add ktav`)
- [`csharp`](https://github.com/ktav-lang/csharp) — C# / .NET (`dotnet add package Ktav`)
- [`java`](https://github.com/ktav-lang/java) — Java / JVM (`io.github.ktav-lang:ktav` on Maven Central)
- [`js`](https://github.com/ktav-lang/js) — JS / TS (`npm install @ktav-lang/ktav`)
- [`php`](https://github.com/ktav-lang/php) — PHP (`composer require ktav-lang/ktav`)
- [`python`](https://github.com/ktav-lang/python) — Python (`pip install ktav`)
