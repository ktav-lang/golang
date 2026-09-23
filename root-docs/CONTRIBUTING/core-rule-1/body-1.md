>>>>> lang=en
## Core rules

### 1. Every bug fix ships with a regression test

When you find a bug, **before fixing it**, write a test that reproduces
it — the test **must fail on `main`** and pass after the fix. Include
both in the same PR.

Tests live at the repo root:

| File                    | Scope                                                      |
| ----------------------- | ---------------------------------------------------------- |
| `ktav_smoke_test.go`    | Loads / Dumps happy paths, bigint, error surface.          |
| `conformance_test.go`   | Cross-language conformance against `ktav-lang/spec`.       |

>>>>> lang=ru
## Основные правила

### 1. Каждый багфикс приходит с регрессионным тестом

Когда нашёл баг, **перед тем как чинить**, напиши тест, который его
воспроизводит — тест **должен падать на `main`** и проходить после
фикса. Оба в одном PR.

Тесты лежат в корне репо:

| Файл                    | Область                                                  |
| ----------------------- | -------------------------------------------------------- |
| `ktav_smoke_test.go`    | Loads / Dumps happy paths, bigint, форма ошибок.         |
| `conformance_test.go`   | Cross-language conformance против `ktav-lang/spec`.      |

>>>>> lang=zh
## 核心规则

### 1. 每个 bugfix 伴随回归测试

发现 bug **修之前**，先写一个能复现它的测试 —— 该测试在 `main` 上
**必须失败**，修复后通过。两者放同一个 PR。

测试位于仓库根：

| 文件                    | 作用域                                                   |
| ----------------------- | -------------------------------------------------------- |
| `ktav_smoke_test.go`    | Loads / Dumps happy path、bigint、错误形态。             |
| `conformance_test.go`   | 针对 `ktav-lang/spec` 的跨语言一致性。                   |

