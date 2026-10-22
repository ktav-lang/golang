>>>>> lang=en
Package functions and `pure.New()` expose all eight operations: `Loads`,
`LoadsStrict`, `LoadsInto`, `Dumps`, `DumpsForceStrings`, `EmitCanonical`,
`FormatSource`, `CanonicalFromSource`, plus full `pure.Error`/`pure.Span` aliases.
The package shares `internal/engine`; its runtime graph is stdlib-only.

Root `ktav.New(ktav.BackendPure)` remains compatible, but root imports require
`github.com/ebitengine/purego v0.8.4` and `golang.org/x/sys v0.28.0` even with
BackendPure. An offline packaged consumer uses a file-only proxy, readonly mode
and an import whitelist to verify no external runtime imports for `/pure`. Its
CI job runs the entire 0.8 conformance corpus through the installed `/pure`
artifact with `CGO_ENABLED=0` and with `rustc`/`cargo` trapped on `PATH`.

Inline compounds have a 128-level limit. Plain-map canonical keys sort
alphabetically (O(n log n)); source canonicalization and formatting preserve
source order. `encoding/json` bridge paths retain depth limits; unlimited depth
is not promised. Node docs and Go build/test commands are development tools,
not `/pure` runtime dependencies.

>>>>> lang=ru
Функции пакета и `pure.New()` предоставляют все восемь операций: `Loads`,
`LoadsStrict`, `LoadsInto`, `Dumps`, `DumpsForceStrings`, `EmitCanonical`,
`FormatSource`, `CanonicalFromSource`, а также полные псевдонимы `pure.Error`/
`pure.Span`. Пакет использует общий `internal/engine`; runtime-граф — только stdlib.

Совместимый root API `ktav.New(ktav.BackendPure)` сохраняется, но корневые импорты
требуют `github.com/ebitengine/purego v0.8.4` и `golang.org/x/sys v0.28.0` даже
при BackendPure. Offline-потребитель использует только файловый proxy, readonly
режим и белый список импортов, проверяя отсутствие внешних runtime-импортов `/pure`.
Его CI-задача прогоняет весь корпус конформности 0.8 через установленный артефакт
`/pure` с `CGO_ENABLED=0` и с перехваченными на `PATH` `rustc`/`cargo`.

Inline-составные значения ограничены 128 уровнями. Ключи обычных карт сортируются
по алфавиту (O(n log n)); канонизация и форматирование сохраняют порядок источника.
Пути `encoding/json` имеют собственные пределы глубины; бесконечная глубина не
обещана. Node-документация и Go-сборка/тесты — инструменты разработки, не runtime
зависимости `/pure`.

>>>>> lang=zh
包级函数和 `pure.New()` 提供全部八项操作：`Loads`、`LoadsStrict`、`LoadsInto`、
`Dumps`、`DumpsForceStrings`、`EmitCanonical`、`FormatSource`、
`CanonicalFromSource`，以及完整的 `pure.Error`/`pure.Span` 别名。该包共享
`internal/engine`；其 runtime 依赖图仅使用标准库。

兼容的根 API `ktav.New(ktav.BackendPure)` 仍保留，但根包导入即需要
`github.com/ebitengine/purego v0.8.4` 和 `golang.org/x/sys v0.28.0`，即使选择
BackendPure。离线打包消费者使用仅文件代理、readonly 模式和导入白名单，验证
`/pure` 没有外部 runtime 导入。其 CI 任务通过已安装的 `/pure` 产物运行完整的
0.8 一致性语料，使用 `CGO_ENABLED=0` 并在 `PATH` 上拦截 `rustc`/`cargo`。

Inline 复合值限制为 128 层。普通映射的规范键按字母排序（O(n log n)）；
源规范化和格式化保留源顺序。`encoding/json` 桥接路径有自身深度限制；不承诺
无限深度。Node 文档和 Go 构建/测试命令是开发工具，而非 `/pure` runtime 依赖。

