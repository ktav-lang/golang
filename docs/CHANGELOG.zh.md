# Changelog

**语言:** [English](../CHANGELOG.md) · [Русский](CHANGELOG.ru.md) · **简体中文**

本文件记录 Go 绑定的所有重要变更。格式基于
[Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/);版本遵循
[Semantic Versioning](https://semver.org/),并采用 pre-1.0 约定:
MINOR 递进视为破坏性变更。

本 changelog 跟踪**绑定发布**,不涉及 Ktav 格式本身的变更 —— 后者见
[`ktav-lang/spec`](https://github.com/ktav-lang/spec/blob/main/CHANGELOG.md).

## 未发布

## 0.8.0 — 2026-09-27

此版本将 Go 绑定从 `ktav 0.7.1` / spec 0.7.0 基线更新至
`ktav 0.8.0` / spec 0.8.0。

### 变更

- 模块版本及原生库回退下载目标为 `v0.8.0`。
- 原生依赖下限从 `ktav 0.7.1` 提升至 `0.8.0`，spec 固定到 0.8.0；
  C ABI 迁移到 `ktav::declare_cabi!()`，导出符号保持不变。
- `FormatSource` 格式化源文本、保留独占整行的注释及源键顺序，并
  规范化空行。
- 错误使用结构化 Ktav 信封。Go 侧写入校验返回带有
  `NonFiniteFloat`、`ScalarRoot` 和 `EmptyKeyName` 原因码的结构化
  `*ktav.Error`。无效 UTF-8 源的 class 是 `InvalidUtf8`，reason 为空，
  它不是 writer reason。库加载/下载错误和 Go JSON 转换错误仍是普通
  Go error，不保证为 `*ktav.Error`。
- 一致性运行器针对 `spec/versions/0.8/tests`，覆盖该版本语料中的
  类别，包括 `strict-lossy`、`unrepresentable` 和
  `parseable-unrepresentable`。运行器会拒绝未知类别，并验证 fixture
  schema、预期数量和必需的 companion 文件。运行 conformance 还需要
  原生库。writer 拒绝及 strict-lossy 测试逐例核对 oracle 中的确切
  原因/错误码。

## 0.6.4 — 2026-08-23

### 新增

- **`LoadsStrict(s string) (any, error)`** —— 通过 Go 绑定和
  `ktav_loads_strict` C ABI 符号暴露 Rust strict parser。

### 变更

- 跟踪 `ktav 0.6.4` 与 spec 0.6.4,包括规范化 float 边界和
  `notation_boundaries` fixture。
- 原生库加载器现在指向精确的 `v0.6.4` release asset。

## [0.6.1] — 2026-06-05

- 文档:将所有 README 示例改写为 spec 0.6 语法(裸数字替代已移除的
  `:i`/`:f` 标记;`##` 注释替代 `#`)。

## 0.6.0 — 2026-06-01

同步至 Ktav 0.6.0 —— 键现在支持转义。

### 新增

- 键处理完整的 §3.7 转义集合,并新增两个转义:
  - `\.` → `.`(字面量点 —— **不**会切分 dotted-path)
  - `\:` → `:`(字面量冒号 —— **不**作为键/值分隔符)
- 示例: `a\.b: v` → `{"a.b": "v"}`,`a\:b: v` → `{"a:b": "v"}`,
  `x.y\.z: v` → `{"x": {"y.z": "v"}}`。

### 破坏性变更

- 键中的字面量反斜杠现在需要写作 `\\`(此前键中的 `\` 是普通字节)。
  实际中很少出现;按 pre-1.0 SemVer 为 MINOR bump。

### 变更

- 跟踪 ktav-rust 0.6.0 / Ktav 规范 0.6.0。绑定源码未改动 —— escape
  语义的变化完全在 Rust 内核中实现,purego FFI 边界对其透明。

---

## 0.5.0 — 2026-05-28

### 破坏性变更

- **Spec 0.5.0**: 类型标记 `:i` / `:f` 不再存在。数字从标量体词法形式
  推断(`42` → Integer,`3.14` → Float)。为 spec 0.1.x 编写的、带显式
  类型标记的文档解析结果不同,必须更新。
- **`##` 注释**: 单 `#` 现为字面字符;注释需使用 `##`。请更新任何
  使用了 `# 注释` 的 Ktav 源码。
- **Float 规范化**: Float 值以 shortest-decimal 规范形式存储(无下划线,
  无前导 `+`)。与旧的序列化输出做字节级比对可能失败。
- **C ABI 新增第六个符号** `ktav_emit_canonical`;将 `KTAV_LIB_PATH`
  指向 pre-0.5.0 的二进制会因符号缺失而失败。

### 新增

- **`EmitCanonical(v any) (string, error)`** — 将 Go 值渲染为规范 Ktav
  (spec § 5.9):字节确定性输出,无内联复合,规范的 integer / float
  正规化。两次相同输入的调用总是产生完全相同的字节。
- **`TestConformanceCanonical`** — 新的一致性测试套件,验证
  `EmitCanonical` 输出与 spec fixtures 中每个 `.canonical.ktav` oracle
  字节一致。

### 变更

- **升级到 `ktav 0.5.0`** — 跟踪上游 Rust crate 的 spec 0.5 实现:
  推断数字类型、`##` 注释、`emit_canonical` API。spec submodule 同步
  至标签 `v0.5.0`。完整变更见
  [`ktav` crate CHANGELOG](https://github.com/ktav-lang/rust/blob/main/CHANGELOG.md#050)。
- **许可证变更为 `MIT OR Apache-2.0`** — 与 `ktav-lang` 生态系统保持
  一致。`LICENSE` 改名为 `LICENSE-MIT`;新增 `LICENSE-APACHE`。
  `Cargo.toml` 中的 SPDX 表达式已相应更新。
- **一致性测试套件更新至 spec 0.5 fixtures** — 路径
  `spec/versions/0.5/tests`;`.canonical.ktav` 文件从 JSON oracle 测试
  中排除,由新的规范测试套件处理。


## 0.3.1 — 2026-05-10

### 新增

- **顶层 Array 支持**(spec 0.1.1,§ 5.0.1)—— 当文档的第一个内容行是
  数组元素行(裸标量、类型标记、单独的 `{`/`[` 或多行开启符)时,
  `Loads` 现在返回 `[]any`。此前顶层 Array 会被拒绝。
- **`Dumps` 接受顶层数组** —— 传入任意切片(`[]any`、`[]string`
  等),渲染出的 Ktav 逐行罗列元素,不带外层 `[...]`。
- **`DumpsForceStrings(v any) (string, error)`** —— 将 Go 值渲染为
  Ktav,每个标量都强制为 String(带类型整数、带类型浮点、布尔、null
  都经由 raw 标记 `::` 摊平为文本形式)。复合结构保持原状。输出经
  `Loads` 读回仍是同一组 String 标量 —— 适用于不理解 `:i` / `:f`
  类型标记的环境或下游消费者。

### 变更

- **升级到 `ktav 0.3.1`** —— 跟踪上游 Rust crate 的顶层 Array 支持与
  `to_string_force_strings` API。spec submodule 同步至 `7256816`
  (spec 0.1.1)。完整变更见
  [`ktav` crate CHANGELOG](https://github.com/ktav-lang/rust/blob/main/CHANGELOG.md#031--2026-05-10)。
- **C ABI 新增第五个符号** `ktav_dumps_force_strings`,与既有符号
  `ktav_loads` / `ktav_dumps` / `ktav_free` / `ktav_version` 并列。
  Go 加载器在首次使用时绑定全部五个符号;将 `KTAV_LIB_PATH` 指向
  pre-0.3.1 二进制会因符号缺失而失败。


## 0.3.0 — 2026-05-08

### 变更

- **升级到 `ktav 0.3.0`** —— 跟踪 ktav 0.3.0(paren 字符串处理收紧:
  行内 `(...)` 形式现在无效,必须改用多行形式)。spec submodule 同步至
  `46d94a7`。完整变更见
  [`ktav` crate CHANGELOG](https://github.com/ktav-lang/rust/blob/main/CHANGELOG.md#030--2026-05-08)。


## 0.2.0 — 2026-05-07

### 变更(破坏性)

- **升级到 `ktav 0.2.0`** —— 多行字符串现在默认序列化为缩进剥离的
  `( ... )` 形式(逐字的 `(( ... ))` 仍作为内容含前导空白或整行仅一个
  `)` 时的回退)。`:f 42` 接受整数字面量(解析为 `42.0`)。完整变更见
  [`ktav` crate CHANGELOG](https://github.com/ktav-lang/rust/blob/main/CHANGELOG.md#020--2026-05-07)。

  将序列化输出与内置的 `((...))` 字面量逐字节比较的代码需要更新。
  Round-trip 不变。

### 规范

- spec submodule 同步(typed_float_without_decimal 从 invalid 移至
  valid/typed_float_integer_body)。


## 0.1.2 — 2026-05-03

### 变更

- **已采用 `ktav 0.1.5`** —— 上游 Rust crate 引入了结构化错误 API
  (`Error::Structured(ErrorKind)` 带字节偏移 span)、对错误枚举追溯
  应用了 `#[non_exhaustive]`,以及公开的事件式解析器 `ktav::thin`。
  Go 绑定对用户可见的行为没有变化:返回的 `error` 值仍携带相同的
  人类可读消息(七个标准类别的 Display 字符串与 ktav 0.1.4 完全
  字节相同,由 ktav 自己的 pinning 测试验证)。将 `ktav::ErrorKind`
  映射到类型化的 Go error 值(以便调用方可以使用
  `errors.As(err, &ktav.MissingSeparatorSpaceError{})` 等)是单独
  的后续工作,记录在
  [`STRUCTURED_ERRORS.md`](https://github.com/ktav-lang/.github/blob/main/STRUCTURED_ERRORS.md)。

`go get github.com/ktav-lang/golang@v0.1.2`。

## 0.1.1 — 2026-04-26

### 变更

- **升级到 `ktav 0.1.4`** —— 上游 Rust crate 中 `cabi` 使用的 untyped
  `parse() → Value` 路径,小文档加速约 30%、大文档加速约 13%,只是
  `Frame::Object` 的初始容量微调(4 → 8)。每次 `ktav.Loads` 都会
  透明地受益。

## 0.1.0 — 首次公开发布

首次发布。面向 **Ktav 格式 0.1**。

### 模块路径

以 `github.com/ktav-lang/golang` 发布。`v0.1.0` git tag 推送后
Go proxy 会自动索引。

### 公开 API

- `Loads(s string) (any, error)` —— 解析 Ktav 文档。
- `LoadsInto(s string, target any) error` —— 通过 `encoding/json` 解析到
  `target`。
- `Dumps(v any) (string, error)` —— 将 Go 值渲染为 Ktav 文本。
- `Error` —— 类型化的 parse / render 错误,可通过 `errors.As` 捕获。

### 架构

- **原生核心** —— 参考 Rust `ktav` crate,通过极小的 `extern "C"` C ABI
  (`crates/cabi`) 封装,以预编译的 `.so` / `.dylib` / `.dll` 分发。
- **Go 加载器** —— `purego`(无 cgo):库在首次调用时从 `$KTAV_LIB_PATH`
  解析,或一次性从匹配的 GitHub Release asset 下载到 `UserCacheDir`。
- **Wire 格式** —— Rust 和 Go 之间用 JSON,用 `{"$i":"..."}` /
  `{"$f":"..."}` 标签包装器保证类型化 integer / float 的无损 round-trip
  和任意精度(`*big.Int`)。

### 类型映射

| Ktav             | Go                                                |
| ---------------- | ------------------------------------------------- |
| `null`           | `nil`                                             |
| `true` / `false` | `bool`                                            |
| `:i <digits>`    | `int64`(安全范围)/ `*big.Int`(更大)           |
| `:f <number>`    | `float64`                                         |
| 裸标量           | `string`                                          |
| `[ ... ]`        | `[]any`                                           |
| `{ ... }`        | `map[string]any`                                |

### 平台

预编译原生二进制:

- `linux/amd64`、`linux/arm64`(glibc)
- `darwin/amd64`、`darwin/arm64`
- `windows/amd64`、`windows/arm64`

Alpine (musl) —— 下一个版本支持。

### 测试覆盖

在 Go 1.21 / 1.22 / 1.23 × Linux / macOS / Windows 上运行 Ktav 0.1
完整 conformance 套件(所有 `valid/` 与 `invalid/` fixture)。

### 致谢

基于参考 `ktav` Rust crate;动态加载使用
[`ebitengine/purego`](https://github.com/ebitengine/purego)。
