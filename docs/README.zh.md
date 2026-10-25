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

`encoding/json` 会对 Go map 的键排序，因此输出顺序是确定的；
`map[string]any` 本身不保留插入顺序。

完整可运行示例:[`examples/basic`](../examples/basic/main.go)。

## API

| 函数 | 用途 |
| --- | --- |
| `Loads(s string) (any, error)` | 将 Ktav 文档解析为原生 Go 值。顶层可以是对象(`map[string]any`)或数组(`[]any`),按 spec § 5.0.1。 |
| `LoadsStrict(s string) (any, error)` | 使用严格数字词法检查解析文档，类型映射与 `Loads` 相同。 |
| `LoadsInto(s string, target any) error` | 解析到任意 `target`(struct、map 等),通过 `encoding/json`。 |
| `Dumps(v any) (string, error)` | 将 Go 值渲染为 Ktav 文本。顶层必须为对象或数组。 |
| `DumpsForceStrings(v any) (string, error)` | 同 `Dumps`,但所有叶标量(integer、float、bool、null)通过 `::` 强制为 String。复合值保留其结构。 |
| `EmitCanonical(v any) (string, error)` | 把 Go 值渲染为规范 Ktav(spec § 5.9)。`encoding/json` 会按字典序排列 Go map 的字符串键。 |
| `CanonicalFromSource(src string) (string, error)` | 解析 Ktav 并立即输出规范形式，保留源文件的键顺序。 |
| `FormatSource(src string) (string, error)` | 把 Ktav 源文本格式化为规范化写法，**保留全部注释**。见下文。 |
| `EditValue(src, path string, replacement any) (string, error)` | 按文本点分路径**在源文本中**替换值，返回修改后的源码。见[源码编辑](#源码编辑)。 |
| `EditValueSegments(src string, segments []string, replacement any) (string, error)` | 同上，但接受已解码的字面量段（不做额外反转义）。 |

## 格式化 —— 规范写法，保留注释

`FormatSource` 与 `EmitCanonical` 是不同的操作：

- **`EmitCanonical`** 接受 Go 值并写出规范形式。值中没有注释，因而
  无法保留注释。
- **`FormatSource`** 接受源文本并规范其写法，同时保留每条独占整行的
  注释（spec § 3.4）。它与 `CanonicalFromSource` 都保留源键顺序。

以下示例使用这些导入：

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

空行用于提示分组：连续两行及以上会缩减为一行，紧贴括号内侧的空白
填充会被删除。

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

格式化会达到不动点。再次传入格式化器之前，先处理 `(string, error)`
返回值：

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

没有注释和空行时，结果与同一文本的 `CanonicalFromSource` 相同。

> **尚未发布（0.9 准备版本）：** 这些编辑 API 和更新后的诊断不包含在已发布的 0.8.1 包中。原生后端需要固定的 0.9 核心（C-FFI 使用 C ABI 2）；纯实现仍完全独立。

## 源码编辑

`EditValue` 与 `EditValueSegments` 直接**在源文本内部**修改一个值，并返回
修改后的源码。这不是 parse → mutate → serialize 的全量重写：只有目标值的
字节区间会被拼接替换（在可读性需要时，连同其所在的内联容器），其余部分
逐字节保留 —— 注释、空行、键的写法、dotted/inline/block 形式、文档的
CR / LF / CRLF 字节、BOM 以及末尾是否有换行。

```go
out, err := ktav.EditValue(src, "targets.storagehost.address", "10.0.0.9:22")
// out == src，仅该标量的字节被替换

out, err := ktav.EditValueSegments(src, []string{"targets", "storagehost", "address"}, "10.0.0.9:22")
```

路径规则：

- `EditValue` 接受文本路径（`key1.key2.0.key4`）。段内 `\.` 表示字面量
  点号，`\\` 表示字面量反斜杠；任何其他转义、结尾的孤立 `\` 或空段都会
  得到 `Class == "EditInvalidPath"` 的 `*ktav.Error`，其 `Body` 是问题在
  路径参数中的 UTF-8 字节偏移。
- `EditValueSegments` 接受已解码的字面量段 —— 不做额外反转义。数字段在
  Object 中是字符串键，在 Array 中是从零开始的 ASCII 数字索引（允许前导
  零，不允许符号）。空段列表为 `"EditNotFound"`。
- 仅支持**替换已有值**：没有 create、remove 或 rename。缺失键、越界索引
  或穿过标量的下降都是 `"EditNotFound"`。

替换值使用常规的 Go 类型映射（`Dumps` 接受的一切，包括标量）；宿主
integer 绝不会退化为 float。

保证：

- no-op 替换（新值等于旧值）逐字节返回源码，无论其写法多么非规范。
- 标记切换最小化：把标量换成数字形或关键字形 String 只切换 `::` 标记的
  第二个 `:`，其周围的原始空格保持不变。
- dotted、inline 与 block 形式均保留，包括整体替换内联容器，以及清空但
  不折叠的 block 容器。
- 仅在可读性需要时才把内联祖先展开为 block 形式：多行替换字符串、行长
  超过 max(100 个 Unicode scalar values, 原物理行长)，或 ≥ 4 个内联转义
  且开销 ≥ 25%。未触及的嵌套分支保持内联；新 block 层级使用四个空格和
  源文件的 EOL。
- `"EditUnsupported"` 类错误表示不存在无损单区间补丁：dotted-key/reopen
  形成的不连续目标、会丢失内部注释行的结构性替换（此类容器内的叶编辑
  仍可用且保留注释）、包含 CR 或等于 `))` 的行的字符串。
- 每个成功补丁在返回前都会重新解析并做语义验证 —— 不可能悄悄损坏文档。
- Native（C ABI）与纯 Go 后端逐字节一致；两者都针对同一回归语料运行。

## 结构化错误

原生解析器和 writer 的失败会携带 Ktav 绑定共用的结构化信封,工具可以
直接处理字段,而不必解析消息文本。其他失败,例如原生库加载/下载和
Go JSON 转换错误,仍是普通 Go error,不保证为 `*ktav.Error`:

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
| `Msg` | 人类可读的错误文本。原生错误使用核心渲染；host 合成的错误使用绑定自身的措辞。 |
| `Class` | 错误类别 —— `DuplicateKey`、`Unrepresentable`、`UnrepresentableAt`、`InvalidUtf8`、`Message`。 |
| `Reason` | Host writer 的原因码包括 `NonFiniteFloat`、`ScalarRoot` 和 `EmptyKeyName`。`InvalidUtf8` 是 source-error class，reason 为空。 |
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
  是**一个**段,不可能与两段路径混淆 —— 没有分隔符,也就无可歧义。

对于原生核心的结构化错误,`Msg` 是逐字取得的,并非在此处拼装,因此同一
份文档在任何 Ktav 绑定中都会产生相同的错误文本。加载器、Go 转换和
host 合成的校验错误不在此跨绑定保证范围内。

## 类型映射

| Ktav             | Go                                               |
| ---------------- | ------------------------------------------------ |
| `null`           | `nil`                                            |
| `true` / `false` | `bool`                                           |
| integer scalar   | `int64`（范围内）；否则为 `string`                |
| float scalar     | `float64`                                        |
| 裸标量           | `string`                                         |
| `[ ... ]`        | `[]any`                                          |
| `{ ... }`        | `map[string]any`（Go map，不保证插入顺序）        |

spec 0.8 中 integer 与 float 从标量体的词法形式推断（无类型标记）。
超出 `int64` 范围的 integer 会解析为 String。Go `int*` / `uint*` /
`*big.Int` 编码为 integer scalar，但超范围的 `*big.Int` 再解析时会
成为 String；`float32` / `float64` 编码为 float scalar，`string` 保持
裸标量。`Dumps` 和 `EmitCanonical` 会拒绝 `NaN` 与 `±Inf`；
`DumpsForceStrings` 会将它们转换为 String 叶值。结构体先走
`encoding/json`，`json:"..."` tag 生效。

## 键的转义

自 spec 0.6.4 起,键段内的字面量 `.` 或 `:` 通过反斜杠书写:

```text
## 键由单个段 "a.b" 组成。
a\.b: v
## 键由单个段 "a:b" 组成。
a\:b: v
## 这是路径 ["x", "y.z"]。
x.y\.z: v
```

键中的字面量反斜杠写作 `\\`。

## 原生库解析顺序

首次调用时，Go 包按以下顺序查找 `ktav_cabi`：

1. **`$KTAV_LIB_PATH`** —— 本地构建的绝对路径，适合开发。
2. **用户缓存** —— `<os.UserCacheDir>/ktav-go/v<version>/…`，此前已下载。
3. **GitHub Release 下载** —— 从
   `github.com/ktav-lang/golang/releases/download/v<version>/<name>`
   下载匹配的资产，缓存到 (2)。安装后首次调用需要联网。

# Pure-Go 后端

使用 `go get github.com/ktav-lang/golang/pure@v0.8.1` 安装，然后导入：

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

包级函数和 `pure.New()` 提供全部八项操作：`Loads`、`LoadsStrict`、`LoadsInto`、
`Dumps`、`DumpsForceStrings`、`EmitCanonical`、`FormatSource`、
`CanonicalFromSource`，以及完整的 `pure.Error`/`pure.Span` 别名。该包共享
`internal/engine`；其 runtime 依赖图仅使用标准库。

兼容的根 API `ktav.New(ktav.BackendPure)` 仍保留，但根包导入即需要
`github.com/ebitengine/purego v0.8.4` 和 `golang.org/x/sys v0.28.0`，即使选择
BackendPure。离线打包消费者使用仅文件代理、readonly 模式和导入白名单，验证
`/pure` 没有外部 runtime 导入。其 CI 任务通过已安装的 `/pure` 产物运行完整的
0.8 一致性语料，使用 `CGO_ENABLED=0` 并在 `PATH` 上拦截 `rustc`/`cargo`。

Inline 复合值限制为 128 层。普通映射的规范键按字母排序（O(n log n)）；
源规范化和格式化保留源顺序。`encoding/json` 桥接路径有自身深度限制；不承诺
无限深度。Node 文档和 Go 构建/测试命令是开发工具，而非 `/pure` runtime 依赖。

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
