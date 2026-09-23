>>>>> lang=en
### 3. Public API changes note compatibility

If you touch anything exported from `ktav`, say in the PR description
whether it is:

- **semver-compatible** (additions, looser signatures, doc changes); or
- **semver-breaking** (renamed / removed items, changed signatures,
  tightened types) — in which case the version bump lands in the next
  MINOR while we are pre-1.0.

Update the CHANGELOG source units under `root-docs/CHANGELOG/` (all
three `>>>>> lang=` blocks) in the same PR and regenerate the output.

>>>>> lang=ru
### 3. Изменения публичного API помечают совместимость

Если трогаешь что-то экспортируемое из `ktav`, в описании PR укажи:

- **semver-совместимо** (добавления, более мягкие сигнатуры, правки
  доков); или
- **semver-ломающе** (переименование / удаление, смена сигнатур,
  более жёсткие типы) — тогда bump версии уезжает в следующий MINOR
  пока мы pre-1.0.

Обновляй CHANGELOG-юниты под `root-docs/CHANGELOG/` (все три блока
`>>>>> lang=`) в том же PR и перегенерируй вывод.

>>>>> lang=zh
### 3. 公开 API 变更标注兼容性

如果改动了 `ktav` 的任何导出项，在 PR 描述里注明属于：

- **semver 兼容**（新增、放宽签名、文档变更）；或
- **semver 破坏**（重命名 / 移除、改签名、收紧类型）—— 在 pre-1.0
  阶段这类改动走下一个 MINOR。

在同一个 PR 中更新 `root-docs/CHANGELOG/` 下的 CHANGELOG 源单元
(全部三个 `>>>>> lang=` 块)并重新生成产物。

