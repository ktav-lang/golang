>>>>> lang=en
## API

| Function | Purpose |
| --- | --- |
| `Loads(s string) (any, error)` | Parse a Ktav document into native Go values. Top-level may be an Object (`map[string]any`) or an Array (`[]any`) per spec § 5.0.1. |
| `LoadsStrict(s string) (any, error)` | Parse with strict numeric spelling checks, using the same Go type mapping as `Loads`. |
| `LoadsInto(s string, target any) error` | Parse into an arbitrary `target` (struct, map, …) via `encoding/json`. |
| `Dumps(v any) (string, error)` | Render a Go value as Ktav text. Top-level must encode to an object or array. |
| `DumpsForceStrings(v any) (string, error)` | Like `Dumps`, but coerces every leaf scalar (integer, float, bool, null) to a String via the raw `::` marker. Compounds preserve their structure. |
| `EmitCanonical(v any) (string, error)` | Render a Go value as canonical Ktav (spec § 5.9). Go maps are encoded through `encoding/json`, which sorts string keys lexicographically. |
| `CanonicalFromSource(src string) (string, error)` | Parse Ktav and immediately emit canonical form, preserving source key order. |
| `FormatSource(src string) (string, error)` | Format Ktav source into its normalised spelling, **keeping every comment**. See below. |
| `EditValue(src, path string, replacement any) (string, error)` | Replace the value addressed by the textual dotted path **in the source text** and return the patched source. See [Source editing](#source-editing). |
| `EditValueSegments(src string, segments []string, replacement any) (string, error)` | Same, with pre-decoded literal segments (no extra unescaping). |
| `InsertValue(src, path string, value any) (string, error)` | Insert a value at the textual dotted path in the source text. |
| `InsertValueSegments(src string, segments []string, value any) (string, error)` | Same, with pre-decoded literal segments (no extra unescaping). |
| `RemoveValue(src, path string) (string, error)` | Remove the value addressed by the textual dotted path from the source text. |
| `RemoveValueSegments(src string, segments []string) (string, error)` | Same, with pre-decoded literal segments (no extra unescaping). |
| `RenameKey(src, path, newKey string) (string, error)` | Rename the key addressed by the textual dotted path in the source text. |
| `RenameKeySegments(src string, segments []string, newKey string) (string, error)` | Same, with pre-decoded literal segments (no extra unescaping). |

>>>>> lang=ru
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
| `InsertValue(src, path string, value any) (string, error)` | Вставить значение по текстовому dotted-пути в исходный текст. |
| `InsertValueSegments(src string, segments []string, value any) (string, error)` | То же с уже декодированными literal-сегментами (без повторного unescape). |
| `RemoveValue(src, path string) (string, error)` | Удалить значение по текстовому dotted-пути из исходного текста. |
| `RemoveValueSegments(src string, segments []string) (string, error)` | То же с уже декодированными literal-сегментами (без повторного unescape). |
| `RenameKey(src, path, newKey string) (string, error)` | Переименовать ключ по текстовому dotted-пути в исходном тексте. |
| `RenameKeySegments(src string, segments []string, newKey string) (string, error)` | То же с уже декодированными literal-сегментами (без повторного unescape). |

>>>>> lang=zh
## API

| 函数 | 用途 |
| --- | --- |
| `Loads(s string) (any, error)` | 将 Ktav 文档解析为原生 Go 值。顶层可以是对象(`map[string]any`)或数组(`[]any`),按 spec § 5.0.1。 |
| `LoadsStrict(s string) (any, error)` | 使用严格数字词法检查解析文档，类型映射与 `Loads` 相同。 |
| `LoadsInto(s string, target any) error` | 解析到任意 `target`(struct、map 等),通过 `encoding/json`。 |
| `Dumps(v any) (string, error)` | 将 Go 值渲染为 Ktav 文本。顶层必须为对象或数组。 |
| `DumpsForceStrings(v any) (string, error)` | 同 `Dumps`,但所有叶标量(integer、float、bool、null)通过 `::` 强制为 String。复合值保留其结构。 |
| `EmitCanonical(v any) (string, error)` | 把 Go 值渲染为规范 Ktav(spec § 5.9)。`encoding/json` 会按字典序排列 Go map 的字符串键。 |
| `CanonicalFromSource(src string) (string, error)` | 解析 Ktav 并立即输出规范形式，保留源文件的键顺序。 |
| `FormatSource(src string) (string, error)` | 把 Ktav 源文本格式化为规范化写法，**保留全部注释**。见下文。 |
| `EditValue(src, path string, replacement any) (string, error)` | 按文本点分路径**在源文本中**替换值，返回修改后的源码。见[源码编辑](#源码编辑)。 |
| `EditValueSegments(src string, segments []string, replacement any) (string, error)` | 同上，但接受已解码的字面量段（不做额外反转义）。 |
| `InsertValue(src, path string, value any) (string, error)` | 按文本点分路径在源文本中插入值。 |
| `InsertValueSegments(src string, segments []string, value any) (string, error)` | 同上，但接受已解码的字面量段（不做额外反转义）。 |
| `RemoveValue(src, path string) (string, error)` | 按文本点分路径从源文本中删除值。 |
| `RemoveValueSegments(src string, segments []string) (string, error)` | 同上，但接受已解码的字面量段（不做额外反转义）。 |
| `RenameKey(src, path, newKey string) (string, error)` | 按文本点分路径在源文本中重命名键。 |
| `RenameKeySegments(src string, segments []string, newKey string) (string, error)` | 同上，但接受已解码的字面量段（不做额外反转义）。 |

