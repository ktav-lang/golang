>>>>> lang=en
### Added

- **`EmitCanonical(v any) (string, error)`** — renders a Go value as
  canonical Ktav (spec § 5.9): byte-deterministic output, no inline
  compounds, canonical integer / float normalisation. Two calls with
  identical inputs always produce identical bytes.
- **`TestConformanceCanonical`** — new conformance suite that verifies
  `EmitCanonical` output matches every `.canonical.ktav` oracle in the
  spec fixtures.

### Changed

- **Picked up `ktav 0.5.0`** — tracks the upstream Rust crate's
  spec 0.5 implementation: inferred numeric types, `##` comments,
  `emit_canonical` API. Spec submodule synced to tag `v0.5.0`. See the
  [`ktav` crate CHANGELOG](https://github.com/ktav-lang/rust/blob/main/CHANGELOG.md#050)
  for the full delta.
- **License changed to `MIT OR Apache-2.0`** — matching the rest of the
  `ktav-lang` ecosystem. `LICENSE` is renamed to `LICENSE-MIT`;
  `LICENSE-APACHE` is added. The SPDX expression in `Cargo.toml` is
  updated accordingly.
- **Conformance test suite updated to spec 0.5 fixtures** — path
  `spec/versions/0.5/tests`; `.canonical.ktav` files are excluded from
  the JSON-oracle test and handled by the new canonical suite.

>>>>> lang=ru
### Добавлено

- **`EmitCanonical(v any) (string, error)`** — рендерит Go-значение в
  канонический Ktav (spec § 5.9): байт-детерминированный вывод, без
  inline-соединений, каноническая нормализация integer / float. Два
  вызова с одинаковыми входами всегда дают идентичные байты.
- **`TestConformanceCanonical`** — новый conformance-набор, проверяющий,
  что вывод `EmitCanonical` совпадает с каждым `.canonical.ktav`-оракулом
  из spec-фикстур.

### Изменено

- **Подхватили `ktav 0.5.0`** — следуем реализации spec 0.5 из upstream
  Rust crate: выводимые числовые типы, комментарии `##`, API
  `emit_canonical`. Submodule spec синхронизирован с тегом `v0.5.0`.
  Полный diff — в
  [`ktav` crate CHANGELOG](https://github.com/ktav-lang/rust/blob/main/CHANGELOG.md#050).
- **Лицензия изменена на `MIT OR Apache-2.0`** — в соответствии с
  остальной экосистемой `ktav-lang`. `LICENSE` переименован в
  `LICENSE-MIT`; добавлен `LICENSE-APACHE`. SPDX-выражение в
  `Cargo.toml` обновлено соответствующим образом.
- **Conformance-набор обновлён до фикстур spec 0.5** — путь
  `spec/versions/0.5/tests`; файлы `.canonical.ktav` исключены из
  JSON-oracle теста и обрабатываются новым canonical-набором.

>>>>> lang=zh
### 新增

- **`EmitCanonical(v any) (string, error)`** — 将 Go 值渲染为规范 Ktav
  (spec § 5.9):字节确定性输出,无内联复合,规范的 integer / float
  正规化。两次相同输入的调用总是产生完全相同的字节。
- **`TestConformanceCanonical`** — 新的一致性测试套件,验证
  `EmitCanonical` 输出与 spec fixtures 中每个 `.canonical.ktav` oracle
  字节一致。

### 变更

- **升级到 `ktav 0.5.0`** — 跟踪上游 Rust crate 的 spec 0.5 实现:
  推断数字类型、`##` 注释、`emit_canonical` API。spec submodule 同步
  至标签 `v0.5.0`。完整变更见
  [`ktav` crate CHANGELOG](https://github.com/ktav-lang/rust/blob/main/CHANGELOG.md#050)。
- **许可证变更为 `MIT OR Apache-2.0`** — 与 `ktav-lang` 生态系统保持
  一致。`LICENSE` 改名为 `LICENSE-MIT`;新增 `LICENSE-APACHE`。
  `Cargo.toml` 中的 SPDX 表达式已相应更新。
- **一致性测试套件更新至 spec 0.5 fixtures** — 路径
  `spec/versions/0.5/tests`;`.canonical.ktav` 文件从 JSON oracle 测试
  中排除,由新的规范测试套件处理。

