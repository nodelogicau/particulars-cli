// Command particulars is a CLI for Dialectical Knowledge Format workspaces.
package main

import (
	"os"

	"github.com/nodelogicau/particulars-cli/internal/cli"
)

func main() {
	os.Exit(cli.Execute(os.Args[1:], os.Stdin, os.Stdout, os.Stderr, stdinIsTerminal))
}

func stdinIsTerminal() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}
