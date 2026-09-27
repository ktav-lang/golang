>>>>> lang=en
### Build

```bash
# 1. Build the native library for your host platform.
cargo build --release -p ktav-cabi

# 2. Point Go at it.
export KTAV_LIB_PATH="$PWD/target/release/libktav_cabi.so"   # Linux
#      ="$PWD/target/release/libktav_cabi.dylib"             # macOS
#      ="$PWD/target/release/ktav_cabi.dll"                  # Windows

# 3. Initialize the pinned spec corpus used by conformance tests.
git submodule update --init
```

`TestMain` uses the fixed path `spec/versions/0.8/tests`; there is no
`KTAV_SPEC_ROOT` override. The conformance guard checks the known category
manifest and fixture structure so corpus changes cannot be silently skipped.

>>>>> lang=ru
### Сборка

```bash
# 1. Собрать нативную либу под хост-платформу.
cargo build --release -p ktav-cabi

# 2. Направить Go на неё.
export KTAV_LIB_PATH="$PWD/target/release/libktav_cabi.so"   # Linux
#      ="$PWD/target/release/libktav_cabi.dylib"             # macOS
#      ="$PWD/target/release/ktav_cabi.dll"                  # Windows

# 3. Инициализировать закреплённый корпус для conformance-тестов.
git submodule update --init
```

`TestMain` использует фиксированный путь `spec/versions/0.8/tests`;
переменной `KTAV_SPEC_ROOT` нет. Conformance guard проверяет известные
категории и структуру fixtures, чтобы изменения корпуса не пропускались молча.

>>>>> lang=zh
### 构建

```bash
# 1. 为本机平台构建原生库。
cargo build --release -p ktav-cabi

# 2. 指向它。
export KTAV_LIB_PATH="$PWD/target/release/libktav_cabi.so"   # Linux
#      ="$PWD/target/release/libktav_cabi.dylib"             # macOS
#      ="$PWD/target/release/ktav_cabi.dll"                  # Windows

# 3. 初始化 conformance 测试使用的固定版本语料。
git submodule update --init
```

`TestMain` 使用固定路径 `spec/versions/0.8/tests`，不支持
`KTAV_SPEC_ROOT` 覆盖。Conformance guard 会检查已知类别清单和 fixture
结构，避免语料变更被静默跳过。

