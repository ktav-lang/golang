>>>>> lang=en
## Source editing

`EditValue` and `EditValueSegments` change one value **inside the source
text** and return the patched source. This is not a parse → mutate →
re-serialize round trip: only the target's byte range (plus, when
readability demands it, the inline wrappers containing it) is spliced, so
everything else survives byte-for-byte — comments, blank lines, key
spelling, dotted/inline/block forms, the document's CR / LF / CRLF bytes,
its BOM, and whether it ends with a newline.

```go
out, err := ktav.EditValue(src, "targets.storagehost.address", "10.0.0.9:22")
// out == src with only that scalar's bytes replaced

out, err := ktav.EditValueSegments(src, []string{"targets", "storagehost", "address"}, "10.0.0.9:22")
```

Path rules:

>>>>> lang=ru
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

>>>>> lang=zh
## 源码编辑

`EditValue` 与 `EditValueSegments` 直接**在源文本内部**修改一个值，并返回
修改后的源码。这不是 parse → mutate → serialize 的全量重写：只有目标值的
字节区间会被拼接替换（在可读性需要时，连同其所在的内联容器），其余部分
逐字节保留 —— 注释、空行、键的写法、dotted/inline/block 形式、文档的
CR / LF / CRLF 字节、BOM 以及末尾是否有换行。

```go
out, err := ktav.EditValue(src, "targets.storagehost.address", "10.0.0.9:22")
// out == src，仅该标量的字节被替换

out, err := ktav.EditValueSegments(src, []string{"targets", "storagehost", "address"}, "10.0.0.9:22")
```

路径规则：

