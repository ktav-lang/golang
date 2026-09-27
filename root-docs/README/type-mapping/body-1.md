>>>>> lang=en
## Type mapping

| Ktav             | Go                                              |
| ---------------- | ----------------------------------------------- |
| `null`           | `nil`                                           |
| `true` / `false` | `bool`                                          |
| integer scalar   | `int64` if it fits; otherwise `string`           |
| float scalar     | `float64`                                       |
| bare scalar      | `string`                                        |
| `[ ... ]`        | `[]any`                                         |
| `{ ... }`        | `map[string]any` (Go map; no insertion-order guarantee) |

Under spec 0.8, integers and floats are inferred from the scalar body's
lexical form (no typed markers). An integer outside `int64` range parses
as a String. On encode, Go `int*` / `uint*` / `*big.Int` become integer
scalars, but an out-of-range `*big.Int` parses back as a String;
`float32` / `float64` become float scalars and `string` stays a bare
scalar. `Dumps` and `EmitCanonical` reject `NaN` and `±Inf`;
`DumpsForceStrings` converts them to String leaves. Structs are serialized
through `encoding/json` first, so `json:"..."` tags are honoured.

>>>>> lang=ru
## Соответствие типов

| Ktav             | Go                                              |
| ---------------- | ----------------------------------------------- |
| `null`           | `nil`                                           |
| `true` / `false` | `bool`                                          |
| integer scalar   | `int64` в диапазоне; иначе `string`             |
| float scalar     | `float64`                                       |
| bare scalar      | `string`                                        |
| `[ ... ]`        | `[]any`                                         |
| `{ ... }`        | `map[string]any` (порядок вставки не гарантирован) |

В spec 0.8 integer и float выводятся из лексической формы скалярного тела
(типизированных маркеров нет). Integer вне диапазона `int64` разбирается
как String. Go `int*` / `uint*` / `*big.Int` сериализуются как integer
scalar, но `*big.Int` вне диапазона при повторном разборе становится
String; `float32` / `float64` становятся float scalar, а `string`
остаётся bare scalar. `Dumps` и `EmitCanonical` отклоняют `NaN` и `±Inf`,
а `DumpsForceStrings` преобразует их в String-leaf. Структуры сначала
проходят через `encoding/json`, теги `json:"..."` учитываются.

>>>>> lang=zh
## 类型映射

| Ktav             | Go                                               |
| ---------------- | ------------------------------------------------ |
| `null`           | `nil`                                            |
| `true` / `false` | `bool`                                           |
| integer scalar   | `int64`（范围内）；否则为 `string`                |
| float scalar     | `float64`                                        |
| 裸标量           | `string`                                         |
| `[ ... ]`        | `[]any`                                          |
| `{ ... }`        | `map[string]any`（Go map，不保证插入顺序）        |

spec 0.8 中 integer 与 float 从标量体的词法形式推断（无类型标记）。
超出 `int64` 范围的 integer 会解析为 String。Go `int*` / `uint*` /
`*big.Int` 编码为 integer scalar，但超范围的 `*big.Int` 再解析时会
成为 String；`float32` / `float64` 编码为 float scalar，`string` 保持
裸标量。`Dumps` 和 `EmitCanonical` 会拒绝 `NaN` 与 `±Inf`；
`DumpsForceStrings` 会将它们转换为 String 叶值。结构体先走
`encoding/json`，`json:"..."` tag 生效。

