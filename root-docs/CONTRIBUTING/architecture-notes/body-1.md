>>>>> lang=en
## Architecture notes

- **Wire format.** Rust and Go exchange JSON over the FFI boundary,
  with `{"$i":"..."}` / `{"$f":"..."}` wrappers for integers / floats
  (spec 0.8: inferred from scalar form, no `:i`/`:f` markers in the
  text). The wire wrappers preserve Go integer precision; Ktav integer
  scalars outside `int64` parse as Strings.
- **Memory ownership.** Rust allocates the output buffer; Go copies it
  into a Go slice and immediately calls `ktav_free` on the Rust side.
  No buffer is long-lived across the FFI boundary.
- **Loader.** `internal/native` dlopens the shared library once per
  process (sync.Once). On Windows we go through
  `golang.org/x/sys/windows`; on Unix through `purego.Dlopen/Dlsym`.

>>>>> lang=ru
## Заметки об архитектуре

- **Wire-формат.** Rust и Go обмениваются JSON через FFI-границу с
  обёртками `{"$i":"..."}` / `{"$f":"..."}` для integer / float
  (spec 0.8: выводится из лексической формы, маркеров `:i`/`:f` в
  тексте больше нет). Wire-обёртки сохраняют точность Go integer, но
  Ktav integer scalar за пределами `int64` разбирается как String.
- **Владение памятью.** Rust аллоцирует выходной буфер; Go копирует в
  slice и сразу дёргает `ktav_free` на Rust-стороне. Сквозь
  FFI-границу не живёт ни один долгий буфер.
- **Лоадер.** `internal/native` один раз на процесс dlopen-ит
  библиотеку (`sync.Once`). На Windows — через
  `golang.org/x/sys/windows`; на Unix — через `purego.Dlopen/Dlsym`.

>>>>> lang=zh
## 架构说明

- **Wire 格式。** Rust 与 Go 在 FFI 边界用 JSON 交换，用
  `{"$i":"..."}` / `{"$f":"..."}` 包装 integer / float（spec 0.8：
  从标量词法形式推断，文本中不再有 `:i`/`:f` 标记）。Wire 包装保留
  Go integer 精度；超出 `int64` 范围的 Ktav integer scalar 会解析为 String。
- **内存所有权。** Rust 分配输出 buffer；Go 拷贝到 slice 后立刻回调
  `ktav_free`。跨 FFI 边界无长期共享内存。
- **加载器。** `internal/native` 通过 `sync.Once` 在每个进程 dlopen
  一次。Windows 走 `golang.org/x/sys/windows`；Unix 走
  `purego.Dlopen/Dlsym`。

