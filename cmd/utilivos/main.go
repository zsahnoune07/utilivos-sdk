package main

import (
	"fmt"
	"os"

	"utilivos/sdk/internal/cli"
)

func main() {
	code, err := cli.Run(os.Args, os.Stdout, os.Stderr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	os.Exit(code)
}
