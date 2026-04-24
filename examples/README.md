# Examples

Minimal programs that exercise the public Go API. Run any of them with
the Rust-built native library pointed to via `KTAV_LIB_PATH` (see
[CONTRIBUTING](../CONTRIBUTING.md#build)):

```bash
cargo build --release -p ktav-cabi
export KTAV_LIB_PATH="$PWD/target/release/libktav_cabi.so"  # adjust for OS

go run ./examples/basic
```

| Directory | Shows                                                    |
| --------- | -------------------------------------------------------- |
| `basic/`  | `Loads` + `Dumps` round-trip with the type-marker demos. |
