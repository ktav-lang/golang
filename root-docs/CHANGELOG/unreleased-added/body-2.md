>>>>> lang=en

### Fixed

- `FormatSource` keeps a whole-line comment that precedes a dotted-key
  pair (`## c` before `a.b: 1`): the comment now attaches to the
  terminal pair of the expanded path instead of being dropped, and
  formatting remains a fixed point.
- Depth safety in pure mode: dotted-key nesting (§5.3.2) is unbounded
  by the spec, so the pure canonical/pretty writer, the formatter, the
  tagged-JSON encoder/decoder and the trivia walk are all iterative
  (explicit work stacks). Deep documents parse, canonicalise and
  format without recursion-depth failures, and no artificial depth
  limit is imposed on ordinary spec inputs.

>>>>> lang=ru
### Исправлено

- `FormatSource` сохраняет строчный комментарий, стоящий перед парой с
  составным (точечным) ключом (`## c` перед `a.b: 1`): комментарий
  теперь привязывается к терминальной паре развёрнутого пути, а не
  теряется; форматирование остаётся неподвижной точкой.
- Безопасность глубины в pure-режиме: вложенность точечных ключей
  (§5.3.2) спецификацией не ограничена, поэтому канонический/красивый
  писатель, форматировщик, кодек tagged-JSON и обход тривий итеративны
  (явные стеки задач). Глубокие документы разбираются, канонизируются
  и форматируются без переполнения стека, и на обычные входы
  спецификации искусственные ограничения глубины не накладываются.

>>>>> lang=zh
### 修复

- `FormatSource` 现在会保留位于点号键对之前的整行注释（`a.b: 1` 之前
  的 `## c`）：注释会附加到展开路径的末端键对上，而不再丢失，且格式
  化仍是不动点。
- Pure 模式的深度安全：点号键嵌套（§5.3.2）不受规范限制，因此 pure
  模式的规范/美观写入器、格式化器、tagged-JSON 编解码器与 trivia 遍历
  全部为迭代实现（显式工作栈）。深层文档可以解析、规范化与格式化而
  不会栈溢出，也不会对普通规范输入施加人为深度限制。

