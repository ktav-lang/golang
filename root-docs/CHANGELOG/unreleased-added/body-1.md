>>>>> lang=en
## Unreleased

### Added

- **`FormatSource(s string) (string, error)`** — the new `ktav_format`
  C ABI symbol, exposed as a comment-preserving formatter over Ktav
  source text. Every comment survives verbatim (spec § 3.4: a comment
  owns a whole line); blank lines survive as a grouping hint, but a
  run of two or more collapses to exactly one and blank padding
  immediately inside a bracket is dropped, so formatting is a fixed
  point: `FormatSource(FormatSource(x)) == FormatSource(x)`. Key order
  is never changed (spec § 5.9); for a document with no comments and
  no blank lines the output equals `CanonicalFromSource` of the same
  text.

>>>>> lang=ru
## Не выпущено

### Добавлено

- **`FormatSource(s string) (string, error)`** — новый символ C ABI
  `ktav_format`, открытый как форматтер Ktav-исходника с сохранением
  комментариев. Каждый комментарий выживает дословно (spec § 3.4:
  комментарий владеет строкой целиком); пустые строки выживают как
  подсказка группировки, но серия из двух и более схлопывается ровно в
  одну, а пустая отбивка непосредственно внутри скобки убирается —
  поэтому форматирование является неподвижной точкой:
  `FormatSource(FormatSource(x)) == FormatSource(x)`. Порядок ключей не
  меняется никогда (spec § 5.9); для документа без комментариев и
  пустых строк результат совпадает с `CanonicalFromSource` того же
  текста.

>>>>> lang=zh
## 未发布

### 新增

- **`FormatSource(s string) (string, error)`** —— 新的 C ABI 符号
  `ktav_format`,作为保留注释的 Ktav 源文本格式化器对外暴露。每条注释
  都逐字保留(spec § 3.4:注释独占一整行);空行作为分组提示保留,但
  连续两行及以上会合并为恰好一行,紧贴括号内侧的空行填充会被丢弃,
  因此格式化是一个不动点:
  `FormatSource(FormatSource(x)) == FormatSource(x)`。键顺序绝不改变
  (spec § 5.9);对于没有注释也没有空行的文档,输出等同于同一文本的
  `CanonicalFromSource`。

