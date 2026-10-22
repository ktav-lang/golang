>>>>> lang=en
For concurrent or library code that must not depend on process-global
state, bind the operations to one backend explicitly:

```go
ops, err := ktav.New(ktav.BackendPure)
ops.Loads(src) // plus LoadsStrict, LoadsInto, Dumps,
               // DumpsForceStrings, EmitCanonical, FormatSource,
               // CanonicalFromSource — all bound to this instance
```

An `Operations` value carries its own backend: its methods never read
or write the global default, so instances stay isolated from each other
and from `UseBackend`. The package-level functions keep the global
default (native) behaviour.

Limits of pure mode: decoded values keep the documented
`map[string]any` contract (decoded key order is not represented);
`EmitCanonical` over plain Go maps keeps the documented alphabetical
order, while `CanonicalFromSource` and `FormatSource` preserve source
insertion order; inline compound nesting is capped at 128 levels
(§5.8.4, implementation-defined, mirroring the reference). Dotted-key nesting (§5.3.2) is NOT capped: the pure writers, the trivia walk, the tagged-JSON codec and the Go-side public bridge conversions (`decodeJSON`, `toTagged`, `prepareWriterValue`, the untagged flattening) are iterative (explicit work stacks), so deep documents and deep programmatic values cannot overflow the stack. A cyclic Go value (a map or slice that contains itself on the active path) is rejected by `Dumps`/`DumpsForceStrings`/`EmitCanonical` with a structured error (`Class: "Message"`, "ktav: cyclic Go value cannot be encoded"); shared sub-maps on sibling branches (a DAG, not a cycle) keep working. Values that fall back through `encoding/json` (arbitrary structs) rely on that package's own nesting limit, which returns an error — not a crash — beyond it; this package does not extend or verify that fallback depth. Cycle detection is ACTIVE-PATH based: an explicit ENTER/EXIT work-stack scheme keeps each container marked active until its entire subtree is converted, so a→b→a loops are rejected on the first revisit while shared DAG nodes (already exited) convert again; slice identity uses (pointer, len, cap), so distinct views of one backing array are never conflated. `Loads`/`LoadsStrict`/`LoadsInto` reject trailing data after the top-level wire value (a second top-level JSON value is an error, not a silent overwrite), and a JSON key `""` is a legitimate key. The scan-probe instrumentation (a nil hook in production) pays one nil-test per coarse event plus per-byte counters inside the string-classification scans — bounded by one pass over the scanned body, documented rather than claimed free.

>>>>> lang=ru
Для параллельного или библиотечного кода, который не должен зависеть от
глобального состояния процесса, привяжите операции к одному бэкенду
явно:

```go
ops, err := ktav.New(ktav.BackendPure)
ops.Loads(src) // плюс LoadsStrict, LoadsInto, Dumps,
               // DumpsForceStrings, EmitCanonical, FormatSource,
               // CanonicalFromSource — все привязаны к этому экземпляру
```

Значение `Operations` несёт собственный бэкенд: его методы не читают и
не изменяют глобальное значение по умолчанию, поэтому экземпляры
изолированы друг от друга и от `UseBackend`. Функции уровня пакета
сохраняют глобальное поведение по умолчанию (нативное).

Ограничения pure-режима: декодированные значения сохраняют
документированный контракт `map[string]any` (порядок ключей не
представим); `EmitCanonical` для обычных Go-карт сохраняет
документированный алфавитный порядок, тогда как `CanonicalFromSource`
и `FormatSource` сохраняют порядок вставки; вложенность инлайн-структур
ограничена 128 уровнями (§5.8.4, определено реализацией, как в
эталоне). Вложенность точечных ключей (§5.3.2) НЕ ограничивается: писатели, обход тривий, tagged-JSON-кодек и преобразования публичного моста на стороне Go (`decodeJSON`, `toTagged`, `prepareWriterValue`, разглаживание) итеративны (явные стеки задач), поэтому глубокие документы и глубоко вложенные программные значения не могут переполнить стек. Циклическое значение Go (карта или срез, содержащие самих себя на активном пути) отклоняется `Dumps`/`DumpsForceStrings`/`EmitCanonical` со структурной ошибкой (`Class: "Message"`, «ktav: cyclic Go value cannot be encoded»); общие вложенные карты на соседних ветвях (DAG, а не цикл) продолжают работать. Значения, уходящие в обход через `encoding/json` (произвольные структуры), полагаются на собственный предел вложенности этого пакета, который возвращает ошибку, а не падение; данный пакет не расширяет и не проверяет глубину этого обхода. Обнаружение циклов работает по АКТИВНОМУ ПУТИ: явная схема ENTER/EXIT на стеке задач помечает контейнер активным, пока не преобразовано всё его поддерево, поэтому цикл a→b→a отклоняется при первом повторном заходе, а общие узлы DAG (уже вышедшие) преобразуются повторно; идентичность срезов определяется (указатель, длина, ёмкость), поэтому разные представления одного массива не смешиваются. `Loads`/`LoadsStrict`/`LoadsInto` отклоняют данные после верхнеуровневого значения провода (второе JSON-значение — ошибка, а не тихая перезапись), а ключ JSON `""` является допустимым ключом. Инструментация scan-probe (нулевой хук в продакшене) платит один nil-тест на крупное событие плюс счётчики байтов внутри сканов классификации строк — ограничено одним проходом по телу; это задокументировано, а не объявлено бесплатным.

>>>>> lang=zh
对于不应依赖进程全局状态的并发或库代码，可将操作显式绑定到某一个后
端：

```go
ops, err := ktav.New(ktav.BackendPure)
ops.Loads(src) // 以及 LoadsStrict、LoadsInto、Dumps、
               // DumpsForceStrings、EmitCanonical、FormatSource、
               // CanonicalFromSource —— 全部绑定到该实例
```

`Operations` 值携带自己的后端：其方法不会读取或修改全局默认值，因此
各实例彼此隔离，也与 `UseBackend` 隔离。包级函数保留全局默认（原生）
行为。

Pure 模式的限制：解码值保持文档规定的 `map[string]any` 契约（解码键
顺序无法表示）；对普通 Go 映射，`EmitCanonical` 保持文档规定的字母
序，而 `CanonicalFromSource` 与 `FormatSource` 保留插入顺序；内联复合
结构嵌套上限为 128 层（§5.8.4，实现定义，与参考实现一致）。点号键嵌套（§5.3.2）不设上限：pure 写入器、trivia 遍历、tagged-JSON 编解码器以及 Go 侧公共桥转换（`decodeJSON`、`toTagged`、`prepareWriterValue`、去标签展平）均为迭代实现（显式工作栈），深层文档与深层程序构造的值不会导致栈溢出。循环引用的 Go 值（映射或切片在活跃路径上包含自身）会被 `Dumps`/`DumpsForceStrings`/`EmitCanonical` 以结构化错误拒绝（`Class: "Message"`，"ktav: cyclic Go value cannot be encoded"）；兄弟分支间共享的子映射（DAG 而非循环）仍可正常工作。经 `encoding/json` 回退的值（任意结构体）依赖该包自身的嵌套上限——超限时返回错误而非崩溃；本包不扩展也不验证该回退路径的深度。循环检测基于活跃路径：显式的 ENTER/EXIT 工作栈方案在容器整个子树转换完成前保持其活跃标记，因此 a→b→a 循环在首次重访即被拒绝，而已退出的共享 DAG 节点会再次转换；切片身份由（指针、长度、容量）决定，同一底层数组的不同视图不会被混淆。`Loads`/`LoadsStrict`/`LoadsInto` 拒绝顶层线格式值之后的尾随数据（第二个顶层 JSON 值是错误而非静默覆盖），且 JSON 键 `""` 是合法键。扫描探针插桩（生产环境中为 nil 钩子）在每个粗粒度事件上付出一次 nil 判断，并在字符串分类扫描内部付出逐字节计数——以对被扫描主体的一次遍历为界；这是如实记录的成本，而非宣称免费。

