package main

import (
	"jianggujin.com/lvs/cmd/custom"
	gom "jianggujin.com/lvs/cmd/go"
)

func init() {
	gom.Init(rootCmd)
	custom.Init(rootCmd)
}
