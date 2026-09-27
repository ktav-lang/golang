>>>>> lang=en
## 0.8.0 — 2026-09-27

This release updates the Go binding from its `ktav 0.7.1` / spec 0.7.0
baseline to `ktav 0.8.0` / spec 0.8.0.

### Changed

- The module and native-library fallback target `v0.8.0`.
- Raised the native dependency floor from `ktav 0.7.1` to `0.8.0`,
  pinned spec 0.8.0, and migrated the C ABI to
  `ktav::declare_cabi!()` without changing the exported symbols.
- `FormatSource` formats source text while preserving whole-line
  comments and source key order while normalizing blank lines.
- Native errors use the structured Ktav envelope. Host-side writer
  validation returns structured `*ktav.Error` values with reason codes
  `NonFiniteFloat`, `ScalarRoot`, and `EmptyKeyName`. Invalid UTF-8 source
  is class `InvalidUtf8` with an empty reason, not a writer reason.
  Library loading/download failures and Go JSON conversion errors remain
  ordinary Go errors and need not be `*ktav.Error`.
- The conformance runner targets `spec/versions/0.8/tests` and covers
  the corpus categories present in that version, including
  `strict-lossy`, `unrepresentable`, and
  `parseable-unrepresentable`. The runner rejects unknown categories and
  validates fixture schemas, expected counts, and required companion
  files. Running conformance also requires the native library. Writer
  rejection and strict-lossy tests assert exact reason/error codes from
  each fixture oracle.

>>>>> lang=ru
## 0.8.0 — 2026-09-27

В этом релизе Go-биндинг обновлён с базовых `ktav 0.7.1` / spec 0.7.0
до `ktav 0.8.0` / spec 0.8.0.

### Изменено

- Версия модуля и цель резервной загрузки нативной библиотеки —
  `v0.8.0`.
- Нижняя граница нативной зависимости повышена с `ktav 0.7.1` до `0.8.0`,
  spec закреплён на 0.8.0, а C ABI переведён на
  `ktav::declare_cabi!()` без изменения экспортируемых символов.
- `FormatSource` форматирует исходный текст, сохраняя комментарии,
  занимающие целую строку, и исходный порядок ключей, нормализуя пустые
  строки.
- Ошибки используют структурированный конверт Ktav. Проверка записи на
  стороне Go возвращает `*ktav.Error` с reason-кодами `NonFiniteFloat`,
  `ScalarRoot` и `EmptyKeyName`. Ошибка UTF-8 в исходнике имеет class
  `InvalidUtf8` и пустой reason, это не writer reason. Ошибки
  загрузки/скачивания библиотеки и преобразования Go JSON остаются
  обычными ошибками Go и не обязаны быть `*ktav.Error`.
- Conformance-раннер нацелен на `spec/versions/0.8/tests` и запускает
  категории корпуса этой версии, включая `strict-lossy`,
  `unrepresentable` и `parseable-unrepresentable`. Раннер отклоняет
  неизвестные категории и проверяет схемы fixtures, ожидаемое количество
  и обязательные companion-файлы. Для conformance-запуска также нужна
  нативная библиотека. Проверки отказов writer'а и strict-lossy сверяют
  точные коды причин/ошибок с oracle каждого fixture.

>>>>> lang=zh
## 0.8.0 — 2026-09-27

此版本将 Go 绑定从 `ktav 0.7.1` / spec 0.7.0 基线更新至
`ktav 0.8.0` / spec 0.8.0。

### 变更

- 模块版本及原生库回退下载目标为 `v0.8.0`。
- 原生依赖下限从 `ktav 0.7.1` 提升至 `0.8.0`，spec 固定到 0.8.0；
  C ABI 迁移到 `ktav::declare_cabi!()`，导出符号保持不变。
- `FormatSource` 格式化源文本、保留独占整行的注释及源键顺序，并
  规范化空行。
- 错误使用结构化 Ktav 信封。Go 侧写入校验返回带有
  `NonFiniteFloat`、`ScalarRoot` 和 `EmptyKeyName` 原因码的结构化
  `*ktav.Error`。无效 UTF-8 源的 class 是 `InvalidUtf8`，reason 为空，
  它不是 writer reason。库加载/下载错误和 Go JSON 转换错误仍是普通
  Go error，不保证为 `*ktav.Error`。
- 一致性运行器针对 `spec/versions/0.8/tests`，覆盖该版本语料中的
  类别，包括 `strict-lossy`、`unrepresentable` 和
  `parseable-unrepresentable`。运行器会拒绝未知类别，并验证 fixture
  schema、预期数量和必需的 companion 文件。运行 conformance 还需要
  原生库。writer 拒绝及 strict-lossy 测试逐例核对 oracle 中的确切
  原因/错误码。

