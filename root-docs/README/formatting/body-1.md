>>>>> lang=en
## Formatting — canonical spelling, comments kept

`FormatSource` and `EmitCanonical` are different operations:

- **`EmitCanonical`** takes a Go value and writes canonical form. Comments
  are not part of a value and cannot survive.
- **`FormatSource`** takes source text and normalizes its spelling while
  preserving every whole-line comment (spec § 3.4). Both it and
  `CanonicalFromSource` preserve source key order.

The snippets below use these imports:

```go
import (
    "fmt"

    ktav "github.com/ktav-lang/golang"
)
```

>>>>> lang=ru
## Форматирование — каноническое написание с сохранением комментариев

`FormatSource` и `EmitCanonical` — разные операции:

- **`EmitCanonical`** принимает Go-значение и пишет каноническую форму.
  В значении нет комментариев, поэтому сохранить их нельзя.
- **`FormatSource`** принимает исходный текст и нормализует написание,
  сохраняя каждый комментарий, занимающий целую строку (spec § 3.4).
  И он, и `CanonicalFromSource` сохраняют исходный порядок ключей.

Ниже приведены используемые в примерах импорты:

```go
import (
    "fmt"

    ktav "github.com/ktav-lang/golang"
)
```

>>>>> lang=zh
## 格式化 —— 规范写法，保留注释

`FormatSource` 与 `EmitCanonical` 是不同的操作：

- **`EmitCanonical`** 接受 Go 值并写出规范形式。值中没有注释，因而
  无法保留注释。
- **`FormatSource`** 接受源文本并规范其写法，同时保留每条独占整行的
  注释（spec § 3.4）。它与 `CanonicalFromSource` 都保留源键顺序。

以下示例使用这些导入：

```go
import (
    "fmt"

    ktav "github.com/ktav-lang/golang"
)
```

