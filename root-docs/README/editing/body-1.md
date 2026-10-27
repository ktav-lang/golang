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

- `EditValue` takes a textual path (`key1.key2.0.key4`). Inside a
  segment `\.` is a literal dot and `\\` a literal backslash; any other
  escape, a trailing lone backslash, or an empty segment is an
  `*ktav.Error` with `Class == "EditInvalidPath"` whose `Body` is the
  UTF-8 byte offset of the problem inside the path argument.
- `EditValueSegments` takes pre-decoded literal segments — no additional
  unescaping. A numeric segment is a string key inside an Object and a
  zero-based ASCII-digit index inside an Array (leading zeros allowed,
  signs are not indexes). An empty segment list is `"EditNotFound"`.
- `EditValue` replaces an existing value in source text. For insertion, removal, or key renaming, see [Insert, remove, and rename](#insert-remove-and-rename). A missing key, an out-of-bounds index, or descent through a scalar is `"EditNotFound"`.

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

- `EditValue` принимает текстовый путь (`key1.key2.0.key4`). Внутри
  сегмента `\.` — literal точка, `\\` — literal backslash; любой другой
  escape, хвостовой `\` или пустой сегмент — `*ktav.Error` с
  `Class == "EditInvalidPath"`, где `Body` — UTF-8 byte-офсет проблемы
  внутри аргумента пути.
- `EditValueSegments` принимает уже декодированные literal-сегменты —
  без повторного unescape. Числовой сегмент — строковый ключ в Object и
  zero-based индекс из ASCII-цифр в Array (ведущие нули допустимы, знаки
  — нет). Пустой список сегментов — `"EditNotFound"`.
- `EditValue` заменяет существующее значение в исходном тексте. Для вставки, удаления и переименования ключа см. [Вставка, удаление и переименование](#вставка-удаление-и-переименование). Отсутствующий ключ, индекс вне диапазона или спуск через скаляр — `"EditNotFound"`.

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

- `EditValue` 接受文本路径（`key1.key2.0.key4`）。段内 `\.` 表示字面量
  点号，`\\` 表示字面量反斜杠；任何其他转义、结尾的孤立 `\` 或空段都会
  得到 `Class == "EditInvalidPath"` 的 `*ktav.Error`，其 `Body` 是问题在
  路径参数中的 UTF-8 字节偏移。
- `EditValueSegments` 接受已解码的字面量段 —— 不做额外反转义。数字段在
  Object 中是字符串键，在 Array 中是从零开始的 ASCII 数字索引（允许前导
  零，不允许符号）。空段列表为 `"EditNotFound"`。
- `EditValue` 在源码文本中替换已有值。插入、删除或重命名键请参见[插入、删除与重命名](#插入删除与重命名)。缺失键、越界索引或穿过标量都会得到 `"EditNotFound"`。

