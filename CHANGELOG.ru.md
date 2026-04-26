# Changelog

**Языки:** [English](CHANGELOG.md) · **Русский** · [简体中文](CHANGELOG.zh.md)

Все значимые изменения Go-биндинга документируются здесь. Формат основан
на [Keep a Changelog](https://keepachangelog.com/ru/1.1.0/);
версионирование — [Semantic Versioning](https://semver.org/) с pre-1.0
соглашением, что MINOR bump — ломающий.

Этот changelog отслеживает **релизы биндинга**, а не изменения самого
формата Ktav — для последнего см.
[`ktav-lang/spec`](https://github.com/ktav-lang/spec/blob/main/CHANGELOG.md).

## 0.1.1 — 2026-04-26

### Изменено

- **Подхватили `ktav 0.1.4`** — untyped путь `parse() → Value` в
  upstream Rust crate (тот, что использует `cabi`) теперь ~30%
  быстрее на маленьких документах и ~13% на больших, благодаря
  однострочной правке initial capacity для `Frame::Object` (4 → 8).
  Каждый `ktav.Loads` получит ускорение прозрачно.

## 0.1.0 — первый публичный релиз

Первый релиз. Цель — **формат Ktav 0.1**.

### Module path

Опубликован как `github.com/ktav-lang/golang`. `go get` подхватит через
Go-прокси автоматически после появления тега `v0.1.0`.

### Публичный API

- `Loads(s string) (any, error)` — разобрать документ Ktav.
- `LoadsInto(s string, target any) error` — разобрать в `target` через
  `encoding/json`.
- `Dumps(v any) (string, error)` — сериализовать Go-значение в Ktav.
- `Error` — типизированная ошибка парсинга/рендера, ловится через
  `errors.As`.

### Архитектура

- **Нативное ядро** — референсный Rust-крейт `ktav`, обёрнутый тонким
  `extern "C"` C ABI (`crates/cabi`) и распространяемый как
  прекомпилированный `.so` / `.dylib` / `.dll`.
- **Go-лоадер** — `purego` (без cgo): библиотека резолвится на первый
  вызов из `$KTAV_LIB_PATH` или скачивается один раз в `UserCacheDir` из
  соответствующего GitHub Release asset.
- **Wire-формат** — JSON между Rust и Go с тегированными обёртками
  `{"$i":"..."}` / `{"$f":"..."}` для lossless round-trip типизированных
  integer / float и произвольной точности (`*big.Int`).

### Соответствие типов

| Ktav             | Go                                              |
| ---------------- | ----------------------------------------------- |
| `null`           | `nil`                                           |
| `true` / `false` | `bool`                                          |
| `:i <digits>`    | `int64` если помещается, иначе `*big.Int`       |
| `:f <number>`    | `float64`                                       |
| scalar без маркера | `string`                                      |
| `[ ... ]`        | `[]any`                                         |
| `{ ... }`        | `map[string]any` (порядок вставки сохраняется)  |

### Платформы

Прекомпилированные нативные бинари:

- `linux/amd64`, `linux/arm64` (glibc)
- `darwin/amd64`, `darwin/arm64`
- `windows/amd64`, `windows/arm64`

Alpine (musl) — в следующем релизе.

### Протестировано на

Полная conformance-сьюта Ktav 0.1 (все `valid/` и `invalid/` фикстуры)
на Go 1.21 / 1.22 / 1.23 × Linux / macOS / Windows.

### Благодарности

Построено поверх reference-Rust-крейта `ktav`. Динамическая загрузка через
[`ebitengine/purego`](https://github.com/ebitengine/purego).
