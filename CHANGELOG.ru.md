# Changelog

**Языки:** [English](CHANGELOG.md) · **Русский** · [简体中文](CHANGELOG.zh.md)

Все значимые изменения Go-биндинга документируются здесь. Формат основан
на [Keep a Changelog](https://keepachangelog.com/ru/1.1.0/);
версионирование — [Semantic Versioning](https://semver.org/) с pre-1.0
соглашением, что MINOR bump — ломающий.

Этот changelog отслеживает **релизы биндинга**, а не изменения самого
формата Ktav — для последнего см.
[`ktav-lang/spec`](https://github.com/ktav-lang/spec/blob/main/CHANGELOG.md).

## 0.5.0 — 2026-05-28

### Ломающие изменения

- **Spec 0.5.0**: типизированные маркеры `:i` / `:f` больше не
  существуют. Числа выводятся из лексической формы тела скаляра
  (`42` → Integer, `3.14` → Float). Документы с явными маркерами
  из spec 0.1.x нужно обновить.
- **Комментарии `##`**: одиночный `#` теперь литеральный символ;
  комментарии требуют `##`. Обновите любой Ktav-исходник с `# комментарий`.
- **Нормализация Float**: значения хранятся в канонической форме
  shortest-decimal. Байт-в-байт сравнение со старым выводом может
  сломаться.
- **C ABI теперь экспортирует шесть символов** — добавлен
  `ktav_emit_canonical`; `KTAV_LIB_PATH` на pre-0.5.0 бинарь упадёт
  с ошибкой отсутствия символа.

### Добавлено

- **`EmitCanonical(v any) (string, error)`** — рендерит Go-значение в
  канонический Ktav (spec § 5.9): детерминированный по байтам вывод,
  без inline-соединений, нормализованные integer / float.
- **`TestConformanceCanonical`** — новый conformance-тест проверяет,
  что `EmitCanonical` даёт вывод, совпадающий с каждым
  `.canonical.ktav`-оракулом из spec-фикстур.

### Изменено

- **Подхватили `ktav 0.5.0`** — следуем upstream Rust crate с
  реализацией spec 0.5: выводимые числовые типы, `##`-комментарии,
  API `emit_canonical`. Submodule spec синхронизирован с тегом
  `v0.5.0`.
- **Лицензия изменена на `MIT OR Apache-2.0`** — в соответствии с
  остальной экосистемой `ktav-lang`. `LICENSE` переименован в
  `LICENSE-MIT`; добавлен `LICENSE-APACHE`.
- **Conformance-тесты обновлены до фикстур spec 0.5** — путь
  `spec/versions/0.5/tests`; `.canonical.ktav` исключены из
  JSON-oracle теста и обрабатываются новым canonical-тестом.


## 0.1.2 — 2026-05-03

### Изменено

- **Подхватили `ktav 0.1.5`** — в upstream Rust crate появился API
  структурированных ошибок (`Error::Structured(ErrorKind)` с
  byte-offset spans), retroactive `#[non_exhaustive]` на error-enum-ах,
  и публичный event-based парсер `ktav::thin`. Поведение Go-биндинга
  для пользователя не меняется: возвращаемые `error`-значения несут то
  же читаемое сообщение (Display-строки семи канонических категорий
  byte-identical к ktav 0.1.4 — проверено собственными pinning-тестами
  ktav). Маппинг `ktav::ErrorKind` на типизированные Go-error-значения
  (чтобы можно было `errors.As(err, &ktav.MissingSeparatorSpaceError{})`
  и т.д.) — отдельная follow-up работа, описанная в
  [`STRUCTURED_ERRORS.md`](https://github.com/ktav-lang/.github/blob/main/STRUCTURED_ERRORS.md).

`go get github.com/ktav-lang/golang@v0.1.2`.

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
