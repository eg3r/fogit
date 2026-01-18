package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/eg3r/fogit/commands"
	"github.com/eg3r/fogit/pkg/fogit"
)

func main() {
	// Set up signal handling for graceful shutdown
	// This creates a context that will be canceled when SIGINT (Ctrl+C) or SIGTERM is received
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := commands.ExecuteContext(ctx); err != nil {
		// Check if the error was due to context cancellation (user pressed Ctrl+C)
		if ctx.Err() == context.Canceled {
			fmt.Fprintln(os.Stderr, "\nOperation canceled")
			os.Exit(130) // Standard exit code for SIGINT
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(fogit.GetExitCode(err))
	}
}
