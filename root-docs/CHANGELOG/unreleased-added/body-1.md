>>>>> lang=en

### Fixed

- `FormatSource` keeps a whole-line comment that precedes a dotted-key pair (`## c` before `a.b: 1`): the comment now attaches to the terminal pair of the expanded path instead of being dropped, and formatting remains a fixed point.
- Depth safety in pure mode: dotted-key nesting (§5.3.2) is unbounded by the spec, so the pure canonical/pretty writer, the formatter, the tagged-JSON encoder/decoder and the trivia walk are all iterative (explicit work stacks); ordinary map/slice emission (`json.Marshal`) and the typed `LoadsInto` decode (`json.Decoder.Decode`) stay on `encoding/json`, whose own nesting limit returns an error, not a crash. Deep documents parse, canonicalise and format without recursion-depth failures, and no artificial depth limit is imposed on ordinary spec inputs.
- Cyclic Go values (a map or slice containing itself on the active path) are now rejected by `Dumps`/`DumpsForceStrings`/`EmitCanonical` with a structured `Message`-class error instead of recursing; shared sub-maps on sibling branches (DAG references) still encode. The public Go-side bridge conversions (`decodeJSON`, `toTagged`, `prepareWriterValue`, untagged flattening) are iterative.
- Cycle detection is now correct for multi-container loops: an explicit ENTER/EXIT active-path scheme rejects a→b→a cycles on the first revisit (slice identity includes len/cap); `Loads`/`LoadsStrict`/`LoadsInto` reject trailing data after the top-level wire value and accept an empty-string JSON key; the `::` raw-marker decision restores the §5.2 keyword/shortcut guards (a Go String `"null"`, `"true"`, `"false"`, `"("`, `"(("`, `"()"`, `"(())"` re-parses as the same String, and force-string of a Bool re-parses as String too).
- Writer conversion is honest O(input + output): `prepareWriterValue` keeps one shared path buffer (snapshot only on the cold error path) so error paths report exact nested keys; scan-probe byte counters run inside the classification scans, bounded by one pass over the body — documented cost, not a claimed zero. Map key sorting in the writer conversion is O(n log n) per map (alphabetical wire order), on top of the linear walk.

>>>>> lang=ru

### Исправлено

- `FormatSource` сохраняет строчный комментарий, стоящий перед парой с составным (точечным) ключом (`## c` перед `a.b: 1`): комментарий теперь привязывается к терминальной паре развёрнутого пути, а не теряется; форматирование остаётся неподвижной точкой.
- Безопасность глубины в pure-режиме: вложенность точечных ключей (§5.3.2) спецификацией не ограничена, поэтому канонический/красивый писатель, форматировщик, кодек tagged-JSON и обход тривий итеративны (явные стеки задач); обычная эмиссия карт/срезов (`json.Marshal`) и типизированное декодирование `LoadsInto` (`json.Decoder.Decode`) остаются на `encoding/json`, чей собственный предел вложенности возвращает ошибку, а не падение. Глубокие документы разбираются, канонизируются и форматируются без переполнения стека, и на обычные входы спецификации искусственные ограничения глубины не накладываются.
- Циклические значения Go (карта или срез, содержащие самих себя на активном пути) теперь отклоняются `Dumps`/`DumpsForceStrings`/`EmitCanonical` структурной ошибкой класса `Message` вместо рекурсии; общие вложенные карты на соседних ветвях (DAG) по-прежнему кодируются. Преобразования публичного моста на стороне Go (`decodeJSON`, `toTagged`, `prepareWriterValue`, разглаживание) итеративны.
- Обнаружение циклов теперь корректно для многоконтурных циклов: явная схема ENTER/EXIT на активном пути отклоняет цикл a→b→a при первом повторном заходе (идентичность срезов учитывает длину и ёмкость); `Loads`/`LoadsStrict`/`LoadsInto` отклоняют данные после верхнеуровневого значения провода и принимают пустой JSON-ключ; решение о `::`-маркере восстанавливает guards §5.2 (Go-строка `"null"`, `"true"`, `"false"`, `"("`, `"(("`, `"()"`, `"(())"` разбирается обратно в ту же строку, принудительная строка Bool — тоже как String).
- Сложность преобразования писателя честно равна O(вход + выход): `prepareWriterValue` держит один общий буфер пути (копия — только на холодном пути ошибки), пути ошибок называют точные вложенные ключи; байтовые счётчики scan-probe выполняются внутри сканов классификации, ограничены одним проходом по телу — задокументированная стоимость, а не заявленный ноль. Сортировка ключей карт в преобразовании писателя — O(n log n) на карту (алфавитный порядок провода), поверх линейного обхода.

>>>>> lang=zh

### 修复

- `FormatSource` 现在会保留位于点号键对之前的整行注释（`a.b: 1` 之前的 `## c`）：注释会附加到展开路径的末端键对上，而不再丢失，且格式化仍是不动点。
- Pure 模式的深度安全：点号键嵌套（§5.3.2）不受规范限制，因此 pure 模式的规范/美观写入器、格式化器、tagged-JSON 编解码器与 trivia 遍历全部为迭代实现（显式工作栈）；普通的映射/切片发射（`json.Marshal`）与 `LoadsInto` 的类型化解码（`json.Decoder.Decode`）仍走 `encoding/json`，其自带嵌套上限，超限返回错误而非崩溃。深层文档可以解析、规范化与格式化而不会栈溢出，也不会对普通规范输入施加人为深度限制。
- 循环引用的 Go 值（映射或切片在活跃路径上包含自身）现在会被 `Dumps`/`DumpsForceStrings`/`EmitCanonical` 以 `Message` 类结构化错误拒绝，而非递归失控；兄弟分支间共享的子映射（DAG 引用）仍可编码。Go 侧公共桥转换（`decodeJSON`、`toTagged`、`prepareWriterValue`、去标签展平）均为迭代实现。
- 循环检测现在对多容器循环也正确：显式的 ENTER/EXIT 活跃路径方案在首次重访即拒绝 a→b→a 循环（切片身份包含长度与容量）；`Loads`/`LoadsStrict`/`LoadsInto` 拒绝顶层线格式值之后的尾随数据并接受空字符串 JSON 键；`::` 原始标记判定恢复了 §5.2 的关键字/捷径守卫（Go 字符串 `"null"`、`"true"`、`"false"`、`"("`、`"(("`、`"()"`、`"(())"` 重解析为同一字符串，Bool 的强制字符串也重解析为 String）。
- 写入器转换的复杂度如实为 O(输入 + 输出)：`prepareWriterValue` 只维护一个共享路径缓冲（仅在冷错误路径上快照），错误路径报告精确的嵌套键；扫描探针字节计数在分类扫描内部执行，以对主体的一次遍历为界——记录在案的成本，而非宣称的零。写入器转换中对每个映射的键排序为 O(n log n)（线格式字母序），叠加在线性遍历之上。

