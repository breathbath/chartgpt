package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var Version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of breathbathChatGPT",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("breathbathChatGPT Version:\n%s", Version)
	},
}

func initVersionCmd() {
	rootCmd.AddCommand(versionCmd)
}
