>>>>> lang=en
# Pure-Go backend

Use `go get github.com/ktav-lang/golang/pure@v0.8.1`, then import:

```go
package main

import (
    "fmt"
    "log"

    "github.com/ktav-lang/golang/pure"
)

func main() {
    value, err := pure.Loads("server.port: 8080\n")
    if err != nil {
        log.Fatal(err)
    }
    text, err := pure.Dumps(value)
    if err != nil {
        log.Fatal(err)
    }
    restored, err := pure.Loads(text)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Print(text)
    fmt.Printf("%#v\n", restored)
}
```

>>>>> lang=ru
# Pure-Go бэкенд

Установите через `go get github.com/ktav-lang/golang/pure@v0.8.1`, затем импортируйте:

```go
package main

import (
    "fmt"
    "log"

    "github.com/ktav-lang/golang/pure"
)

func main() {
    value, err := pure.Loads("server.port: 8080\n")
    if err != nil {
        log.Fatal(err)
    }
    text, err := pure.Dumps(value)
    if err != nil {
        log.Fatal(err)
    }
    restored, err := pure.Loads(text)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Print(text)
    fmt.Printf("%#v\n", restored)
}
```

>>>>> lang=zh
# Pure-Go 后端

使用 `go get github.com/ktav-lang/golang/pure@v0.8.1` 安装，然后导入：

```go
package main

import (
    "fmt"
    "log"

    "github.com/ktav-lang/golang/pure"
)

func main() {
    value, err := pure.Loads("server.port: 8080\n")
    if err != nil {
        log.Fatal(err)
    }
    text, err := pure.Dumps(value)
    if err != nil {
        log.Fatal(err)
    }
    restored, err := pure.Loads(text)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Print(text)
    fmt.Printf("%#v\n", restored)
}
```

