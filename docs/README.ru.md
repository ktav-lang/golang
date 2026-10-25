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
| `EditValue(src, path string, replacement any) (string, error)` | Заменить значение по текстовому dotted-пути **в исходном тексте** и вернуть исправленный источник. См. [Правка исходника](#правка-исходника). |
| `EditValueSegments(src string, segments []string, replacement any) (string, error)` | То же с уже декодированными literal-сегментами (без повторного unescape). |

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

> **Ещё не выпущено (подготовка 0.9):** Эти API редактирования и обновлённая диагностика не входят в опубликованные пакеты 0.8.1. Native-backend требует закреплённое ядро 0.9 (C ABI 2 для C-FFI); pure-backend остаётся самостоятельным.

## Правка исходника

`EditValue` и `EditValueSegments` меняют одно значение **внутри
исходного текста** и возвращают исправленный источник. Это не
parse → mutate → serialize: сплайсятся только байты цели (плюс, когда
требует читаемость, содержащие её inline-обёртки), поэтому всё остальное
сохраняется байт-в-байт — комментарии, пустые строки, написание ключей,
dotted/inline/block-формы, байты CR / LF / CRLF документа, его BOM и
наличие финального перевода строки.

```go
out, err := ktav.EditValue(src, "targets.storagehost.address", "10.0.0.9:22")
// out == src, где заменены байты только этого скаляра

out, err := ktav.EditValueSegments(src, []string{"targets", "storagehost", "address"}, "10.0.0.9:22")
```

Правила путей:

- `EditValue` принимает текстовый путь (`key1.key2.0.key4`). Внутри
  сегмента `\.` — literal точка, `\\` — literal backslash; любой другой
  escape, хвостовой `\` или пустой сегмент — `*ktav.Error` с
  `Class == "EditInvalidPath"`, где `Body` — UTF-8 byte-офсет проблемы
  внутри аргумента пути.
- `EditValueSegments` принимает уже декодированные literal-сегменты —
  без повторного unescape. Числовой сегмент — строковый ключ в Object и
  zero-based индекс из ASCII-цифр в Array (ведущие нули допустимы, знаки
  — нет). Пустой список сегментов — `"EditNotFound"`.
- Поддерживается только **замена существующего значения**: без create,
  remove и rename. Отсутствующий ключ, индекс вне диапазона или спуск
  через скаляр — `"EditNotFound"`.

Значения замены используют обычную Go-таблицу типов (всё, что принимает
`Dumps`, включая скаляры); integer хоста никогда не превращается во
float.

Гарантии:

- No-op (замена равна старому значению) возвращает исходник
  байт-в-байт, каким бы неканоничным ни было его написание.
- Переходы маркеров минимальны: замена скаляра на числоподобную или
  keyword-подобную String переключает только второй `:` маркера `::`,
  исходные пробелы вокруг сохраняются.
- Dotted-, inline- и блочные формы сохраняются, включая замену целого
  inline-контейнера и опустошённый (но не свернувшийся) блочный
  контейнер.
- Inline-предки раскрываются в блочную форму только когда требует
  читаемость: многострочная строка замены, рост строки сверх
  max(100 Unicode scalar values, прежняя длина физической строки) или
  ≥ 4 inline-escapes с накладными расходами ≥ 25%. Незатронутые вложенные
  ветки остаются inline; новые блочные уровни — четыре пробела и EOL
  источника.
- Ошибки класса `"EditUnsupported"` описывают пары цель/замена без
  lossless-патча: noncontiguous dotted-key/reopen-цели, структурная
  замена, которая потеряла бы внутренние комментарии (правка листа
  внутри такого контейнера работает и сохраняет комментарии), значения без
  lossless-представления в контексте цели.
- Каждый успешный патч перед возвратом повторно парсится и семантически
  верифицируется — тихое повреждение невозможно.
- Native (C ABI) и pure-Go бэкенды совпадают байт-в-байт; оба
  прогоняются через один и тот же regression-корпус.

### Практические примеры правки исходника

Примеры относятся к ещё не выпущенному API 0.9, а не к опубликованному пакету 0.8.1. Они заменяют только существующие значения: не переименовывают ключи, не создают отсутствующие пути и не удаляют записи. Фрагменты независимы; Rust `?` предполагает внешнюю функцию с результатом `Result`. В Go panic используется только для фиксированных корректных данных примера; приложение должно обработать возвращённую ошибку.

Примеры привязок используют самостоятельный pure-бэкенд без native-бинарника. Нативные точки входа принимают те же пути и значения: `ktav.EditValue` (Go), `Ktav.editValue` (Java), `KtavApi.EditValue(..., KtavBackend.Native)` (C#), `Ktav::editValue` (PHP), native/FFI-экспорты пакета (JS), `ktav.edit_value` (Python). C ABI требует версию 2; PyO3/N-API/WASM — совместимое ядро подготовки 0.9. FFI-вызовы Bun/Deno возвращают promise и требуют `await`; показанные pure/Node/WASM-варианты синхронны.

```go
import "github.com/ktav-lang/golang/pure"
```

#### 1. Настройка развёртывания через dotted-ключ

Меняется только адрес; комментарий, dotted-форма и соседнее поле пользователя сохраняются байт-в-байт.

```go
source := "## deployment\ntargets.storagehost.address: old\ntargets.storagehost.user: mcp-agent\n"
updated, err := pure.EditValue(source, "targets.storagehost.address", "192.168.0.88:22")
if err != nil {
    panic(err)
}
if updated != "## deployment\ntargets.storagehost.address: 192.168.0.88:22\ntargets.storagehost.user: mcp-agent\n" { panic("unexpected edit") }
```

Результат:

```ktav
## deployment
targets.storagehost.address: 192.168.0.88:22
targets.storagehost.user: mcp-agent
```

#### 2. Буквальные точки: текстовый путь или сегменты

Ключ исходника в кавычках буквально называется `a.b`. В текстовом пути точка экранируется; API сегментов получает декодированное имя напрямую. Оба вызова возвращают одинаковый результат.

```go
source := "\"a.b\": {port: 80}\n"
updated, err := pure.EditValue(source, "a\\.b.port", int64(90))
if err != nil {
    panic(err)
}
same, err := pure.EditValueSegments(source, []string{"a.b", "port"}, int64(90))
if err != nil {
    panic(err)
}
if updated != "\"a.b\": {port: 90}\n" { panic("unexpected edit") }
if same != updated { panic("path APIs differ") }
```

Результат:

```ktav
"a.b": {port: 90}
```

#### 3. Лист внутри массива со смешанными формами

Здесь `0` — индекс массива. В объекте такой же сегмент обозначал бы ключ с именем `0`; нетронутые контейнеры остаются inline.

```go
source := "servers: [{host: old, keep: yes}, standby]\n"
updated, err := pure.EditValueSegments(source, []string{"servers", "0", "host"}, "new")
if err != nil {
    panic(err)
}
if updated != "servers: [{host: new, keep: yes}, standby]\n" { panic("unexpected edit") }
```

Результат:

```ktav
servers: [{host: new, keep: yes}, standby]
```

#### 4. Похожая на число строка остаётся строкой

Замена имеет тип String, а не Integer. Raw-маркер сохраняет это различие. Для числового значения передавайте типизированный Integer.

```go
source := "port: 8080\n"
updated, err := pure.EditValue(source, "port", "9090")
if err != nil {
    panic(err)
}
if updated != "port:: 9090\n" { panic("unexpected edit") }
```

Результат:

```ktav
port:: 9090
```

#### 5. Многострочная замена: раскрывается только необходимое

Изменённый массив раскрывается с отступом четыре пробела. Нетронутый объект остаётся inline, содержимое verbatim-строки не переотступается.

```go
source := "a: [old, {keep: yes}]\n"
updated, err := pure.EditValue(source, "a.0", "x\ny")
if err != nil {
    panic(err)
}
if updated != "a: [\n    ((\nx\ny\n    ))\n    {keep: yes}\n]\n" { panic("unexpected edit") }
```

Результат:

```ktav
a: [
    ((
x
y
    ))
    {keep: yes}
]
```

#### 6. No-op сохраняет неканоническое написание

Значение Integer уже равно 1. Проверка равенства возвращает исходные байты, включая `+1` и пробелы разделителя; критерии читаемости не применяются.

```go
source := "n  : +1\n"
updated, err := pure.EditValue(source, "n", int64(1))
if err != nil {
    panic(err)
}
if updated != source { panic("no-op changed source") }
```

Результат:

```ktav
n  : +1
```

#### 7. BOM, CRLF и последняя строка без перевода

Результат сохраняет BOM и CRLF, не добавляя финальный перевод строки. Экранированное представление делает эти байты видимыми.

```go
source := "\ufeffport: 8080\r\nhost: db"
updated, err := pure.EditValue(source, "port", int64(9090))
if err != nil {
    panic(err)
}
if updated != "\ufeffport: 9090\r\nhost: db" { panic("unexpected edit") }
```

Результат:

```text
"\uFEFFport: 9090\r\nhost: db"
```

Критерии раскрытия фиксированы: многострочная замена, рост сверх `max(100, исходная длина строки)` либо не менее четырёх экранирований с накладными расходами от 25% для отдельной строки замены. Ширина считается в Unicode scalar values, а не байтах UTF-8 или колонках терминала. Уже длинная строка без роста сохраняется. Новые уровни используют четыре пробела и EOL исходника. Если CR или строка данных `))` мешают lossless-кодированию вне inline, сохраняется корректный inline-вариант; эти символы не означают автоматический отказ любой inline-правки.

#### 8. Точная замена Integer

Домен Integer в исходнике — знаковый 64-битный. Это значение больше безопасного диапазона JavaScript Number, но помещается в i64; используйте точное host-представление, а не округлённое число с плавающей точкой. Типизированный Integer вне i64 невозможно сохранить в исходнике как Integer: правка отказывает, не обрезая значение. Для текстового большого идентификатора используйте String. Float различает отрицательный и положительный нуль; no-op для Object не зависит от порядка ключей, порядок Array значим.

```go
updated, err := pure.EditValue("n: 1\n", "n", int64(9007199254740993))
if err != nil { panic(err) }
if updated != "n: 9007199254740993\n" { panic("precision lost") }
```

```ktav
n: 9007199254740993
```

### Диагностика и безопасные отказы в 0.9

Координаты исходника — байтовые смещения UTF-8 с учётом BOM; spans полуоткрыты `[start, end)`, строки нумеруются с 1. Поддерживаются LF, отдельный CR и CRLF; CRLF считается одним переводом строки. Один Unicode-символ может занимать несколько байтов.

Незавершённый `a: [\n` даёт `UnclosedCompound`, строку 1 и span `3..5`: начало — самая внутренняя незакрытая скобка, а не начало ключа или строки. Для сырых байтов `61 3A 20 FF` (`a: ` и невалидный UTF-8) валидация сообщает `InvalidUtf8`, строку 1 и покрывающий span `3..4`; `line_text` равен null, а не придуманному тексту после lossy-декодирования. Проверка UTF-8 предшествует грамматике.

Некорректный текстовый путь, включая пустой путь или пустой текстовый сегмент, даёт `EditInvalidPath`; смещение считается в байтах UTF-8 самого пути. Пустой список сегментов, отсутствующий ключ, индекс вне диапазона или спуск через скаляр дают `EditNotFound`. В объекте числовой сегмент остаётся ключом; индекс массива содержит ASCII-цифры, допускает ведущие нули, но не знаки.

Целые промежуточные объекты из dotted/reopened-определений могут занимать несколько физических диапазонов и давать `EditUnsupported`; их существующие листья остаются редактируемыми. Структурная замена контейнера не должна удалять внутренние комментарии. Ошибка не возвращает частичный патч и не меняет переданный исходник. Результат повторно разбирается и семантически проверяется: цель должна равняться замене, остальное дерево значений — исходному.

В привязках ошибочный исходник даёт обычный конверт парсера до проверки пути. Некорректный wire-payload — ошибка `Message` без выдуманных координат. Wire-конверт содержит `error`, `reason`, `line`, `line_text`, `span`, `path`, `body`, `canonical`, `spec_section`, `message`; имена host-accessors описаны в разделе ошибок. В `body` находятся десятичное байтовое смещение пути для `EditInvalidPath`, null для `EditNotFound` и причина отказа для `EditUnsupported`.

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

# Pure-Go бэкенд

Установите через `go get github.com/ktav-lang/golang/pure@v0.8.1`, затем импортируйте:

```go
package main

import (
    "fmt"
    "log"

    "github.com/ktav-lang/golang/pure"
)

func main() {
    value, err := pure.Loads("server.port: 8080\n")
    if err != nil {
        log.Fatal(err)
    }
    text, err := pure.Dumps(value)
    if err != nil {
        log.Fatal(err)
    }
    restored, err := pure.Loads(text)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Print(text)
    fmt.Printf("%#v\n", restored)
}
```

Функции пакета и `pure.New()` предоставляют все восемь операций: `Loads`,
`LoadsStrict`, `LoadsInto`, `Dumps`, `DumpsForceStrings`, `EmitCanonical`,
`FormatSource`, `CanonicalFromSource`, а также полные псевдонимы `pure.Error`/
`pure.Span`. Пакет использует общий `internal/engine`; runtime-граф — только stdlib.

Совместимый root API `ktav.New(ktav.BackendPure)` сохраняется, но корневые импорты
требуют `github.com/ebitengine/purego v0.8.4` и `golang.org/x/sys v0.28.0` даже
при BackendPure. Offline-потребитель использует только файловый proxy, readonly
режим и белый список импортов, проверяя отсутствие внешних runtime-импортов `/pure`.
Его CI-задача прогоняет весь корпус конформности 0.8 через установленный артефакт
`/pure` с `CGO_ENABLED=0` и с перехваченными на `PATH` `rustc`/`cargo`.

Inline-составные значения ограничены 128 уровнями. Ключи обычных карт сортируются
по алфавиту (O(n log n)); канонизация и форматирование сохраняют порядок источника.
Пути `encoding/json` имеют собственные пределы глубины; бесконечная глубина не
обещана. Node-документация и Go-сборка/тесты — инструменты разработки, не runtime
зависимости `/pure`.

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
