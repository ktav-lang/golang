>>>>> lang=en
## 0.8.1 — 2026-10-22

### Added

- `pure` is a separate stdlib-only package: `import "github.com/ktav-lang/golang/pure"`; package functions and `pure.New()` expose all eight operations and `Error`/`Span` aliases. The legacy root `BackendPure` API remains, but importing the root module retains its native `purego` and `x/sys` dependencies.
- The native core is upgraded to the published Rust `ktav` 0.8.1 with spec 0.8.1, covering all five corpus categories (valid, invalid, unrepresentable, parseable-unrepresentable, strict-lossy).
- The packaged pure consumer CI job runs the entire 0.8 conformance corpus through the installed `/pure` artifact with `CGO_ENABLED=0` and with `rustc`/`cargo` trapped on `PATH`, in addition to the file-only proxy, readonly mode, and import whitelist.

### Fixed

- Empty source buffers now pass a non-null pointer through the C ABI, preserving the empty Object and zero-byte canonical document contract with Rust debug builds.
- Pure parser/writer traversal of dotted-key nesting is iterative. Public map/slice emission and typed `LoadsInto` decoding still use `encoding/json`; this release does not promise unlimited nesting through that bridge.
- `FormatSource` preserves comments preceding dotted-key pairs and keeps trailing comments inside wrapped root Arrays; formatting remains a fixed point.
- Cyclic Go values are rejected by `Dumps`/`DumpsForceStrings`/`EmitCanonical` with a structured `Message`-class error instead of recursing; shared sub-maps on sibling branches still encode.
- Cycle detection is correct for multi-container loops; `Loads`/`LoadsStrict`/`LoadsInto` reject trailing data after the top-level wire value and accept an empty-string JSON key; the `::` raw-marker decision restores the §5.2 keyword/shortcut guards.
- Writer key sorting in the writer conversion is O(n log n) per map (alphabetical wire order), on top of the linear walk.

>>>>> lang=ru
## 0.8.1 — 2026-10-22

### Добавлено

- `pure` — отдельный пакет только со стандартной библиотекой: `import "github.com/ktav-lang/golang/pure"`; функции пакета и `pure.New()` предоставляют все восемь операций и псевдонимы `Error`/`Span`. Совместимый API `BackendPure` корневого пакета сохранён, но импорт корневого модуля сохраняет native-зависимости `purego` и `x/sys`.
- Нативное ядро обновлено до опубликованного Rust `ktav` 0.8.1 со спецификацией 0.8.1 и всеми пятью категориями корпуса (valid, invalid, unrepresentable, parseable-unrepresentable, strict-lossy).
- CI-задача упакованного pure-потребителя прогоняет весь корпус конформности 0.8 через установленный артефакт `/pure` с `CGO_ENABLED=0` и с перехваченными на `PATH` `rustc`/`cargo`, в дополнение к файловому proxy, readonly-режиму и белому списку импортов.

### Исправлено

- Пустой исходный буфер передаётся через C ABI с ненулевым указателем: debug-сборки Rust сохраняют контракт пустого Object и canonical-документа из нуля байтов.
- Обход вложенности точечных ключей в pure-парсере и генераторе итеративен. Публичная сериализация карт/срезов и типизированное декодирование `LoadsInto` по-прежнему используют `encoding/json`; выпуск не гарантирует неограниченную вложенность через этот мост.
- `FormatSource` сохраняет комментарии перед парами с точечными ключами и завершающие комментарии внутри явного корневого Array; форматирование остаётся неподвижной точкой.
- Циклические значения Go отклоняются `Dumps`/`DumpsForceStrings`/`EmitCanonical` структурной ошибкой класса `Message` вместо рекурсии; общие вложенные карты на соседних ветвях по-прежнему кодируются.
- Обнаружение циклов корректно для многоконтурных циклов; `Loads`/`LoadsStrict`/`LoadsInto` отклоняют данные после верхнеуровневого значения провода и принимают пустой JSON-ключ; решение о `::`-маркере восстанавливает guards §5.2.
- Сортировка ключей карт в преобразовании писателя — O(n log n) на карту (алфавитный порядок провода), поверх линейного обхода.

>>>>> lang=zh
## 0.8.1 — 2026-10-22

### 新增

- `pure` 是独立的仅标准库包：`import "github.com/ktav-lang/golang/pure"`；包级函数和 `pure.New()` 提供全部八项操作及 `Error`/`Span` 别名。根包的兼容 `BackendPure` API 保留，但导入根模块仍会带入 native 依赖 `purego` 和 `x/sys`。
- 原生核心升级到已发布的 Rust `ktav` 0.8.1，配套规范 0.8.1，覆盖全部五类语料（valid、invalid、unrepresentable、parseable-unrepresentable、strict-lossy）。
- 打包 pure 消费者的 CI 任务在仅文件代理、readonly 模式和导入白名单之外，还通过已安装的 `/pure` 产物运行完整的 0.8 一致性语料，使用 `CGO_ENABLED=0` 并在 `PATH` 上拦截 `rustc`/`cargo`。

### 修复

- 空源缓冲区通过 C ABI 传递非空指针，使 Rust debug 构建保持空 Object 和零字节 canonical 文档的约定。
- Pure 解析器和写入器以迭代方式遍历点号键嵌套。公开的映射/切片序列化与 `LoadsInto` 类型化解码仍使用 `encoding/json`；本版本不保证通过该桥接层支持无限嵌套。
- `FormatSource` 保留点号键对之前的注释，以及带括号根 Array 内的尾随注释；格式化仍是不动点。
- 循环引用的 Go 值会被 `Dumps`/`DumpsForceStrings`/`EmitCanonical` 以 `Message` 类结构化错误拒绝，而非递归失控；兄弟分支间共享的子映射仍可编码。
- 循环检测对多容器循环也正确；`Loads`/`LoadsStrict`/`LoadsInto` 拒绝顶层线格式值之后的尾随数据并接受空字符串 JSON 键；`::` 原始标记判定恢复 §5.2 的关键字/捷径守卫。
- 写入器转换中对每个映射的键排序为 O(n log n)（线格式字母序），叠加在线性遍历之上。

