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
| `InsertValue(src, path string, value any) (string, error)` | 按文本点分路径在源文本中插入值。 |
| `InsertValueSegments(src string, segments []string, value any) (string, error)` | 同上，但接受已解码的字面量段（不做额外反转义）。 |
| `RemoveValue(src, path string) (string, error)` | 按文本点分路径从源文本中删除值。 |
| `RemoveValueSegments(src string, segments []string) (string, error)` | 同上，但接受已解码的字面量段（不做额外反转义）。 |
| `RenameKey(src, path, newKey string) (string, error)` | 按文本点分路径在源文本中重命名键。 |
| `RenameKeySegments(src string, segments []string, newKey string) (string, error)` | 同上，但接受已解码的字面量段（不做额外反转义）。 |

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
- `EditValue` 在源码文本中替换已有值。插入、删除或重命名键请参见[插入、删除与重命名](#插入删除与重命名)。缺失键、越界索引或穿过标量都会得到 `"EditNotFound"`。

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
  仍可用且保留注释）、或无法在目标上下文中无损表示的值。
- 每个成功补丁在返回前都会重新解析并做语义验证 —— 不可能悄悄损坏文档。
- Native（C ABI）与纯 Go 后端逐字节一致；两者都针对同一回归语料运行。

## 插入、删除与重命名

根 `ktav` 包和独立的 `pure` 包提供相同 API：

```go
func InsertValue(src, textualPath string, replacement any) (string, error)
func InsertValueSegments(src string, segments []string, replacement any) (string, error)
func RemoveValue(src, textualPath string) (string, error)
func RemoveValueSegments(src string, segments []string) (string, error)
func RenameKey(src, textualPath, newKey string) (string, error)
func RenameKeySegments(src string, segments []string, newKey string) (string, error)
```

文本路径使用 `.` 分隔；`\.` 和 `\\` 分别表示字面点号与反斜杠。Segments API 接受字面量、预解码的段，因此 `[]string{"a.b"}` 指向单个键 `a.b`。例如：

```go
out, err := ktav.InsertValue(src, `list\.item.new`, "value")
out, err := ktav.RemoveValue(src, `list\.item.0`)
out, err := ktav.RenameKeySegments(src, []string{"list.item", "old"}, "new.name")
```

Insert 的最后一段是新 Object 键；父级必须存在且为 Object，新键值对追加在末尾。Remove 可删除叶值或子树；删除数组元素时也会删除其分隔符。插入和删除都不会清除空祖先：例如，从 dotted 源 `a.b: 1\n` 删除最后一个 `b` 后得到 `a: {}\n`（空父级会被显式物化），而不是删除 `a`。Rename 原位修改一个 Object 键，保留键值对的值与位置；`newKey` 是单个字面键，不是路径。重命名为原键是逐字节不变的 no-op。

文本路径无效时返回 `Class == "EditInvalidPath"` 的 `*ktav.Error`。目标/父级不存在、数组索引无效或穿过标量会得到 `EditNotFound`；插入已有键或重命名为同级现有键会得到 `EditAlreadyExists`。向数组插入、重命名数组索引，以及无法用单一区间无损表示的结构情形会得到 `EditUnsupported`。数组索引为从零开始的 ASCII 数字（允许前导零，不允许符号）；Object 中的数字段仍是普通字符串键。Insert 值遵循 `Dumps` 的常规 Go 类型映射。

C-FFI ABI 2 库必须导出全部六个符号：`ktav_insert_value`、`ktav_insert_value_segments`、`ktav_remove_value`、`ktav_remove_value_segments`、`ktav_rename_key` 和 `ktav_rename_key_segments`。缺少这些符号的旧 ABI 2 库与此 Go 绑定不兼容，无法支持这些 API。纯 Go API 独立运行，不加载或依赖原生库；native 与 pure 是相互独立的实现后端。

### 源文本编辑示例

以下示例适用于尚未发布的 0.9 API，而非已发布的 0.8.1 包。只能替换已有值：不会重命名键、创建缺失路径或删除条目。各代码片段相互独立；Rust 的 `?` 假定外层函数返回 `Result`。Go 示例仅针对固定且有效的输入使用 panic；应用程序应处理返回的错误。

绑定示例使用无需原生二进制的独立纯实现。原生入口接收相同的路径和值：Go 的 `ktav.EditValue`、Java 的 `Ktav.editValue`、C# 的 `KtavApi.EditValue(..., KtavBackend.Native)`、PHP 的 `Ktav::editValue`、JS 包的 native/FFI 导出及 Python 的 `ktav.edit_value`。C ABI 后端要求 ABI 2；PyO3/N-API/WASM 要求兼容的 0.9 准备核心。Bun/Deno FFI 返回 promise，需要 `await`；此处 pure/Node/WASM 示例为同步调用。

```go
import "github.com/ktav-lang/golang/pure"
```

#### 1. 通过点分键修改部署配置

只修改地址；注释、点分形式及相邻的用户字段逐字节保持不变。

```go
source := "## deployment\ntargets.storagehost.address: old\ntargets.storagehost.user: mcp-agent\n"
updated, err := pure.EditValue(source, "targets.storagehost.address", "192.168.0.88:22")
if err != nil {
    panic(err)
}
if updated != "## deployment\ntargets.storagehost.address: 192.168.0.88:22\ntargets.storagehost.user: mcp-agent\n" { panic("unexpected edit") }
```

结果:

```ktav
## deployment
targets.storagehost.address: 192.168.0.88:22
targets.storagehost.user: mcp-agent
```

#### 2. 字面点号：文本路径或路径段

源码中带引号的键名就是 `a.b`。文本路径需要转义点号；路径段 API 直接接收解码后的键名。两个调用返回相同结果。

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

结果:

```ktav
"a.b": {port: 90}
```

#### 3. 修改混合形式数组中的叶子值

此处 `0` 是数组索引；在对象中，同一个路径段表示名称为 `0` 的键。未修改的容器保持行内形式。

```go
source := "servers: [{host: old, keep: yes}, standby]\n"
updated, err := pure.EditValueSegments(source, []string{"servers", "0", "host"}, "new")
if err != nil {
    panic(err)
}
if updated != "servers: [{host: new, keep: yes}, standby]\n" { panic("unexpected edit") }
```

结果:

```ktav
servers: [{host: new, keep: yes}, standby]
```

#### 4. 看似数字的字符串仍然是字符串

替换值是 String，而不是 Integer。raw 标记保留这种区别；需要数值时请传入类型化 Integer。

```go
source := "port: 8080\n"
updated, err := pure.EditValue(source, "port", "9090")
if err != nil {
    panic(err)
}
if updated != "port:: 9090\n" { panic("unexpected edit") }
```

结果:

```ktav
port:: 9090
```

#### 5. 多行替换：仅展开必要的容器

修改的数组以四空格缩进展开。未修改的对象保持行内形式，verbatim 字符串内容不会重新缩进。

```go
source := "a: [old, {keep: yes}]\n"
updated, err := pure.EditValue(source, "a.0", "x\ny")
if err != nil {
    panic(err)
}
if updated != "a: [\n    ((\nx\ny\n    ))\n    {keep: yes}\n]\n" { panic("unexpected edit") }
```

结果:

```ktav
a: [
    ((
x
y
    ))
    {keep: yes}
]
```

#### 6. 无变化时保留非规范拼写

Integer 的值已经是 1。相等判断直接返回原始字节，包括 `+1` 及分隔符周围的空白，不应用可读性阈值。

```go
source := "n  : +1\n"
updated, err := pure.EditValue(source, "n", int64(1))
if err != nil {
    panic(err)
}
if updated != source { panic("no-op changed source") }
```

结果:

```ktav
n  : +1
```

#### 7. BOM、CRLF 及无换行符的末行

返回值保留 BOM 和 CRLF，且不会增加末尾换行。以下转义表示使这些字节清晰可见。

```go
source := "\ufeffport: 8080\r\nhost: db"
updated, err := pure.EditValue(source, "port", int64(9090))
if err != nil {
    panic(err)
}
if updated != "\ufeffport: 9090\r\nhost: db" { panic("unexpected edit") }
```

结果:

```text
"\uFEFFport: 9090\r\nhost: db"
```

展开条件固定：替换中包含多行字符串、修改后行长增长并超过 `max(100, 原始行长)`，或单个替换字符串需要至少四个转义且编码开销达到 25%。宽度按 Unicode 标量值计算，而非 UTF-8 字节或终端列。原本已很长但没有增长的行保持不变。新层级使用四空格及源文本的 EOL。若 CR 或数据行 `))` 无法无损转换为独立行，则保留有效的行内候选；这些字符不会使所有行内编辑自动失败。

#### 8. 精确替换 Integer

源文本的 Integer 域是有符号 64 位。此值超过 JavaScript Number 的安全整数范围，但仍在 i64 内；应使用精确的宿主表示，而非已舍入的浮点值。超出 i64 的类型化 Integer 无法在源码中保持 Integer 身份，因此会被拒绝而不是截断。大型文本标识符应使用 String。Float 区分负零和正零；Object 的 no-op 相等忽略键顺序，而 Array 顺序仍然重要。

```go
updated, err := pure.EditValue("n: 1\n", "n", int64(9007199254740993))
if err != nil { panic(err) }
if updated != "n: 9007199254740993\n" { panic("precision lost") }
```

```ktav
n: 9007199254740993
```

### 0.9 的诊断与安全失败

源码坐标为包含 BOM 的 UTF-8 字节偏移，span 使用半开区间 `[start, end)`，行号从 1 开始。支持 LF、独立 CR 和 CRLF，且 CRLF 计为一个换行。一个 Unicode 字符可能占用多个字节。

未完成的 `a: [\n` 报告 `UnclosedCompound`、第 1 行和 span `3..5`：区间从最内层未闭合的起始括号开始，而非键或行的开头。原始字节 `61 3A 20 FF`（`a: ` 后接无效 UTF-8）报告 `InvalidUtf8`、第 1 行及覆盖区间 `3..4`；`line_text` 为 null，不会通过有损解码伪造文本。UTF-8 校验优先于语法处理。

非法文本路径（包括空路径或空文本段）报告 `EditInvalidPath`，偏移按路径本身的 UTF-8 字节计算。空路径段列表、缺失键、索引越界或穿过标量下钻报告 `EditNotFound`。对象中的数字段仍是键；数组索引仅接受 ASCII 数字，允许前导零，不允许正负号。

由 dotted/reopened 定义拼成的整个中间对象可能占据多个物理范围，因此报告 `EditUnsupported`；已有叶子值仍可修改。容器结构替换不能删除内部注释。失败调用不会返回部分补丁，也不会修改传入的源码。返回文本会重新解析并进行语义检查：目标必须等于替换值，其余值树必须与原文相等。

在绑定中，无效源码会先于路径验证报告普通解析错误信封。非法 wire 载荷是无虚构源码坐标的 `Message` 错误。wire 信封包含 `error`、`reason`、`line`、`line_text`、`span`、`path`、`body`、`canonical`、`spec_section`、`message`；宿主语言的访问器名称见错误章节。`body` 分别承载 `EditInvalidPath` 的十进制路径字节偏移、`EditNotFound` 的 null，以及 `EditUnsupported` 的拒绝原因。

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
