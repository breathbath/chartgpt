package chatgpt

import (
	"context"
	"fmt"
	"strings"
	"time"

	"breathbathChatGPT/pkg/help"

	"breathbathChatGPT/pkg/msg"
	"breathbathChatGPT/pkg/storage"
	"breathbathChatGPT/pkg/utils"

	"github.com/sirupsen/logrus"
)

const (
	defaultConversationValidity = time.Minute * 30
	conversationVersion         = "v1"
)

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

func (sc *SetConversationContextHandler) CanHandle(_ context.Context, req *msg.Request) (bool, error) {
	return strings.HasPrefix(req.Message, sc.command), nil
}

func getConversationKey(req *msg.Request) string {
	return storage.GenerateCacheKey(conversationVersion, "chatgpt", "conversation", req.GetConversationID())
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

	conversation.Context = &Context{
		Message:            conversationContext,
		CreatedAtTimestamp: time.Now().Unix(),
	}
	log.Debugf("Going to save conversation context: %q", conversationContext)
	conversation.ID = req.GetConversationID()

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

func (sc *SetConversationContextHandler) GetHelp(context.Context, *msg.Request) help.Result {
	text := fmt.Sprintf(
		"%s #text#: to set context for the current conversation (see setting system role message "+
			"https://platform.openai.com/docs/guides/chat/introduction)",
		sc.command,
	)

	return help.Result{Text: text}
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

func (sc *ResetConversationHandler) CanHandle(_ context.Context, req *msg.Request) (bool, error) {
	return utils.MatchesCommand(req.Message, sc.command), nil
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

func (sc *ResetConversationHandler) GetHelp(context.Context, *msg.Request) help.Result {
	text := fmt.Sprintf("%s: to reset your conversation", sc.command)

	return help.Result{Text: text, PredefinedOption: sc.command}
}
