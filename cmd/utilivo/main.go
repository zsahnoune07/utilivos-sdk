package main

import (
	"fmt"
	"os"

	"github.com/utilivo/utilivo-sdk/internal/cli"
)

func main() {
	code, err := cli.Run(os.Args, os.Stdout, os.Stderr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	os.Exit(code)
}
