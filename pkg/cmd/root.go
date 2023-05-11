package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "bgpt",
	Short: "Breathbath ChatGPT integrates it into popular bots like Telegram",
}

func Execute() error {
	initVersionCmd()
	initTelegramCmd()
	initBcryptCmd()

	return rootCmd.Execute()
}
