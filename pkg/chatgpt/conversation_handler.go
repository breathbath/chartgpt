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
		command:       "/setsm",
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

func (sc *SetConversationContextHandler) Handle(ctx context.Context, req *msg.Request) (*msg.Response, error) {
	log := logrus.WithContext(ctx)

	log.Debug("will set conversation context")

	conversationContextText := utils.ExtractCommandValue(req.Message, sc.command)
	if conversationContextText == "" {
		return &msg.Response{
			Messages: []msg.ResponseMessage{
				{
					Message: "empty conversation context provided",
					Type:    msg.Error,
				},
			},
		}, nil
	}

	conversationContext := &Context{
		Message:            conversationContextText,
		CreatedAtTimestamp: time.Now().Unix(),
	}

	conversationContextKey := getConversationContextKey(req)
	err := sc.db.Save(ctx, conversationContextKey, conversationContext, 0)
	if err != nil {
		return nil, err
	}

	log.Debugf("Saved conversation context: %q", conversationContext)

	return &msg.Response{
		Messages: []msg.ResponseMessage{
			{
				Message: fmt.Sprintf("Saved system message %q", conversationContextText),
				Type:    msg.Success,
			},
		},
	}, nil
}

func (sc *SetConversationContextHandler) GetHelp(context.Context, *msg.Request) help.Result {
	text := fmt.Sprintf(
		"%s #text#: to set context for the current conversation (see setting system role message ",
		sc.command,
	)

	return help.Result{Text: text}
}

type GetConversationContextHandler struct {
	db            storage.Client
	command       string
	adminDetector func(req *msg.Request) bool
}

func NewGetConversationContextCommand(
	db storage.Client,
	adminDetector func(req *msg.Request) bool,
) *GetConversationContextHandler {
	return &GetConversationContextHandler{
		db:            db,
		command:       "/getsm",
		adminDetector: adminDetector,
	}
}

func (sc *GetConversationContextHandler) CanHandle(_ context.Context, req *msg.Request) (bool, error) {
	if !strings.HasPrefix(req.Message, sc.command) {
		return false, nil
	}

	if !sc.adminDetector(req) {
		return false, nil
	}

	return true, nil
}

func (sc *GetConversationContextHandler) GetConversationContext(ctx context.Context, req *msg.Request) (*Context, error) {
	log := logrus.WithContext(ctx)

	log.Debug("will get conversation context")
	conversationContextKey := getConversationContextKey(req)
	conversationContext := &Context{}
	found, err := sc.db.Load(ctx, conversationContextKey, conversationContext)
	if err != nil {
		return conversationContext, err
	}

	if found {
		log.Debugf("got overriding context %q under %q", conversationContext.Message, conversationContextKey)
		return conversationContext, nil
	}

	log.Debugf("using default context %q", SystemMessage)

	conversationContext.Message = SystemMessage
	conversationContext.CreatedAtTimestamp = time.Now().UTC().Unix()

	return conversationContext, nil
}

func (sc *GetConversationContextHandler) Handle(ctx context.Context, req *msg.Request) (*msg.Response, error) {
	log := logrus.WithContext(ctx)

	log.Debug("will get conversation context")

	messages := []string{
		fmt.Sprintf("GLOBAL SYSTEM MESSAGE:\n%s", SystemMessage),
	}
	overridingConversationContext, err := sc.GetConversationContext(ctx, req)
	if err != nil {
		return nil, err
	}

	overriddingContext := overridingConversationContext.GetMessage()
	if overriddingContext != SystemMessage {
		messages = append(messages, fmt.Sprintf("OVERRIDDING SYSTEM MESSAGE:\n%s", overriddingContext))
	}
	log.Debugf("Got conversation contexts: %+v", messages)

	return &msg.Response{
		Messages: []msg.ResponseMessage{
			{
				Message: strings.Join(messages, "\n"),
				Type:    msg.Success,
			},
		},
	}, nil
}

func (sc *GetConversationContextHandler) GetHelp(context.Context, *msg.Request) help.Result {
	text := fmt.Sprintf(
		"%s: to get context for the current conversation (global and Overridding)",
		sc.command,
	)

	return help.Result{Text: text}
}

type ResetConversationHandler struct {
	command          string
	db               storage.Client
	isScopedModeFunc func() bool
	adminDetector    func(req *msg.Request) bool
}

func NewResetConversationHandler(
	db storage.Client,
	isScopedModeFunc func() bool,
	adminDetector func(req *msg.Request) bool,
) *ResetConversationHandler {
	return &ResetConversationHandler{
		command:          "/reset",
		db:               db,
		isScopedModeFunc: isScopedModeFunc,
		adminDetector:    adminDetector,
	}
}

func (sc *ResetConversationHandler) CanHandle(_ context.Context, req *msg.Request) (bool, error) {
	if !utils.MatchesCommand(req.Message, sc.command) {
		return false, nil
	}

	return true, nil
}

func (sc *ResetConversationHandler) Handle(ctx context.Context, req *msg.Request) (*msg.Response, error) {
	log := logrus.WithContext(ctx)

	conversationKey := getConversationKey(req)
	err := sc.db.Delete(ctx, conversationKey)
	if err != nil {
		return nil, err
	}

	log.Debugf("deleted conversation under %q", conversationKey)
	if sc.adminDetector(req) {
		conversationContextKey := getConversationContextKey(req)
		err := sc.db.Delete(ctx, conversationContextKey)
		if err != nil {
			return nil, err
		}
		log.Debugf("deleted conversation context under %q", conversationContextKey)
	}

	return &msg.Response{
		Messages: []msg.ResponseMessage{
			{
				Message: "Reset success",
				Type:    msg.Success,
			},
		},
	}, nil
}

func (sc *ResetConversationHandler) GetHelp(context.Context, *msg.Request) help.Result {
	text := fmt.Sprintf("%s: to reset your conversation history", sc.command)

	return help.Result{Text: text}
}
