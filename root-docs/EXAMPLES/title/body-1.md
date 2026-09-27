>>>>> lang=en
# Examples

**Languages:** **English** · [Русский](README.ru.md) · [简体中文](README.zh.md)

Minimal programs that exercise the public Go API. Run any of them with
the Rust-built native library pointed to via `KTAV_LIB_PATH` (see
[CONTRIBUTING](../docs/CONTRIBUTING.md#build)):

```bash
cargo build --release -p ktav-cabi
export KTAV_LIB_PATH="$PWD/target/release/libktav_cabi.so"  # adjust for OS

go run ./examples/basic
```

| Directory | Shows                                                    |
| --------- | -------------------------------------------------------- |
| `basic/`  | `Loads` + `Dumps` round-trip with inferred scalar types. |
>>>>> lang=ru
# Примеры

**Языки:** [English](README.md) · **Русский** · [简体中文](README.zh.md)

Минимальные программы, которые задействуют публичный Go-API. Любую из
них можно запустить, указав через `KTAV_LIB_PATH` нативную библиотеку,
собранную на Rust (см. [CONTRIBUTING](../docs/CONTRIBUTING.md#build)):

```bash
cargo build --release -p ktav-cabi
export KTAV_LIB_PATH="$PWD/target/release/libktav_cabi.so"  # adjust for OS

go run ./examples/basic
```

| Каталог   | Что показывает                                           |
| --------- | -------------------------------------------------------- |
| `basic/`  | Round-trip `Loads` + `Dumps` с выводом типов скаляров.   |
>>>>> lang=zh
# 示例

**语言:** [English](README.md) · [Русский](README.ru.md) · **简体中文**

一些最小程序，用于演练公共 Go API。通过 `KTAV_LIB_PATH` 指向以 Rust 构建的
原生库，即可运行其中任意一个（见 [CONTRIBUTING](../docs/CONTRIBUTING.md#build)）:

```bash
cargo build --release -p ktav-cabi
export KTAV_LIB_PATH="$PWD/target/release/libktav_cabi.so"  # adjust for OS

go run ./examples/basic
```

| 目录        | 演示内容                                                     |
| --------- | -------------------------------------------------------- |
| `basic/`  | `Loads` + `Dumps` 往返转换，演示标量类型推断。                           |
