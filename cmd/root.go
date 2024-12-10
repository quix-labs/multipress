package cmd

import (
	"github.com/quix-labs/multipress/cmd/env"
	"github.com/quix-labs/multipress/pkg/app"
	"github.com/spf13/cobra"
)

// Cobra command setup
var rootCmd = &cobra.Command{
	Use:   "multipress",
	Short: "Multipress CLI for managing templates and projects",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		configDir, _ := cmd.Flags().GetString("config")
		return app.LoadApp(configDir)
	},
}

func init() {
	rootCmd.AddCommand(env.RootCmd)
	rootCmd.PersistentFlags().StringP("config", "c", "", "configuration folder")
}

// Execute starts the root Cobra command.
func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}
