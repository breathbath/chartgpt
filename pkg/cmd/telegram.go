package cmd

import (
	"breathbathChartGPT/pkg/auth"
	"breathbathChartGPT/pkg/chartgpt"
	"breathbathChartGPT/pkg/redis"
	"breathbathChartGPT/pkg/telegram"
	logging "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
	"os/signal"
	"syscall"
)

var telegramCmd = &cobra.Command{
	Use:   "telegram",
	Short: "Starts a Telegram bot",
	RunE: func(cmd *cobra.Command, args []string) error {
		bot, err := buildTelegram()
		if err != nil {
			return err
		}
		go bot.Start()

		waitForSignal(bot)

		return nil
	},
}

func initTelegramCmd() {
	rootCmd.AddCommand(telegramCmd)
}

func waitForSignal(server *telegram.Bot) {
	terminateSignals := make(chan os.Signal, 1)

	signal.Notify(terminateSignals, syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM) //NOTE:: syscall.SIGKILL we cannot catch kill -9 as its force kill signal.

	s := <-terminateSignals
	logging.Infof("Got one of stop signals, shutting down server gracefully, SIGNAL NAME : %v", s)
	server.Stop()
}

func buildTelegram() (*telegram.Bot, error) {
	chartgptMsgHandler, err := chartgpt.BuildHandler()
	if err != nil {
		return nil, err
	}

	storage, err := redis.BuildStorage()
	if err != nil {
		return nil, err
	}

	authHandler, err := auth.BuildHandler(chartgptMsgHandler, storage)
	if err != nil {
		return nil, err
	}

	telegramBot, err := telegram.BuildBot(authHandler)
	if err != nil {
		return nil, err
	}

	return telegramBot, nil
}
