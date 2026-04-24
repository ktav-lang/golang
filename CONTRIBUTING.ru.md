# Contributing

**Языки:** [English](CONTRIBUTING.md) · **Русский** · [简体中文](CONTRIBUTING.zh.md)

Спасибо за помощь. Репо содержит **две вещи**, сшитые вместе:

- Небольшой Rust-крейт (`crates/cabi/`), оборачивающий референсный
  парсер `ktav` за `extern "C"` API.
- Go-пакет в корне репо, который загружает скомпилированную библиотеку
  в рантайме через [`purego`](https://github.com/ebitengine/purego).

## Локальная разработка

Требуется: Rust (stable), Go 1.21+, git.

```bash
# 1. Собрать нативную библиотеку под хост-платформу.
cargo build --release -p ktav-cabi

# 2. Указать Go путь к ней и прогнать тесты.
export KTAV_LIB_PATH="$PWD/target/release/libktav_cabi.so"   # Linux
#      ="$PWD/target/release/libktav_cabi.dylib"             # macOS
#      ="$PWD/target/release/ktav_cabi.dll"                  # Windows

# 3. Для conformance-тестов указать путь к spec-фикстурам.
export KTAV_SPEC_ROOT="$PWD/spec/versions/0.1/tests"

go test ./...
```

`git submodule update --init` подтягивает общие spec-фикстуры.

## Заметки об архитектуре

- **Wire-формат.** Rust и Go обмениваются JSON с обёртками
  `{"$i":"..."}` / `{"$f":"..."}` для типизированных integer / float.
  Это сохраняет произвольную точность и различение `:i` vs `:f` через
  FFI-границу.
- **Память.** Rust аллоцирует выходной буфер; Go копирует в slice и
  сразу вызывает `ktav_free` на Rust-стороне. Через FFI-границу
  не живёт ни один долгий буфер.
- **Лоадер.** На Go-стороне `internal/native` один раз на процесс
  dlopen-ит библиотеку (sync.Once). На Windows — через
  `golang.org/x/sys/windows`; на Unix — через `purego.Dlopen/Dlsym`.

## Релизы

Тег `v<X.Y.Z>` на `main`. Release workflow кросс-компилирует шесть
платформенных бинарей, прикрепляет как assets GitHub Release, а Go proxy
подхватывает тег сам. Константа `LibVersion` в
`internal/native/loader.go` должна совпадать с тегом — меняется в том же
коммите что и тег-message.

## Лицензия

Внося изменения, вы соглашаетесь, что код публикуется под MIT (см.
[LICENSE](LICENSE)).
