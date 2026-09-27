>>>>> lang=en
## Dev setup

You need:

- Go **1.21+**.
- A Rust toolchain via [`rustup`](https://rustup.rs/). MSRV: **1.71**.
- `git`.

Layout during development — the Go package loads the Rust-built
`ktav_cabi` cdylib via `purego`. Initialise the pinned spec submodule;
an adjacent spec checkout is not used:

```
ktav-lang/
└── golang/    ← this repo
    ├── spec/  ← pinned submodule with conformance fixtures
    └── crates/cabi/
```

```bash
git submodule update --init spec
cargo build --release -p ktav-cabi
```

The library is loaded from `target/release` (or
`$CARGO_TARGET_DIR/release`). Set `KTAV_LIB_PATH` only to override its path.

The Rust C ABI crate (`crates/cabi/`) depends on the published `ktav`
crate on crates.io by default. For local cross-repo edits, switch the
`workspace.dependencies.ktav` entry in `Cargo.toml` to
`{ path = "../rust" }`.

>>>>> lang=ru
## Локальная разработка

Нужно:

- Go **1.21+**.
- Rust toolchain через [`rustup`](https://rustup.rs/). MSRV: **1.71**.
- `git`.

Рабочая раскладка — Go-пакет грузит собранный Rust-ом `ktav_cabi` cdylib
через `purego`. Инициализируй закреплённый submodule spec; соседняя
копия spec не используется:

```
ktav-lang/
└── golang/    ← this repo
    ├── spec/  ← закреплённый submodule с conformance-фикстурами
    └── crates/cabi/
```

```bash
git submodule update --init spec
cargo build --release -p ktav-cabi
```

Библиотека загружается из `target/release` (или
`$CARGO_TARGET_DIR/release`). `KTAV_LIB_PATH` задаёт путь только при
необходимости переопределить его.

Rust C ABI крейт (`crates/cabi/`) по умолчанию тянет опубликованный
`ktav` с crates.io. Для локальных правок между репо переключи
`workspace.dependencies.ktav` в `Cargo.toml` на `{ path = "../rust" }`.

>>>>> lang=zh
## 开发环境

需要：

- Go **1.21+**。
- 通过 [`rustup`](https://rustup.rs/) 安装的 Rust toolchain。MSRV: **1.71**。
- `git`。

开发布局 —— Go 包通过 `purego` 加载 Rust 编译出的 `ktav_cabi` cdylib。
初始化固定版本的 spec submodule；旁边的 spec 副本不会被使用：

```
ktav-lang/
└── golang/    ← this repo
    ├── spec/  ← 含 conformance fixtures 的固定版本 submodule
    └── crates/cabi/
```

```bash
git submodule update --init spec
cargo build --release -p ktav-cabi
```

库从 `target/release` 加载（或从 `$CARGO_TARGET_DIR/release` 加载）。
只有需要覆盖默认路径时才设置 `KTAV_LIB_PATH`。

Rust C ABI crate（`crates/cabi/`）默认依赖 crates.io 上发布的 `ktav`。
跨仓库本地改动时，把 `Cargo.toml` 的
`workspace.dependencies.ktav` 切换为 `{ path = "../rust" }`。

