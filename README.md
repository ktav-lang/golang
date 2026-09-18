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
// port: 8443
// tls: true
// ratio: 0.95
// upstreams: [
//     { host: a.example  port: 1080 }
//     { host: b.example  port: 1080 }
// ]
// notes: null
```

A complete runnable version lives in [`examples/basic`](examples/basic/main.go).

## API

| Function | Purpose |
| --- | --- |
| `Loads(s string) (any, error)` | Parse a Ktav document into native Go values. Top-level may be an Object (`map[string]any`) or an Array (`[]any`) per spec § 5.0.1. |
| `LoadsStrict(s string) (any, error)` | Parse with strict numeric spelling checks, using the same Go type mapping as `Loads`. |
| `LoadsInto(s string, target any) error` | Parse into an arbitrary `target` (struct, map, …) via `encoding/json`. |
| `Dumps(v any) (string, error)` | Render a Go value as Ktav text. Top-level must encode to an object or array. |
| `DumpsForceStrings(v any) (string, error)` | Like `Dumps`, but coerces every leaf scalar (integer, float, bool, null) to a String via the raw `::` marker. Compounds preserve their structure. |
| `EmitCanonical(v any) (string, error)` | Render a Go value as canonical Ktav (spec § 5.9). Key order follows Go map iteration (alphabetical for `map[string]any`). |
| `CanonicalFromSource(src string) (string, error)` | Parse Ktav and immediately emit canonical form, preserving source key order. |
| `FormatSource(src string) (string, error)` | Format Ktav source into its normalised spelling, **keeping every comment**. See below. |

## Formatting — canonical spelling, comments kept

`FormatSource` and `EmitCanonical` are different operations and it is
worth being clear which you want:

- **`EmitCanonical`** takes a Go value and writes the canonical form.
  Trivia does not exist in a value, so none survives.
- **`FormatSource`** takes source *text* and rewrites its spelling while
  **preserving every comment verbatim** (spec § 3.4: a comment owns a
  whole line). Key order is never changed.

```go
out, err := ktav.FormatSource("## why\na:   {x: 1}\n")
// "## why\na: {\n    x: 1\n}\n"
// the comment survives; the inline compound is expanded to canonical
// multi-line form and re-indented
```

Blank lines survive as a grouping hint, but a run of two or more
collapses to exactly one, and blank padding immediately inside a bracket
is dropped:

```go
ktav.FormatSource("## keep me\n\n\nport: 8080\n")
// "## keep me\n\nport: 8080\n" — two blank lines became one
```

That makes formatting a fixed point:

```go
ktav.FormatSource(ktav.FormatSource(x)) == ktav.FormatSource(x)
```

For a document with no comments and no blank lines, the result equals
`CanonicalFromSource` of the same text.

## Structured errors

Every failure carries the same structured envelope the other Ktav
bindings carry, so a tool can act on the fields instead of parsing a
message. `Loads`, `Dumps` and the rest return an `*Error`:

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
| `Msg` | The core's own rendering of the error. Never empty; this is what `Error()` returns. |
| `Class` | Structured error class — `DuplicateKey`, `Unrepresentable`, `Message`. |
| `Reason` | Writer-time reason code (spec § 5.9.0) such as `NonFiniteFloat`; `""` when the envelope carries null. |
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

`Msg` is taken from the core verbatim rather than assembled here, so the
same document produces the same error text in every Ktav binding.

## Type mapping

| Ktav             | Go                                              |
| ---------------- | ----------------------------------------------- |
| `null`           | `nil`                                           |
| `true` / `false` | `bool`                                          |
| integer scalar   | `int64` if it fits, else `*big.Int`             |
| float scalar     | `float64`                                       |
| bare scalar      | `string`                                        |
| `[ ... ]`        | `[]any`                                         |
| `{ ... }`        | `map[string]any` (insertion order preserved)    |

Under spec 0.5 integers and floats are inferred from the scalar body's
lexical form (no typed markers). On encode, Go `int*` / `uint*` /
`*big.Int` become integer scalars; `float32` / `float64` become float
scalars; `string` stays a bare scalar. `NaN` and `±Inf` are rejected.
Structs are serialized through `encoding/json` first, so `json:"..."`
tags are honoured.

## Key escaping

Since spec 0.6.4 a literal `.` or `:` inside a key segment is written
with a backslash:

```text
a\.b: v        // key is the single segment "a.b" -> map["a.b"] = "v"
a\:b: v        // key contains a colon            -> map["a:b"] = "v"
x.y\.z: v      // split on the first dot only     -> map["x"]["y.z"] = "v"
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
