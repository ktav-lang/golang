>>>>> lang=en
## Type mapping

| Ktav             | Go                                              |
| ---------------- | ----------------------------------------------- |
| `null`           | `nil`                                           |
| `true` / `false` | `bool`                                          |
| integer scalar   | `int64` if it fits, else `*big.Int`             |
| float scalar     | `float64`                                       |
| bare scalar      | `string`                                        |
| `[ ... ]`        | `[]any`                                         |
| `{ ... }`        | `map[string]any` (insertion order preserved)    |

Under spec 0.5 integers and floats are inferred from the scalar body's
lexical form (no typed markers). On encode, Go `int*` / `uint*` /
`*big.Int` become integer scalars; `float32` / `float64` become float
scalars; `string` stays a bare scalar. `NaN` and `±Inf` are rejected.
Structs are serialized through `encoding/json` first, so `json:"..."`
tags are honoured.

>>>>> lang=ru
## Соответствие типов

| Ktav             | Go                                              |
| ---------------- | ----------------------------------------------- |
| `null`           | `nil`                                           |
| `true` / `false` | `bool`                                          |
| integer scalar   | `int64` если помещается, иначе `*big.Int`       |
| float scalar     | `float64`                                       |
| bare scalar      | `string`                                        |
| `[ ... ]`        | `[]any`                                         |
| `{ ... }`        | `map[string]any` (порядок вставки сохраняется)  |

В spec 0.5 integer и float выводятся из лексической формы скалярного тела
(типизированных маркеров нет). На сериализации Go `int*` / `uint*` /
`*big.Int` → integer scalar; `float32` / `float64` → float scalar;
`string` остаётся bare scalar. `NaN` и `±Inf` отвергаются. Структуры
сначала проходят через `encoding/json`, теги `json:"..."` учитываются.

>>>>> lang=zh
## 类型映射

| Ktav             | Go                                               |
| ---------------- | ------------------------------------------------ |
| `null`           | `nil`                                            |
| `true` / `false` | `bool`                                           |
| integer scalar   | `int64`（可容纳）或 `*big.Int`                   |
| float scalar     | `float64`                                        |
| 裸标量           | `string`                                         |
| `[ ... ]`        | `[]any`                                          |
| `{ ... }`        | `map[string]any`（保留插入顺序）                 |

spec 0.5 中 integer 与 float 从标量体的词法形式推断（无类型标记）。编码时
Go `int*` / `uint*` / `*big.Int` → integer scalar；`float32` / `float64`
→ float scalar；`string` 保持裸标量。`NaN` 与 `±Inf` 会被拒绝。结构体先
走 `encoding/json`，`json:"..."` tag 生效。

