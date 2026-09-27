>>>>> lang=en
Formatting reaches a fixed point. Handle each `(string, error)` result
before passing the next string to the formatter:

```go
func isFixedPoint(src string) (bool, error) {
	once, err := ktav.FormatSource(src)
	if err != nil {
		return false, err
	}
	twice, err := ktav.FormatSource(once)
	if err != nil {
		return false, err
	}
	return once == twice, nil
}
```

Without comments or blank lines, the result equals `CanonicalFromSource`
for the same text.

>>>>> lang=ru
Форматирование достигает неподвижной точки. Обработайте результат
`(string, error)`, прежде чем передавать строку форматтеру снова:

```go
func isFixedPoint(src string) (bool, error) {
	once, err := ktav.FormatSource(src)
	if err != nil {
		return false, err
	}
	twice, err := ktav.FormatSource(once)
	if err != nil {
		return false, err
	}
	return once == twice, nil
}
```

Без комментариев и пустых строк результат совпадает с
`CanonicalFromSource` того же текста.

>>>>> lang=zh
格式化会达到不动点。再次传入格式化器之前，先处理 `(string, error)`
返回值：

```go
func isFixedPoint(src string) (bool, error) {
	once, err := ktav.FormatSource(src)
	if err != nil {
		return false, err
	}
	twice, err := ktav.FormatSource(once)
	if err != nil {
		return false, err
	}
	return once == twice, nil
}
```

没有注释和空行时，结果与同一文本的 `CanonicalFromSource` 相同。

