>>>>> lang=en
## Choosing the pure-Go backend

By default the package dispatches to the native `ktav_cabi` shared
library (purego, no cgo). A second, fully self-contained implementation
— parser and generator written in Go — is available and covers every
public operation with identical signatures:

```go
ktav.UsePure()                      // or:
ktav.UseBackend(ktav.BackendPure)
```

- `UsePure` / `UseBackend(ktav.BackendPure)`: pure-Go path. It never
  loads, downloads or requires the native library and never spawns a
  subprocess, so it also works on machines where no `ktav_cabi`
  artifact is installed.
- `UseNative` / `UseBackend(ktav.BackendNative)`: the purego `ktav_cabi`
  path. This is the default and the historical behaviour.
- There is no automatic fallback in either direction: if the selected
  backend fails, you get an error, not the other implementation.

Call the selector once at startup, before the first parse. The
implementation can be switched between calls; a call in flight always
finishes on the backend it started on.

>>>>> lang=ru
## Выбор pure-Go бэкенда

По умолчанию пакет направляет вызовы в нативную разделяемую библиотеку
`ktav_cabi` (purego, без cgo). Доступна и вторая, полностью
самодостаточная реализация — парсер и генератор, написанные на Go. Она
покрывает все публичные операции с теми же сигнатурами:

```go
ktav.UsePure()                      // или:
ktav.UseBackend(ktav.BackendPure)
```

- `UsePure` / `UseBackend(ktav.BackendPure)`: pure-Go путь. Он не
  загружает, не скачивает и не требует нативную библиотеку и не
  порождает подпроцессов, поэтому работает и там, где артефакт
  `ktav_cabi` не установлен.
- `UseNative` / `UseBackend(ktav.BackendNative)`: путь через purego
  `ktav_cabi`. Это значение по умолчанию и историческое поведение.
- Автоматического переключения нет ни в одну сторону: если выбранный
  бэкенд завершается ошибкой, вы получаете ошибку, а не другую
  реализацию.

Выберите бэкенд один раз при старте, до первого разбора. Реализацию
можно переключать между вызовами; начатый вызов всегда завершается на
том бэкенде, на котором начался.

>>>>> lang=zh
## 选择 pure-Go 后端

默认情况下，包会把调用分发到原生 `ktav_cabi` 共享库（purego，无
cgo）。另一个完全自包含的实现——以 Go 编写的解析器和生成器——同样
可用，以相同签名覆盖所有公开操作：

```go
ktav.UsePure()                      // 或：
ktav.UseBackend(ktav.BackendPure)
```

- `UsePure` / `UseBackend(ktav.BackendPure)`：pure-Go 路径。它不加载、
  不下载、也不需要原生库，且不启动子进程，因此在未安装 `ktav_cabi`
  制品的机器上同样可用。
- `UseNative` / `UseBackend(ktav.BackendNative)`：经 purego `ktav_cabi`
  的路径。这是默认值，也是历史行为。
- 两个方向都没有自动回退：所选后端失败时，你得到的是错误，而不是
  另一个实现。

请在启动时、第一次解析之前选择一次后端。两次调用之间可以切换实现；
进行中的调用总是在它开始时所用的后端上完成。

