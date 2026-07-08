package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	"github.com/cleura/cleura-cli/internal/cli"
)

// version is set at build time via -ldflags "-X main.version=...". Module
// builds (go install ...@version) apply no ldflags, so fall back to the
// version recorded in the build info.
var version = "dev"

func main() {
	v := version
	if v == "dev" {
		if bi, ok := debug.ReadBuildInfo(); ok && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
			v = bi.Main.Version
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := cli.NewRootCommand(v).ExecuteContext(ctx); err != nil {
		// Treat cancellation-caused errors as an interrupt: from Go 1.26,
		// NotifyContext cancels with a cause ("interrupt signal received")
		// that in-flight errors wrap instead of context.Canceled. Errors
		// unrelated to the cancellation — e.g. a contractual exit code that
		// raced a signal — keep their own exit code.
		if errors.Is(err, context.Canceled) || (ctx.Err() != nil && errors.Is(err, context.Cause(ctx))) {
			os.Exit(130)
		}
		exitCode := 1
		var coded *cli.ExitCodeError
		if errors.As(err, &coded) {
			exitCode = coded.Code
		}
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(exitCode)
	}
}
