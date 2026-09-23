>>>>> lang=en
### Public API

- `Loads(s string) (any, error)` — parse a Ktav document.
- `LoadsInto(s string, target any) error` — parse into `target` via
  `encoding/json`.
- `Dumps(v any) (string, error)` — render a Go value as Ktav text.
- `Error` — typed parse / render error surfaced through `errors.As`.

>>>>> lang=ru
### Путь модуля

Опубликован как `github.com/ktav-lang/golang`. `go get` подхватит через
Go-прокси автоматически после появления тега `v0.1.0`.

### Публичный API

- `Loads(s string) (any, error)` — разобрать документ Ktav.
- `LoadsInto(s string, target any) error` — разобрать в `target` через
  `encoding/json`.
- `Dumps(v any) (string, error)` — сериализовать Go-значение в Ktav.
- `Error` — типизированная ошибка парсинга/рендера, ловится через
  `errors.As`.

>>>>> lang=zh
### 模块路径

以 `github.com/ktav-lang/golang` 发布。`v0.1.0` git tag 推送后
Go proxy 会自动索引。

### 公开 API

- `Loads(s string) (any, error)` —— 解析 Ktav 文档。
- `LoadsInto(s string, target any) error` —— 通过 `encoding/json` 解析到
  `target`。
- `Dumps(v any) (string, error)` —— 将 Go 值渲染为 Ktav 文本。
- `Error` —— 类型化的 parse / render 错误,可通过 `errors.As` 捕获。

