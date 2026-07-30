package cmd

import (
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/cobra"
)

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check that the local environment is ready to run DeployOS",
		RunE: func(cmd *cobra.Command, _ []string) error {
			out := cmd.OutOrStdout()

			if _, err := fmt.Fprintf(out, "go runtime:  %s (%s/%s)\n", runtime.Version(), runtime.GOOS, runtime.GOARCH); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(out, "config.yaml: %s\n", describePresence("config.yaml")); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(out, ".env:        %s\n", describePresence(".env")); err != nil {
				return err
			}

			return nil
		},
	}
}

func describePresence(path string) string {
	if _, err := os.Stat(path); err != nil {
		return "not found (defaults will be used)"
	}
	return "found"
}
