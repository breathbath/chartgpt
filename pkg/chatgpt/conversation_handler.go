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
	db            storage.Client
	command       string
	isScopedMode  func() bool
	adminDetector func(req *msg.Request) bool
}

func NewSetConversationContextCommand(
	db storage.Client,
	isScopedMode func() bool,
	adminDetector func(req *msg.Request) bool,
) *SetConversationContextHandler {
	return &SetConversationContextHandler{
		db:            db,
		command:       "/context",
		isScopedMode:  isScopedMode,
		adminDetector: adminDetector,
	}
}

func (sc *SetConversationContextHandler) CanHandle(_ context.Context, req *msg.Request) (bool, error) {
	if !strings.HasPrefix(req.Message, sc.command) {
		return false, nil
	}

	if sc.isScopedMode() && !sc.adminDetector(req) {
		return false, nil
	}

	return true, nil
}

func getConversationKey(req *msg.Request) string {
	return storage.GenerateCacheKey(conversationVersion, "chatgpt", "conversation", req.GetConversationID())
}

func getConversationContextKey(req *msg.Request) string {
	return storage.GenerateCacheKey(conversationVersion, "chatgpt", "conversation_context", req.GetConversationID())
}

func getGlobalConversationContextKey() string {
	return storage.GenerateCacheKey(conversationVersion, "chatgpt", "conversation_context_glob")
}

func (sc *SetConversationContextHandler) Handle(ctx context.Context, req *msg.Request) (*msg.Response, error) {
	log := logrus.WithContext(ctx)

	log.Debug("will set conversation context")

	conversationContextText := utils.ExtractCommandValue(req.Message, sc.command)
	if conversationContextText == "" {
		return &msg.Response{
			Message: "empty conversation context provided",
			Type:    msg.Error,
		}, nil
	}

	conversationContext := &Context{
		Message:            conversationContextText,
		CreatedAtTimestamp: time.Now().Unix(),
	}

	if sc.isScopedMode() && sc.adminDetector(req) {
		conversationContextKey := getGlobalConversationContextKey()
		err := sc.db.Save(ctx, conversationContextKey, conversationContext, 0)
		if err != nil {
			return nil, err
		}
		return &msg.Response{
			Message: fmt.Sprintf("Remembered conversation context %q", conversationContextText),
			Type:    msg.Success,
		}, nil
	}

	conversationContextKey := getConversationContextKey(req)
	err := sc.db.Save(ctx, conversationContextKey, conversationContext, defaultConversationValidity)
	if err != nil {
		return nil, err
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

	log.Debugf("Going to save conversation context: %q", conversationContextText)
	conversation.ID = req.GetConversationID()

	err = sc.db.Save(ctx, cacheKey, conversation, defaultConversationValidity)
	if err != nil {
		return nil, err
	}

	log.Debugf("Saved conversation context: %q", conversationContext)
	return &msg.Response{
		Message: fmt.Sprintf("Remembered conversation context %q", conversationContextText),
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
	command       string
	db            storage.Client
	modeDetector  func() bool
	adminDetector func(req *msg.Request) bool
}

func NewResetConversationHandler(
	db storage.Client,
	modeDetector func() bool,
	adminDetector func(req *msg.Request) bool,
) *ResetConversationHandler {
	return &ResetConversationHandler{
		command:       "/reset",
		db:            db,
		modeDetector:  modeDetector,
		adminDetector: adminDetector,
	}
}

func (sc *ResetConversationHandler) CanHandle(_ context.Context, req *msg.Request) (bool, error) {
	if !utils.MatchesCommand(req.Message, sc.command) {
		return false, nil
	}

	if sc.modeDetector() && !sc.adminDetector(req) {
		return false, nil
	}

	return true, nil
}

func (sc *ResetConversationHandler) Handle(ctx context.Context, req *msg.Request) (*msg.Response, error) {
	log := logrus.WithContext(ctx)

	log.Debug("will reset conversation")

	cacheKey := getConversationKey(req)
	err := sc.db.Delete(ctx, cacheKey)
	if err != nil {
		return nil, err
	}

	if !sc.modeDetector() {
		conversationContextKey := getGlobalConversationContextKey()
		err := sc.db.Delete(ctx, conversationContextKey)
		if err != nil {
			return nil, err
		}
		return &msg.Response{
			Message: "Successfully reset your current conversation",
			Type:    msg.Success,
		}, nil
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
