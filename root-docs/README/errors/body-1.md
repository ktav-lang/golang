>>>>> lang=en
## Structured errors

Native parser and writer failures carry the structured envelope shared
by Ktav bindings, so tools can act on fields instead of parsing a
message. Other failures, such as native-library loading/download and Go
JSON conversion errors, are ordinary Go errors and need not be
`*ktav.Error`:

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

| Field | Meaning |
| --- | --- |
| `Msg` | Human-readable error text. Native-origin errors use the core's rendering; host-synthesized errors use binding wording. |
| `Class` | Structured error class — `DuplicateKey`, `Unrepresentable`, `UnrepresentableAt`, `InvalidUtf8`, `Message`. |
| `Reason` | Host writer reasons include `NonFiniteFloat`, `ScalarRoot`, and `EmptyKeyName`. `InvalidUtf8` is a source-error class with an empty reason. |
| `Line` | 1-based source line; `0` when the envelope carries null. |
| `LineText` | Text of the offending line. |
| `Span` | `*Span` — byte offsets into the UTF-8 source. |
| `Path` | Exact decoded key segments. |
| `Body` | The offending value as written. |
| `Canonical` | What the canonical form would have been. |
| `SpecSection` | The clause violated, e.g. `§3.6/§5.2`. |

>>>>> lang=ru
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

>>>>> lang=zh
## 结构化错误

原生解析器和 writer 的失败会携带 Ktav 绑定共用的结构化信封,工具可以
直接处理字段,而不必解析消息文本。其他失败,例如原生库加载/下载和
Go JSON 转换错误,仍是普通 Go error,不保证为 `*ktav.Error`:

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

| 字段 | 含义 |
| --- | --- |
| `Msg` | 人类可读的错误文本。原生错误使用核心渲染；host 合成的错误使用绑定自身的措辞。 |
| `Class` | 错误类别 —— `DuplicateKey`、`Unrepresentable`、`UnrepresentableAt`、`InvalidUtf8`、`Message`。 |
| `Reason` | Host writer 的原因码包括 `NonFiniteFloat`、`ScalarRoot` 和 `EmptyKeyName`。`InvalidUtf8` 是 source-error class，reason 为空。 |
| `Line` | 1 起算的源行号;信封为 null 时是 `0`。 |
| `LineText` | 出错那一行的文本。 |
| `Span` | `*Span` —— UTF-8 源文本中的字节偏移。 |
| `Path` | 精确解码后的键段。 |
| `Body` | 出错的值,按写法原样。 |
| `Canonical` | 规范形式本应是什么。 |
| `SpecSection` | 被违反的条款,如 `§3.6/§5.2`。 |

