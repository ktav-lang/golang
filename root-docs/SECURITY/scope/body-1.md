>>>>> lang=en
## Scope

Issues that count as security problems for this package:

- Out-of-bounds reads / writes or panics in the native `ktav_cabi`
  shared library that crash or hang the consumer Go process. The
  library is loaded via `purego` / `LoadLibrary`, so a crash is
  directly visible — no Go-side catch exists.
- Runaway memory or CPU when parsing crafted input.
- Incorrect FFI memory handling (double-free, missing free, reading
  freed buffers across the `ktav_free` boundary).
- Any behaviour that allows crafted Ktav input to escape the expected
  value-domain (arbitrary code execution in the loaded library,
  uninitialised-memory disclosure, etc.).
- A download-time vector: the `internal/native` loader fetches the
  prebuilt binary from the matching GitHub Release. Reports about
  TLS / integrity-check gaps in that path belong here.

Issues that are **not** security problems here — please use regular
issues for these:

- Performance regressions without crash / hang characteristics.
- Behavioural mismatches that aren't exploitable.
- Problems in the Ktav format itself — those belong in
  [`ktav-lang/spec`](https://github.com/ktav-lang/spec).
>>>>> lang=ru
## Область

Что считается проблемой безопасности для этого пакета:

- Out-of-bounds чтения / записи или паники в нативной библиотеке
  `ktav_cabi`, ведущие к падению или зависанию consumer-процесса Go.
  Библиотека грузится через `purego` / `LoadLibrary` — падение видно
  напрямую, на Go-стороне его не поймать.
- Неконтролируемое потребление памяти или CPU при разборе специально
  сформированного входа.
- Некорректное обращение с памятью на FFI-границе (double-free,
  missing free, чтение освобождённого буфера после `ktav_free`).
- Любое поведение, при котором сформированный Ktav-вход выходит за
  ожидаемый value-домен (произвольное выполнение кода в загруженной
  библиотеке, раскрытие неинициализированной памяти и т. п.).
- Download-time вектор: лоадер `internal/native` качает прекомпилированный
  бинарь из соответствующего GitHub Release. Репорты про TLS /
  проверку целостности этой цепочки сюда.

Что **не** считается проблемой безопасности здесь — пожалуйста,
используйте обычные issue:

- Регрессии производительности без характеристик crash / hang.
- Поведенческие расхождения, которые не эксплуатируются.
- Проблемы в самом формате Ktav — им место в
  [`ktav-lang/spec`](https://github.com/ktav-lang/spec).
>>>>> lang=zh
## 范围

以下问题会按本包的安全问题处理:

- 原生 `ktav_cabi` 库中的越界读写或 panic，导致 consumer Go 进程
  崩溃或挂起。库通过 `purego` / `LoadLibrary` 加载 —— 崩溃对调用方
  直接可见，Go 侧无法捕获。
- 解析构造输入时出现失控的内存或 CPU 消耗。
- FFI 边界上的内存处理错误（double-free、missing free、`ktav_free`
  后读取已释放缓冲区）。
- 任何允许构造的 Ktav 输入逃逸出预期值域的行为（加载库内的任意代码
  执行、未初始化内存泄露等）。
- 下载期向量：`internal/native` 加载器会从对应的 GitHub Release
  拉取预编译二进制。该路径上的 TLS / 完整性校验缺陷也上报到这里。

以下**不**算本包的安全问题 —— 请走普通 issue:

- 没有崩溃 / 挂起特征的性能回归。
- 不可利用的行为差异。
- Ktav 格式本身的问题 —— 这类问题属于
  [`ktav-lang/spec`](https://github.com/ktav-lang/spec)。
