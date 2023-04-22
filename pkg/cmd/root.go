package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "bgpt",
	Short: "Breathbath ChartGPT integrates it into popular bots like Telegram",
}

func Execute() error {
	initVersionCmd()
	initTelegramCmd()

	return rootCmd.Execute()
}
