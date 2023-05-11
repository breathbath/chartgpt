package chatgpt

import (
	"breathbathChatGPT/pkg/msg"
	"breathbathChatGPT/pkg/storage"
	"breathbathChatGPT/pkg/utils"
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"time"
)

const defaultConversationValidity = time.Minute * 30

type SetConversationContextCommand struct {
	operators []string
	db        storage.Client
}

func NewSetConversationContextCommand(db storage.Client) *SetConversationContextCommand {
	return &SetConversationContextCommand{
		operators: []string{"mode", "context"},
		db:        db,
	}
}

func (sc *SetConversationContextCommand) CanHandle(ctx context.Context, req *msg.Request) bool {
	return utils.MatchesAny(req.Message, CommandPrefix, sc.operators)
}

func getConversationKey(req *msg.Request) string {
	return "chatgpt/conversation/" + req.GetConversationId()
}

func (sc *SetConversationContextCommand) Handle(ctx context.Context, req *msg.Request) (*msg.Response, error) {
	log := logrus.WithContext(ctx)

	log.Debug("will set conversation context")

	conversationContext := extractCommandValue(req.Message, sc.operators)
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

func (sc *SetConversationContextCommand) GetHelp() string {
	operatorHelp := buildHelpFromOperators(sc.operators)

	return fmt.Sprintf("%s #text#: to set context for the current conversation (see setting system role message https://platform.openai.com/docs/guides/chat/introduction)", operatorHelp)
}

type ResetConversationCommand struct {
	operators []string
	db        storage.Client
}

func NewResetConversationCommand(db storage.Client) *ResetConversationCommand {
	return &ResetConversationCommand{
		operators: []string{"reset"},
		db:        db,
	}
}

func (sc *ResetConversationCommand) CanHandle(ctx context.Context, req *msg.Request) bool {
	return utils.MatchesAny(req.Message, CommandPrefix, sc.operators)
}

func (sc *ResetConversationCommand) Handle(ctx context.Context, req *msg.Request) (*msg.Response, error) {
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

func (sc *ResetConversationCommand) GetHelp() string {
	operatorHelp := buildHelpFromOperators(sc.operators)

	return fmt.Sprintf("%s: to reset your conversation", operatorHelp)
}
