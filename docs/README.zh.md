# ktav — Go 绑定

[![Go Reference](https://pkg.go.dev/badge/github.com/ktav-lang/golang.svg)](https://pkg.go.dev/github.com/ktav-lang/golang)
[![CI](https://img.shields.io/github/actions/workflow/status/ktav-lang/golang/ci.yml?style=flat-square&logo=github&label=CI)](https://github.com/ktav-lang/golang/actions)
![License: MIT OR Apache-2.0](https://img.shields.io/badge/license-MIT%20OR%20Apache--2.0-blue?style=flat-square)
[![Playground](https://img.shields.io/badge/playground-try%20online-7c3aed?style=flat-square&logo=rocket&logoColor=white)](https://ktav-lang.github.io/)

**语言:** [English](../README.md) · [Русский](README.ru.md) · **简体中文**

**演练场：** 在浏览器中互转 JSON / YAML / TOML / INI ⇄ Ktav — **[ktav-lang.github.io](https://ktav-lang.github.io/)**。

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

完整可运行示例:[`examples/basic`](../examples/basic/main.go)。

## API

| 函数 | 用途 |
| --- | --- |
| `Loads(s string) (any, error)` | 将 Ktav 文档解析为原生 Go 值。 |
| `LoadsStrict(s string) (any, error)` | 使用严格数字词法检查解析文档，类型映射与 `Loads` 相同。 |
| `LoadsInto(s string, target any) error` | 通过 `encoding/json` 解析到任意 `target`（struct、map 等）。 |
| `Dumps(v any) (string, error)` | 将 Go 值渲染为 Ktav 文本。顶层必须为对象或数组。 |
| `DumpsForceStrings(v any) (string, error)` | 同 `Dumps`，但所有叶标量（integer、float、bool、null）通过 `::` 强制为 String。 |
| `EmitCanonical(v any) (string, error)` | 输出规范 Ktav（spec § 5.9 — 字节确定性，无内联复合）。 |
| `CanonicalFromSource(src string) (string, error)` | 解析 Ktav 并立即输出规范形式，保留源文件的键顺序。 |
| `FormatSource(src string) (string, error)` | 把 Ktav 源文本格式化为规范化写法，**保留全部注释**。见下文。 |

## 格式化 —— 规范写法，保留注释

`FormatSource` 与 `EmitCanonical` 是两种不同的操作,值得分清需要哪一个:

- **`EmitCanonical`** 接受 Go 值并写出规范形式。值里不存在注释,
  因此没有任何注释可以保留。
- **`FormatSource`** 接受源**文本**,重写其写法,同时**逐字保留每条
  注释**(spec § 3.4:注释独占一整行)。键顺序绝不改变。

```go
out, err := ktav.FormatSource("## why\na:   {x: 1}\n")
// "## why\na: {\n    x: 1\n}\n"
// 注释保留;inline 复合值被展开为规范的多行形式并重新缩进
```

空行作为分组提示保留下来,但连续两行及以上会合并为恰好一行,紧贴括号
内侧的空行填充会被丢弃:

```go
ktav.FormatSource("## keep me\n\n\nport: 8080\n")
// "## keep me\n\nport: 8080\n" —— 两个空行变成一个
```

因此格式化是一个不动点:

```go
ktav.FormatSource(ktav.FormatSource(x)) == ktav.FormatSource(x)
```

对于没有注释也没有空行的文档,结果与同一文本的 `CanonicalFromSource`
相同。

## 结构化错误

每一次失败都携带与其他 Ktav 绑定相同的结构化信封,因此工具可以针对
字段做处理,而不必解析消息文本。`Loads`、`Dumps` 等返回 `*Error`:

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

| 字段 | 含义 |
| --- | --- |
| `Msg` | 来自核心自身的错误渲染文本。永不为空;`Error()` 返回的就是它。 |
| `Class` | 结构化错误类别 —— `DuplicateKey`、`Unrepresentable`、`Message`。 |
| `Reason` | writer 侧的原因码(spec § 5.9.0),如 `NonFiniteFloat`;信封为 null 时是 `""`。 |
| `Line` | 1 起算的源行号;信封为 null 时是 `0`。 |
| `LineText` | 出错那一行的文本。 |
| `Span` | `*Span` —— UTF-8 源文本中的字节偏移。 |
| `Path` | 精确解码后的键段。 |
| `Body` | 出错的值,按写法原样。 |
| `Canonical` | 规范形式本应是什么。 |
| `SpecSection` | 被违反的条款,如 `§3.6/§5.2`。 |

两处容易弄错的细节:

- **`Span` 是 UTF-8 中的字节偏移**,不是 UTF-16 码元。LSP 消费方要么
  自行转换,要么协商 `positionEncoding: "utf-8"`。
- **`Path` 是键段切片,绝不是拼接后的字符串。** 字面名为 `a.b` 的键
  是**一个**段,不可能与两段路径混淆 —— 提供在线契约中并不存在可供
  歧义解读的分隔符。

`Msg` 是从核心逐字取得的,并非在此处拼装,因此同一份文档在任何 Ktav
绑定中都产生相同的错误文本。

## 类型映射

| Ktav             | Go                                               |
| ---------------- | ------------------------------------------------ |
| `null`           | `nil`                                            |
| `true` / `false` | `bool`                                           |
| integer scalar   | `int64`（可容纳）或 `*big.Int`                   |
| float scalar     | `float64`                                        |
| 裸标量           | `string`                                         |
| `[ ... ]`        | `[]any`                                          |
| `{ ... }`        | `map[string]any`（保留插入顺序）                 |

spec 0.5 中 integer 与 float 从标量体的词法形式推断（无类型标记）。编码时
Go `int*` / `uint*` / `*big.Int` → integer scalar；`float32` / `float64`
→ float scalar；`string` 保持裸标量。`NaN` 与 `±Inf` 会被拒绝。结构体先
走 `encoding/json`，`json:"..."` tag 生效。

## 键的转义

自 spec 0.6.4 起,键段内的字面量 `.` 或 `:` 通过反斜杠书写:

```text
a\.b: v        // 键是单个段 "a.b"        -> map["a.b"] = "v"
a\:b: v        // 键中包含冒号            -> map["a:b"] = "v"
x.y\.z: v      // 只按第一个点切分        -> map["x"]["y.z"] = "v"
```

键中的字面量反斜杠写作 `\\`。

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

MIT OR Apache-2.0 —— 详见 [LICENSE-MIT](../LICENSE-MIT) 和
[LICENSE-APACHE](../LICENSE-APACHE)。

## 其他 Ktav 实现

- [`spec`](https://github.com/ktav-lang/spec) —— 规范 + 一致性测试套件
- [`rust`](https://github.com/ktav-lang/rust) —— 参考 Rust crate(`cargo add ktav`)
- [`csharp`](https://github.com/ktav-lang/csharp) —— C# / .NET(`dotnet add package Ktav`)
- [`java`](https://github.com/ktav-lang/java) —— Java / JVM(`io.github.ktav-lang:ktav`,Maven Central)
- [`js`](https://github.com/ktav-lang/js) —— JS / TS(`npm install @ktav-lang/ktav`)
- [`php`](https://github.com/ktav-lang/php) —— PHP(`composer require ktav-lang/ktav`)
- [`python`](https://github.com/ktav-lang/python) —— Python(`pip install ktav`)
