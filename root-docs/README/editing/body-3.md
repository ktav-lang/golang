>>>>> lang=en
- A no-op replacement (the new value equals the old one) returns the
  source byte-identical, however noncanonical its spelling.
- Marker transitions are minimal: switching a scalar to a numeric- or
  keyword-like String toggles only the second `:` of the `::` marker,
  and the original spacing around it is preserved.
- Dotted, inline, and block forms are preserved, including a whole
  inline container replacement and emptied-but-not-collapsed block
  containers.
- Inline ancestors expand to block form only when readability requires:
  a multiline replacement string, growth beyond max(100 Unicode scalar
  values, the old physical line length), or ≥ 4 inline escapes with ≥ 25%
  overhead. Untouched nested branches stay inline; new block levels use
  four spaces and the source EOL.
- Errors of class `"EditUnsupported"` name pairs with no lossless
  single-range patch: noncontiguous dotted-key/reopen targets, structural
  replacement that would drop interior comment lines (leaf edits inside
  such containers still work and keep the comments), or values without a
  lossless representation in the target context.
- Every successful patch is re-parsed and semantically verified before
  it is returned, so silent corruption is impossible.
- The native (C ABI) and pure-Go backends agree byte-for-byte; both are
  exercised against the same regression corpus.

>>>>> lang=ru
- No-op (замена равна старому значению) возвращает исходник
  байт-в-байт, каким бы неканоничным ни было его написание.
- Переходы маркеров минимальны: замена скаляра на числоподобную или
  keyword-подобную String переключает только второй `:` маркера `::`,
  исходные пробелы вокруг сохраняются.
- Dotted-, inline- и блочные формы сохраняются, включая замену целого
  inline-контейнера и опустошённый (но не свернувшийся) блочный
  контейнер.
- Inline-предки раскрываются в блочную форму только когда требует
  читаемость: многострочная строка замены, рост строки сверх
  max(100 Unicode scalar values, прежняя длина физической строки) или
  ≥ 4 inline-escapes с накладными расходами ≥ 25%. Незатронутые вложенные
  ветки остаются inline; новые блочные уровни — четыре пробела и EOL
  источника.
- Ошибки класса `"EditUnsupported"` описывают пары цель/замена без
  lossless-патча: noncontiguous dotted-key/reopen-цели, структурная
  замена, которая потеряла бы внутренние комментарии (правка листа
  внутри такого контейнера работает и сохраняет комментарии), значения без
  lossless-представления в контексте цели.
- Каждый успешный патч перед возвратом повторно парсится и семантически
  верифицируется — тихое повреждение невозможно.
- Native (C ABI) и pure-Go бэкенды совпадают байт-в-байт; оба
  прогоняются через один и тот же regression-корпус.

>>>>> lang=zh
- no-op 替换（新值等于旧值）逐字节返回源码，无论其写法多么非规范。
- 标记切换最小化：把标量换成数字形或关键字形 String 只切换 `::` 标记的
  第二个 `:`，其周围的原始空格保持不变。
- dotted、inline 与 block 形式均保留，包括整体替换内联容器，以及清空但
  不折叠的 block 容器。
- 仅在可读性需要时才把内联祖先展开为 block 形式：多行替换字符串、行长
  超过 max(100 个 Unicode scalar values, 原物理行长)，或 ≥ 4 个内联转义
  且开销 ≥ 25%。未触及的嵌套分支保持内联；新 block 层级使用四个空格和
  源文件的 EOL。
- `"EditUnsupported"` 类错误表示不存在无损单区间补丁：dotted-key/reopen
  形成的不连续目标、会丢失内部注释行的结构性替换（此类容器内的叶编辑
  仍可用且保留注释）、或无法在目标上下文中无损表示的值。
- 每个成功补丁在返回前都会重新解析并做语义验证 —— 不可能悄悄损坏文档。
- Native（C ABI）与纯 Go 后端逐字节一致；两者都针对同一回归语料运行。

