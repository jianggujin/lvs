package util

import (
	"fmt"
	"jianggujin.com/lvs/internal/invoke"
	"os"
	"os/exec"
)

type ErrorWrapper struct {
	Msg string
	Err error
}

func (e *ErrorWrapper) SetMsg(msg string, args ...any) *ErrorWrapper {
	if len(args) > 0 {
		e.Msg = fmt.Sprintf(msg, args...)
	} else {
		e.Msg = msg
	}

	return e
}
func (e *ErrorWrapper) SetErr(err error) *ErrorWrapper {
	e.Err = err
	return e
}

func (e *ErrorWrapper) String() string {
	if e.Err != nil {
		if e.Msg != "" {
			return e.Msg + ": " + e.Err.Error()
		}
		return e.Err.Error()
	}
	return e.Msg
}

func (e *ErrorWrapper) Error() string { return e.String() }

func (e *ErrorWrapper) Unwrap() error {
	return e.Err
}

func (e *ErrorWrapper) Unwraps() error {
	if e.Err == nil {
		return e
	}
	if next, ok := e.Err.(*ErrorWrapper); ok {
		return next.Unwraps()
	}
	return e.Err
}

func WrapError(err error) *ErrorWrapper {
	return new(ErrorWrapper).SetErr(err)
}
func WrapErrorMsg(msg string, args ...any) *ErrorWrapper {
	return new(ErrorWrapper).SetMsg(msg, args...)
}

func Sudo(err error) error {
	if err != nil {
		if wrapper, ok := err.(*ErrorWrapper); ok {
			err = wrapper.Unwraps()
		}
		if err != nil && os.IsPermission(err) && isSudoAvailable() {
			sudoErr := invoke.GetInvoker().CommandOptions("sudo", append([]string{"--preserve-env"}, os.Args...), invoke.WithStd())
			if sudoErr != nil {
				return WrapError(err).SetMsg("permission elevation failed(%v)", sudoErr)
			}
			os.Exit(0)
		}
	}
	return err
}

func isSudoAvailable() bool {
	_, err := exec.LookPath("sudo")
	return err == nil
}
