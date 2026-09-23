>>>>> lang=en
## How the native library is resolved

At first call the Go package resolves `ktav_cabi` in this order:

1. **`$KTAV_LIB_PATH`** — absolute path to a local build. Most useful
   for development.
2. **User cache** — `<os.UserCacheDir>/ktav-go/v<version>/…`, downloaded
   on a previous call.
3. **GitHub Release download** — the matching asset is fetched once
   from `github.com/ktav-lang/golang/releases/download/v<version>/<name>`
   and cached under (2). Requires network on first call after install.

>>>>> lang=ru
## Как резолвится нативная библиотека

На первый вызов Go-пакет ищет `ktav_cabi` в следующем порядке:

1. **`$KTAV_LIB_PATH`** — абсолютный путь к локальной сборке. Удобно для
   разработки.
2. **User cache** — `<os.UserCacheDir>/ktav-go/v<version>/…`, скачано на
   предыдущем вызове.
3. **Скачивание с GitHub Release** — подходящий asset качается один раз
   с `github.com/ktav-lang/golang/releases/download/v<version>/<name>` и
   кэшируется в (2). Нужен интернет на первый вызов после установки.

>>>>> lang=zh
## 原生库解析顺序

首次调用时，Go 包按以下顺序查找 `ktav_cabi`：

1. **`$KTAV_LIB_PATH`** —— 本地构建的绝对路径，适合开发。
2. **用户缓存** —— `<os.UserCacheDir>/ktav-go/v<version>/…`，此前已下载。
3. **GitHub Release 下载** —— 从
   `github.com/ktav-lang/golang/releases/download/v<version>/<name>`
   下载匹配的资产，缓存到 (2)。安装后首次调用需要联网。

