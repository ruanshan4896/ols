package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var Version = "0.1.0"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Hiển thị phiên bản của ols-cli",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintf(cmd.OutOrStdout(), "ols-cli version %s\n", Version)
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
}
