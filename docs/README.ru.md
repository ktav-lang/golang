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
// name: frontend
// notes: null
// port: 8443
// ratio: 0.95
// tls: true
// upstreams: [
//     {
//         host: a.example
//         port: 1080
//     }
//     {
//         host: b.example
//         port: 1080
//     }
// ]
```

`encoding/json` сортирует ключи Go-map, поэтому порядок вывода
детерминирован; сам `map[string]any` не сохраняет порядок вставки.

Полный запускаемый пример — в [`examples/basic`](../examples/basic/main.go).

## API

| Функция | Назначение |
| --- | --- |
| `Loads(s string) (any, error)` | Разобрать документ Ktav в нативные Go-значения. Верхний уровень может быть объектом (`map[string]any`) или массивом (`[]any`) согласно spec § 5.0.1. |
| `LoadsStrict(s string) (any, error)` | Разобрать документ в strict-режиме с проверкой канонической записи чисел. Та же Go-таблица типов, что и у `Loads`. |
| `LoadsInto(s string, target any) error` | Разобрать в произвольный `target` (struct, map, …) через `encoding/json`. |
| `Dumps(v any) (string, error)` | Сериализовать Go-значение в Ktav-текст. Верхний уровень должен сериализоваться в объект или массив. |
| `DumpsForceStrings(v any) (string, error)` | Как `Dumps`, но все leaf-скаляры (integer, float, bool, null) приводятся к String через `::`. Составные значения сохраняют свою структуру. |
| `EmitCanonical(v any) (string, error)` | Вывести Go-значение в канонический Ktav (spec § 5.9). `encoding/json` сортирует строковые ключи Go-map лексикографически. |
| `CanonicalFromSource(src string) (string, error)` | Разобрать Ktav и сразу вывести каноническую форму, сохраняя порядок ключей источника. |
| `FormatSource(src string) (string, error)` | Отформатировать Ktav-источник в нормализованное написание, **сохраняя все комментарии**. См. ниже. |

## Форматирование — каноническое написание с сохранением комментариев

`FormatSource` и `EmitCanonical` — разные операции:

- **`EmitCanonical`** принимает Go-значение и пишет каноническую форму.
  В значении нет комментариев, поэтому сохранить их нельзя.
- **`FormatSource`** принимает исходный текст и нормализует написание,
  сохраняя каждый комментарий, занимающий целую строку (spec § 3.4).
  И он, и `CanonicalFromSource` сохраняют исходный порядок ключей.

Ниже приведены используемые в примерах импорты:

```go
import (
    "fmt"

    ktav "github.com/ktav-lang/golang"
)
```

```go
func formatExample() error {
	out, err := ktav.FormatSource("## why\na:   {x: 1}\n")
	if err != nil {
		return err
	}
	fmt.Print(out)
	return nil
}
```

Пустые строки служат подсказками группировки: серия из двух и более
сокращается до одной, а пустая отбивка непосредственно внутри скобок
удаляется.

```go
func formatBlankLinesExample() error {
	out, err := ktav.FormatSource("## keep me\n\n\nport: 8080\n")
	if err != nil {
		return err
	}
	fmt.Print(out)
	return nil
}
```

Форматирование достигает неподвижной точки. Обработайте результат
`(string, error)`, прежде чем передавать строку форматтеру снова:

```go
func isFixedPoint(src string) (bool, error) {
	once, err := ktav.FormatSource(src)
	if err != nil {
		return false, err
	}
	twice, err := ktav.FormatSource(once)
	if err != nil {
		return false, err
	}
	return once == twice, nil
}
```

Без комментариев и пустых строк результат совпадает с
`CanonicalFromSource` того же текста.

## Структурированные ошибки

Сбои нативного парсера и writer'а несут структурированный конверт,
общий для биндингов Ktav, поэтому инструмент может работать с полями, а
не разбирать сообщение. Другие сбои, например загрузка/скачивание
библиотеки и преобразование Go JSON, остаются обычными Go-ошибками и не
обязаны быть `*ktav.Error`:

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
| `Msg` | Текст ошибки. Для ошибок native-ядра используется рендеринг ядра; host-синтезированные ошибки используют формулировки биндинга. |
| `Class` | Класс ошибки — `DuplicateKey`, `Unrepresentable`, `UnrepresentableAt`, `InvalidUtf8`, `Message`. |
| `Reason` | Host-writer использует, например, `NonFiniteFloat`, `ScalarRoot` и `EmptyKeyName`. `InvalidUtf8` — класс ошибки источника с пустым reason. |
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

Для структурированных ошибок native-ядра `Msg` берётся дословно, а не
собирается здесь, поэтому один и тот же документ даёт одинаковый текст
ошибки во всех биндингах Ktav. Ошибки загрузчика, преобразования Go и
host-синтезированной проверки не входят в эту межъязыковую гарантию.

## Соответствие типов

| Ktav             | Go                                              |
| ---------------- | ----------------------------------------------- |
| `null`           | `nil`                                           |
| `true` / `false` | `bool`                                          |
| integer scalar   | `int64` в диапазоне; иначе `string`             |
| float scalar     | `float64`                                       |
| bare scalar      | `string`                                        |
| `[ ... ]`        | `[]any`                                         |
| `{ ... }`        | `map[string]any` (порядок вставки не гарантирован) |

В spec 0.8 integer и float выводятся из лексической формы скалярного тела
(типизированных маркеров нет). Integer вне диапазона `int64` разбирается
как String. Go `int*` / `uint*` / `*big.Int` сериализуются как integer
scalar, но `*big.Int` вне диапазона при повторном разборе становится
String; `float32` / `float64` становятся float scalar, а `string`
остаётся bare scalar. `Dumps` и `EmitCanonical` отклоняют `NaN` и `±Inf`,
а `DumpsForceStrings` преобразует их в String-leaf. Структуры сначала
проходят через `encoding/json`, теги `json:"..."` учитываются.

## Экранирование в ключах

Начиная со spec 0.6.4 литеральные `.` или `:` внутри сегмента ключа
записываются через backslash:

```text
## Ключ состоит из одного сегмента "a.b".
a\.b: v
## Ключ состоит из одного сегмента "a:b".
a\:b: v
## Это путь ["x", "y.z"].
x.y\.z: v
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
