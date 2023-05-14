package chatgpt

import (
	"breathbathChatGPT/pkg/msg"
	"breathbathChatGPT/pkg/storage"
	"breathbathChatGPT/pkg/utils"
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"strings"
	"time"
)

const defaultConversationValidity = time.Minute * 30

type SetConversationContextHandler struct {
	db      storage.Client
	command string
}

func NewSetConversationContextCommand(db storage.Client) *SetConversationContextHandler {
	return &SetConversationContextHandler{
		db:      db,
		command: "/context",
	}
}

func (sc *SetConversationContextHandler) CanHandle(ctx context.Context, req *msg.Request) (bool, error) {
	return strings.HasPrefix(req.Message, sc.command), nil
}

func getConversationKey(req *msg.Request) string {
	return "chatgpt/conversation/" + req.GetConversationId()
}

func (sc *SetConversationContextHandler) Handle(ctx context.Context, req *msg.Request) (*msg.Response, error) {
	log := logrus.WithContext(ctx)

	log.Debug("will set conversation context")

	conversationContext := utils.ExtractCommandValue(req.Message, sc.command)
	if conversationContext == "" {
		return &msg.Response{
			Message: "empty conversation context provided",
			Type:    msg.Error,
		}, nil
	}

	cacheKey := getConversationKey(req)
	conversation := new(Conversation)
	found, err := sc.db.Load(ctx, cacheKey, conversation)
	if err != nil {
		return nil, err
	}

	if found {
		conversation.Messages = []ConversationMessage{}
	}

	conversation.Context = conversationContext
	log.Debugf("Going to save conversation context: %q", conversationContext)
	conversation.ID = req.GetConversationId()

	err = sc.db.Save(ctx, cacheKey, conversation, defaultConversationValidity)
	if err != nil {
		return nil, err
	}

	log.Debugf("Saved conversation context: %q", conversationContext)
	return &msg.Response{
		Message: fmt.Sprintf("Remembered conversation context %q", conversationContext),
		Type:    msg.Success,
	}, nil
}

func (sc *SetConversationContextHandler) GetHelp() string {
	return fmt.Sprintf(
		"%s #text#: to set context for the current conversation (see setting system role message https://platform.openai.com/docs/guides/chat/introduction)",
		sc.command,
	)
}

type ResetConversationHandler struct {
	command string
	db      storage.Client
}

func NewResetConversationHandler(db storage.Client) *ResetConversationHandler {
	return &ResetConversationHandler{
		command: "/reset",
		db:      db,
	}
}

func (sc *ResetConversationHandler) CanHandle(ctx context.Context, req *msg.Request) (bool, error) {
	return strings.HasPrefix(req.Message, sc.command), nil
}

func (sc *ResetConversationHandler) Handle(ctx context.Context, req *msg.Request) (*msg.Response, error) {
	log := logrus.WithContext(ctx)

	log.Debug("will reset conversation")

	cacheKey := getConversationKey(req)
	err := sc.db.Delete(ctx, cacheKey)
	if err != nil {
		return nil, err
	}

	log.Debug("successfully reset conversation")

	return &msg.Response{
		Message: "Successfully reset your current conversation",
		Type:    msg.Success,
	}, nil
}

func (sc *ResetConversationHandler) GetHelp() string {
	return fmt.Sprintf("%s: to reset your conversation", sc.command)
}
