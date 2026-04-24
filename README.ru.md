# ktav — Go биндинги

**Языки:** [English](README.md) · **Русский** · [简体中文](README.zh.md)

Go биндинги для [формата конфигурации Ktav](https://github.com/ktav-lang/spec).
Тонкая обёртка вокруг референсного Rust-парсера, который грузится в
рантайме через [`purego`](https://github.com/ebitengine/purego) — поэтому
**`cgo` у потребителя не нужен**, обычный `go build` работает из коробки.

```bash
go get github.com/ktav-lang/golang
```

## Быстрый старт

```go
package main

import (
    "fmt"

    ktav "github.com/ktav-lang/golang"
)

func main() {
    src := `
service: web
port:i 8080
ratio:f 0.75
tls: true
tags: [
    prod
    eu-west-1
]
db.host: primary.internal
db.timeout:i 30
`
    cfg, err := ktav.Loads(src)
    if err != nil {
        panic(err)
    }
    fmt.Printf("%#v\n", cfg)
}
```

Декодирование в структуру:

```go
type Config struct {
    Service string   `json:"service"`
    Port    int64    `json:"port"`
    Ratio   float64  `json:"ratio"`
    TLS     bool     `json:"tls"`
    Tags    []string `json:"tags"`
    DB      struct {
        Host    string `json:"host"`
        Timeout int64  `json:"timeout"`
    } `json:"db"`
}

var cfg Config
if err := ktav.LoadsInto(src, &cfg); err != nil {
    // ...
}
```

## API

| Функция | Назначение |
| --- | --- |
| `Loads(s string) (any, error)` | Разобрать документ Ktav в нативные Go-значения. |
| `LoadsInto(s string, target any) error` | Разобрать в произвольный `target` (struct, map, …) через `encoding/json`. |
| `Dumps(v any) (string, error)` | Сериализовать Go-значение в Ktav-текст. Верхний уровень должен быть объектом. |

## Соответствие типов

| Ktav             | Go                                              |
| ---------------- | ----------------------------------------------- |
| `null`           | `nil`                                           |
| `true` / `false` | `bool`                                          |
| `:i <digits>`    | `int64` если помещается, иначе `*big.Int`       |
| `:f <number>`    | `float64`                                       |
| scalar без маркера | `string`                                      |
| `[ ... ]`        | `[]any`                                         |
| `{ ... }`        | `map[string]any` (порядок вставки сохраняется)  |

На сериализации Go `int*` / `uint*` / `*big.Int` → `:i`; `float32` /
`float64` → `:f`; `string` остаётся bare scalar. `NaN` и `±Inf`
отвергаются. Структуры сначала проходят через `encoding/json`, так что
теги `json:"..."` учитываются.

## Как резолвится нативная библиотека

На первый вызов Go-пакет ищет `ktav_cabi` в следующем порядке:

1. **`$KTAV_LIB_PATH`** — абсолютный путь к локальной сборке. Удобно для
   разработки.
2. **User cache** — `<os.UserCacheDir>/ktav-go/v<version>/…`, скачано на
   предыдущем вызове.
3. **Скачивание с GitHub Release** — подходящий asset качается один раз
   с `github.com/ktav-lang/golang/releases/download/v<version>/<name>` и
   кэшируется в (2). Нужен интернет на первый вызов после установки.

## Поддержка рантаймов

- Go 1.21+.
- Прекомпилированные бинари: `linux/amd64`, `linux/arm64`,
  `darwin/amd64`, `darwin/arm64`, `windows/amd64`, `windows/arm64`.
- Linux-дистрибутивы должны использовать glibc 2.17+ (дефолтный
  Rust-таргет). Alpine (musl) — в планах.

## Лицензия

MIT — см. [LICENSE](LICENSE).

Спецификация Ktav: [ktav-lang/spec](https://github.com/ktav-lang/spec).
Референсный Rust-крейт: [ktav-lang/rust](https://github.com/ktav-lang/rust).
