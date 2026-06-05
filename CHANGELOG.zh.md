# Changelog

**语言:** [English](CHANGELOG.md) · [Русский](CHANGELOG.ru.md) · **简体中文**

本文档记录 Go 绑定的所有重要变更。格式基于
[Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/);版本采用
[Semantic Versioning](https://semver.org/),遵循 pre-1.0 约定:
MINOR 版本升级视为破坏性。

本 changelog 跟踪**绑定发布**,不涉及 Ktav 格式本身的变更 —— 后者见
[`ktav-lang/spec`](https://github.com/ktav-lang/spec/blob/main/CHANGELOG.md)。

## [0.6.1] — 2026-06-05

- 文档：将所有 README 示例改写为 spec 0.6 语法（裸数字替代已移除的 `:i`/`:f` 标记；`##` 注释替代 `#`）。

## 0.6.0 —— 2026-06-01

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

## 0.5.0 —— 2026-05-28

### 破坏性变更

- **Spec 0.5.0**：类型标记 `:i` / `:f` 不再存在。数字从标量体词法形式
  推断（`42` → Integer，`3.14` → Float）。使用旧标记的 spec 0.1.x
  文档需更新。
- **`##` 注释**：单 `#` 现为字面字符；注释需使用 `##`。
- **Float 规范化**：Float 值以 shortest-decimal 规范形式存储。旧序列化
  输出的字节级比对可能失败。
- **C ABI 新增第六个符号** `ktav_emit_canonical`；旧二进制会报符号缺失。

### 新增

- **`EmitCanonical(v any) (string, error)`** — 输出规范 Ktav（spec § 5.9）：
  字节确定性，无内联复合，规范 integer / float 正规化。
- **`TestConformanceCanonical`** — 验证 `EmitCanonical` 输出与每个
  `.canonical.ktav` oracle 字节一致。

### 变更

- **升级到 `ktav 0.5.0`** — 跟踪上游 Rust crate 的 spec 0.5 实现：
  推断数字类型、`##` 注释、`emit_canonical` API。spec submodule 同步
  至标签 `v0.5.0`。
- **许可证变更为 `MIT OR Apache-2.0`** — 与 `ktav-lang` 生态系统保持
  一致。`LICENSE` 改名为 `LICENSE-MIT`；新增 `LICENSE-APACHE`。
- **一致性测试更新至 spec 0.5 fixtures** — 路径
  `spec/versions/0.5/tests`；`.canonical.ktav` 从 JSON oracle 测试
  中排除，由新规范测试处理。


## 0.1.2 —— 2026-05-03

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

## 0.1.1 —— 2026-04-26

### 变更

- **升级到 `ktav 0.1.4`** —— 上游 Rust crate 中 `cabi` 使用的 untyped
  `parse() → Value` 路径,小文档加速约 30%、大文档加速约 13%,只是
  `Frame::Object` 的初始容量微调(4 → 8)。每次 `ktav.Loads` 都会
  透明地受益。

## 0.1.0 —— 首次公开发布

首次发布。面向 **Ktav 格式 0.1**。

### Module 路径

以 `github.com/ktav-lang/golang` 发布。`v0.1.0` git tag 推送后
Go proxy 会自动索引。

### 公开 API

- `Loads(s string) (any, error)` —— 解析 Ktav 文档。
- `LoadsInto(s string, target any) error` —— 通过 `encoding/json` 解析到
  `target`。
- `Dumps(v any) (string, error)` —— 将 Go 值渲染为 Ktav 文本。
- `Error` —— 类型化的 parse / render 错误,可通过 `errors.As` 捕获。

### 架构

- **原生核心** —— 参考 Rust `ktav` crate，通过极小的 `extern "C"` C ABI
  (`crates/cabi`) 封装，以预编译的 `.so` / `.dylib` / `.dll` 分发。
- **Go 加载器** —— `purego`（无 cgo）: 库在首次调用时从 `$KTAV_LIB_PATH`
  解析，或一次性从匹配的 GitHub Release asset 下载到 `UserCacheDir`。
- **Wire 格式** —— Rust 和 Go 之间用 JSON，用 `{"$i":"..."}` /
  `{"$f":"..."}` 标签包装器保证类型化 integer / float 的无损 round-trip
  和任意精度（`*big.Int`）。

### 类型映射

| Ktav             | Go                                                |
| ---------------- | ------------------------------------------------- |
| `null`           | `nil`                                             |
| `true` / `false` | `bool`                                            |
| `:i <digits>`    | `int64`（安全范围）/ `*big.Int`（更大）           |
| `:f <number>`    | `float64`                                         |
| 裸标量           | `string`                                          |
| `[ ... ]`        | `[]any`                                           |
| `{ ... }`        | `map[string]any`（保留插入顺序）                  |

### 平台

预编译原生二进制:

- `linux/amd64`、`linux/arm64`（glibc）
- `darwin/amd64`、`darwin/arm64`
- `windows/amd64`、`windows/arm64`

Alpine (musl) —— 下一个版本支持。

### 测试覆盖

在 Go 1.21 / 1.22 / 1.23 × Linux / macOS / Windows 上运行 Ktav 0.1
完整 conformance 套件（所有 `valid/` 与 `invalid/` fixture）。

### 致谢

基于参考 `ktav` Rust crate；动态加载使用
[`ebitengine/purego`](https://github.com/ebitengine/purego)。
