package telegram

import (
	"breathbathChatGPT/pkg/errs"
	"breathbathChatGPT/pkg/msg"
	"context"
	"fmt"
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

func (b *Bot) botMsgToRequest(telegramCtx telebot.Context) (*msg.Request, error) {
	sender := new(msg.Sender)
	telegramSender := telegramCtx.Sender()
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
	chat := telegramCtx.Chat()
	if chat != nil {
		conversationID = chat.ID
	}

	telegramMsg := telegramCtx.Message()
	if telegramMsg == nil {
		return nil, errors.New("no message received")
	}

	meta := map[string]interface{}{
		"payload":         telegramMsg.Payload,
		"timestamp":       telegramMsg.Unixtime,
		"conversation_id": conversationID,
	}

	req := &msg.Request{
		Platform: "telegram",
		ID:       fmt.Sprint(telegramCtx.Message().ID),
		Sender:   sender,
		Message:  telegramCtx.Text(),
		Meta:     meta,
	}
	if telegramMsg.Voice != nil {
		voiceFile := telegramMsg.Voice.MediaFile()
		reader, err := b.baseBot.File(voiceFile)
		if err != nil {
			return nil, err
		}

		req.File = msg.File{
			FileID:     voiceFile.FileID,
			UniqueID:   voiceFile.UniqueID,
			FileSize:   voiceFile.FileSize,
			FilePath:   voiceFile.FilePath,
			FileLocal:  voiceFile.FileLocal,
			FileURL:    voiceFile.FileURL,
			FileReader: reader,
			Format:     msg.FormatVoice,
		}
	}

	return req, nil
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

	replyButtonsGroups := [][]telebot.ReplyButton{}
	for i, predefinedResp := range resp.Options.GetPredefinedResponses() {
		if predefinedResp == "" {
			continue
		}
		if i%3 == 0 {
			replyButtonsGroups = append(replyButtonsGroups, []telebot.ReplyButton{})
		}

		lastGroupIndex := len(replyButtonsGroups) - 1
		replyButtonsGroups[lastGroupIndex] = append(
			replyButtonsGroups[lastGroupIndex],
			telebot.ReplyButton{Text: string(predefinedResp)},
		)
	}

	if len(replyButtonsGroups) > 0 {
		rm := &telebot.ReplyMarkup{
			OneTimeKeyboard: resp.Options.IsTempPredefinedResponse(),
			ReplyKeyboard:   replyButtonsGroups,
			ResizeKeyboard:  true,
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

	req, err := b.botMsgToRequest(c)
	if err != nil {
		return err
	}

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

	b.baseBot.Handle(telebot.OnVoice, func(c telebot.Context) error {
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
