>>>>> lang=en
## Insert, remove, and rename

The root `ktav` package and standalone `pure` package expose the same APIs:

```go
func InsertValue(src, textualPath string, replacement any) (string, error)
func InsertValueSegments(src string, segments []string, replacement any) (string, error)
func RemoveValue(src, textualPath string) (string, error)
func RemoveValueSegments(src string, segments []string) (string, error)
func RenameKey(src, textualPath, newKey string) (string, error)
func RenameKeySegments(src string, segments []string, newKey string) (string, error)
```

Textual paths use `.` separators; `\.` and `\\` encode a literal dot and backslash. Segment APIs take literal, pre-decoded segments, so `[]string{"a.b"}` addresses the single key `a.b`. For example:

```go
out, err := ktav.InsertValue(src, `list\.item.new`, "value")
out, err := ktav.RemoveValue(src, `list\.item.0`)
out, err := ktav.RenameKeySegments(src, []string{"list.item", "old"}, "new.name")
```

Insert's final segment is a new Object key, and the parent must exist and be an Object; it appends the pair. Remove accepts a leaf or subtree, and removes an array element together with its delimiter. Neither insertion nor removal prunes empty ancestors: for example, removing the last `b` from dotted source `a.b: 1\n` yields `a: {}\n` (the empty parent is materialized), rather than pruning `a`. Rename changes one Object key in place, preserving the pair's value and position; `newKey` is one literal key, not a path. Renaming to the same key is a byte-identical no-op.

Invalid textual paths return `*ktav.Error` with `Class == "EditInvalidPath"`. Missing targets/parents, invalid array indexes, or descent through scalars produce `EditNotFound`; inserting an existing key or renaming to a sibling key produces `EditAlreadyExists`. Array insertion and array-key rename, and structural cases that cannot be represented by a lossless single-range splice, produce `EditUnsupported`. Array indexes are zero-based ASCII digits (leading zeros allowed; signs are not indexes); numeric segments remain ordinary string keys in Objects. Insert values use the regular `Dumps` Go type mapping.

The C-FFI ABI 2 library must export all six symbols: `ktav_insert_value`, `ktav_insert_value_segments`, `ktav_remove_value`, `ktav_remove_value_segments`, `ktav_rename_key`, and `ktav_rename_key_segments`. An older ABI 2 library without these symbols is incompatible with this Go binding and cannot support these APIs. The pure-Go APIs are standalone and do not load or depend on the native library; native and pure implementations are independent backends.

>>>>> lang=ru
## Вставка, удаление и переименование

Пакеты верхнего уровня `ktav` и самостоятельный `pure` предоставляют одинаковые API:

```go
func InsertValue(src, textualPath string, replacement any) (string, error)
func InsertValueSegments(src string, segments []string, replacement any) (string, error)
func RemoveValue(src, textualPath string) (string, error)
func RemoveValueSegments(src string, segments []string) (string, error)
func RenameKey(src, textualPath, newKey string) (string, error)
func RenameKeySegments(src string, segments []string, newKey string) (string, error)
```

Текстовый путь использует разделитель `.`; `\.` и `\\` задают literal-точку и обратную косую черту. API с сегментами принимают literal, заранее декодированные сегменты: `[]string{"a.b"}` адресует один ключ `a.b`. Например:

```go
out, err := ktav.InsertValue(src, `list\.item.new`, "value")
out, err := ktav.RemoveValue(src, `list\.item.0`)
out, err := ktav.RenameKeySegments(src, []string{"list.item", "old"}, "new.name")
```

Последний сегмент Insert — новый ключ Object; родитель должен существовать и быть Object, пара добавляется в конец. Remove удаляет лист или поддерево; элемент массива удаляется вместе с разделителем. Вставка и удаление не удаляют пустых предков: например, удаление последнего `b` из dotted-источника `a.b: 1\n` даёт `a: {}\n` (пустой родитель материализуется), а не удаляет `a`. Rename переименовывает один ключ Object на месте, сохраняя значение и позицию пары; `newKey` — один literal-ключ, не путь. Переименование в тот же ключ — побайтный no-op.

Некорректный текстовый путь даёт `*ktav.Error` с `Class == "EditInvalidPath"`. Отсутствующие цель/родитель, неверный индекс массива или спуск через скаляр дают `EditNotFound`; существующий ключ при вставке или совпадающий ключ-сосед при переименовании — `EditAlreadyExists`. Вставка в массив, переименование ключа массива и структурные случаи без lossless-патча одним диапазоном дают `EditUnsupported`. Индексы массива — zero-based ASCII-цифры (ведущие нули разрешены, знаки не являются индексами); числовые сегменты в Object остаются обычными строковыми ключами. Значения Insert используют стандартное Go-сопоставление типов `Dumps`.

Библиотека C-FFI ABI 2 должна экспортировать все шесть символов: `ktav_insert_value`, `ktav_insert_value_segments`, `ktav_remove_value`, `ktav_remove_value_segments`, `ktav_rename_key` и `ktav_rename_key_segments`. Старая библиотека ABI 2 без этих символов несовместима с этой Go-привязкой и не поддерживает эти API. API pure-Go самостоятельны и не загружают native-библиотеку; native и pure — независимые реализации.

>>>>> lang=zh
## 插入、删除与重命名

根 `ktav` 包和独立的 `pure` 包提供相同 API：

```go
func InsertValue(src, textualPath string, replacement any) (string, error)
func InsertValueSegments(src string, segments []string, replacement any) (string, error)
func RemoveValue(src, textualPath string) (string, error)
func RemoveValueSegments(src string, segments []string) (string, error)
func RenameKey(src, textualPath, newKey string) (string, error)
func RenameKeySegments(src string, segments []string, newKey string) (string, error)
```

文本路径使用 `.` 分隔；`\.` 和 `\\` 分别表示字面点号与反斜杠。Segments API 接受字面量、预解码的段，因此 `[]string{"a.b"}` 指向单个键 `a.b`。例如：

```go
out, err := ktav.InsertValue(src, `list\.item.new`, "value")
out, err := ktav.RemoveValue(src, `list\.item.0`)
out, err := ktav.RenameKeySegments(src, []string{"list.item", "old"}, "new.name")
```

Insert 的最后一段是新 Object 键；父级必须存在且为 Object，新键值对追加在末尾。Remove 可删除叶值或子树；删除数组元素时也会删除其分隔符。插入和删除都不会清除空祖先：例如，从 dotted 源 `a.b: 1\n` 删除最后一个 `b` 后得到 `a: {}\n`（空父级会被显式物化），而不是删除 `a`。Rename 原位修改一个 Object 键，保留键值对的值与位置；`newKey` 是单个字面键，不是路径。重命名为原键是逐字节不变的 no-op。

文本路径无效时返回 `Class == "EditInvalidPath"` 的 `*ktav.Error`。目标/父级不存在、数组索引无效或穿过标量会得到 `EditNotFound`；插入已有键或重命名为同级现有键会得到 `EditAlreadyExists`。向数组插入、重命名数组索引，以及无法用单一区间无损表示的结构情形会得到 `EditUnsupported`。数组索引为从零开始的 ASCII 数字（允许前导零，不允许符号）；Object 中的数字段仍是普通字符串键。Insert 值遵循 `Dumps` 的常规 Go 类型映射。

C-FFI ABI 2 库必须导出全部六个符号：`ktav_insert_value`、`ktav_insert_value_segments`、`ktav_remove_value`、`ktav_remove_value_segments`、`ktav_rename_key` 和 `ktav_rename_key_segments`。缺少这些符号的旧 ABI 2 库与此 Go 绑定不兼容，无法支持这些 API。纯 Go API 独立运行，不加载或依赖原生库；native 与 pure 是相互独立的实现后端。

