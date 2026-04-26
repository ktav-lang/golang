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

### Парсинг — декод сразу в типизированную структуру

```go
package main

import (
    "fmt"

    ktav "github.com/ktav-lang/golang"
)

const src = `
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

func main() {
    var cfg Config
    if err := ktav.LoadsInto(src, &cfg); err != nil {
        panic(err)
    }
    fmt.Printf("port=%d host=%s timeout=%ds\n",
        cfg.Port, cfg.DB.Host, cfg.DB.Timeout)
}
```

### Обход — динамическая форма с диспатчем по типу

```go
dyn, _ := ktav.Loads(src)
for k, v := range dyn.(map[string]any) {
    switch x := v.(type) {
    case bool:           fmt.Printf("%s is bool=%v\n", k, x)
    case int64:          fmt.Printf("%s is int=%d\n", k, x)
    case float64:        fmt.Printf("%s is float=%g\n", k, x)
    case string:         fmt.Printf("%s is str=%q\n", k, x)
    case []any:          fmt.Printf("%s is array(%d)\n", k, len(x))
    case map[string]any: fmt.Printf("%s is object(%d)\n", k, len(x))
    case nil:            fmt.Printf("%s is null\n", k)
    }
}
```

### Билд + рендер — собираем документ в коде

```go
doc := map[string]any{
    "name":  "frontend",
    "port":  int64(8443),
    "tls":   true,
    "ratio": 0.95,
    "upstreams": []any{
        map[string]any{"host": "a.example", "port": int64(1080)},
        map[string]any{"host": "b.example", "port": int64(1080)},
    },
    "notes": nil,
}
out, _ := ktav.Dumps(doc)
fmt.Print(out)
```

Полный запускаемый пример — в [`examples/basic`](examples/basic/main.go).

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

## Другие реализации Ktav

- [`spec`](https://github.com/ktav-lang/spec) — спецификация + conformance-тесты
- [`rust`](https://github.com/ktav-lang/rust) — эталонный Rust crate (`cargo add ktav`)
- [`csharp`](https://github.com/ktav-lang/csharp) — C# / .NET (`dotnet add package Ktav`)
- [`java`](https://github.com/ktav-lang/java) — Java / JVM (`io.github.ktav-lang:ktav` на Maven Central)
- [`js`](https://github.com/ktav-lang/js) — JS / TS (`npm install @ktav-lang/ktav`)
- [`php`](https://github.com/ktav-lang/php) — PHP (`composer require ktav-lang/ktav`)
- [`python`](https://github.com/ktav-lang/python) — Python (`pip install ktav`)
