# Contributing

**语言:** [English](CONTRIBUTING.md) · [Русский](CONTRIBUTING.ru.md) · **简体中文**

感谢参与。本仓库把两件东西缝合在一起:

- 一个小的 Rust crate（`crates/cabi/`），把参考 `ktav` 解析器封装在
  `extern "C"` API 之后。
- 仓库根目录的一个 Go 包，通过
  [`purego`](https://github.com/ebitengine/purego) 在运行时加载编译产物。

## 本地开发

依赖: Rust（stable）、Go 1.21+、git。

```bash
# 1. 为本机平台构建原生库。
cargo build --release -p ktav-cabi

# 2. 指向它并运行测试。
export KTAV_LIB_PATH="$PWD/target/release/libktav_cabi.so"   # Linux
#      ="$PWD/target/release/libktav_cabi.dylib"             # macOS
#      ="$PWD/target/release/ktav_cabi.dll"                  # Windows

# 3. 运行 conformance 测试时指向 spec 子模块路径。
export KTAV_SPEC_ROOT="$PWD/spec/versions/0.1/tests"

go test ./...
```

`git submodule update --init` 拉取共享的 spec fixture。

## 架构说明

- **Wire 格式。** Rust 与 Go 交换 JSON，用 `{"$i":"..."}` /
  `{"$f":"..."}` 包装类型化 integer / float，保留任意精度以及跨 FFI
  边界的 `:i` / `:f` 区分。
- **内存。** Rust 分配输出 buffer；Go 拷贝到 slice 后立刻回调
  `ktav_free`。跨 FFI 边界无长期共享内存。
- **加载器。** Go 这边 `internal/native` 通过 `sync.Once` 在每个进程
  dlopen 一次。Windows 走 `golang.org/x/sys/windows`；Unix 走
  `purego.Dlopen/Dlsym`。

## 发布

在 `main` 上打 `v<X.Y.Z>` 标签。Release workflow 交叉编译 6 个平台的
二进制，作为 GitHub Release assets 附加，Go proxy 会自动索引 tag。
`internal/native/loader.go` 中的 `LibVersion` 常量必须与 tag 对齐 ——
在打 tag 的同一 commit 里更新。

## 许可

贡献即同意以 MIT 协议发布（见 [LICENSE](LICENSE)）。
