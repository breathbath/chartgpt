package telegram

import (
	"breathbathChatGPT/pkg/errs"
	"breathbathChatGPT/pkg/logging"
	"breathbathChatGPT/pkg/msg"
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/telebot.v3"
	"gorm.io/gorm"
	"strings"
)

type Bot struct {
	conf       *Config
	baseBot    *telebot.Bot
	msgHandler *msg.Router
	dbConn     *gorm.DB
}

func NewBot(c *Config, r *msg.Router, dbConn *gorm.DB) (*Bot, error) {
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

	return &Bot{conf: c, baseBot: botAPI, msgHandler: r, dbConn: dbConn}, nil
}

func (b *Bot) botMsgToRequest(ctx context.Context, telegramCtx telebot.Context) (*msg.Request, error) {
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

	callback := telegramCtx.Callback()
	if callback != nil {
		req.ID = callback.MessageID
		meta["callback"] = map[string]interface{}{
			"MessageID": callback.MessageID,
			"Message":   callback.Data,
			"Unique":    callback.Unique,
			"Data":      callback.Data,
		}
		req.Message = strings.TrimLeft(callback.Data, "\f")
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

func (b *Bot) guessParseMode(resp *msg.ResponseMessage) telebot.ParseMode {
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
	resp *msg.ResponseMessage,
	senderOpts *telebot.SendOptions,
) error {
	log := logrus.WithContext(ctx)

	if resp.Media != nil && resp.Media.IsBeforeMessage {
		senderOptsMedia := &telebot.SendOptions{
			ParseMode: senderOpts.ParseMode,
		}
		err := b.sendMedia(ctx, telegramMsg, resp, senderOptsMedia)
		if err != nil {
			return errors.Wrapf(err, "failed to send media:\n%+v", resp.Media)
		}
	}

	if resp.Message != "" {
		_, err := b.baseBot.Send(telegramMsg.Sender(), resp.Message, senderOpts)
		if err != nil {
			return errors.Wrapf(err, "failed to send success message:\n%s", resp.Message)
		}
	}

	if resp.Media != nil && !resp.Media.IsBeforeMessage {
		err := b.sendMedia(ctx, telegramMsg, resp, senderOpts)
		if err != nil {
			return errors.Wrapf(err, "failed to send media:\n%+v", resp.Media)
		}
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

func (b *Bot) sendMedia(
	ctx context.Context,
	telegramMsg telebot.Context,
	resp *msg.ResponseMessage,
	senderOpts *telebot.SendOptions,
) error {
	log := logrus.WithContext(ctx)
	if resp.Media == nil {
		return nil
	}

	var mediaFile *telebot.File
	if resp.Media.Path != "" {
		if resp.Media.PathType == msg.MediaPathTypeFile {
			f := telebot.FromDisk(resp.Media.Path)
			mediaFile = &f
		} else if resp.Media.PathType == msg.MediaPathTypeUrl {
			f := telebot.FromURL(resp.Media.Path)
			mediaFile = &f
		} else {
			log.Infof("Unknown media path type %q", resp.Media.PathType)
		}
	}

	var sendable telebot.Sendable
	if mediaFile != nil {
		if resp.Media.Type == msg.MediaTypeImage {
			sendable = &telebot.Photo{File: *mediaFile}
		} else {
			log.Infof("Unknown media type %q", resp.Media.Type)
		}
	}

	if sendable == nil {
		log.Info("Nothing to send from media")
		return nil
	}

	_, err := b.baseBot.Send(telegramMsg.Sender(), sendable, senderOpts)
	if err != nil {
		return errors.Wrap(err, "failed to send media")
	}

	log.Infof("Successfully sent media %+v", resp.Media)

	return nil
}

func (b *Bot) processResponseMessage(
	ctx context.Context,
	telegramMsg telebot.Context,
	resp *msg.ResponseMessage,
) error {
	log := logrus.WithContext(ctx)

	if resp == nil || (resp.Message == "" && resp.Media == nil) {
		log.Info("response message is empty, will send nothing to the sender")
		return nil
	}

	senderOpts := &telebot.SendOptions{
		ParseMode: b.guessParseMode(resp),
	}

	log.Debugf("telegram sender options: %+v", senderOpts)
	log.Debugf("telegram message:\n%q", resp.Message)
	log.Debugf("telegram media:\n%+v", resp.Media)

	replyButtonsGroups := [][]telebot.ReplyButton{}
	inlineButtonGroups := [][]telebot.InlineButton{}
	replyButtonsGroup := []telebot.ReplyButton{}
	inlineButtonGroup := []telebot.InlineButton{}

	for _, predefinedResp := range resp.Options.GetPredefinedResponses() {
		if predefinedResp.Type == msg.PredefinedResponseInline {
			inlineButtonGroup = append(inlineButtonGroup, telebot.InlineButton{Text: predefinedResp.Text, Data: predefinedResp.Data})
		} else {
			replyButtonsGroup = append(replyButtonsGroup, telebot.ReplyButton{Text: predefinedResp.Text})
		}

		if len(inlineButtonGroup) == 3 {
			inlineButtonGroups = append(inlineButtonGroups, inlineButtonGroup)
			inlineButtonGroup = []telebot.InlineButton{}
		}

		if len(replyButtonsGroup) == 3 {
			replyButtonsGroups = append(replyButtonsGroups, replyButtonsGroup)
			replyButtonsGroup = []telebot.ReplyButton{}
		}
	}

	if len(inlineButtonGroup) > 0 {
		inlineButtonGroups = append(inlineButtonGroups, inlineButtonGroup)
	}

	if len(replyButtonsGroup) > 0 {
		replyButtonsGroups = append(replyButtonsGroups, replyButtonsGroup)
	}

	if len(replyButtonsGroups) > 0 {
		rm := &telebot.ReplyMarkup{
			OneTimeKeyboard: resp.Options.IsTempPredefinedResponse(),
			ReplyKeyboard:   replyButtonsGroups,
			ResizeKeyboard:  true,
		}
		senderOpts.ReplyMarkup = rm
	}

	if len(inlineButtonGroups) > 0 {
		rm := &telebot.ReplyMarkup{
			OneTimeKeyboard: resp.Options.IsTempPredefinedResponse(),
			InlineKeyboard:  inlineButtonGroups,
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

func (b *Bot) handle(ctx context.Context, telegramContext telebot.Context) error {
	log := logrus.WithContext(ctx)

	log.Debugf("got telegram message: %q", telegramContext.Text())

	req, err := b.botMsgToRequest(ctx, telegramContext)
	if err != nil {
		return err
	}

	resp, err := b.msgHandler.Route(ctx, req)
	if err != nil {
		_, sendErr := b.baseBot.Send(telegramContext.Sender(), "Unexpected error", &telebot.SendOptions{})
		if sendErr != nil {
			log.Errorf("failed to send error message to the sender: %v", sendErr)
		}

		return err
	}

	for _, respMsg := range resp.Messages {
		err = b.processResponseMessage(ctx, telegramContext, &respMsg)
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

		ctxT := logging.WithTrackingId(ctx)

		err := b.handle(ctxT, c)
		if err != nil {
			return err
		}

		return nil
	})

	b.baseBot.Handle(telebot.OnVoice, func(c telebot.Context) error {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		ctxT := logging.WithTrackingId(ctx)
		err := b.handle(ctxT, c)
		if err != nil {
			return err
		}
		return nil
	})

	b.baseBot.Handle(telebot.OnCallback, func(c telebot.Context) error {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		ctxT := logging.WithTrackingId(ctx)
		err := b.handle(ctxT, c)
		if err != nil {
			return err
		}
		return nil
	})

	b.baseBot.Handle(&telebot.InlineButton{
		Unique: "",
	}, func(c telebot.Context) error {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		ctxT := logging.WithTrackingId(ctx)

		return b.handle(ctxT, c)
	})

	b.baseBot.Start()
}

func (b *Bot) Stop() {
	logrus.Info("will stop telegram bot")
	b.baseBot.Stop()
	logrus.Info("stopped telegram bot")
}
