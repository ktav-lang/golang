>>>>> lang=en
### Build & render — construct a document in code

```go
doc := map[string]any{
    "name":  "frontend",
    "port":  int64(8443),
    "tls":   true,
    "ratio": 0.95,
    "upstreams": []any{
        map[string]any{"host": "a.example", "port": int64(1080)},
        map[string]any{"host": "b.example", "port": int64(1080)},
    },
    "notes": nil,
}
out, _ := ktav.Dumps(doc)
fmt.Print(out)
// name: frontend
// notes: null
// port: 8443
// ratio: 0.95
// tls: true
// upstreams: [
//     {
//         host: a.example
//         port: 1080
//     }
//     {
//         host: b.example
//         port: 1080
//     }
// ]
```

`encoding/json` sorts Go map keys, so this output order is deterministic;
the `map[string]any` itself does not preserve insertion order.

A complete runnable version lives in [`examples/basic`](examples/basic/main.go).

>>>>> lang=ru
### Билд + рендер — собираем документ в коде

```go
doc := map[string]any{
    "name":  "frontend",
    "port":  int64(8443),
    "tls":   true,
    "ratio": 0.95,
    "upstreams": []any{
        map[string]any{"host": "a.example", "port": int64(1080)},
        map[string]any{"host": "b.example", "port": int64(1080)},
    },
    "notes": nil,
}
out, _ := ktav.Dumps(doc)
fmt.Print(out)
// name: frontend
// notes: null
// port: 8443
// ratio: 0.95
// tls: true
// upstreams: [
//     {
//         host: a.example
//         port: 1080
//     }
//     {
//         host: b.example
//         port: 1080
//     }
// ]
```

`encoding/json` сортирует ключи Go-map, поэтому порядок вывода
детерминирован; сам `map[string]any` не сохраняет порядок вставки.

Полный запускаемый пример — в [`examples/basic`](../examples/basic/main.go).

>>>>> lang=zh
### 构建并渲染 —— 用代码搭建文档

```go
doc := map[string]any{
    "name":  "frontend",
    "port":  int64(8443),
    "tls":   true,
    "ratio": 0.95,
    "upstreams": []any{
        map[string]any{"host": "a.example", "port": int64(1080)},
        map[string]any{"host": "b.example", "port": int64(1080)},
    },
    "notes": nil,
}
out, _ := ktav.Dumps(doc)
fmt.Print(out)
// name: frontend
// notes: null
// port: 8443
// ratio: 0.95
// tls: true
// upstreams: [
//     {
//         host: a.example
//         port: 1080
//     }
//     {
//         host: b.example
//         port: 1080
//     }
// ]
```

`encoding/json` 会对 Go map 的键排序，因此输出顺序是确定的；
`map[string]any` 本身不保留插入顺序。

完整可运行示例:[`examples/basic`](../examples/basic/main.go)。

