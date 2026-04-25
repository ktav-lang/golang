# ktav — Go bindings

**Languages:** **English** · [Русский](README.ru.md) · [简体中文](README.zh.md)

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
port:i 8080
ratio:f 0.75
tls: true
tags: [
    prod
    eu-west-1
]
db.host: primary.internal
db.timeout:i 30
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
// port:i 8443
// tls: true
// ratio:f 0.95
// upstreams: [
//     { host: a.example  port:i 1080 }
//     { host: b.example  port:i 1080 }
// ]
// notes: null
```

A complete runnable version lives in [`examples/basic`](examples/basic/main.go).

## API

| Function | Purpose |
| --- | --- |
| `Loads(s string) (any, error)` | Parse a Ktav document into native Go values. |
| `LoadsInto(s string, target any) error` | Parse into an arbitrary `target` (struct, map, …) via `encoding/json`. |
| `Dumps(v any) (string, error)` | Render a Go value as Ktav text. Top-level must encode to an object. |

## Type mapping

| Ktav             | Go                                              |
| ---------------- | ----------------------------------------------- |
| `null`           | `nil`                                           |
| `true` / `false` | `bool`                                          |
| `:i <digits>`    | `int64` if it fits, else `*big.Int`             |
| `:f <number>`    | `float64`                                       |
| bare scalar      | `string`                                        |
| `[ ... ]`        | `[]any`                                         |
| `{ ... }`        | `map[string]any` (insertion order preserved)    |

On encode, Go `int*` / `uint*` / `*big.Int` become `:i`; `float32` /
`float64` become `:f`; `string` stays a bare scalar. `NaN` and `±Inf`
are rejected. Structs are serialized through `encoding/json` first, so
`json:"..."` tags are honoured.

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

MIT — see [LICENSE](LICENSE).

Ktav spec: [ktav-lang/spec](https://github.com/ktav-lang/spec).
Reference Rust crate: [ktav-lang/rust](https://github.com/ktav-lang/rust).
