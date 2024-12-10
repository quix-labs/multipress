package cmd

import (
	"github.com/quix-labs/multipress/pkg/app"
	"github.com/spf13/cobra"
)

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use: "start",
	RunE: func(cmd *cobra.Command, args []string) error {
		return app.GetApplication().Start()
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
}
