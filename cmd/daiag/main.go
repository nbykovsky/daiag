package main

import (
	"context"
	"os"

	"daiag/internal/cli"
)

func main() {
	app := cli.NewDefault(os.Stdout, os.Stderr)
	os.Exit(app.Run(context.Background(), os.Args[1:]))
}
