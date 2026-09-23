>>>>> lang=en
### Lint

```bash
go vet -unsafeptr=false ./...
gofmt -l .                            # should print nothing
cargo fmt --all --check
cargo clippy --release -p ktav-cabi -- -D warnings
```

CI runs the same commands; run them locally before pushing.

>>>>> lang=ru
### Линт

```bash
go vet -unsafeptr=false ./...
gofmt -l .                            # должен вернуть пустоту
cargo fmt --all --check
cargo clippy --release -p ktav-cabi -- -D warnings
```

CI гоняет то же; прогоняй локально перед push.

>>>>> lang=zh
### Lint

```bash
go vet -unsafeptr=false ./...
gofmt -l .                            # 应当无输出
cargo fmt --all --check
cargo clippy --release -p ktav-cabi -- -D warnings
```

CI 跑同样的命令；push 前本地先过一遍。

