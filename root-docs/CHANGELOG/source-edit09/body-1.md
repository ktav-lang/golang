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
- 接入规范 0.9 语料（412 个夹具；invalid 增至 162，含 72 个新的
  diagnostics09 EOF + 非法 UTF-8 用例），并新增以夹具驱动的回归运行器，
  在两个后端上验证 `expected_location`（line、原始字节 start、精确 end
  或 minimum_end）。

### 文档

- 扩展 README 示例：点分路径与字面键、混合数组、String/Integer 区别、多行展开、逐字节 no-op 及 BOM/CRLF/EOF 保留。明确 0.9 字节诊断、安全拒绝、ABI 2 和独立纯实现用法；这些新增功能不属于已发布的 0.8.1。

