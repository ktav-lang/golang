>>>>> lang=en
- **Breaking:** the error channel now surfaces the structured ktav
  error envelope. `ktav.Error` carries the envelope's fields as
  first-class members — `Class`, `Reason`, `Line`, `LineText`, `Span`,
  `Path` (exact decoded key segments, never a joined string), `Body`,
  `Canonical`, `SpecSection` — plus `Msg`, which `Error()` returns.
  `Msg` is the core's own rendering, taken verbatim rather than
  reassembled here, so the same document produces the same error text
  in every Ktav binding. Surface error text therefore changes; matching
  on raw message strings is not stable across this boundary. Intended
  change.
- The conformance runner reads `spec/versions/0.8/tests` (it silently
  kept reading the stale `0.7` corpus after the submodule was re-pinned
  to `0.8.0` — the path was hardcoded, not derived from the pin) and
  executes every fixture category the corpus ships: `unrepresentable/`
  and `parseable-unrepresentable/` (a JSON value / parseable value the
  writer must refuse, asserted on both `Dumps` and `EmitCanonical`) and
  the new `strict-lossy/` (`Loads` must equal the lax value, `LoadsStrict`
  must refuse with the matching reason, body and canonical form), each
  with per-case reason assertions. A guard test fails the build if an
  unrecognized category directory appears under the corpus, so a future
  addition can't repeat this silently.

>>>>> lang=ru
- **Ломающее:** канал ошибок теперь несёт структурированный конверт
  ошибки ktav. `ktav.Error` держит поля конверта первоклассными членами
  — `Class`, `Reason`, `Line`, `LineText`, `Span`, `Path` (точные
  декодированные сегменты ключа, а не склеенная строка), `Body`,
  `Canonical`, `SpecSection` — плюс `Msg`, который и возвращает
  `Error()`. `Msg` — собственный рендеринг ядра, взятый дословно, а не
  пересобранный здесь, поэтому один и тот же документ даёт один и тот же
  текст ошибки в любом биндинге Ktav. Внешний текст ошибки,
  следовательно, меняется; сопоставление с сырыми строками сообщений
  через эту границу нестабильно. Изменение намеренное.
- Conformance-раннер читает `spec/versions/0.8/tests` (после
  перезакрепления сабмодуля на `0.8.0` он молча продолжал читать
  устаревший корпус `0.7` — путь был захардкожен, а не выведен из
  пина) и исполняет все категории корпуса: `unrepresentable/` и
  `parseable-unrepresentable/` (JSON-значение / разбираемое значение,
  которое writer обязан отвергнуть — проверяется и на `Dumps`, и на
  `EmitCanonical`), а также новую `strict-lossy/` (`Loads` обязан
  совпасть с lax-значением, `LoadsStrict` обязан отказать с
  соответствующей причиной, телом и канонической формой), каждая с
  проверкой кода причины по случаям. Guard-тест обрушивает сборку при
  появлении нераспознанной категории в корпусе, чтобы это не
  повторилось молча.

>>>>> lang=zh
- **破坏性:** 错误通道现在呈现结构化的 ktav 错误信封。`ktav.Error`
  以一等成员携带信封字段 —— `Class`、`Reason`、`Line`、`LineText`、
  `Span`、`Path`(精确解码后的键段,绝不是拼接字符串)、`Body`、
  `Canonical`、`SpecSection` —— 外加 `Msg`,`Error()` 返回的正是它。
  `Msg` 是核心自身的渲染文本,逐字取得而非在此处重新拼装,因此同一
  份文档在任何 Ktav 绑定中都产生相同的错误文本。对外的错误文本因此
  发生变化;跨越这一边界去匹配原始消息字符串并不稳定。此变更是有意
  为之。
- Conformance 运行器读取 `spec/versions/0.8/tests`(子模块重新固定到
  `0.8.0` 之后,它一直静默读取过期的 `0.7` 语料——路径是硬编码的,
  并非从固定版本推导而来),并执行语料中的每个类别:
  `unrepresentable/` 与 `parseable-unrepresentable/`(writer 必须拒绝
  的 JSON 值 / 可解析的值——对 `Dumps` 与 `EmitCanonical` 都作断言),
  以及新增的 `strict-lossy/`(`Loads` 必须等于 lax 值,`LoadsStrict`
  必须以匹配的原因、body 与规范形式拒绝),每个类别都逐例断言原因码。
  一个 guard 测试会在语料中出现无法识别的类别目录时使构建失败,以
  防止这个问题再次悄然发生。

