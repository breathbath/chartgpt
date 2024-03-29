package telegram

import (
	"context"
	"fmt"

	"breathbathChatGPT/pkg/errs"
	"breathbathChatGPT/pkg/msg"

	"github.com/pkg/errors"
	logging "github.com/sirupsen/logrus"
	"gopkg.in/telebot.v3"
)

type Bot struct {
	conf       *Config
	baseBot    *telebot.Bot
	msgHandler *msg.Router
}

func NewBot(c *Config, r *msg.Router) (*Bot, error) {
	validationErr := c.Validate()
	if validationErr.HasErrors() {
		return nil, validationErr
	}

	botAPI, err := telebot.NewBot(telebot.Settings{
		Token: c.APIToken,
		OnError: func(err error, c telebot.Context) {
			errs.Handle(err, false)
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create telegram bot")
	}

	return &Bot{conf: c, baseBot: botAPI, msgHandler: r}, nil
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

	var conversationID int64
	chat := telegramMsg.Chat()
	if chat != nil {
		conversationID = chat.ID
	}

	return &msg.Request{
		Platform: "telegram",
		ID:       fmt.Sprint(telegramMsg.Message().ID),
		Sender:   sender,
		Message:  telegramMsg.Text(),
		Meta: map[string]interface{}{
			"payload":         telegramMsg.Message().Payload,
			"timestamp":       telegramMsg.Message().Unixtime,
			"conversation_id": conversationID,
		},
	}
}

func (b *Bot) guessParseMode(resp *msg.Response) telebot.ParseMode {
	switch resp.Options.GetFormat() {
	case msg.OutputFormatMarkdown1:
		return telebot.ModeMarkdown
	case msg.OutputFormatMarkdown2:
		return telebot.ModeMarkdownV2
	case msg.OutputFormatHTML:
		return telebot.ModeHTML
	case msg.OutputFormatUndefined:
		return telebot.ModeDefault
	default:
		return telebot.ModeDefault
	}
}

func (b *Bot) sendMessageSuccess(
	ctx context.Context,
	telegramMsg telebot.Context,
	resp *msg.Response,
	senderOpts *telebot.SendOptions,
) error {
	log := logging.WithContext(ctx)

	_, err := b.baseBot.Send(telegramMsg.Sender(), resp.Message, senderOpts)
	if err != nil {
		return errors.Wrapf(err, "failed to send success message:\n%s", resp.Message)
	}

	if resp.Options.IsResponseToHiddenMessage() {
		originalMsg := telegramMsg.Message()
		deleteErr := b.baseBot.Delete(originalMsg)
		if deleteErr != nil {
			log.Errorf("failed to delete user message %d: %v", originalMsg.ID, deleteErr)
		} else {
			log.Infof("deleted user message %d as it contained a sensitive data", originalMsg.ID)
		}
	}

	return nil
}

func (b *Bot) processResponseMessage(
	ctx context.Context,
	telegramMsg telebot.Context,
	resp *msg.Response,
) error {
	log := logging.WithContext(ctx)

	if resp == nil || resp.Message == "" {
		log.Info("response message is empty, will send nothing to the sender")
		return nil
	}

	senderOpts := &telebot.SendOptions{
		ParseMode: b.guessParseMode(resp),
	}

	log.Debugf("telegram sender options: %+v", senderOpts)
	log.Debugf("telegram message:\n%q", resp.Message)

	replyButtons := make([]telebot.ReplyButton, 0)
	for _, predefinedResp := range resp.Options.GetPredefinedResponses() {
		if predefinedResp == "" {
			continue
		}
		replyButtons = append(replyButtons, telebot.ReplyButton{
			Text: string(predefinedResp),
		})
	}

	if len(replyButtons) > 0 {
		rm := &telebot.ReplyMarkup{
			OneTimeKeyboard: resp.Options.IsTempPredefinedResponse(),
			ReplyKeyboard: [][]telebot.ReplyButton{
				replyButtons,
			},
			ResizeKeyboard: true,
		}
		senderOpts.ReplyMarkup = rm
	}

	var err error
	switch resp.Type {
	case msg.Error:
		_, err = b.baseBot.Send(
			telegramMsg.Sender(),
			`❗`+resp.Message+`❗`,
			senderOpts,
		)

		if err != nil {
			return errors.Wrapf(err, "failed to send error message: %s", resp.Message)
		}
	case msg.Success:
		return b.sendMessageSuccess(ctx, telegramMsg, resp, senderOpts)
	case msg.Undefined:
		return b.sendMessageSuccess(ctx, telegramMsg, resp, senderOpts)
	default:
		return b.sendMessageSuccess(ctx, telegramMsg, resp, senderOpts)
	}

	return nil
}

func (b *Bot) handle(ctx context.Context, c telebot.Context) error {
	log := logging.WithContext(ctx)

	log.Debugf("got telegram message: %q", c.Text())

	req := b.botMsgToRequest(c)

	resp, err := b.msgHandler.Route(ctx, req)
	if err != nil {
		_, sendErr := b.baseBot.Send(c.Sender(), "Unexpected error", &telebot.SendOptions{})
		if sendErr != nil {
			log.Errorf("failed to send error message to the sender: %v", sendErr)
		}

		return err
	}

	err = b.processResponseMessage(ctx, c, resp)
	if err != nil {
		return err
	}

	return nil
}

func (b *Bot) Start() {
	b.baseBot.Handle(telebot.OnText, func(c telebot.Context) error {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		return b.handle(ctx, c)
	})

	b.baseBot.Handle(&telebot.InlineButton{
		Unique: "",
	}, func(c telebot.Context) error {
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
