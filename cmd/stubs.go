package cmd

import (
	"github.com/quix-labs/multipress/pkg/app"
	"github.com/spf13/cobra"
)

// publishStubsCmd represents the publishStubs command
var publishStubsCmd = &cobra.Command{
	Use: "stubs",
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO PARAMETER OVERRIDE AND DISPLAY ALL STUBS PUBLISHED
		return app.GetApplication().PublishStubs()
	},
}

func init() {
	rootCmd.AddCommand(publishStubsCmd)
}
