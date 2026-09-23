>>>>> lang=en
### Test

```bash
go test -v ./...                # full suite
go test -run TestSmoke ./...    # filter by name
go test -run Conformance ./...  # spec fixtures only
```

When either `KTAV_LIB_PATH` or `KTAV_SPEC_ROOT` is unset, the relevant
tests **skip** rather than fail — so `go test` in a bare checkout stays
green.

>>>>> lang=ru
### Тесты

```bash
go test -v ./...                # полный прогон
go test -run TestSmoke ./...    # фильтр по имени
go test -run Conformance ./...  # только spec-фикстуры
```

Если `KTAV_LIB_PATH` или `KTAV_SPEC_ROOT` не заданы — соответствующие
тесты **скипаются**, а не падают, чтобы `go test` на голой копии
оставался зелёным.

>>>>> lang=zh
### 测试

```bash
go test -v ./...                # 完整套件
go test -run TestSmoke ./...    # 按名称过滤
go test -run Conformance ./...  # 只跑 spec fixtures
```

当 `KTAV_LIB_PATH` 或 `KTAV_SPEC_ROOT` 未设置时，相关测试会**跳过**
而不是失败 —— 纯净 checkout 下 `go test` 也保持绿色。

