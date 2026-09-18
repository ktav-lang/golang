# ktav — Go биндинги

[![Go Reference](https://pkg.go.dev/badge/github.com/ktav-lang/golang.svg)](https://pkg.go.dev/github.com/ktav-lang/golang)
[![CI](https://img.shields.io/github/actions/workflow/status/ktav-lang/golang/ci.yml?style=flat-square&logo=github&label=CI)](https://github.com/ktav-lang/golang/actions)
![License: MIT OR Apache-2.0](https://img.shields.io/badge/license-MIT%20OR%20Apache--2.0-blue?style=flat-square)
[![Playground](https://img.shields.io/badge/playground-try%20online-7c3aed?style=flat-square&logo=rocket&logoColor=white)](https://ktav-lang.github.io/)

**Языки:** [English](../README.md) · **Русский** · [简体中文](README.zh.md)

**Песочница:** конвертация JSON / YAML / TOML / INI ⇄ Ktav прямо в браузере — **[ktav-lang.github.io](https://ktav-lang.github.io/)**.

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
port: 8080
ratio: 0.75
tls: true
tags: [
    prod
    eu-west-1
]
db.host: primary.internal
db.timeout: 30
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

Полный запускаемый пример — в [`examples/basic`](../examples/basic/main.go).

## API

| Функция | Назначение |
| --- | --- |
| `Loads(s string) (any, error)` | Разобрать документ Ktav в нативные Go-значения. |
| `LoadsStrict(s string) (any, error)` | Разобрать документ в strict-режиме с проверкой канонической записи чисел. |
| `LoadsInto(s string, target any) error` | Разобрать в произвольный `target` (struct, map, …) через `encoding/json`. |
| `Dumps(v any) (string, error)` | Сериализовать Go-значение в Ktav-текст. Верхний уровень — объект или массив. |
| `DumpsForceStrings(v any) (string, error)` | Как `Dumps`, но все leaf-скаляры (integer, float, bool, null) приводятся к String через `::`. |
| `EmitCanonical(v any) (string, error)` | Канонический Ktav (spec § 5.9 — байт-детерминированный, без inline-соединений). |
| `CanonicalFromSource(src string) (string, error)` | Разобрать Ktav и сразу вывести каноническую форму, сохраняя порядок ключей источника. |
| `FormatSource(src string) (string, error)` | Отформатировать Ktav-источник в нормализованное написание, **сохраняя все комментарии**. См. ниже. |

## Форматирование — каноническое написание с сохранением комментариев

`FormatSource` и `EmitCanonical` — разные операции, и стоит понимать,
какая нужна:

- **`EmitCanonical`** принимает Go-значение и пишет каноническую форму.
  В значении комментариев не существует, поэтому сохранять нечего.
- **`FormatSource`** принимает исходный *текст* и переписывает его
  написание, **сохраняя каждый комментарий дословно** (spec § 3.4:
  комментарий занимает строку целиком). Порядок ключей не меняется.

```go
out, err := ktav.FormatSource("## why\na:   {x: 1}\n")
// "## why\na: {\n    x: 1\n}\n"
// комментарий сохранён; inline-составное развёрнуто в каноническую
// многострочную форму и переотступлено
```

Пустые строки выживают как подсказка группировки, но серия из двух и
более схлопывается ровно в одну, а пустой отбивки сразу внутри скобки не
остаётся:

```go
ktav.FormatSource("## keep me\n\n\nport: 8080\n")
// "## keep me\n\nport: 8080\n" — две пустых строки стали одной
```

Поэтому форматирование — неподвижная точка:

```go
ktav.FormatSource(ktav.FormatSource(x)) == ktav.FormatSource(x)
```

Для документа без комментариев и без пустых строк результат совпадает с
`CanonicalFromSource` того же текста.

## Структурированные ошибки

Каждый сбой несёт тот же структурированный конверт, что и остальные
биндинги Ktav, поэтому инструмент может работать с полями, а не
разбирать сообщение. `Loads`, `Dumps` и прочие возвращают `*Error`:

```go
if _, err := ktav.Loads("a: 1\na: 2\n"); err != nil {
    var kerr *ktav.Error
    if errors.As(err, &kerr) {
        fmt.Println(kerr.Class)       // "DuplicateKey"
        fmt.Println(kerr.Line)        // 2
        fmt.Println(kerr.SpecSection) // "§6.2"
    }
}
```

| Поле | Значение |
| --- | --- |
| `Msg` | Собственный рендеринг ошибки, полученный из ядра. Никогда не пустой; именно его возвращает `Error()`. |
| `Class` | Класс структурированной ошибки — `DuplicateKey`, `Unrepresentable`, `Message`. |
| `Reason` | Код причины на стороне writer'а (spec § 5.9.0), например `NonFiniteFloat`; `""`, когда в конверте null. |
| `Line` | Строка источника, счёт с 1; `0`, когда в конверте null. |
| `LineText` | Текст ошибочной строки. |
| `Span` | `*Span` — байтовые смещения в UTF-8-источнике. |
| `Path` | Точные декодированные сегменты ключа. |
| `Body` | Ошибочное значение как написано. |
| `Canonical` | Какой была бы каноническая форма. |
| `SpecSection` | Нарушенный пункт, например `§3.6/§5.2`. |

Две детали, в которых легко ошибиться:

- **`Span` — это байтовые смещения в UTF-8**, а не кодовые единицы
  UTF-16. Потребитель LSP либо преобразует их, либо договаривается о
  `positionEncoding: "utf-8"`.
- **`Path` — срез сегментов, а не склеенная строка.** Ключ, буквально
  названный `a.b`, — это один сегмент, и его нельзя спутать с
  двухсегментным путём: в проводном контракте нет разделителя, о котором
  можно было бы двусмысленно судить.

`Msg` берётся из ядра дословно, а не собирается здесь, поэтому один и
тот же документ даёт один и тот же текст ошибки в любом биндинге Ktav.

## Соответствие типов

| Ktav             | Go                                              |
| ---------------- | ----------------------------------------------- |
| `null`           | `nil`                                           |
| `true` / `false` | `bool`                                          |
| integer scalar   | `int64` если помещается, иначе `*big.Int`       |
| float scalar     | `float64`                                       |
| bare scalar      | `string`                                        |
| `[ ... ]`        | `[]any`                                         |
| `{ ... }`        | `map[string]any` (порядок вставки сохраняется)  |

В spec 0.5 integer и float выводятся из лексической формы скалярного тела
(типизированных маркеров нет). На сериализации Go `int*` / `uint*` /
`*big.Int` → integer scalar; `float32` / `float64` → float scalar;
`string` остаётся bare scalar. `NaN` и `±Inf` отвергаются. Структуры
сначала проходят через `encoding/json`, теги `json:"..."` учитываются.

## Экранирование в ключах

Начиная со spec 0.6.4 литеральные `.` или `:` внутри сегмента ключа
записываются через backslash:

```text
a\.b: v        // ключ — один сегмент "a.b"     -> map["a.b"] = "v"
a\:b: v        // двоеточие внутри ключа        -> map["a:b"] = "v"
x.y\.z: v      // делим только по первой точке  -> map["x"]["y.z"] = "v"
```

Литеральный backslash в ключе пишется как `\\`.

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

MIT OR Apache-2.0 — см. [LICENSE-MIT](../LICENSE-MIT) и
[LICENSE-APACHE](../LICENSE-APACHE).

## Другие реализации Ktav

- [`spec`](https://github.com/ktav-lang/spec) — спецификация + conformance-тесты
- [`rust`](https://github.com/ktav-lang/rust) — эталонный Rust crate (`cargo add ktav`)
- [`csharp`](https://github.com/ktav-lang/csharp) — C# / .NET (`dotnet add package Ktav`)
- [`java`](https://github.com/ktav-lang/java) — Java / JVM (`io.github.ktav-lang:ktav` на Maven Central)
- [`js`](https://github.com/ktav-lang/js) — JS / TS (`npm install @ktav-lang/ktav`)
- [`php`](https://github.com/ktav-lang/php) — PHP (`composer require ktav-lang/ktav`)
- [`python`](https://github.com/ktav-lang/python) — Python (`pip install ktav`)
