package telegram

import (
	"breathbathChatGPT/pkg/msg"
	"breathbathChatGPT/pkg/storage"
	"context"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"time"
)

const DelayedKeysPrefix = "delayed_"

type DelayType uint

const (
	DelayTypeMessage = iota + 1
	DelayTypeCallback
)

type DelayedMessageCtx struct {
	Message              *msg.ResponseMessage
	SendCtx              SendCtx
	Timeout              time.Duration
	DueTime              time.Time
	Key                  string
	DelayedCallbackInput json.RawMessage
	DelayType            DelayType
}

type DelayedMessageSender struct {
	inputs   chan DelayedMessageCtx
	resets   chan string
	sendFunc func(ctx context.Context, sendCtx SendCtx, resp *msg.ResponseMessage) error
	callback func(input json.RawMessage) ([]msg.ResponseMessage, error)
	cache    storage.Client
}

func NewDelayedMessageSender(
	sendFunc func(ctx context.Context, sendCtx SendCtx, resp *msg.ResponseMessage) error,
	callback func(input json.RawMessage) ([]msg.ResponseMessage, error),
	cache storage.Client,
) *DelayedMessageSender {
	de := &DelayedMessageSender{
		inputs:   make(chan DelayedMessageCtx, 10),
		resets:   make(chan string, 10),
		sendFunc: sendFunc,
		cache:    cache,
		callback: callback,
	}
	de.start()

	return de
}

func (de *DelayedMessageSender) buildDelayedKey(key string) string {
	return fmt.Sprintf("%s%s", DelayedKeysPrefix, key)
}

func (de *DelayedMessageSender) start() {
	defer logrus.Debug("started delayed message sender")

	go func() {
		for ipt := range de.inputs {
			ipt.DueTime = time.Now().UTC().Add(ipt.Timeout)
			err := de.cache.Save(context.Background(), de.buildDelayedKey(ipt.Key), ipt, time.Hour*240)
			if err != nil {
				logrus.Errorf("failed to store delayed data under %s: %v", ipt.Key, err)
			}
		}
	}()

	go func() {
		for r := range de.resets {
			err := de.cache.Delete(context.Background(), de.buildDelayedKey(r))
			if err != nil {
				logrus.Errorf("Failed to delete key %s from cache: %v", r, err)
			}
		}
	}()

	go func() {
		for {
			time.Sleep(10 * time.Second)

			keys, err := de.cache.FindKeys(context.Background(), DelayedKeysPrefix+"*")
			if err != nil {
				logrus.Errorf("Failed to read keys %s from cache: %v", DelayedKeysPrefix+"*", err)
				continue
			}

			for _, key := range keys {
				var delayedMessage DelayedMessageCtx
				_, err := de.cache.Load(context.Background(), key, &delayedMessage)
				if err != nil {
					logrus.Errorf("failed to load delayed context data under key %q: %v", key, err)
					continue
				}

				if delayedMessage.DueTime.Before(time.Now().UTC()) {
					if delayedMessage.DelayType == DelayTypeMessage {
						go func() {
							err := de.sendFunc(context.Background(), delayedMessage.SendCtx, delayedMessage.Message)
							if err != nil {
								logrus.Errorf("failed to process delayed message %+v: %v", delayedMessage, err)
							} else {
								logrus.Debugf("processed delayed message %q", delayedMessage.Message.Message)
							}
						}()
					} else if delayedMessage.DelayType == DelayTypeCallback {
						go de.processDelayedCallback(delayedMessage)
					}

					err = de.cache.Delete(context.Background(), key)
					if err != nil {
						logrus.Error("failed to delete cached data under key %q: %v", key, err)
						continue
					}
				}
			}
		}
	}()
}

func (de *DelayedMessageSender) processDelayedCallback(delayedMessage DelayedMessageCtx) {
	respMessages, err := de.callback(delayedMessage.DelayedCallbackInput)
	if err != nil {
		logrus.Errorf("delayed callback failed: %v", err)
		return
	}

	for i := range respMessages {
		err := de.sendFunc(context.Background(), delayedMessage.SendCtx, &respMessages[i])
		if err != nil {
			logrus.Errorf("failed to process delayed message %q: %v", string(delayedMessage.DelayedCallbackInput), err)
		} else {
			logrus.Debugf("processed delayed message %q", string(delayedMessage.DelayedCallbackInput))
		}
	}
}

func (de *DelayedMessageSender) Plan(execCtx DelayedMessageCtx) {
	de.inputs <- execCtx
}

func (de *DelayedMessageSender) Reset(id string) {
	de.resets <- id
}
