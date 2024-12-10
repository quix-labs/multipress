package env

import (
	"fmt"

	"github.com/spf13/cobra"
)

// RootCmd represents the env command
var RootCmd = &cobra.Command{
	Use: "env",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("env called")
	},
}
