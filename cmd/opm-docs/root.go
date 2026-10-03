package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"

	"github.com/spf13/cobra"
)

// Exit codes.
const (
	exitOK    = 0
	exitUsage = 1
	exitError = 2
)

// usageError marks an error as the caller's: an unknown flag, a missing
// argument, an unreadable or invalid config. It exits 1.
type usageError struct{ err error }

func (e *usageError) Error() string { return e.err.Error() }
func (e *usageError) Unwrap() error { return e.err }

func usage(format string, args ...any) error {
	return &usageError{fmt.Errorf(format, args...)}
}

func asUsage(err error) error {
	if err == nil {
		return nil
	}
	return &usageError{err}
}

// silentError is an execution failure whose details were already printed
// (lint violations). It exits 2 with only its summary.
type silentError struct{ msg string }

func (e *silentError) Error() string { return e.msg }

func newRoot(stdout, stderr io.Writer) *cobra.Command {
	root := &cobra.Command{
		Use:           "opm-docs",
		Short:         "Build, lint, publish and pull OPM docs bundles",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetFlagErrorFunc(func(_ *cobra.Command, err error) error { return asUsage(err) })
	root.AddCommand(
		newVersionCmd(),
		newLintCmd(),
		newBuildCmd(),
		newCheckCmd(),
		newPushCmd(),
		newPromoteCmd(),
		newPullCmd(),
		newReviseCmd(),
	)
	return root
}

func run(args []string, stdout, stderr io.Writer) int {
	root := newRoot(stdout, stderr)
	root.SetArgs(args)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	err := root.ExecuteContext(ctx)
	if err == nil {
		return exitOK
	}
	fmt.Fprintln(stderr, "opm-docs:", strings.TrimSpace(err.Error()))
	var ue *usageError
	if errors.As(err, &ue) || isCobraUsage(err) {
		return exitUsage
	}
	return exitError
}

// isCobraUsage recognizes the errors cobra returns for an unknown command
// or a wrong number of arguments, which never pass through the flag error
// function.
func isCobraUsage(err error) bool {
	msg := err.Error()
	for _, p := range []string{"unknown command", "accepts ", "requires at least", "requires at most", "unknown flag", "unknown shorthand flag", "flag needs an argument", "invalid argument"} {
		if strings.Contains(msg, p) {
			return true
		}
	}
	return false
}
