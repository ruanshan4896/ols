package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var RootCmd = &cobra.Command{
	Use:   "ols",
	Short: "WordPress & OpenLiteSpeed Docker Multi-Site Management CLI",
	Long:  `ols là công cụ tự động hóa quản lý nhiều website WordPress độc lập với OpenLiteSpeed và Docker trên VPS.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunInteractiveMenu(os.Stdin, os.Stdout)
	},
}

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
