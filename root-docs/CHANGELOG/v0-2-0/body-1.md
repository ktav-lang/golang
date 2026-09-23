>>>>> lang=en

## 0.2.0 — 2026-05-07

### Changed (breaking)

- **Picked up `ktav 0.2.0`** — multi-line strings now serialize in the
  indented stripped `( ... )` form by default (verbatim `(( ... ))`
  remains as fallback for content with leading whitespace or sole-`)`
  lines). `:f 42` accepts integer literals (parsed as `42.0`).
  See the
  [`ktav` crate CHANGELOG](https://github.com/ktav-lang/rust/blob/main/CHANGELOG.md#020--2026-05-07)
  for the full delta.

  Code comparing serialized output byte-for-byte to a baked-in
  `((...))` literal must be updated. Round-trip is unchanged.

### Spec

- spec submodule synced (typed_float_without_decimal moved invalid →
  valid/typed_float_integer_body).

>>>>> lang=ru

## 0.2.0 — 2026-05-07

### Изменено (ломающее)

- **Подхватили `ktav 0.2.0`** — многострочные строки теперь
  сериализуются в отступлённую очищенную форму `( ... )` по умолчанию
  (дословная `(( ... ))` остаётся запасным вариантом для содержимого с
  ведущими пробелами или строками из одной `)`). `:f 42` принимает
  целые литералы (разбираются как `42.0`). Полный diff — в
  [`ktav` crate CHANGELOG](https://github.com/ktav-lang/rust/blob/main/CHANGELOG.md#020--2026-05-07).

  Код, сравнивающий сериализованный вывод побайтово с зашитым
  литералом `((...))`, придётся обновить. Round-trip не изменился.

### Спецификация

- submodule spec синхронизирован (typed_float_without_decimal перенесён
  из invalid → valid/typed_float_integer_body).

>>>>> lang=zh

## 0.2.0 — 2026-05-07

### 变更(破坏性)

- **升级到 `ktav 0.2.0`** —— 多行字符串现在默认序列化为缩进剥离的
  `( ... )` 形式(逐字的 `(( ... ))` 仍作为内容含前导空白或整行仅一个
  `)` 时的回退)。`:f 42` 接受整数字面量(解析为 `42.0`)。完整变更见
  [`ktav` crate CHANGELOG](https://github.com/ktav-lang/rust/blob/main/CHANGELOG.md#020--2026-05-07)。

  将序列化输出与内置的 `((...))` 字面量逐字节比较的代码需要更新。
  Round-trip 不变。

### 规范

- spec submodule 同步(typed_float_without_decimal 从 invalid 移至
  valid/typed_float_integer_body)。

