package main

import (
	"fmt"
	"runtime"
)

func main() {
	fmt.Printf("%s %s/%s", runtime.Version(), runtime.GOOS, runtime.GOARCH)
}
