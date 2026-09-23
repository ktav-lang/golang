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
// port: 8443
// tls: true
// ratio: 0.95
// upstreams: [
//     { host: a.example  port: 1080 }
//     { host: b.example  port: 1080 }
// ]
// notes: null
```

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
// port: 8443
// tls: true
// ratio: 0.95
// upstreams: [
//     { host: a.example  port: 1080 }
//     { host: b.example  port: 1080 }
// ]
// notes: null
```

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
// port: 8443
// tls: true
// ratio: 0.95
// upstreams: [
//     { host: a.example  port: 1080 }
//     { host: b.example  port: 1080 }
// ]
// notes: null
```

完整可运行示例:[`examples/basic`](../examples/basic/main.go)。

