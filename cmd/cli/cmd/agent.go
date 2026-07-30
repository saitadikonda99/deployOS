package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newAgentCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "agent",
		Short: "Manage the DeployOS node agent running on this machine",
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := fmt.Fprintln(cmd.OutOrStdout(),
				"agent management via the CLI is not implemented yet; run the deployos-agent binary directly (cmd/agent) to start an agent.")
			return err
		},
	}
}
