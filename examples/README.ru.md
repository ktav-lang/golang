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
| `basic/`  | Round-trip `Loads` + `Dumps` с демо типовых маркеров.    |
