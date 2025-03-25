package invoke

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"time"
)

var (
	Timeout = 3 * time.Second
)

type Option func(*exec.Cmd)

func WithStd() Option {
	return func(cmd *exec.Cmd) {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
}

func WithDir(dir string) Option {
	return func(cmd *exec.Cmd) {
		cmd.Dir = dir
	}
}

type Invoker interface {
	Command(string, ...string) ([]byte, error)
	CommandWithContext(context.Context, string, ...string) ([]byte, error)

	CommandOptions(string, []string, ...Option) error
	CommandOptionsWithContext(context.Context, string, []string, ...Option) error
}
type Invoke struct{}

func (i Invoke) Command(name string, args ...string) ([]byte, error) {
	return i.CommandWithContext(context.Background(), name, args...)
}

func (i Invoke) CommandWithContext(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	err := cmd.Run()
	return buf.Bytes(), err
}

func (i Invoke) CommandOptions(name string, args []string, opts ...Option) error {
	return i.CommandOptionsWithContext(context.Background(), name, args, opts...)
}

func (i Invoke) CommandOptionsWithContext(ctx context.Context, name string, args []string, opts ...Option) error {
	cmd := exec.CommandContext(ctx, name, args...)
	for _, opt := range opts {
		opt(cmd)
	}
	if cmd.Stdout == nil || cmd.Stderr == nil {
		var buf bytes.Buffer
		if cmd.Stdout == nil {
			cmd.Stdout = &buf
		}
		if cmd.Stderr == nil {
			cmd.Stderr = &buf
		}
	}

	return cmd.Run()
}

var invoke Invoker = Invoke{}

func GetInvoker() Invoker {
	return invoke
}
