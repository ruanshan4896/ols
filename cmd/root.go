package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var RootCmd = &cobra.Command{
	Use:   "ols-cli",
	Short: "WordPress & OpenLiteSpeed Docker Multi-Site Management CLI",
	Long:  `ols-cli là công cụ tự động hóa quản lý nhiều website WordPress độc lập với OpenLiteSpeed và Docker trên VPS.`,
}

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
