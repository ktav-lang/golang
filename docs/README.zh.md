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

## 选择 pure-Go 后端

默认情况下，包会把调用分发到原生 `ktav_cabi` 共享库（purego，无
cgo）。另一个完全自包含的实现——以 Go 编写的解析器和生成器——同样
可用，以相同签名覆盖所有公开操作：

```go
ktav.UsePure()                      // 或：
ktav.UseBackend(ktav.BackendPure)
```

- `UsePure` / `UseBackend(ktav.BackendPure)`：pure-Go 路径。它不加载、
  不下载、也不需要原生库，且不启动子进程，因此在未安装 `ktav_cabi`
  制品的机器上同样可用。
- `UseNative` / `UseBackend(ktav.BackendNative)`：经 purego `ktav_cabi`
  的路径。这是默认值，也是历史行为。
- 两个方向都没有自动回退：所选后端失败时，你得到的是错误，而不是
  另一个实现。

请在启动时、第一次解析之前选择一次后端。两次调用之间可以切换实现；
进行中的调用总是在它开始时所用的后端上完成。

对于不应依赖进程全局状态的并发或库代码，可将操作显式绑定到某一个后
端：

```go
ops, err := ktav.New(ktav.BackendPure)
ops.Loads(src) // 以及 LoadsStrict、LoadsInto、Dumps、
               // DumpsForceStrings、EmitCanonical、FormatSource、
               // CanonicalFromSource —— 全部绑定到该实例
```

`Operations` 值携带自己的后端：其方法不会读取或修改全局默认值，因此
各实例彼此隔离，也与 `UseBackend` 隔离。包级函数保留全局默认（原生）
行为。

Pure 模式的限制：解码值保持文档规定的 `map[string]any` 契约（解码键
顺序无法表示）；对普通 Go 映射，`EmitCanonical` 保持文档规定的字母
序，而 `CanonicalFromSource` 与 `FormatSource` 保留插入顺序；内联复合
结构嵌套上限为 128 层（§5.8.4，实现定义，与参考实现一致）。点号键嵌套（§5.3.2）不设上限：pure 写入器、trivia 遍历、tagged-JSON 编解码器以及 Go 侧公共桥转换（`decodeJSON`、`toTagged`、`prepareWriterValue`、去标签展平）均为迭代实现（显式工作栈），深层文档与深层程序构造的值不会导致栈溢出。循环引用的 Go 值（映射或切片在活跃路径上包含自身）会被 `Dumps`/`DumpsForceStrings`/`EmitCanonical` 以结构化错误拒绝（`Class: "Message"`，"ktav: cyclic Go value cannot be encoded"）；兄弟分支间共享的子映射（DAG 而非循环）仍可正常工作。经 `encoding/json` 回退的值（任意结构体）依赖该包自身的嵌套上限——超限时返回错误而非崩溃；本包不扩展也不验证该回退路径的深度。循环检测基于活跃路径：显式的 ENTER/EXIT 工作栈方案在容器整个子树转换完成前保持其活跃标记，因此 a→b→a 循环在首次重访即被拒绝，而已退出的共享 DAG 节点会再次转换；切片身份由（指针、长度、容量）决定，同一底层数组的不同视图不会被混淆。`Loads`/`LoadsStrict`/`LoadsInto` 拒绝顶层线格式值之后的尾随数据（第二个顶层 JSON 值是错误而非静默覆盖），且 JSON 键 `""` 是合法键。扫描探针插桩（生产环境中为 nil 钩子）在每个粗粒度事件上付出一次 nil 判断，并在字符串分类扫描内部付出逐字节计数——以对被扫描主体的一次遍历为界；这是如实记录的成本，而非宣称免费。

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
