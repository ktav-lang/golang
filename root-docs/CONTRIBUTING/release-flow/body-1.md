>>>>> lang=en
## Release flow

Tag `v<X.Y.Z>` on `main`. The release workflow cross-compiles six
platform binaries (`linux` amd64/arm64, `darwin` amd64/arm64, `windows`
amd64/arm64), attaches them as GitHub Release assets, and the Go proxy
picks up the tag automatically. The embedded `LibVersion` constant in
`internal/native/loader.go` must match the tag — change it in the same
commit as the tag message.

>>>>> lang=ru
## Релизный процесс

Тег `v<X.Y.Z>` на `main`. Release workflow кросс-компилирует шесть
платформенных бинарей (`linux` amd64/arm64, `darwin` amd64/arm64,
`windows` amd64/arm64), прикрепляет как assets GitHub Release, а Go
proxy подхватит тег сам. Константа `LibVersion` в
`internal/native/loader.go` должна совпадать с тегом — меняется в том же
коммите, что и сообщение тега.

>>>>> lang=zh
## 发布流程

在 `main` 上打 `v<X.Y.Z>` tag。Release workflow 交叉编译六个平台的
二进制（`linux` amd64/arm64、`darwin` amd64/arm64、`windows` amd64/arm64），
作为 GitHub Release assets 附加，Go proxy 会自动索引 tag。
`internal/native/loader.go` 中的 `LibVersion` 常量必须与 tag 对齐 ——
在打 tag 消息的同一 commit 里更新。

