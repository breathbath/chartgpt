package telegram

import (
	"breathbathChatGPT/pkg/msg"
	"context"
	"github.com/sirupsen/logrus"
	"gopkg.in/telebot.v3"
	"sync"
	"time"
)

type DelayedMessageCtx struct {
	Ctx         context.Context
	Message     *msg.ResponseMessage
	TelegramCtx telebot.Context
	Timeout     time.Duration
	dueTime     time.Time
}

type DelayedMessageSender struct {
	inputs   chan DelayedMessageCtx
	resets   chan string
	sendFunc func(ctx context.Context, telegramMsg telebot.Context, resp *msg.ResponseMessage) error
}

func NewDelayedMessageSender(
	sendFunc func(ctx context.Context, telegramMsg telebot.Context, resp *msg.ResponseMessage) error,
) *DelayedMessageSender {
	de := &DelayedMessageSender{
		inputs:   make(chan DelayedMessageCtx, 10),
		resets:   make(chan string, 10),
		sendFunc: sendFunc,
	}
	de.start()

	return de
}

func (de *DelayedMessageSender) start() {
	defer logrus.Debug("started delayed message sender")
	data := &sync.Map{}

	go func() {
		for ipt := range de.inputs {
			ipt.dueTime = time.Now().UTC().Add(ipt.Timeout)
			data.Store(
				ipt.TelegramCtx.Sender().Recipient(),
				ipt,
			)
		}
	}()

	go func() {
		for r := range de.resets {
			data.Delete(r)
		}
	}()

	go func() {
		for {
			data.Range(func(key, value any) bool {
				delayedMessage := value.(DelayedMessageCtx)
				if delayedMessage.dueTime.Before(time.Now().UTC()) {
					go func() {
						err := de.sendFunc(delayedMessage.Ctx, delayedMessage.TelegramCtx, delayedMessage.Message)
						log := logrus.WithContext(delayedMessage.Ctx)
						if err != nil {
							log.Errorf("failed to process delayed message %+v: %v", delayedMessage, err)
						}
						log.Debugf("processed delayed message %q", delayedMessage.Message.Message)
					}()
					data.Delete(key)
				}
				return true
			})
			time.Sleep(time.Second)
		}
	}()
}

func (de *DelayedMessageSender) Plan(execCtx DelayedMessageCtx) {
	de.inputs <- execCtx
}

func (de *DelayedMessageSender) Reset(id string) {
	de.resets <- id
}
