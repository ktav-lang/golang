>>>>> lang=en
### Type mapping

| Ktav             | Go                                              |
| ---------------- | ----------------------------------------------- |
| `null`           | `nil`                                           |
| `true` / `false` | `bool`                                          |
| `:i <digits>`    | `int64` if it fits, else `*big.Int`             |
| `:f <number>`    | `float64`                                       |
| bare scalar      | `string`                                        |
| `[ ... ]`        | `[]any`                                         |
| `{ ... }`        | `map[string]any` (insertion order preserved)    |

>>>>> lang=ru
### Соответствие типов

| Ktav             | Go                                              |
| ---------------- | ----------------------------------------------- |
| `null`           | `nil`                                           |
| `true` / `false` | `bool`                                          |
| `:i <digits>`    | `int64` если помещается, иначе `*big.Int`       |
| `:f <number>`    | `float64`                                       |
| scalar без маркера | `string`                                      |
| `[ ... ]`        | `[]any`                                         |
| `{ ... }`        | `map[string]any` (порядок вставки сохраняется)  |

>>>>> lang=zh
### 类型映射

| Ktav             | Go                                                |
| ---------------- | ------------------------------------------------- |
| `null`           | `nil`                                             |
| `true` / `false` | `bool`                                            |
| `:i <digits>`    | `int64`(安全范围)/ `*big.Int`(更大)           |
| `:f <number>`    | `float64`                                         |
| 裸标量           | `string`                                          |
| `[ ... ]`        | `[]any`                                           |
| `{ ... }`        | `map[string]any`(保留插入顺序)                  |

