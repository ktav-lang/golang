>>>>> lang=en
## Dev setup

You need:

- Go **1.21+**.
- A Rust toolchain via [`rustup`](https://rustup.rs/). MSRV: **1.70**.
- `git`.

Layout during development — the Go package loads the Rust-built
`ktav_cabi` cdylib via `purego`. Clone the sibling spec repo (used by
conformance tests) next to this one or initialise the submodule:

```
ktav-lang/
├── golang/    ← this repo
├── rust/      ← sibling Rust crate (path dep for local dev)
└── spec/      ← conformance fixtures (git submodule at golang/spec/)
```

The Rust C ABI crate (`crates/cabi/`) depends on the published `ktav`
crate on crates.io by default. For local cross-repo edits, switch the
`workspace.dependencies.ktav` entry in `Cargo.toml` to
`{ path = "../rust" }`.

>>>>> lang=ru
## Локальная разработка

Нужно:

- Go **1.21+**.
- Rust toolchain через [`rustup`](https://rustup.rs/). MSRV: **1.70**.
- `git`.

Рабочая раскладка — Go-пакет грузит собранный Rust-ом `ktav_cabi` cdylib
через `purego`. Рядом с репо клонируй spec (используется conformance-
тестами), либо подними submodule:

```
ktav-lang/
├── golang/    ← this repo
├── rust/      ← sibling Rust crate (path dep for local dev)
└── spec/      ← conformance fixtures (git submodule at golang/spec/)
```

Rust C ABI крейт (`crates/cabi/`) по умолчанию тянет опубликованный
`ktav` с crates.io. Для локальных правок между репо переключи
`workspace.dependencies.ktav` в `Cargo.toml` на `{ path = "../rust" }`.

>>>>> lang=zh
## 开发环境

需要：

- Go **1.21+**。
- 通过 [`rustup`](https://rustup.rs/) 安装的 Rust toolchain。MSRV: **1.70**。
- `git`。

开发布局 —— Go 包通过 `purego` 加载 Rust 编译出的 `ktav_cabi` cdylib。
把 spec 仓库（conformance 测试用）克隆到旁边，或初始化 submodule：

```
ktav-lang/
├── golang/    ← this repo
├── rust/      ← sibling Rust crate (path dep for local dev)
└── spec/      ← conformance fixtures (git submodule at golang/spec/)
```

Rust C ABI crate（`crates/cabi/`）默认依赖 crates.io 上发布的 `ktav`。
跨仓库本地改动时，把 `Cargo.toml` 的
`workspace.dependencies.ktav` 切换为 `{ path = "../rust" }`。

