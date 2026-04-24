// Basic example: parse a Ktav document, print the native Go shape,
// then re-render it back to text.
//
// Run with the repo-built native library:
//
//	cargo build --release -p ktav-cabi
//	KTAV_LIB_PATH=$PWD/target/release/libktav_cabi.so \
//	    go run ./examples/basic
package main

import (
	"fmt"

	ktav "github.com/ktav-lang/golang"
)

const src = `
service: web
port:i 8080
ratio:f 0.75
tls: true
tags: [
    prod
    eu-west-1
]
db.host: primary.internal
db.timeout:i 30
`

func main() {
	cfg, err := ktav.Loads(src)
	if err != nil {
		panic(err)
	}
	fmt.Printf("parsed: %#v\n\n", cfg)

	out, err := ktav.Dumps(cfg)
	if err != nil {
		panic(err)
	}
	fmt.Printf("re-rendered:\n%s", out)
}
