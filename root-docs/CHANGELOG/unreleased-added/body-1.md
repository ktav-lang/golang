>>>>> lang=en
## Unreleased

### Added

- A full pure-Go backend: parser and generator implemented natively in
  Go, usable as a complete alternative to the Rust-backed path for
  every public operation (`Loads`, `LoadsStrict`, `LoadsInto`, `Dumps`,
  `DumpsForceStrings`, `EmitCanonical`, `FormatSource`,
  `CanonicalFromSource`). Select it explicitly with `ktav.UsePure()`
  or `ktav.UseBackend(ktav.BackendPure)`; `UseNative` selects the
  purego `ktav_cabi` path, which remains the default. There is no
  automatic fallback in either direction.
- Pure mode loads no shared library, downloads nothing and spawns no
  subprocess, so it works in environments where no `ktav_cabi`
  artifact is available.
- Per-instance backend binding: `ktav.New(ktav.Backend)` returns an
  `Operations` value whose `Loads`, `LoadsStrict`, `LoadsInto`, `Dumps`,
  `DumpsForceStrings`, `EmitCanonical`, `FormatSource` and
  `CanonicalFromSource` methods are bound to that backend and never
  touch the process-global default — safe for concurrent use. The
  package-level functions keep the global-default (native) behaviour.
- The native backend and the bundled Rust crate are upgraded to
  `ktav 0.8.1` (spec 0.8.1).
- The native loader now verifies the loaded `ktav_cabi` artifact: after
  loading it calls `ktav_version` and fails with
  "ktav_cabi version X does not match expected 0.8.1" on any mismatch —
  including a `$KTAV_LIB_PATH` override pointing at an out-of-sync
  build (overrides still resolve, they are just verified like any other
  artifact).
- Known limits of pure mode: decoded values keep the existing
  `map[string]any` contract (key order of decoded maps is not
  represented); `EmitCanonical` on plain Go maps keeps the documented
  alphabetical key order while `CanonicalFromSource` and
  `FormatSource` preserve source insertion order; inline compound
  nesting is capped at 128 levels (§5.8.4, implementation-defined,
  mirroring the reference).

>>>>> lang=ru
## Не выпущено

### Добавлено

- Полноценный pure-Go бэкенд: парсер и генератор реализованы на Go и
  являются полной альтернативой пути через Rust для всех публичных
  операций (`Loads`, `LoadsStrict`, `LoadsInto`, `Dumps`,
  `DumpsForceStrings`, `EmitCanonical`, `FormatSource`,
  `CanonicalFromSource`). Выбирается явно через `ktav.UsePure()` или
  `ktav.UseBackend(ktav.BackendPure)`; `UseNative` выбирает путь через
  purego `ktav_cabi`, который остаётся значением по умолчанию.
  Автоматического переключения нет ни в одну сторону.
- Pure-режим не загружает разделяемую библиотеку, ничего не скачивает
  и не порождает подпроцессов, поэтому работает там, где артефакт
  `ktav_cabi` недоступен.
- Привязка бэкенда к экземпляру: `ktav.New(ktav.Backend)` возвращает
  значение `Operations`, методы которого (`Loads`, `LoadsStrict`,
  `LoadsInto`, `Dumps`, `DumpsForceStrings`, `EmitCanonical`,
  `FormatSource`, `CanonicalFromSource`) привязаны к этому бэкенду и не
  затрагивают глобальное состояние процесса — безопасно для
  параллельного использования. Функции уровня пакета сохраняют
  глобальное поведение по умолчанию (нативное).
- Нативный бэкенд и встроенный Rust-crate обновлены до `ktav 0.8.1`
  (spec 0.8.1).
- Загрузчик нативной библиотеки теперь проверяет загруженный артефакт
  `ktav_cabi`: после загрузки он вызывает `ktav_version` и при любом
  несовпадении завершается ошибкой "ktav_cabi version X does not match
  expected 0.8.1" — в том числе для переопределения через
  `$KTAV_LIB_PATH`, указывающего на рассинхронизированную сборку
  (переопределение по-прежнему работает, просто проверяется, как и
  любой другой артефакт).
- Известные ограничения pure-режима: декодированные значения сохраняют
  существующий контракт `map[string]any` (порядок ключей декодированных
  карт не представим); `EmitCanonical` для обычных Go-карт сохраняет
  документированный алфавитный порядок ключей, тогда как
  `CanonicalFromSource` и `FormatSource` сохраняют порядок вставки;
  вложенность инлайн-структур ограничена 128 уровнями (§5.8.4,
  определено реализацией, как в эталоне).

>>>>> lang=zh
## 未发布

### 新增

- 完整的 pure-Go 后端：解析器和生成器以 Go 原生实现，可作为基于
  Rust 路径的完整替代，覆盖所有公开操作（`Loads`、`LoadsStrict`、
  `LoadsInto`、`Dumps`、`DumpsForceStrings`、`EmitCanonical`、
  `FormatSource`、`CanonicalFromSource`）。通过 `ktav.UsePure()` 或
  `ktav.UseBackend(ktav.BackendPure)` 显式选择；`UseNative` 选择
  purego `ktav_cabi` 路径，它仍是默认值。两个方向都没有自动回退。
- Pure 模式不加载共享库、不下载任何内容、不启动子进程，因此在
  没有 `ktav_cabi` 制品的环境中也可用。
- 按实例绑定后端：`ktav.New(ktav.Backend)` 返回一个 `Operations` 值，
  其 `Loads`、`LoadsStrict`、`LoadsInto`、`Dumps`、`DumpsForceStrings`、
  `EmitCanonical`、`FormatSource` 与 `CanonicalFromSource` 方法均绑定
  到该后端，不触碰进程级全局默认值——可安全并发使用。包级函数保留
  全局默认（原生）行为。
- 原生后端与内置 Rust crate 升级至 `ktav 0.8.1`（spec 0.8.1）。
- 原生加载器现在会校验已加载的 `ktav_cabi` 制品：加载后调用
  `ktav_version`，任何不匹配都会以 "ktav_cabi version X does not match
  expected 0.8.1" 失败——包括 `$KTAV_LIB_PATH` 指向版本不一致构建的
  情况（覆盖路径仍然可用，只是与其他制品一样接受校验）。
- Pure 模式的已知限制：解码值保持现有 `map[string]any` 契约（解码
  映射的键顺序无法表示）；对普通 Go 映射，`EmitCanonical` 保持
  文档规定的字母序，而 `CanonicalFromSource` 与 `FormatSource` 保留
  插入顺序；内联复合结构嵌套上限为 128 层（§5.8.4，实现定义，与
  参考实现一致）。

