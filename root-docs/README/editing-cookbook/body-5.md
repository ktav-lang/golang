>>>>> lang=en
#### 6. No-op preserves noncanonical spelling

The Integer value is already 1. Equality returns the original bytes, including `+1` and the original separator spacing; readability thresholds are bypassed.

```go
source := "n  : +1\n"
updated, err := pure.EditValue(source, "n", int64(1))
if err != nil {
    panic(err)
}
if updated != source { panic("no-op changed source") }
```

Result:

```ktav
n  : +1
```

#### 7. BOM, CRLF and an unterminated final line

The returned string keeps the BOM and CRLF, and does not gain a final newline. Its escaped representation is shown to make the bytes visible.

>>>>> lang=ru
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

>>>>> lang=zh
#### 6. 无变化时保留非规范拼写

Integer 的值已经是 1。相等判断直接返回原始字节，包括 `+1` 及分隔符周围的空白，不应用可读性阈值。

```go
source := "n  : +1\n"
updated, err := pure.EditValue(source, "n", int64(1))
if err != nil {
    panic(err)
}
if updated != source { panic("no-op changed source") }
```

结果:

```ktav
n  : +1
```

#### 7. BOM、CRLF 及无换行符的末行

返回值保留 BOM 和 CRLF，且不会增加末尾换行。以下转义表示使这些字节清晰可见。

