>>>>> lang=en
- `EditValue` takes a textual path (`key1.key2.0.key4`). Inside a
  segment `\.` is a literal dot and `\\` a literal backslash; any other
  escape, a trailing lone backslash, or an empty segment is an
  `*ktav.Error` with `Class == "EditInvalidPath"` whose `Body` is the
  UTF-8 byte offset of the problem inside the path argument.
- `EditValueSegments` takes pre-decoded literal segments — no additional
  unescaping. A numeric segment is a string key inside an Object and a
  zero-based ASCII-digit index inside an Array (leading zeros allowed,
  signs are not indexes). An empty segment list is `"EditNotFound"`.
- Only **replacement of an existing value** is supported: no create,
  remove, or rename. A missing key, an out-of-bounds index, or descent
  through a scalar is `"EditNotFound"`.

Replacement values use the ordinary Go type mapping (`Dumps` accepts
them, scalars included); a host integer never degrades to a float.

Guarantees:

>>>>> lang=ru
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

>>>>> lang=zh
- `EditValue` 接受文本路径（`key1.key2.0.key4`）。段内 `\.` 表示字面量
  点号，`\\` 表示字面量反斜杠；任何其他转义、结尾的孤立 `\` 或空段都会
  得到 `Class == "EditInvalidPath"` 的 `*ktav.Error`，其 `Body` 是问题在
  路径参数中的 UTF-8 字节偏移。
- `EditValueSegments` 接受已解码的字面量段 —— 不做额外反转义。数字段在
  Object 中是字符串键，在 Array 中是从零开始的 ASCII 数字索引（允许前导
  零，不允许符号）。空段列表为 `"EditNotFound"`。
- 仅支持**替换已有值**：没有 create、remove 或 rename。缺失键、越界索引
  或穿过标量的下降都是 `"EditNotFound"`。

替换值使用常规的 Go 类型映射（`Dumps` 接受的一切，包括标量）；宿主
integer 绝不会退化为 float。

保证：

