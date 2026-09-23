>>>>> lang=en
## 0.1.1 — 2026-04-26

### Changed

- **Picked up `ktav 0.1.4`** — the upstream Rust crate's untyped
  `parse() → Value` path (which `cabi` uses) is now ~30% faster on
  small documents and ~13% faster on large ones, just from a one-
  line `Frame::Object` capacity tweak (4 → 8). Every `ktav.Loads`
  call benefits transparently.

>>>>> lang=ru
## 0.1.1 — 2026-04-26

### Изменено

- **Подхватили `ktav 0.1.4`** — untyped путь `parse() → Value` в
  upstream Rust crate (тот, что использует `cabi`) теперь ~30% быстрее
  на маленьких документах и ~13% на больших — только за счёт
  однострочной правки initial capacity для `Frame::Object` (4 → 8).
  Каждый вызов `ktav.Loads` выигрывает от этого прозрачно.

>>>>> lang=zh
## 0.1.1 — 2026-04-26

### 变更

- **升级到 `ktav 0.1.4`** —— 上游 Rust crate 中 `cabi` 使用的 untyped
  `parse() → Value` 路径,小文档加速约 30%、大文档加速约 13%,只是
  `Frame::Object` 的初始容量微调(4 → 8)。每次 `ktav.Loads` 都会
  透明地受益。

