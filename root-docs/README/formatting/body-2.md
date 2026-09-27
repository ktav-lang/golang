>>>>> lang=en
```go
func formatExample() error {
	out, err := ktav.FormatSource("## why\na:   {x: 1}\n")
	if err != nil {
		return err
	}
	fmt.Print(out)
	return nil
}
```

Blank lines are grouping hints: a run of two or more becomes one, and
blank padding immediately inside a bracket is dropped.

```go
func formatBlankLinesExample() error {
	out, err := ktav.FormatSource("## keep me\n\n\nport: 8080\n")
	if err != nil {
		return err
	}
	fmt.Print(out)
	return nil
}
```

>>>>> lang=ru
```go
func formatExample() error {
	out, err := ktav.FormatSource("## why\na:   {x: 1}\n")
	if err != nil {
		return err
	}
	fmt.Print(out)
	return nil
}
```

Пустые строки служат подсказками группировки: серия из двух и более
сокращается до одной, а пустая отбивка непосредственно внутри скобок
удаляется.

```go
func formatBlankLinesExample() error {
	out, err := ktav.FormatSource("## keep me\n\n\nport: 8080\n")
	if err != nil {
		return err
	}
	fmt.Print(out)
	return nil
}
```

>>>>> lang=zh
```go
func formatExample() error {
	out, err := ktav.FormatSource("## why\na:   {x: 1}\n")
	if err != nil {
		return err
	}
	fmt.Print(out)
	return nil
}
```

空行用于提示分组：连续两行及以上会缩减为一行，紧贴括号内侧的空白
填充会被删除。

```go
func formatBlankLinesExample() error {
	out, err := ktav.FormatSource("## keep me\n\n\nport: 8080\n")
	if err != nil {
		return err
	}
	fmt.Print(out)
	return nil
}
```

