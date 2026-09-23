>>>>> lang=en

## 0.3.1 — 2026-05-10

### Added

- **Top-level Array support** (spec 0.1.1, § 5.0.1) — `Loads` now
  returns a `[]any` when the document's first content line is an
  array-item line (bare scalar, typed marker, lone `{`/`[`, or
  multi-line opener). Previously top-level Arrays were rejected.
- **`Dumps` accepts top-level arrays** — pass any slice (`[]any`,
  `[]string`, etc.) and the rendered Ktav has bare item-per-line at
  the top, no surrounding `[...]`.
- **`DumpsForceStrings(v any) (string, error)`** — renders a Go value
  as Ktav with every scalar coerced to a String (typed integers,
  typed floats, booleans, null are flattened to their textual form
  via the raw-marker `::`). Compounds preserve their structure.
  The output round-trips back through `Loads` as the same set of
  String scalars — useful for environments or downstream consumers
  that don't understand the `:i` / `:f` typed markers.

### Changed

- **Picked up `ktav 0.3.1`** — tracks the upstream Rust crate's
  top-level Array support and `to_string_force_strings` API. Spec
  submodule synced to `7256816` (spec 0.1.1). See the
  [`ktav` crate CHANGELOG](https://github.com/ktav-lang/rust/blob/main/CHANGELOG.md#031--2026-05-10)
  for the full delta.
- **C ABI now exports five symbols** — added `ktav_dumps_force_strings`
  alongside the existing `ktav_loads` / `ktav_dumps` / `ktav_free` /
  `ktav_version`. The Go loader binds all five at first use; pinning
  `KTAV_LIB_PATH` to a pre-0.3.1 binary will fail with a missing
  symbol error.

>>>>> lang=ru

## 0.3.1 — 2026-05-10

### Добавлено

- **Поддержка top-level Array** (spec 0.1.1, § 5.0.1) — `Loads` теперь
  возвращает `[]any`, когда первая содержательная строка документа —
  строка элемента массива (голый скаляр, типизированный маркер, одинокий
  `{`/`[` или многострочный открыватель). Раньше top-level Array
  отвергались.
- **`Dumps` принимает top-level массивы** — передайте любой срез
  (`[]any`, `[]string` и т.д.), и отрендеренный Ktav будет содержать
  элементы построчно, без обрамляющих `[...]`.
- **`DumpsForceStrings(v any) (string, error)`** — рендерит Go-значение
  в Ktav, приводя каждый скаляр к String (типизированные целые,
  типизированные float, булевы и null сворачиваются к текстовой форме
  через raw-маркер `::`). Составные структуры сохраняют вид. Вывод
  считывается обратно через `Loads` как тот же набор String-скаляров —
  полезно для сред и потребителей, не понимающих типизированные маркеры
  `:i` / `:f`.

### Изменено

- **Подхватили `ktav 0.3.1`** — поддержка top-level Array и API
  `to_string_force_strings` из upstream Rust crate. Submodule spec
  синхронизирован с `7256816` (spec 0.1.1). Полный diff — в
  [`ktav` crate CHANGELOG](https://github.com/ktav-lang/rust/blob/main/CHANGELOG.md#031--2026-05-10).
- **C ABI теперь экспортирует пять символов** — добавлен
  `ktav_dumps_force_strings` рядом с существующими `ktav_loads` /
  `ktav_dumps` / `ktav_free` / `ktav_version`. Go-загрузчик связывает
  все пять при первом использовании; `KTAV_LIB_PATH`, закреплённый за
  pre-0.3.1 binary, упадёт с ошибкой отсутствия символа.

>>>>> lang=zh

## 0.3.1 — 2026-05-10

### 新增

- **顶层 Array 支持**(spec 0.1.1,§ 5.0.1)—— 当文档的第一个内容行是
  数组元素行(裸标量、类型标记、单独的 `{`/`[` 或多行开启符)时,
  `Loads` 现在返回 `[]any`。此前顶层 Array 会被拒绝。
- **`Dumps` 接受顶层数组** —— 传入任意切片(`[]any`、`[]string`
  等),渲染出的 Ktav 逐行罗列元素,不带外层 `[...]`。
- **`DumpsForceStrings(v any) (string, error)`** —— 将 Go 值渲染为
  Ktav,每个标量都强制为 String(带类型整数、带类型浮点、布尔、null
  都经由 raw 标记 `::` 摊平为文本形式)。复合结构保持原状。输出经
  `Loads` 读回仍是同一组 String 标量 —— 适用于不理解 `:i` / `:f`
  类型标记的环境或下游消费者。

### 变更

- **升级到 `ktav 0.3.1`** —— 跟踪上游 Rust crate 的顶层 Array 支持与
  `to_string_force_strings` API。spec submodule 同步至 `7256816`
  (spec 0.1.1)。完整变更见
  [`ktav` crate CHANGELOG](https://github.com/ktav-lang/rust/blob/main/CHANGELOG.md#031--2026-05-10)。
- **C ABI 新增第五个符号** `ktav_dumps_force_strings`,与既有符号
  `ktav_loads` / `ktav_dumps` / `ktav_free` / `ktav_version` 并列。
  Go 加载器在首次使用时绑定全部五个符号;将 `KTAV_LIB_PATH` 指向
  pre-0.3.1 二进制会因符号缺失而失败。

