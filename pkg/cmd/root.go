package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

func NewRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "configism",
	}
	cmd.AddCommand(NewVersionCommand())
	return cmd
}

func Execute() int {
	rootCmd := NewRootCommand()
	err := rootCmd.Execute()
	if err != nil {
		fmt.Println(err)
		return 1
	}
	return 0
}
