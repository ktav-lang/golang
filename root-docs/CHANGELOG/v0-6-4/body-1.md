>>>>> lang=en
## 0.6.4 — 2026-08-23

### Added

- **`LoadsStrict(s string) (any, error)`** — exposes the Rust strict parser
  through the Go binding and the `ktav_loads_strict` C ABI symbol.

### Changed

- Tracks `ktav 0.6.4` and spec 0.6.4, including the normative float
  canonicalisation boundary and the `notation_boundaries` fixture.
- Native-library resolution now targets the exact `v0.6.4` release asset.

>>>>> lang=ru
## 0.6.4 — 2026-08-23

### Добавлено

- **`LoadsStrict(s string) (any, error)`** — открывает strict-парсер Rust
  через Go-биндинг и символ C ABI `ktav_loads_strict`.

### Изменено

- Go-биндинг отслеживает `ktav 0.6.4` и spec 0.6.4, включая нормативную
  границу канонической записи float и фикстуру `notation_boundaries`.
- Загрузчик нативной библиотеки теперь нацелен на точный asset релиза
  `v0.6.4`.

>>>>> lang=zh
## 0.6.4 — 2026-08-23

### 新增

- **`LoadsStrict(s string) (any, error)`** —— 通过 Go 绑定和
  `ktav_loads_strict` C ABI 符号暴露 Rust strict parser。

### 变更

- 跟踪 `ktav 0.6.4` 与 spec 0.6.4,包括规范化 float 边界和
  `notation_boundaries` fixture。
- 原生库加载器现在指向精确的 `v0.6.4` release asset。

