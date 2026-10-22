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
(§5.8.4, implementation-defined, mirroring the reference). Dotted-key nesting (§5.3.2) is NOT capped: the pure writers, the tagged-JSON bridge and the trivia walk are iterative (explicit work stacks), so deep documents cannot overflow the stack.

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
эталоне). Вложенность точечных ключей (§5.3.2) НЕ ограничивается: писатели, tagged-JSON-мост и обход тривий итеративны (явные стеки задач), поэтому глубокие документы не могут переполнить стек.

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
结构嵌套上限为 128 层（§5.8.4，实现定义，与参考实现一致）。点号键嵌套（§5.3.2）不设上限：pure 写入器、tagged-JSON 桥与 trivia 遍历均为迭代实现（显式工作栈），深层文档不会导致栈溢出。

