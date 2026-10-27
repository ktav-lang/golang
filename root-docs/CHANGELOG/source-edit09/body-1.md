>>>>> lang=en
## Unreleased — 0.9 preparation

- Source editing (0.9): `EditValue(src, path, replacement)` and
  `EditValueSegments(src, segments, replacement)` splice exactly the
  target's byte range inside the source text — comments, key spelling,
  dotted/inline/block forms, BOM and CR/LF/CRLF/EOF bytes outside the
  edit survive; no-op replacements are byte-identical; inline ancestors
  expand minimally when readability requires. New `EditInvalidPath` /
  `EditNotFound` / `EditUnsupported` error classes carry only `message`
  and `body` (the UTF-8 byte offset or the reason). Native and pure
  backends agree byte-for-byte; `pure` gains the same two entry points.
- The C ABI grows `ktav_edit_value` / `ktav_edit_value_segments`; the
  loader now gates on `ktav_abi_version()` == 2 (user-approved bump
  from 1) and refuses older libraries with a clear version error —
  never a silent pure fallback. Package versions are unchanged.
- Source editing also adds `InsertValue` / `InsertValueSegments`,
  `RemoveValue` / `RemoveValueSegments`, and `RenameKey` /
  `RenameKeySegments` to both the root Go package and standalone `pure`.
  Insert appends a new Object pair; remove deletes array elements with
  their delimiter and never prunes empty ancestors (removing the last
  `b` from `a.b: 1\n` yields `a: {}\n`). Rename changes an Object key
  in place. The C ABI 2 library must export `ktav_insert_value`,
  `ktav_insert_value_segments`, `ktav_remove_value`,
  `ktav_remove_value_segments`, `ktav_rename_key`, and
  `ktav_rename_key_segments`; an older ABI 2 library lacking any of
  them is incompatible. Pure-Go remains independent of native.
- Exact API signatures (in both packages): `InsertValue(src, textualPath string, replacement any) (string, error)`, `InsertValueSegments(src string, segments []string, replacement any) (string, error)`, `RemoveValue(src, textualPath string) (string, error)`, `RemoveValueSegments(src string, segments []string) (string, error)`, `RenameKey(src, textualPath, newKey string) (string, error)`, and `RenameKeySegments(src string, segments []string, newKey string) (string, error)`. Textual paths use `.` with `\.`/`\\` escaping; segment arrays and `newKey` are literal keys. Invalid textual paths return `EditInvalidPath`; missing targets/parents, bad array indexes, or scalar descent return `EditNotFound`; collisions return `EditAlreadyExists`. Array insertion/rename and non-lossless structural splices return `EditUnsupported`. Array indexes are zero-based ASCII digits (leading zeroes allowed); numeric Object segments are strings. Insert takes the normal `Dumps` Go type mapping. All 18 exported ABI symbols require ABI version 2; the six mutation symbols are listed above. Native and pure are independent backends.
- Spec 0.9 corpus (412 fixtures; invalid grew to 162 with 72 new
  diagnostics09 EOF + malformed-UTF-8 cases) is wired in with a
  fixture-driven regression runner validating `expected_location`
  (line, raw-byte start, exact end or minimum_end) on both backends,
  native and pure.

### Documentation

- Expand README examples for dotted and literal-key paths, mixed arrays, String/Integer identity, multiline unfolding, byte-identical no-op and BOM/CRLF/EOF preservation. Clarify 0.9 byte diagnostics, safe refusal, ABI 2 and standalone pure usage; these additions are not part of published 0.8.1.

>>>>> lang=ru
## Ещё не выпущено — подготовка 0.9

- Правка исходника (0.9): `EditValue(src, path, replacement)` и
  `EditValueSegments(src, segments, replacement)` сплайсят ровно
  байтовый диапазон цели внутри исходного текста — комментарии,
  написание ключей, dotted/inline/block-формы, BOM и байты
  CR/LF/CRLF/EOF вне правки сохраняются; no-op-замены байт-в-байт
  идентичны исходнику; inline-предки раскрываются минимально, когда
  требует читаемость. Новые классы ошибок `EditInvalidPath` /
  `EditNotFound` / `EditUnsupported` несут только `message` и `body`
  (UTF-8 byte-офсет или причина). Native и pure бэкенды совпадают
  байт-в-байт; в `pure` добавлены те же две точки входа.
- C ABI пополнили `ktav_edit_value` / `ktav_edit_value_segments`;
  загрузчик теперь проверяет `ktav_abi_version()` == 2 (одобренный
  пользователем подъём с 1) и отклоняет старые библиотеки понятной
  ошибкой версии — никогда не подменяя их тихим pure-fallback.
  Версии пакетов не менялись.
- В root Go-пакете и самостоятельном `pure` добавлены `InsertValue` /
  `InsertValueSegments`, `RemoveValue` / `RemoveValueSegments` и
  `RenameKey` / `RenameKeySegments`. Insert добавляет пару Object;
  remove удаляет элемент массива вместе с разделителем и не удаляет
  пустых предков (удаление последнего `b` из `a.b: 1\n` даёт `a: {}\n`).
  Rename меняет ключ Object на месте. C ABI 2 должна экспортировать
  `ktav_insert_value`, `ktav_insert_value_segments`, `ktav_remove_value`,
  `ktav_remove_value_segments`, `ktav_rename_key` и
  `ktav_rename_key_segments`; старая ABI 2 без любого из них несовместима.
  Pure-Go не зависит от native.
- Точные сигнатуры (в обоих пакетах): `InsertValue(src, textualPath string, replacement any) (string, error)`, `InsertValueSegments(src string, segments []string, replacement any) (string, error)`, `RemoveValue(src, textualPath string) (string, error)`, `RemoveValueSegments(src string, segments []string) (string, error)`, `RenameKey(src, textualPath, newKey string) (string, error)` и `RenameKeySegments(src string, segments []string, newKey string) (string, error)`. В текстовом пути `.` — разделитель, `\.`/`\\` — экранирование; сегменты и `newKey` являются literal-ключами. Неверный текстовый путь даёт `EditInvalidPath`; отсутствующая цель/родитель, неверный индекс массива или спуск через скаляр — `EditNotFound`; конфликт ключей — `EditAlreadyExists`. Вставка в массив, переименование индекса массива и структурная правка без lossless-сплайса дают `EditUnsupported`. Индексы массива — ASCII-цифры с нуля (ведущие нули допустимы); числовой сегмент Object — строковый ключ. Insert использует обычное Go-сопоставление типов `Dumps`. Все 18 экспортируемых ABI-символов требуют версии ABI 2; шесть mutation-символов перечислены выше. Native и pure — независимые бэкенды.
- Подключён корпус спецификации 0.9 (412 фикстур; invalid вырос до 162
  за счёт 72 новых diagnostics09 — EOF + malformed UTF-8) с
  fixture-driven regression-runner, проверяющим `expected_location`
  (line, raw-byte start, точный end или minimum_end) на обоих бэкендах.

### Документация

- Дополнены примеры README: dotted-пути и буквальные ключи, смешанные массивы, различие String/Integer, раскрытие многострочной замены, побайтовый no-op и сохранение BOM/CRLF/EOF. Уточнены байтовая диагностика 0.9, безопасные отказы, ABI 2 и самостоятельный pure-бэкенд; дополнения не относятся к опубликованному 0.8.1.

>>>>> lang=zh
## 尚未发布 — 0.9 准备版本

- 源码编辑（0.9）：`EditValue(src, path, replacement)` 与
  `EditValueSegments(src, segments, replacement)` 在源文本内只拼接目标值
  的字节区间 —— 编辑区外的注释、键写法、dotted/inline/block 形式、BOM
  和 CR/LF/CRLF/EOF 字节全部保留；no-op 替换逐字节等同源码；内联祖先在
  可读性需要时最小化展开。新增 `EditInvalidPath` / `EditNotFound` /
  `EditUnsupported` 错误类，仅携带 `message` 与 `body`（UTF-8 字节偏移或
  原因）。Native 与纯 Go 后端逐字节一致；`pure` 同样获得这两个入口。
- C ABI 新增 `ktav_edit_value` / `ktav_edit_value_segments`；加载器现在
  以 `ktav_abi_version()` == 2 为门禁（经用户批准从 1 升级），以清晰的
  版本错误拒绝旧库 —— 绝不静默回退到 pure。包版本未变。
- 根 Go 包和独立 `pure` 新增 `InsertValue` / `InsertValueSegments`、
  `RemoveValue` / `RemoveValueSegments` 与 `RenameKey` / `RenameKeySegments`。
  Insert 追加 Object 键值对；remove 删除数组元素时连同分隔符一起删除，且
  不清除空祖先（从 `a.b: 1\n` 删除最后一个 `b` 会得到 `a: {}\n`）。
  Rename 原位更改 Object 键。C ABI 2 库必须导出 `ktav_insert_value`、
  `ktav_insert_value_segments`、`ktav_remove_value`、
  `ktav_remove_value_segments`、`ktav_rename_key` 和
  `ktav_rename_key_segments`；缺少其中任何符号的旧 ABI 2 库均不兼容。
  纯 Go 实现不依赖 native。
- 两个包中的精确签名为：`InsertValue(src, textualPath string, replacement any) (string, error)`、`InsertValueSegments(src string, segments []string, replacement any) (string, error)`、`RemoveValue(src, textualPath string) (string, error)`、`RemoveValueSegments(src string, segments []string) (string, error)`、`RenameKey(src, textualPath, newKey string) (string, error)` 和 `RenameKeySegments(src string, segments []string, newKey string) (string, error)`。文本路径以 `.` 分隔，使用 `\.`/`\\` 转义；segments 与 `newKey` 均为字面键。无效文本路径返回 `EditInvalidPath`；目标/父级不存在、数组索引无效或穿过标量返回 `EditNotFound`；键冲突返回 `EditAlreadyExists`。向数组插入、重命名数组索引以及无法无损拼接的结构修改返回 `EditUnsupported`。数组索引为从零开始的 ASCII 数字（允许前导零）；Object 数字段是字符串键。Insert 遵循 `Dumps` 的 Go 类型映射。全部 18 个导出 ABI 符号要求 ABI 2；上文列出六个 mutation 符号。Native 与 pure 是独立后端。
- 接入规范 0.9 语料（412 个夹具；invalid 增至 162，含 72 个新的
  diagnostics09 EOF + 非法 UTF-8 用例），并新增以夹具驱动的回归运行器，
  在两个后端上验证 `expected_location`（line、原始字节 start、精确 end
  或 minimum_end）。

### 文档

- 扩展 README 示例：点分路径与字面键、混合数组、String/Integer 区别、多行展开、逐字节 no-op 及 BOM/CRLF/EOF 保留。明确 0.9 字节诊断、安全拒绝、ABI 2 和独立纯实现用法；这些新增功能不属于已发布的 0.8.1。

