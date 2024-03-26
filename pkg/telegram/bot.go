package telegram

import (
	"breathbathChatGPT/pkg/errs"
	"breathbathChatGPT/pkg/logging"
	"breathbathChatGPT/pkg/msg"
	"breathbathChatGPT/pkg/storage"
	"context"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/telebot.v3"
	"gorm.io/gorm"
	"strings"
)

type SendCtx struct {
	Receiver     string
	ReceiverName string
	MessageID    string
	ChatID       int64
}

func (sc SendCtx) Recipient() string {
	return sc.Receiver
}

func (sc SendCtx) Validate() error {
	ers := []string{}
	if sc.Receiver == "" {
		ers = append(ers, "Empty receiver info")
	}
	if sc.MessageID == "" {
		ers = append(ers, "Empty message ID")
	}
	if sc.ChatID == 0 {
		ers = append(ers, "Empty chat ID")
	}

	if len(ers) == 0 {
		return nil
	}

	return errors.New(strings.Join(ers, ". "))
}

func (sc SendCtx) MessageSig() (messageID string, chatID int64) {
	return sc.MessageID, sc.ChatID
}

type Bot struct {
	conf                 *Config
	baseBot              *telebot.Bot
	msgHandler           *msg.Router
	dbConn               *gorm.DB
	delayedMessageSender *DelayedMessageSender
}

func NewBot(c *Config, r *msg.Router, dbConn *gorm.DB, cache storage.Client, delayedMessagesCallback func(input json.RawMessage) ([]msg.ResponseMessage, error)) (*Bot, error) {
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

	b := &Bot{conf: c, baseBot: botAPI, msgHandler: r, dbConn: dbConn}
	delayedMessageSender := NewDelayedMessageSender(b.processResponseMessage, delayedMessagesCallback, cache)
	b.delayedMessageSender = delayedMessageSender

	return b, nil
}

func (b *Bot) botMsgToRequest(ctx context.Context, telegramCtx telebot.Context) (*msg.Request, error) {
	sender := new(msg.Sender)
	telegramSender := telegramCtx.Sender()
	if telegramSender != nil {
		sender.ID = fmt.Sprint(telegramSender.ID)
		sender.LastName = telegramSender.LastName
		sender.FirstName = telegramSender.FirstName

		if telegramSender.Username != "" {
			sender.UserName = telegramSender.Username
		} else {
			sender.UserName = telegramSender.Recipient()
		}

		sender.Language = telegramSender.LanguageCode
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
	sendCtx SendCtx,
	resp *msg.ResponseMessage,
	senderOpts *telebot.SendOptions,
) error {
	log := logrus.WithContext(ctx)

	if resp.Media != nil && resp.Media.IsBeforeMessage {
		senderOptsMedia := &telebot.SendOptions{
			ParseMode: senderOpts.ParseMode,
		}
		err := b.sendMedia(ctx, sendCtx, resp, senderOptsMedia)
		if err != nil {
			return errors.Wrapf(err, "failed to send media:\n%+v", resp.Media)
		}
	}

	if resp.Message != "" {
		_, err := b.baseBot.Send(sendCtx, resp.Message, senderOpts)
		if err != nil {
			return errors.Wrapf(err, "failed to send success message:\n%s", resp.Message)
		}
	}

	if resp.Media != nil && !resp.Media.IsBeforeMessage {
		err := b.sendMedia(ctx, sendCtx, resp, senderOpts)
		if err != nil {
			return errors.Wrapf(err, "failed to send media:\n%+v", resp.Media)
		}
	}

	if resp.Options.IsResponseToHiddenMessage() {
		deleteErr := b.baseBot.Delete(sendCtx)
		if deleteErr != nil {
			log.Errorf("failed to delete user message %d: %v", sendCtx.MessageID, deleteErr)
		} else {
			log.Infof("deleted user message %d as it contained a sensitive data", sendCtx.MessageID)
		}
	}

	return nil
}

func (b *Bot) sendMedia(
	ctx context.Context,
	senderCtx SendCtx,
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

	_, err := b.baseBot.Send(senderCtx, sendable, senderOpts)
	if err != nil {
		return errors.Wrap(err, "failed to send media")
	}

	log.Infof("Successfully sent media %+v", resp.Media)

	return nil
}

func (b *Bot) processResponseMessage(
	ctx context.Context,
	sendCtx SendCtx,
	resp *msg.ResponseMessage,
) error {
	log := logrus.WithContext(ctx)

	if resp.DelayedOptions != nil {
		log.Debugf(
			"Delayed message %q, timeout: %v, sender: %s, callback: %q",
			resp.Message,
			resp.DelayedOptions.Timeout,
			sendCtx.Receiver,
			string(resp.DelayedOptions.CallbackPayload),
		)

		delayedMessage := &msg.ResponseMessage{
			Message: resp.Message,
			Type:    resp.Type,
			Options: resp.Options,
			Media:   resp.Media,
		}
		delayedMessageCtx := DelayedMessageCtx{
			Message:              delayedMessage,
			SendCtx:              sendCtx,
			Timeout:              resp.DelayedOptions.Timeout,
			Key:                  fmt.Sprintf("%s-%s", resp.DelayedOptions.Key, sendCtx.Receiver),
			DelayedCallbackInput: resp.DelayedOptions.CallbackPayload,
		}

		if resp.DelayedOptions.DelayType == msg.DelayTypeMessage {
			delayedMessageCtx.DelayType = DelayTypeMessage
		} else if resp.DelayedOptions.DelayType == msg.DelayTypeCallback {
			delayedMessageCtx.DelayType = DelayTypeCallback
		}

		b.delayedMessageSender.Plan(delayedMessageCtx)

		return nil
	}

	if resp == nil || (resp.Message == "" && resp.Media == nil) {
		log.Info("response message is empty, will send nothing to the sender")
		return nil
	}

	senderOpts := &telebot.SendOptions{
		ParseMode: b.guessParseMode(resp),
	}

	log.Debugf("telegram sender options: %+v", senderOpts)
	log.Debugf("telegram message:\n%q", resp.Message)

	replyButtonsGroups := [][]telebot.ReplyButton{}
	inlineButtonGroups := [][]telebot.InlineButton{}
	replyButtonsGroup := []telebot.ReplyButton{}
	inlineButtonGroup := []telebot.InlineButton{}

	for _, predefinedResp := range resp.Options.GetPredefinedResponses() {
		if predefinedResp.Type == msg.PredefinedResponseInline {
			button := telebot.InlineButton{
				Text: predefinedResp.Text,
			}
			if predefinedResp.Link != "" {
				button.URL = predefinedResp.Link
			}
			if predefinedResp.Data != "" {
				button.Data = predefinedResp.Data
			}
			inlineButtonGroup = append(inlineButtonGroup, button)
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
			sendCtx,
			`❗`+resp.Message+`❗`,
			senderOpts,
		)

		if err != nil {
			return errors.Wrapf(err, "failed to send error message: %s", resp.Message)
		}
	case msg.Success:
		return b.sendMessageSuccess(ctx, sendCtx, resp, senderOpts)
	case msg.Undefined:
		return b.sendMessageSuccess(ctx, sendCtx, resp, senderOpts)
	default:
		return b.sendMessageSuccess(ctx, sendCtx, resp, senderOpts)
	}

	return nil
}

func (b *Bot) handle(ctx context.Context, telegramContext telebot.Context) error {
	log := logrus.WithContext(ctx)

	sender := telegramContext.Sender()
	log.Debugf("got telegram message: %q from %+v", telegramContext.Text(), sender)
	if sender.IsBot {
		log.Warn("A bot conversation is detected, skipping talking to bot")
		return nil
	}

	sendCtx := b.TelegramContextToSendCtx(telegramContext)
	err := sendCtx.Validate()
	if err != nil {
		log.Error(err)
		return nil
	}

	req, err := b.botMsgToRequest(ctx, telegramContext)
	if err != nil {
		return err
	}

	log.Debugf("resetting delayed conversation with %q", telegramContext.Sender().Recipient())
	b.delayedMessageSender.Reset(telegramContext.Sender().Recipient())
	b.delayedMessageSender.Reset(fmt.Sprintf("%s-%s", "delayed_like", telegramContext.Sender().Recipient()))
	b.delayedMessageSender.Reset(fmt.Sprintf("%s-%s", "delayed_recommend", telegramContext.Sender().Recipient()))

	resp, err := b.msgHandler.Route(ctx, req)
	if err != nil {
		_, sendErr := b.baseBot.Send(telegramContext.Sender(), "Unexpected error", &telebot.SendOptions{})
		if sendErr != nil {
			log.Errorf("failed to send error message to the sender: %v", sendErr)
		}

		return err
	}

	for _, respMsg := range resp.Messages {
		sendCtx := b.TelegramContextToSendCtx(telegramContext)
		err = b.processResponseMessage(ctx, sendCtx, &respMsg)
		if err != nil {
			return err
		}
	}

	return nil
}

func (b *Bot) TelegramContextToSendCtx(telegramContext telebot.Context) SendCtx {
	messageID, chatID := telegramContext.Message().MessageSig()
	return SendCtx{
		Receiver:  telegramContext.Sender().Recipient(),
		MessageID: messageID,
		ChatID:    chatID,
		ReceiverName: fmt.Sprintf(
			"%s %s %s",
			telegramContext.Sender().Username,
			telegramContext.Sender().FirstName,
			telegramContext.Sender().LastName,
		),
	}
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
