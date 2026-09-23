>>>>> lang=en
### Architecture

- **Native core** — the reference Rust `ktav` crate, wrapped with a tiny
  `extern "C"` C ABI (`crates/cabi`) and distributed as a prebuilt
  `.so` / `.dylib` / `.dll`.
- **Go loader** — `purego` (no cgo): the library is resolved at first
  call from `$KTAV_LIB_PATH` or downloaded once into `UserCacheDir`
  from the matching GitHub Release asset.
- **Wire format** — JSON between Rust and Go, with `{"$i":"..."}` /
  `{"$f":"..."}` tagged wrappers for lossless typed-integer / typed-float
  round-trips and arbitrary-precision integers (`*big.Int`).

>>>>> lang=ru
### Архитектура

- **Нативное ядро** — референсный Rust-крейт `ktav`, обёрнутый тонким
  `extern "C"` C ABI (`crates/cabi`) и распространяемый как
  прекомпилированный `.so` / `.dylib` / `.dll`.
- **Go-лоадер** — `purego` (без cgo): библиотека резолвится на первый
  вызов из `$KTAV_LIB_PATH` или скачивается один раз в `UserCacheDir` из
  соответствующего GitHub Release asset.
- **Wire-формат** — JSON между Rust и Go с тегированными обёртками
  `{"$i":"..."}` / `{"$f":"..."}` для lossless round-trip типизированных
  integer / float и произвольной точности (`*big.Int`).

>>>>> lang=zh
### 架构

- **原生核心** —— 参考 Rust `ktav` crate,通过极小的 `extern "C"` C ABI
  (`crates/cabi`) 封装,以预编译的 `.so` / `.dylib` / `.dll` 分发。
- **Go 加载器** —— `purego`(无 cgo):库在首次调用时从 `$KTAV_LIB_PATH`
  解析,或一次性从匹配的 GitHub Release asset 下载到 `UserCacheDir`。
- **Wire 格式** —— Rust 和 Go 之间用 JSON,用 `{"$i":"..."}` /
  `{"$f":"..."}` 标签包装器保证类型化 integer / float 的无损 round-trip
  和任意精度(`*big.Int`)。

