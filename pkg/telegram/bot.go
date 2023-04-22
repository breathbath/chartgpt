package telegram

import (
	"breathbathChartGPT/pkg/errs"
	"breathbathChartGPT/pkg/msg"
	"context"
	"fmt"
	"github.com/pkg/errors"
	logging "github.com/sirupsen/logrus"
	"gopkg.in/telebot.v3"
)

type Bot struct {
	conf       *Config
	baseBot    *telebot.Bot
	msgHandler msg.Handler
}

func NewBot(c *Config, h msg.Handler) (*Bot, error) {
	validationErr := c.Validate()
	if validationErr.HasErrors() {
		return nil, validationErr
	}

	botApi, err := telebot.NewBot(telebot.Settings{
		Token: c.APIToken,
		OnError: func(err error, c telebot.Context) {
			errs.Handle(err, false)
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create telegram bot")
	}

	return &Bot{conf: c, baseBot: botApi, msgHandler: h}, nil
}

func (b *Bot) botMsgToRequest(telegramMsg telebot.Context) *msg.Request {
	sender := new(msg.Sender)
	telegramSender := telegramMsg.Sender()
	if telegramSender != nil {
		id := telegramSender.Username
		if id == "" {
			id = fmt.Sprint(telegramSender.ID)
		}

		sender.ID = id
		sender.LastName = telegramSender.LastName
		sender.FirstName = telegramSender.FirstName
	}

	return &msg.Request{
		Source:  "telegram",
		ID:      fmt.Sprint(telegramMsg.Message().ID),
		Sender:  sender,
		Message: telegramMsg.Text(),
		Meta: map[string]interface{}{
			"payload":   telegramMsg.Message().Payload,
			"timestamp": telegramMsg.Message().Unixtime,
		},
	}
}

func (b *Bot) processResponseMessage(
	telegramMsg telebot.Context,
	respMsg msg.ResponseMessage,
	keyboard *telebot.ReplyMarkup,
) error {
	if respMsg.Message == "" {
		return nil
	}

	var err error
	switch respMsg.Type {
	case msg.Success:
		_, err = b.baseBot.Send(telegramMsg.Sender(), respMsg.Message, &telebot.SendOptions{})
		if err != nil {
			return err
		}
	case msg.Error:
		_, err = b.baseBot.Send(
			telegramMsg.Sender(),
			fmt.Sprintf(`<font color="red">%s</font>`, respMsg.Message),
			&telebot.SendOptions{
				ParseMode: telebot.ModeHTML,
			},
		)

		if err != nil {
			return err
		}
	case msg.Prompt:
		if len(keyboard.InlineKeyboard) == 0 {
			keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, []telebot.InlineButton{})
		}

		keyboard.InlineKeyboard[0] = append(keyboard.InlineKeyboard[0], telebot.InlineButton{
			Text: respMsg.Message,
			Data: fmt.Sprint(respMsg.Meta["data"]),
		})
	}

	return nil
}

func (b *Bot) handle(ctx context.Context, c telebot.Context) error {
	log := logging.WithContext(ctx)

	log = log.WithField("message.id", fmt.Sprintf("%d_%d", c.Chat().ID, c.Message().ID))
	log.Infof("got telegram message: %q", c.Text())

	req := b.botMsgToRequest(c)

	resp, err := b.msgHandler.Handle(ctx, req)
	if err != nil {
		_, sendErr := b.baseBot.Send(c.Sender(), "Unexpected error", &telebot.SendOptions{})
		if sendErr != nil {
			log.Errorf("failed to send error message to the sender: %v", sendErr)
		}

		return err
	}

	if resp == nil || len(resp.Messages) == 0 {
		log.Info("message is ignored")
		return nil
	}

	keyboard := &telebot.ReplyMarkup{
		InlineKeyboard: [][]telebot.InlineButton{},
	}

	for _, respMsg := range resp.Messages {
		err := b.processResponseMessage(c, respMsg, keyboard)
		if err != nil {
			return err
		}
	}

	if len(keyboard.InlineKeyboard) > 0 {
		log.Infof("will send prompt options: %v", keyboard.InlineKeyboard)
		_, err := b.baseBot.Send(c.Message().Chat, "here are some options", keyboard)
		if err != nil {
			return err
		}
	}

	return nil
}

func (b *Bot) Start() {
	b.baseBot.Handle(telebot.OnText, func(c telebot.Context) error {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		return b.handle(ctx, c)
	})

	b.baseBot.Start()
}

func (b *Bot) Stop() {
	logging.Info("will stop telegram bot")
	b.baseBot.Stop()
	logging.Info("stopped telegram bot")
}
