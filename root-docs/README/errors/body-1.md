>>>>> lang=en
## Structured errors

Every failure carries the same structured envelope the other Ktav
bindings carry, so a tool can act on the fields instead of parsing a
message. `Loads`, `Dumps` and the rest return an `*Error`:

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
| `Msg` | The core's own rendering of the error. Never empty; this is what `Error()` returns. |
| `Class` | Structured error class — `DuplicateKey`, `Unrepresentable`, `Message`. |
| `Reason` | Writer-time reason code (spec § 5.9.0) such as `NonFiniteFloat`; `""` when the envelope carries null. |
| `Line` | 1-based source line; `0` when the envelope carries null. |
| `LineText` | Text of the offending line. |
| `Span` | `*Span` — byte offsets into the UTF-8 source. |
| `Path` | Exact decoded key segments. |
| `Body` | The offending value as written. |
| `Canonical` | What the canonical form would have been. |
| `SpecSection` | The clause violated, e.g. `§3.6/§5.2`. |

>>>>> lang=ru
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

>>>>> lang=zh
## 结构化错误

每一次失败都携带与其他 Ktav 绑定相同的结构化信封,因此工具可以针对
字段做处理,而不必解析消息文本。`Loads`、`Dumps` 等返回 `*Error`:

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
| `Msg` | 来自核心自身的错误渲染文本。永不为空;`Error()` 返回的就是它。 |
| `Class` | 结构化错误类别 —— `DuplicateKey`、`Unrepresentable`、`Message`。 |
| `Reason` | writer 侧的原因码(spec § 5.9.0),如 `NonFiniteFloat`;信封为 null 时是 `""`。 |
| `Line` | 1 起算的源行号;信封为 null 时是 `0`。 |
| `LineText` | 出错那一行的文本。 |
| `Span` | `*Span` —— UTF-8 源文本中的字节偏移。 |
| `Path` | 精确解码后的键段。 |
| `Body` | 出错的值,按写法原样。 |
| `Canonical` | 规范形式本应是什么。 |
| `SpecSection` | 被违反的条款,如 `§3.6/§5.2`。 |

