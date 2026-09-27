>>>>> lang=en
Two details that are easy to get wrong:

- **`Span` is byte offsets into UTF-8**, not UTF-16 code units. An LSP
  consumer either converts or negotiates `positionEncoding: "utf-8"`.
- **`Path` is a slice of segments, never a joined string.** A key
  literally named `a.b` is one segment and cannot be confused with a
  two-segment path — there is no separator to be ambiguous about.

For native structured errors, `Msg` is taken from the core verbatim
rather than assembled here, so the same document produces the same error
text in every Ktav binding. Loader, Go conversion, and host-synthesized
validation errors are outside that cross-binding guarantee.

>>>>> lang=ru
Две детали, в которых легко ошибиться:

- **`Span` — это байтовые смещения в UTF-8**, а не кодовые единицы
  UTF-16. Потребитель LSP либо преобразует их, либо договаривается о
  `positionEncoding: "utf-8"`.
- **`Path` — срез сегментов, а не склеенная строка.** Ключ, буквально
  названный `a.b`, — это один сегмент, и его нельзя спутать с
  двухсегментным путём: в проводном контракте нет разделителя, о котором
  можно было бы двусмысленно судить.

Для структурированных ошибок native-ядра `Msg` берётся дословно, а не
собирается здесь, поэтому один и тот же документ даёт одинаковый текст
ошибки во всех биндингах Ktav. Ошибки загрузчика, преобразования Go и
host-синтезированной проверки не входят в эту межъязыковую гарантию.

>>>>> lang=zh
两处容易弄错的细节:

- **`Span` 是 UTF-8 中的字节偏移**,不是 UTF-16 码元。LSP 消费方要么
  自行转换,要么协商 `positionEncoding: "utf-8"`。
- **`Path` 是键段切片,绝不是拼接后的字符串。** 字面名为 `a.b` 的键
  是**一个**段,不可能与两段路径混淆 —— 没有分隔符,也就无可歧义。

对于原生核心的结构化错误,`Msg` 是逐字取得的,并非在此处拼装,因此同一
份文档在任何 Ktav 绑定中都会产生相同的错误文本。加载器、Go 转换和
host 合成的校验错误不在此跨绑定保证范围内。

