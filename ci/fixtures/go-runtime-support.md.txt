>>>>> lang=en
## Runtime support

- Go 1.21+.
- Prebuilt binaries for: `linux/amd64`, `linux/arm64`, `darwin/amd64`,
  `darwin/arm64`, `windows/amd64`, `windows/arm64`.
- Linux distros must use glibc 2.17+ (Rust's default target). Alpine
  (musl) support is planned.

The prepared 0.8.1 Windows native distribution uses GNU/LLVM targets
`x86_64-pc-windows-gnullvm` and `aarch64-pc-windows-gnullvm`.
The existing `windows-amd64` and `windows-arm64` asset names remain
compatibility identifiers, not MSVC compiler claims. These DLLs require
neither Rust/LLVM build tools nor `VCRUNTIME140.dll` at runtime; they use
Windows system APIs and the system UCRT. Go dependencies are unchanged; consumers need neither Rust nor cgo.
Complete original compiler/runtime copyright and licence texts are embedded
in the DLL and retained in the adjacent `.NOTICE.json`; distribute the
notice and `.NOTICE.manifest.json` sidecars with the DLL. This describes
the prepared candidate, not a repair of previously published binaries.

>>>>> lang=ru
## Поддержка рантаймов

- Go 1.21+.
- Прекомпилированные бинари: `linux/amd64`, `linux/arm64`,
  `darwin/amd64`, `darwin/arm64`, `windows/amd64`, `windows/arm64`.
- Linux-дистрибутивы должны использовать glibc 2.17+ (дефолтный
  Rust-таргет). Alpine (musl) — в планах.

Подготовленная Windows native distribution 0.8.1 использует GNU/LLVM targets
`x86_64-pc-windows-gnullvm` и `aarch64-pc-windows-gnullvm`.
Прежние имена assets `windows-amd64` и `windows-arm64` сохраняются как
идентификаторы совместимости, а не как указание на компилятор MSVC.
Во время работы этим DLL не нужны инструменты сборки Rust/LLVM или
`VCRUNTIME140.dll`: используются системные Windows API и UCRT.
Зависимости Go не изменены; потребителям не нужны Rust или cgo. Полные исходные copyright и лицензионные
тексты compiler/runtime встроены в DLL и сохранены в соседнем
`.NOTICE.json`; распространяйте notice и `.NOTICE.manifest.json` вместе
с DLL. Это описание подготовленного candidate, а не исправление ранее
опубликованных бинарников.

>>>>> lang=zh
## 运行时支持

- Go 1.21+。
- 预编译二进制: `linux/amd64`、`linux/arm64`、`darwin/amd64`、
  `darwin/arm64`、`windows/amd64`、`windows/arm64`。
- Linux 发行版需 glibc 2.17+（Rust 默认 target）。Alpine（musl）支持
  在计划中。

准备中的 0.8.1 Windows 原生发行包使用 GNU/LLVM 目标
`x86_64-pc-windows-gnullvm` 和 `aarch64-pc-windows-gnullvm`。
现有的 `windows-amd64`、`windows-arm64` 文件名保留为兼容性标识，
并不表示使用 MSVC 编译。这些 DLL 在运行时不需要 Rust/LLVM 构建工具
或 `VCRUNTIME140.dll`，而是使用 Windows 系统 API 和系统 UCRT。
Go 依赖保持不变，使用者无需 Rust 或 cgo。完整的编译器及运行时原始版权和许可证文本嵌入 DLL，
并保留在相邻的 `.NOTICE.json` 中；分发 DLL 时也应保留 notice 和
`.NOTICE.manifest.json`。此说明仅适用于准备中的候选发行包，
不表示已修复先前发布的二进制文件。

