package cmd

import (
	"breathbathChatGPT/pkg/auth"
	"breathbathChatGPT/pkg/chatgpt"
	"breathbathChatGPT/pkg/msg"
	"breathbathChatGPT/pkg/storage"
	"breathbathChatGPT/pkg/telegram"
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

		logging.Info("started telegram bot")

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
	db, err := storage.BuildRedisClient()
	if err != nil {
		return nil, err
	}

	chatgptMsgHandler, handlerHelp, err := chatgpt.BuildChatCompletionHandler(db)
	if err != nil {
		return nil, err
	}

	commandsHandler := &msg.CommandsHandler{
		PassHandler: chatgptMsgHandler,
		DynamicHelp: handlerHelp,
	}

	authHandler, err := auth.BuildHandler(commandsHandler, db)
	if err != nil {
		return nil, err
	}

	telegramBot, err := telegram.BuildBot(authHandler)
	if err != nil {
		return nil, err
	}

	return telegramBot, nil
}
