>>>>> lang=en
`0` is an array index here. Inside an Object, the same segment would address the key named `0`; untouched containers remain inline.

```go
source := "servers: [{host: old, keep: yes}, standby]\n"
updated, err := pure.EditValueSegments(source, []string{"servers", "0", "host"}, "new")
if err != nil {
    panic(err)
}
if updated != "servers: [{host: new, keep: yes}, standby]\n" { panic("unexpected edit") }
```

Result:

```ktav
servers: [{host: new, keep: yes}, standby]
```

#### 4. A numeric-looking String stays a String

The replacement is a String, not an Integer. The raw marker preserves that distinction. Supply a typed Integer when a numeric value is intended.

```go
source := "port: 8080\n"
updated, err := pure.EditValue(source, "port", "9090")
if err != nil {
    panic(err)
}
if updated != "port:: 9090\n" { panic("unexpected edit") }
```

>>>>> lang=ru
Здесь `0` — индекс массива. В объекте такой же сегмент обозначал бы ключ с именем `0`; нетронутые контейнеры остаются inline.

```go
source := "servers: [{host: old, keep: yes}, standby]\n"
updated, err := pure.EditValueSegments(source, []string{"servers", "0", "host"}, "new")
if err != nil {
    panic(err)
}
if updated != "servers: [{host: new, keep: yes}, standby]\n" { panic("unexpected edit") }
```

Результат:

```ktav
servers: [{host: new, keep: yes}, standby]
```

#### 4. Похожая на число строка остаётся строкой

Замена имеет тип String, а не Integer. Raw-маркер сохраняет это различие. Для числового значения передавайте типизированный Integer.

```go
source := "port: 8080\n"
updated, err := pure.EditValue(source, "port", "9090")
if err != nil {
    panic(err)
}
if updated != "port:: 9090\n" { panic("unexpected edit") }
```

>>>>> lang=zh
此处 `0` 是数组索引；在对象中，同一个路径段表示名称为 `0` 的键。未修改的容器保持行内形式。

```go
source := "servers: [{host: old, keep: yes}, standby]\n"
updated, err := pure.EditValueSegments(source, []string{"servers", "0", "host"}, "new")
if err != nil {
    panic(err)
}
if updated != "servers: [{host: new, keep: yes}, standby]\n" { panic("unexpected edit") }
```

结果:

```ktav
servers: [{host: new, keep: yes}, standby]
```

#### 4. 看似数字的字符串仍然是字符串

替换值是 String，而不是 Integer。raw 标记保留这种区别；需要数值时请传入类型化 Integer。

```go
source := "port: 8080\n"
updated, err := pure.EditValue(source, "port", "9090")
if err != nil {
    panic(err)
}
if updated != "port:: 9090\n" { panic("unexpected edit") }
```

