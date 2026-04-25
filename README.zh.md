# ktav — Go 绑定

**语言:** [English](README.md) · [Русский](README.ru.md) · **简体中文**

[Ktav 配置格式](https://github.com/ktav-lang/spec)的 Go 绑定。
在参考 Rust 解析器之上做了薄封装，通过
[`purego`](https://github.com/ebitengine/purego) 在运行时动态加载 ——
因此**使用方无需 `cgo`**，普通的 `go build` 开箱即用。

```bash
go get github.com/ktav-lang/golang
```

## 快速开始

### 解析 —— 直接解码到类型化结构体

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

### 遍历 —— 动态形态,按类型分派

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

### 构建并渲染 —— 用代码搭建文档

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
```

完整可运行示例:[`examples/basic`](examples/basic/main.go)。

## API

| 函数 | 用途 |
| --- | --- |
| `Loads(s string) (any, error)` | 将 Ktav 文档解析为原生 Go 值。 |
| `LoadsInto(s string, target any) error` | 通过 `encoding/json` 解析到任意 `target`（struct、map 等）。 |
| `Dumps(v any) (string, error)` | 将 Go 值渲染为 Ktav 文本。顶层必须为对象。 |

## 类型映射

| Ktav             | Go                                               |
| ---------------- | ------------------------------------------------ |
| `null`           | `nil`                                            |
| `true` / `false` | `bool`                                           |
| `:i <digits>`    | `int64`（可容纳）或 `*big.Int`                   |
| `:f <number>`    | `float64`                                        |
| 裸标量           | `string`                                         |
| `[ ... ]`        | `[]any`                                          |
| `{ ... }`        | `map[string]any`（保留插入顺序）                 |

编码时 Go `int*` / `uint*` / `*big.Int` → `:i`；`float32` / `float64` →
`:f`；`string` 保持裸标量。`NaN` 与 `±Inf` 会被拒绝。结构体先走
`encoding/json`，所以 `json:"..."` tag 生效。

## 原生库解析顺序

首次调用时，Go 包按以下顺序查找 `ktav_cabi`：

1. **`$KTAV_LIB_PATH`** —— 本地构建的绝对路径，适合开发。
2. **用户缓存** —— `<os.UserCacheDir>/ktav-go/v<version>/…`，此前已下载。
3. **GitHub Release 下载** —— 从
   `github.com/ktav-lang/golang/releases/download/v<version>/<name>`
   下载匹配的资产，缓存到 (2)。安装后首次调用需要联网。

## 运行时支持

- Go 1.21+。
- 预编译二进制: `linux/amd64`、`linux/arm64`、`darwin/amd64`、
  `darwin/arm64`、`windows/amd64`、`windows/arm64`。
- Linux 发行版需 glibc 2.17+（Rust 默认 target）。Alpine（musl）支持
  在计划中。

## 许可

MIT —— 详见 [LICENSE](LICENSE)。

Ktav 规范: [ktav-lang/spec](https://github.com/ktav-lang/spec)。
参考 Rust crate: [ktav-lang/rust](https://github.com/ktav-lang/rust)。
