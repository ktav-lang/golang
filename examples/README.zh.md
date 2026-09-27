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
