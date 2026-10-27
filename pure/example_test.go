package pure_test

import (
	"fmt"
	"reflect"

	"github.com/ktav-lang/golang/pure"
)

func ExampleLoads() {
	value, err := pure.Loads("server.port: 8080\n")
	if err != nil {
		panic(err)
	}
	text, err := pure.Dumps(value)
	if err != nil {
		panic(err)
	}
	restored, err := pure.Loads(text)
	if err != nil {
		panic(err)
	}
	want := map[string]any{
		"server": map[string]any{"port": int64(8080)},
	}
	fmt.Println(reflect.DeepEqual(value, want))
	fmt.Println(reflect.DeepEqual(restored, want))
	// Output:
	// true
	// true
}

func ExampleInsertValue() {
	out, err := pure.InsertValue("server: {port: 8080}\n", `server.host`, "localhost")
	if err != nil {
		panic(err)
	}
	fmt.Print(out)
	// Output:
	// server: {port: 8080, host: localhost}
}

func ExampleRenameKey() {
	out, err := pure.RenameKey("a: 1\n", "a", `new.key`)
	if err != nil {
		panic(err)
	}
	fmt.Print(out)
	// Output:
	// "new.key": 1
}
