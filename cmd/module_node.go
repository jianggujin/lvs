//go:build (windows && (amd64 || 386 || arm64)) || (linux && (amd64 || arm || armv7l || arm64 || ppc64le || s390x)) || (darwin && (amd64 || arm64))

package main

import "jianggujin.com/lvs/cmd/node"

func init() {
	node.Init(rootCmd)
}
