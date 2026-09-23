>>>>> lang=en
### 5. Native library stays in lockstep with the Go module

The embedded `LibVersion` constant (`internal/native/loader.go`) **must**
match the git tag used to cut the release. If you bump the Go module
version, update `LibVersion` in the same commit. Mismatched values cause
consumers to download a library that doesn't match their code.

>>>>> lang=ru
### 5. Нативная библиотека идёт в ногу с Go-модулем

Константа `LibVersion` (`internal/native/loader.go`) **должна** совпадать
с тегом релиза. Если поднимаешь версию Go-модуля — поднимай
`LibVersion` в том же коммите. Рассинхрон приводит к тому, что потребитель
качает библиотеку не от той версии кода.

>>>>> lang=zh
### 5. 原生库与 Go 模块版本对齐

`LibVersion` 常量（`internal/native/loader.go`）**必须**与 release
tag 对齐。升级 Go 模块版本时，在同一个 commit 里更新 `LibVersion`。
错位会让使用者下载到与代码不匹配的库。

