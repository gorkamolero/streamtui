package main

import (
	"os"

	"streamtui/internal/cli"
)

func main() {
	code := cli.Run(os.Args[1:], os.Stdout, os.Stderr)
	os.Exit(int(code))
}
