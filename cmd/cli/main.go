// Command deployos is the DeployOS command-line interface.
package main

import (
	"fmt"
	"os"

	"github.com/saitadikonda99/deployOS/cmd/cli/cmd"
)

// version is set at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	if err := cmd.Execute(version); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
