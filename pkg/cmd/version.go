package cmd

import (
	"fmt"
	"github.com/amannm/configism/internal/build"
	"github.com/spf13/cobra"
)

func NewVersionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version number",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\n", build.Version)
			return err
		},
	}
	return cmd
}
