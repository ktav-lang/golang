>>>>> lang=en
## Formatting — canonical spelling, comments kept

`FormatSource` and `EmitCanonical` are different operations and it is
worth being clear which you want:

- **`EmitCanonical`** takes a Go value and writes the canonical form.
  Trivia does not exist in a value, so none survives.
- **`FormatSource`** takes source *text* and rewrites its spelling while
  **preserving every comment verbatim** (spec § 3.4: a comment owns a
  whole line). Key order is never changed.

```go
out, err := ktav.FormatSource("## why\na:   {x: 1}\n")
// "## why\na: {\n    x: 1\n}\n"
// the comment survives; the inline compound is expanded to canonical
// multi-line form and re-indented
```

Blank lines survive as a grouping hint, but a run of two or more
collapses to exactly one, and blank padding immediately inside a bracket
is dropped:

```go
ktav.FormatSource("## keep me\n\n\nport: 8080\n")
// "## keep me\n\nport: 8080\n" — two blank lines became one
```

That makes formatting a fixed point:

```go
ktav.FormatSource(ktav.FormatSource(x)) == ktav.FormatSource(x)
```

For a document with no comments and no blank lines, the result equals
`CanonicalFromSource` of the same text.

>>>>> lang=ru
## Форматирование — каноническое написание с сохранением комментариев

`FormatSource` и `EmitCanonical` — разные операции, и стоит понимать,
какая нужна:

- **`EmitCanonical`** принимает Go-значение и пишет каноническую форму.
  В значении комментариев не существует, поэтому сохранять нечего.
- **`FormatSource`** принимает исходный *текст* и переписывает его
  написание, **сохраняя каждый комментарий дословно** (spec § 3.4:
  комментарий занимает строку целиком). Порядок ключей не меняется.

```go
out, err := ktav.FormatSource("## why\na:   {x: 1}\n")
// "## why\na: {\n    x: 1\n}\n"
// the comment survives; the inline compound is expanded to canonical
// multi-line form and re-indented
```

Пустые строки сохраняются как подсказка группировки, но серия из двух и
более схлопывается ровно в одну, а пустые строки непосредственно внутри
скобки отбрасываются:

```go
ktav.FormatSource("## keep me\n\n\nport: 8080\n")
// "## keep me\n\nport: 8080\n" — two blank lines became one
```

Поэтому форматирование — неподвижная точка:

```go
ktav.FormatSource(ktav.FormatSource(x)) == ktav.FormatSource(x)
```

Для документа без комментариев и без пустых строк результат совпадает с
`CanonicalFromSource` того же текста.

>>>>> lang=zh
## 格式化 —— 规范写法，保留注释

`FormatSource` 与 `EmitCanonical` 是两种不同的操作,值得分清需要哪一个:

- **`EmitCanonical`** 接受 Go 值并写出规范形式。值里不存在注释,
  因此没有任何注释可以保留。
- **`FormatSource`** 接受源**文本**,重写其写法,同时**逐字保留每条
  注释**(spec § 3.4:注释独占一整行)。键顺序绝不改变。

```go
out, err := ktav.FormatSource("## why\na:   {x: 1}\n")
// "## why\na: {\n    x: 1\n}\n"
// the comment survives; the inline compound is expanded to canonical
// multi-line form and re-indented
```

空行作为分组提示保留下来,但连续两行及以上会合并为恰好一行,紧贴括号
内侧的空行填充会被丢弃:

```go
ktav.FormatSource("## keep me\n\n\nport: 8080\n")
// "## keep me\n\nport: 8080\n" — two blank lines became one
```

因此格式化是一个不动点:

```go
ktav.FormatSource(ktav.FormatSource(x)) == ktav.FormatSource(x)
```

对于没有注释也没有空行的文档,结果与同一文本的 `CanonicalFromSource`
相同。

