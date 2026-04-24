# Contributing в ktav (Go)

**Языки:** [English](CONTRIBUTING.md) · **Русский** · [简体中文](CONTRIBUTING.zh.md)

## Основные правила

### 1. Каждый багфикс приходит с регрессионным тестом

Когда нашёл баг, **перед тем как чинить**, напиши тест, который его
воспроизводит — тест **должен падать на `main`** и проходить после
фикса. Оба в одном PR.

Тесты лежат в корне репо:

| Файл                    | Область                                                  |
| ----------------------- | -------------------------------------------------------- |
| `ktav_smoke_test.go`    | Loads / Dumps happy paths, bigint, форма ошибок.         |
| `conformance_test.go`   | Cross-language conformance против `ktav-lang/spec`.      |

### 2. Не переизобретай формат в биндингах

Этот Go-пакет намеренно тонкая обёртка. Поведение парсера и формата —
в Rust-крейте ([`ktav-lang/rust`](https://github.com/ktav-lang/rust));
правки там автоматически обновляют все языковые биндинги. В этом репо
только **Go-специфичная эргономика**: type mapping, purego-лоадер,
cache / download логика.

Если изменение требует правки формата — сначала обсуждение в
[`ktav-lang/spec`](https://github.com/ktav-lang/spec).

### 3. Изменения публичного API помечают совместимость

Если трогаешь что-то экспортируемое из `ktav`, в описании PR укажи:

- **semver-совместимо** (добавления, более мягкие сигнатуры, правки
  доков); или
- **semver-ломающе** (переименование / удаление, смена сигнатур,
  более жёсткие типы) — тогда bump версии уезжает в следующий MINOR
  пока мы pre-1.0.

Обновляй `CHANGELOG.md` и оба перевода в том же PR.

### 4. Один концепт на коммит

Коммиты атомарны: багфикс и его тест — вместе, фича и её тесты — вместе,
переименование — отдельно, рефактор — отдельно. `git log --oneline`
должен читаться как changelog. Префиксы `feat:` / `fix:` не используем
— conventional commits здесь не применяем.

### 5. Нативная библиотека идёт в ногу с Go-модулем

Константа `LibVersion` (`internal/native/loader.go`) **должна** совпадать
с тегом релиза. Если поднимаешь версию Go-модуля — поднимай
`LibVersion` в том же коммите. Рассинхрон приводит к тому, что потребитель
качает библиотеку не от той версии кода.

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
├── golang/    ← этот репо
├── rust/      ← соседний Rust-крейт (path-зависимость для local dev)
└── spec/      ← conformance-фикстуры (git submodule в golang/spec/)
```

Rust C ABI крейт (`crates/cabi/`) по умолчанию тянет опубликованный
`ktav` с crates.io. Для локальных правок между репо переключи
`workspace.dependencies.ktav` в `Cargo.toml` на `{ path = "../rust" }`.

### Сборка

```bash
# 1. Собрать нативную либу под хост-платформу.
cargo build --release -p ktav-cabi

# 2. Направить Go на неё.
export KTAV_LIB_PATH="$PWD/target/release/libktav_cabi.so"   # Linux
#      ="$PWD/target/release/libktav_cabi.dylib"             # macOS
#      ="$PWD/target/release/ktav_cabi.dll"                  # Windows

# 3. Для conformance-тестов указать путь к spec-submodule.
git submodule update --init
export KTAV_SPEC_ROOT="$PWD/spec/versions/0.1/tests"
```

### Тесты

```bash
go test -v ./...                # полный прогон
go test -run TestSmoke ./...    # фильтр по имени
go test -run Conformance ./...  # только spec-фикстуры
```

Если `KTAV_LIB_PATH` или `KTAV_SPEC_ROOT` не заданы — соответствующие
тесты **скипаются**, а не падают, чтобы `go test` на голой копии
оставался зелёным.

### Линт

```bash
go vet -unsafeptr=false ./...
gofmt -l .                            # должен вернуть пустоту
cargo fmt --all --check
cargo clippy --release -p ktav-cabi -- -D warnings
```

CI гоняет то же; прогоняй локально перед push.

## Заметки об архитектуре

- **Wire-формат.** Rust и Go обмениваются JSON через FFI-границу с
  обёртками `{"$i":"..."}` / `{"$f":"..."}` для типизированных
  integer / float. Это сохраняет произвольную точность и различение
  `:i` vs `:f` сквозь encode / decode.
- **Владение памятью.** Rust аллоцирует выходной буфер; Go копирует в
  slice и сразу дёргает `ktav_free` на Rust-стороне. Сквозь
  FFI-границу не живёт ни один долгий буфер.
- **Лоадер.** `internal/native` один раз на процесс dlopen-ит
  библиотеку (`sync.Once`). На Windows — через
  `golang.org/x/sys/windows`; на Unix — через `purego.Dlopen/Dlsym`.

## Релизный процесс

Тег `v<X.Y.Z>` на `main`. Release workflow кросс-компилирует шесть
платформенных бинарей (`linux` amd64/arm64, `darwin` amd64/arm64,
`windows` amd64/arm64), прикрепляет как assets GitHub Release, а Go
proxy подхватит тег сам. Константа `LibVersion` в
`internal/native/loader.go` должна совпадать с тегом — меняется в том же
коммите.

## Философия

Девиз Ktav: **"быть другом конфигу, а не его экзаменатором."** Перед тем
как предложить Go-специфичную фичу, спроси:

- Добавляет ли это новое правило, которое читатель должен держать в
  голове?
- Может ли это жить в пользовательском коде вместо библиотеки?
- Не размывает ли это принцип "no magic types"?

Новые правила дороги. Отвергай всё, что не принадлежит сюда очевидно.

## Языковая политика

Этот репо участвует в трёхъязычной политике org-уровня (EN / RU / ZH).
Каждый prose-файл живёт в трёх параллельных версиях — см.
[`ktav-lang/.github/AGENTS.md`](https://github.com/ktav-lang/.github/blob/main/AGENTS.md)
про naming convention и правило "обновлять все три в одном коммите".
