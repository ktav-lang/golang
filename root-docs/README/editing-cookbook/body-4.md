>>>>> lang=en
Result:

```ktav
port:: 9090
```

#### 5. Multiline replacement: expand only what is necessary

The changed array expands with four-space indentation. The untouched object stays inline, and verbatim string contents are not reindented.

```go
source := "a: [old, {keep: yes}]\n"
updated, err := pure.EditValue(source, "a.0", "x\ny")
if err != nil {
    panic(err)
}
if updated != "a: [\n    ((\nx\ny\n    ))\n    {keep: yes}\n]\n" { panic("unexpected edit") }
```

Result:

```ktav
a: [
    ((
x
y
    ))
    {keep: yes}
]
```

>>>>> lang=ru
Результат:

```ktav
port:: 9090
```

#### 5. Многострочная замена: раскрывается только необходимое

Изменённый массив раскрывается с отступом четыре пробела. Нетронутый объект остаётся inline, содержимое verbatim-строки не переотступается.

```go
source := "a: [old, {keep: yes}]\n"
updated, err := pure.EditValue(source, "a.0", "x\ny")
if err != nil {
    panic(err)
}
if updated != "a: [\n    ((\nx\ny\n    ))\n    {keep: yes}\n]\n" { panic("unexpected edit") }
```

Результат:

```ktav
a: [
    ((
x
y
    ))
    {keep: yes}
]
```

>>>>> lang=zh
结果:

```ktav
port:: 9090
```

#### 5. 多行替换：仅展开必要的容器

修改的数组以四空格缩进展开。未修改的对象保持行内形式，verbatim 字符串内容不会重新缩进。

```go
source := "a: [old, {keep: yes}]\n"
updated, err := pure.EditValue(source, "a.0", "x\ny")
if err != nil {
    panic(err)
}
if updated != "a: [\n    ((\nx\ny\n    ))\n    {keep: yes}\n]\n" { panic("unexpected edit") }
```

结果:

```ktav
a: [
    ((
x
y
    ))
    {keep: yes}
]
```

