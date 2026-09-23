>>>>> lang=en
### Changed

- Tracks `ktav 0.7.1` and spec **0.7.0** (spec submodule pinned to
  `v0.7.0`); the dependency floor is raised from `0.7` and the
  binding's own version is unchanged. The Go API is otherwise
  unchanged — quoted key segments (spec 0.7 § 5.3.3) and `\uXXXX`
  escapes (§ 3.7.1) arrive through the Rust core, so the new 0.7 error
  kinds (§§ 6.11–6.16) need no new Go-side surface.
- Migrated `crates/cabi` to a single `ktav::declare_cabi!()` invocation
  (ktav's `cabi` feature) instead of a hand-rolled C ABI shim; the
  exported symbol surface (`ktav_loads`, `ktav_loads_strict`,
  `ktav_dumps`, `ktav_dumps_force_strings`, `ktav_emit_canonical`,
  `ktav_format`, `ktav_canonical_from_source`, `ktav_free`,
  `ktav_version`, `ktav_abi_version`) is unchanged, so the Go API is
  unaffected. Dependency floor raised to `ktav 0.8`, spec submodule
  re-pinned to `v0.8.0` (adds § 5.2: a decimal with a redundant leading
  zero parses as a String, not an Integer).
- The module version moves to **0.8.0**, in step with the core and the
  specification; the prebuilt-library download fallback now targets the
  `v0.8.0` release asset.

>>>>> lang=ru
### Изменено

- Отслеживает `ktav 0.7.1` и spec **0.7.0** (сабмодуль spec закреплён на
  `v0.7.0`); нижняя граница зависимости поднята с `0.7`, собственная
  версия биндинга не менялась. В остальном Go API прежний —
  квотированные сегменты ключей (spec 0.7 § 5.3.3) и escape-формы
  `\uXXXX` (§ 3.7.1) приходят через Rust-ядро, поэтому новые классы
  ошибок 0.7 (§§ 6.11–6.16) не требуют новой поверхности со стороны Go.
- `crates/cabi` переведён на один вызов `ktav::declare_cabi!()` (фича
  `cabi` крейта `ktav`) вместо рукописной C ABI-прослойки; набор
  экспортируемых символов (`ktav_loads`, `ktav_loads_strict`,
  `ktav_dumps`, `ktav_dumps_force_strings`, `ktav_emit_canonical`,
  `ktav_format`, `ktav_canonical_from_source`, `ktav_free`,
  `ktav_version`, `ktav_abi_version`) не изменился, поэтому Go API не
  затронут. Нижняя граница зависимости поднята до `ktav 0.8`, сабмодуль
  spec перезакреплён на `v0.8.0` (добавлен § 5.2: десятичное число с
  избыточным ведущим нулём разбирается как String, а не Integer).
- Версия модуля — **0.8.0**, синхронно с ядром и спецификацией; резервная
  загрузка предсобранной библиотеки теперь нацелена на ассет релиза
  `v0.8.0`.

>>>>> lang=zh
### 变更

- 跟随 `ktav 0.7.1` 与 spec **0.7.0**(spec 子模块固定在 `v0.7.0`);
  依赖下限从 `0.7` 提高,绑定自身的版本未变。Go API 其余部分不变 ——
  quoted 键段(spec 0.7 § 5.3.3)与 `\uXXXX` escape(§ 3.7.1)经由
  Rust 核心到达,因此 0.7 新增的错误类别(§§ 6.11–6.16)不需要新的
  Go 侧接口。
- `crates/cabi` 改为单次调用 `ktav::declare_cabi!()`(ktav 的 `cabi`
  特性),取代手写的 C ABI 垫片;导出的符号集(`ktav_loads`、
  `ktav_loads_strict`、`ktav_dumps`、`ktav_dumps_force_strings`、
  `ktav_emit_canonical`、`ktav_format`、`ktav_canonical_from_source`、
  `ktav_free`、`ktav_version`、`ktav_abi_version`)不变,因此绑定 API
  不受影响。依赖下限提升至 `ktav 0.8`,spec 子模块重新固定到 `v0.8.0`
  (新增 § 5.2:带有多余前导零的十进制数解析为 String,而非 Integer)。
- 包版本升至 **0.8.0**,与核心和规范同步;预编译库的回退下载现在指向
  `v0.8.0` 发布资产。

