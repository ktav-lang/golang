>>>>> lang=en
### Build

```bash
# 1. Build the native library for your host platform.
cargo build --release -p ktav-cabi

# 2. Point Go at it.
export KTAV_LIB_PATH="$PWD/target/release/libktav_cabi.so"   # Linux
#      ="$PWD/target/release/libktav_cabi.dylib"             # macOS
#      ="$PWD/target/release/ktav_cabi.dll"                  # Windows

# 3. For conformance tests, point at the spec submodule.
git submodule update --init
export KTAV_SPEC_ROOT="$PWD/spec/versions/0.5/tests"
```

>>>>> lang=ru
### Сборка

```bash
# 1. Собрать нативную либу под хост-платформу.
cargo build --release -p ktav-cabi

# 2. Направить Go на неё.
export KTAV_LIB_PATH="$PWD/target/release/libktav_cabi.so"   # Linux
#      ="$PWD/target/release/libktav_cabi.dylib"             # macOS
#      ="$PWD/target/release/ktav_cabi.dll"                  # Windows

# 3. Для conformance-тестов указать путь к spec-submodule.
git submodule update --init
export KTAV_SPEC_ROOT="$PWD/spec/versions/0.5/tests"
```

>>>>> lang=zh
### 构建

```bash
# 1. 为本机平台构建原生库。
cargo build --release -p ktav-cabi

# 2. 指向它。
export KTAV_LIB_PATH="$PWD/target/release/libktav_cabi.so"   # Linux
#      ="$PWD/target/release/libktav_cabi.dylib"             # macOS
#      ="$PWD/target/release/ktav_cabi.dll"                  # Windows

# 3. 运行 conformance 测试时指向 spec submodule。
git submodule update --init
export KTAV_SPEC_ROOT="$PWD/spec/versions/0.5/tests"
```

