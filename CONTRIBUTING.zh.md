# Contributing 到 ktav（Go）

**语言:** [English](CONTRIBUTING.md) · [Русский](CONTRIBUTING.ru.md) · **简体中文**

## 核心规则

### 1. 每个 bugfix 伴随回归测试

发现 bug **修之前**，先写一个能复现它的测试 —— 该测试在 `main` 上
**必须失败**，修复后通过。两者放同一个 PR。

测试位于仓库根：

| 文件                    | 作用域                                                   |
| ----------------------- | -------------------------------------------------------- |
| `ktav_smoke_test.go`    | Loads / Dumps happy path、bigint、错误形态。             |
| `conformance_test.go`   | 针对 `ktav-lang/spec` 的跨语言一致性。                   |

### 2. 不要在绑定中重新发明格式

这个 Go 包刻意是薄封装。解析器与格式行为属于 Rust crate
（[`ktav-lang/rust`](https://github.com/ktav-lang/rust)）—— 在那里改动
会同步更新所有语言绑定。本仓库只接纳**Go 专有的人体工学**：类型映射、
purego 加载器、缓存 / 下载逻辑。

若改动需要调整格式，先在
[`ktav-lang/spec`](https://github.com/ktav-lang/spec) 发起讨论。

### 3. 公开 API 变更标注兼容性

如果改动了 `ktav` 的任何导出项，在 PR 描述里注明属于：

- **semver 兼容**（新增、放宽签名、文档变更）；或
- **semver 破坏**（重命名 / 移除、改签名、收紧类型）—— 在 pre-1.0
  阶段这类改动走下一个 MINOR。

`CHANGELOG.md` 和两个翻译版本在同一个 PR 里更新。

### 4. 一个 commit 一个概念

Commit 保持原子：bugfix 和它的测试在一起、feature 和它的测试在一起、
重命名单独、重构单独。`git log --oneline` 应当像 changelog 一样阅读。
不要在 commit message 里加 `feat:` / `fix:` 前缀 —— 这里不用 conventional
commits。

### 5. 原生库与 Go 模块版本对齐

`LibVersion` 常量（`internal/native/loader.go`）**必须**与 release
tag 对齐。升级 Go 模块版本时，在同一个 commit 里更新 `LibVersion`。
错位会让使用者下载到与代码不匹配的库。

## 开发环境

需要：

- Go **1.21+**。
- 通过 [`rustup`](https://rustup.rs/) 安装的 Rust toolchain。MSRV: **1.70**。
- `git`。

开发布局 —— Go 包通过 `purego` 加载 Rust 编译出的 `ktav_cabi` cdylib。
把 spec 仓库（conformance 测试用）克隆到旁边，或初始化 submodule：

```
ktav-lang/
├── golang/    ← 本仓库
├── rust/      ← 相邻 Rust crate（本地开发的 path 依赖）
└── spec/      ← conformance fixtures（golang/spec/ 的 submodule）
```

Rust C ABI crate（`crates/cabi/`）默认依赖 crates.io 上发布的 `ktav`。
跨仓库本地改动时，把 `Cargo.toml` 的
`workspace.dependencies.ktav` 切换为 `{ path = "../rust" }`。

### 构建

```bash
# 1. 为本机平台构建原生库。
cargo build --release -p ktav-cabi

# 2. 指向它。
export KTAV_LIB_PATH="$PWD/target/release/libktav_cabi.so"   # Linux
#      ="$PWD/target/release/libktav_cabi.dylib"             # macOS
#      ="$PWD/target/release/ktav_cabi.dll"                  # Windows

# 3. 运行 conformance 测试时指向 spec submodule。
git submodule update --init
export KTAV_SPEC_ROOT="$PWD/spec/versions/0.1/tests"
```

### 测试

```bash
go test -v ./...                # 完整套件
go test -run TestSmoke ./...    # 按名称过滤
go test -run Conformance ./...  # 只跑 spec fixtures
```

当 `KTAV_LIB_PATH` 或 `KTAV_SPEC_ROOT` 未设置时，相关测试会**跳过**
而不是失败 —— 纯净 checkout 下 `go test` 也保持绿色。

### Lint

```bash
go vet -unsafeptr=false ./...
gofmt -l .                            # 应当无输出
cargo fmt --all --check
cargo clippy --release -p ktav-cabi -- -D warnings
```

CI 跑同样的命令；push 前本地先过一遍。

## 架构说明

- **Wire 格式。** Rust 与 Go 在 FFI 边界用 JSON 交换，用
  `{"$i":"..."}` / `{"$f":"..."}` 包装类型化 integer / float。
  保留任意精度以及 encode / decode 时的 `:i` / `:f` 区分。
- **内存所有权。** Rust 分配输出 buffer；Go 拷贝到 slice 后立刻回调
  `ktav_free`。跨 FFI 边界无长期共享内存。
- **加载器。** `internal/native` 通过 `sync.Once` 在每个进程 dlopen
  一次。Windows 走 `golang.org/x/sys/windows`；Unix 走
  `purego.Dlopen/Dlsym`。

## 发布流程

在 `main` 上打 `v<X.Y.Z>` tag。Release workflow 交叉编译六个平台的
二进制（`linux` amd64/arm64、`darwin` amd64/arm64、`windows` amd64/arm64），
作为 GitHub Release assets 附加，Go proxy 会自动索引 tag。
`internal/native/loader.go` 中的 `LibVersion` 常量必须与 tag 对齐 ——
在打 tag 的同一 commit 里更新。

## 哲学

Ktav 的座右铭：**"做配置的朋友，而非考官。"** 提议 Go 专有特性前先问：

- 这是否新增了读者必须记住的规则？
- 这能否放进用户代码而非库？
- 这是否侵蚀了"no magic types"原则？

新规则代价高。对不明显归属这里的东西一律拒绝。

## 语言策略

本仓库参与 org 级别的三语言策略（EN / RU / ZH）。每个 prose 文件都有
三个并行版本 —— 命名约定和"三个翻译在同一 commit 更新"的规则见
[`ktav-lang/.github/AGENTS.md`](https://github.com/ktav-lang/.github/blob/main/AGENTS.md)。
