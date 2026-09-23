>>>>> lang=en
## 0.5.0 — 2026-05-28

### Breaking

- **Spec 0.5.0**: typed markers `:i` / `:f` no longer exist. Numbers
  are inferred from the scalar body's lexical form (`42` → Integer,
  `3.14` → Float). Documents written for spec 0.1.x with explicit
  typed markers parse differently and must be updated.
- **`##` comments**: single `#` is now a literal character; comments
  require `##`. Update any Ktav source that used `# comment`.
- **Float normalisation**: Float values are stored in canonical
  shortest-decimal form (no underscores, no leading `+`). Byte-for-byte
  comparison against old serialised output may fail.
- **C ABI now exports six symbols** — `ktav_emit_canonical` is added;
  pinning `KTAV_LIB_PATH` to a pre-0.5.0 binary will fail with a
  missing symbol error.

>>>>> lang=ru
## 0.5.0 — 2026-05-28

### Ломающие изменения

- **Spec 0.5.0**: типизированные маркеры `:i` / `:f` больше не
  существуют. Числа выводятся из лексической формы тела скаляра
  (`42` → Integer, `3.14` → Float). Документы, написанные для spec 0.1.x
  с явными типизированными маркерами, разбираются иначе и должны быть
  обновлены.
- **Комментарии `##`**: одиночный `#` теперь литеральный символ;
  комментарии требуют `##`. Обновите любой Ktav-исходник, в котором
  использовался `# комментарий`.
- **Нормализация Float**: значения Float хранятся в канонической форме
  shortest-decimal (без подчёркиваний, без ведущего `+`). Побайтовое
  сравнение со старым сериализованным выводом может ломаться.
- **C ABI теперь экспортирует шесть символов** — добавлен
  `ktav_emit_canonical`; привязка `KTAV_LIB_PATH` к библиотеке версии
  до 0.5.0 упадёт с ошибкой отсутствия символа.

>>>>> lang=zh
## 0.5.0 — 2026-05-28

### 破坏性变更

- **Spec 0.5.0**: 类型标记 `:i` / `:f` 不再存在。数字从标量体词法形式
  推断(`42` → Integer,`3.14` → Float)。为 spec 0.1.x 编写的、带显式
  类型标记的文档解析结果不同,必须更新。
- **`##` 注释**: 单 `#` 现为字面字符;注释需使用 `##`。请更新任何
  使用了 `# 注释` 的 Ktav 源码。
- **Float 规范化**: Float 值以 shortest-decimal 规范形式存储(无下划线,
  无前导 `+`)。与旧的序列化输出做字节级比对可能失败。
- **C ABI 新增第六个符号** `ktav_emit_canonical`;将 `KTAV_LIB_PATH`
  指向 pre-0.5.0 的二进制会因符号缺失而失败。

