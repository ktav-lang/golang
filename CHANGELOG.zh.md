# Changelog

**语言:** [English](CHANGELOG.md) · [Русский](CHANGELOG.ru.md) · **简体中文**

本文档记录 Go 绑定的所有重要变更。格式基于
[Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/);版本采用
[Semantic Versioning](https://semver.org/),遵循 pre-1.0 约定:
MINOR 版本升级视为破坏性。

本 changelog 跟踪**绑定发布**,不涉及 Ktav 格式本身的变更 —— 后者见
[`ktav-lang/spec`](https://github.com/ktav-lang/spec/blob/main/CHANGELOG.md)。

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
