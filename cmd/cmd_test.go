package main

import (
	"os"
	"testing"
)

func TestConfig(t *testing.T) {
	execute(t, "config")
}

func TestVersion(t *testing.T) {
	execute(t, "version")
}

func TestInstall(t *testing.T) {
	execute(t, "install")
}

func TestUninstall(t *testing.T) {
	execute(t, "uninstall")
}

func TestGoInstall(t *testing.T) {
	execute(t, "go", "install", "latest")
}

func TestNodeList(t *testing.T) {
	execute(t, "node", "ls")
}

func TestNodeExec(t *testing.T) {
	execute(t, "node", "exec", "18.20.3", "node", "-v")
	execute(t, "node", "current")
}

func TestNodeAlias(t *testing.T) {
	execute(t, "node", "alias", "Default", "1.18.0")
	execute(t, "node", "alias", "default")
	execute(t, "node", "alias")
}

func TestNodeUnAlias(t *testing.T) {
	execute(t, "node", "unalias", "default")
}

func TestEnv(t *testing.T) {
	t.Log(os.Getenv("Path"))
}

func execute(t *testing.T, args ...string) {
	rootCmd.SetArgs(args)
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
}
