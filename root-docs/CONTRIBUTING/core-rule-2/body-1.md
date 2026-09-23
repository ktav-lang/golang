>>>>> lang=en
### 2. Don't reinvent the format in the bindings

This Go package is deliberately a thin wrapper. Parser and format
behaviour belong in the Rust crate
([`ktav-lang/rust`](https://github.com/ktav-lang/rust)) — changing it
there updates every language binding at once. Only **Go-specific
ergonomics** (type mapping, purego loader, cache / download logic)
belong in this repo.

If your change requires a format change, start a discussion in
[`ktav-lang/spec`](https://github.com/ktav-lang/spec) first.

>>>>> lang=ru
### 2. Не переизобретай формат в биндингах

Этот Go-пакет намеренно тонкая обёртка. Поведение парсера и формата —
в Rust-крейте ([`ktav-lang/rust`](https://github.com/ktav-lang/rust));
правки там автоматически обновляют все языковые биндинги. В этом репо
только **Go-специфичная эргономика**: type mapping, purego-лоадер,
cache / download логика.

Если изменение требует правки формата — сначала обсуждение в
[`ktav-lang/spec`](https://github.com/ktav-lang/spec).

>>>>> lang=zh
### 2. 不要在绑定中重新发明格式

这个 Go 包刻意是薄封装。解析器与格式行为属于 Rust crate
（[`ktav-lang/rust`](https://github.com/ktav-lang/rust)）—— 在那里改动
会同步更新所有语言绑定。本仓库只接纳**Go 专有的人体工学**：类型映射、
purego 加载器、缓存 / 下载逻辑。

若改动需要调整格式，先在
[`ktav-lang/spec`](https://github.com/ktav-lang/spec) 发起讨论。

