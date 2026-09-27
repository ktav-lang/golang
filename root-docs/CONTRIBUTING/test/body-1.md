>>>>> lang=en
### Test

```bash
go test -v ./...                # full suite
go test -run TestSmoke ./...    # filter by name
go test -run Conformance ./...  # spec fixtures only
```

Before any tests run, `TestMain` validates the corpus at
`spec/versions/0.8/tests`; missing files, companions, schema, or count
errors fail immediately. There is no `KTAV_SPEC_ROOT` override. The native library defaults to
`target/release` (or `$CARGO_TARGET_DIR/release`); `KTAV_LIB_PATH` is an
optional path override. Tests requiring the C ABI fail if the library is
missing; they do not skip.

>>>>> lang=ru
### Тесты

```bash
go test -v ./...                # полный прогон
go test -run TestSmoke ./...    # фильтр по имени
go test -run Conformance ./...  # только spec-фикстуры
```

До запуска тестов `TestMain` проверяет корпус в
`spec/versions/0.8/tests`; отсутствие файлов или companions, ошибки
схемы или количества сразу приводят к ошибке. Переопределения
`KTAV_SPEC_ROOT` нет. По умолчанию
нативная библиотека ищется в `target/release` (или
`$CARGO_TARGET_DIR/release`); `KTAV_LIB_PATH` позволяет переопределить
путь. Тесты, которым нужен C ABI, завершаются ошибкой, если библиотеки
нет, а не пропускаются.

>>>>> lang=zh
### 测试

```bash
go test -v ./...                # 完整套件
go test -run TestSmoke ./...    # 按名称过滤
go test -run Conformance ./...  # 只跑 spec fixtures
```

运行任何测试前，`TestMain` 都会检查
`spec/versions/0.8/tests` 中的语料；文件、配套文件缺失或 schema、
数量错误都会立即失败。没有 `KTAV_SPEC_ROOT` 覆盖项。原生库默认从
`target/release` 加载
（或从 `$CARGO_TARGET_DIR/release` 加载）；`KTAV_LIB_PATH` 可选地覆盖
库路径。需要 C ABI 的测试在库缺失时会失败，不会跳过。

