>>>>> lang=en
#### 2. Literal dots: textual path or segments

The quoted source key is literally `a.b`. The textual path escapes its dot; the segments call passes the decoded key directly. Both calls return the same result.

```go
source := "\"a.b\": {port: 80}\n"
updated, err := pure.EditValue(source, "a\\.b.port", int64(90))
if err != nil {
    panic(err)
}
same, err := pure.EditValueSegments(source, []string{"a.b", "port"}, int64(90))
if err != nil {
    panic(err)
}
if updated != "\"a.b\": {port: 90}\n" { panic("unexpected edit") }
if same != updated { panic("path APIs differ") }
```

Result:

```ktav
"a.b": {port: 90}
```

#### 3. A leaf inside a mixed-form array

>>>>> lang=ru
#### 2. Буквальные точки: текстовый путь или сегменты

Ключ исходника в кавычках буквально называется `a.b`. В текстовом пути точка экранируется; API сегментов получает декодированное имя напрямую. Оба вызова возвращают одинаковый результат.

```go
source := "\"a.b\": {port: 80}\n"
updated, err := pure.EditValue(source, "a\\.b.port", int64(90))
if err != nil {
    panic(err)
}
same, err := pure.EditValueSegments(source, []string{"a.b", "port"}, int64(90))
if err != nil {
    panic(err)
}
if updated != "\"a.b\": {port: 90}\n" { panic("unexpected edit") }
if same != updated { panic("path APIs differ") }
```

Результат:

```ktav
"a.b": {port: 90}
```

#### 3. Лист внутри массива со смешанными формами

>>>>> lang=zh
#### 2. 字面点号：文本路径或路径段

源码中带引号的键名就是 `a.b`。文本路径需要转义点号；路径段 API 直接接收解码后的键名。两个调用返回相同结果。

```go
source := "\"a.b\": {port: 80}\n"
updated, err := pure.EditValue(source, "a\\.b.port", int64(90))
if err != nil {
    panic(err)
}
same, err := pure.EditValueSegments(source, []string{"a.b", "port"}, int64(90))
if err != nil {
    panic(err)
}
if updated != "\"a.b\": {port: 90}\n" { panic("unexpected edit") }
if same != updated { panic("path APIs differ") }
```

结果:

```ktav
"a.b": {port: 90}
```

#### 3. 修改混合形式数组中的叶子值

