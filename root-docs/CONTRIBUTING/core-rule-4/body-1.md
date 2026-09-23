>>>>> lang=en
### 4. One concept per commit

Commits should be atomic: a bug fix and its test together, a feature
and its tests together, a rename on its own, a refactor on its own.
`git log --oneline` should read like a changelog. Don't prefix commit
messages with `feat:` / `fix:` — no conventional commits here.

>>>>> lang=ru
### 4. Один концепт на коммит

Коммиты атомарны: багфикс и его тест — вместе, фича и её тесты — вместе,
переименование — отдельно, рефактор — отдельно. `git log --oneline`
должен читаться как changelog. Префиксы `feat:` / `fix:` не используем
— conventional commits здесь не применяем.

>>>>> lang=zh
### 4. 一个 commit 一个概念

Commit 保持原子：bugfix 和它的测试在一起、feature 和它的测试在一起、
重命名单独、重构单独。`git log --oneline` 应当像 changelog 一样阅读。
不要在 commit message 里加 `feat:` / `fix:` 前缀 —— 这里不用 conventional
commits。

