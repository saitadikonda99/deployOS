// Package cmd implements the deployos CLI's commands using Cobra.
package cmd

import "github.com/spf13/cobra"

func newRootCmd(version string) *cobra.Command {
	root := &cobra.Command{
		Use:           "deployos",
		Short:         "DeployOS - a personal cloud operating system",
		Long:          "DeployOS turns a machine into a secure, production-ready cloud server.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(newVersionCmd(version))
	root.AddCommand(newDoctorCmd())
	root.AddCommand(newAgentCmd())

	return root
}

// Execute runs the deployos CLI's root command.
func Execute(version string) error {
	return newRootCmd(version).Execute()
}
