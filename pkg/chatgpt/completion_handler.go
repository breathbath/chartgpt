package chatgpt

import (
	"context"
	"strings"
	"time"

	"breathbathChatGPT/pkg/msg"
	"breathbathChatGPT/pkg/rest"
	"breathbathChatGPT/pkg/storage"

	logging "github.com/sirupsen/logrus"
)

const (
	URL                 = "https://api.openai.com"
	CompletionsURL      = URL + "/v1/chat/completions"
	ModelsURL           = URL + "/v1/models"
	ConversationTimeout = time.Minute * 10
)

type ChatCompletionHandler struct {
	cfg            *Config
	settingsLoader *Loader
	db             storage.Client
	isScopedMode   func() bool
}

func NewChatCompletionHandler(
	cfg *Config,
	db storage.Client,
	loader *Loader,
	isScopedMode func() bool,
) (h *ChatCompletionHandler, err error) {
	e := cfg.Validate()
	if e.HasErrors() {
		return nil, e
	}

	return &ChatCompletionHandler{
		cfg:            cfg,
		db:             db,
		settingsLoader: loader,
		isScopedMode:   isScopedMode,
	}, nil
}

func (h *ChatCompletionHandler) buildConversation(ctx context.Context, req *msg.Request) (*Conversation, error) {
	log := logging.WithContext(ctx)

	conversationContext, err := h.buildConversationContext(ctx)
	if err != nil {
		return nil, err
	}

	cacheKey := getConversationKey(req)
	conversation := new(Conversation)
	found, err := h.db.Load(ctx, cacheKey, conversation)
	if err != nil {
		return nil, err
	}

	if !found || h.isConversationOutdated(conversation) {
		log.Debug("the conversation is not found or outdated, will start a new conversation")
		return &Conversation{ID: req.GetConversationID(), Context: conversationContext}, nil
	}

	conversation.Context = conversationContext

	return conversation, nil
}

func (h *ChatCompletionHandler) buildConversationContext(ctx context.Context) (*Context, error) {
	key := ""
	if h.isScopedMode() {
		key = getGlobalConversationContextKey()
	}

	conversationContext := new(Context)
	found, err := h.db.Load(ctx, key, conversationContext)
	if err != nil {
		return nil, err
	}

	if !found {
		return &Context{}, nil
	}

	return conversationContext, nil
}

func (h *ChatCompletionHandler) getLastMessageTime(msgs []ConversationMessage) time.Time {
	lastMessageTime := int64(0)
	for _, message := range msgs {
		if message.CreatedAt <= lastMessageTime {
			continue
		}
		lastMessageTime = message.CreatedAt
	}

	return time.Unix(lastMessageTime, 0)
}

func (h *ChatCompletionHandler) isConversationOutdated(conv *Conversation) bool {
	// for the case when we started a conversation with a context but didn't send any messages yet
	if len(conv.Messages) == 0 && conv.Context.GetMessage() != "" {
		contextCreatedAt := time.Unix(conv.Context.GetCreatedAt(), 0)
		return contextCreatedAt.Add(ConversationTimeout).Before(time.Now())
	}

	lastMessageTime := h.getLastMessageTime(conv.Messages)
	return lastMessageTime.Add(ConversationTimeout).Before(time.Now())
}

func (h *ChatCompletionHandler) Handle(ctx context.Context, req *msg.Request) (*msg.Response, error) {
	log := logging.WithContext(ctx)

	model := h.settingsLoader.LoadModel(ctx, req)

	conversation, err := h.buildConversation(ctx, req)
	if err != nil {
		return nil, err
	}
	conversation.Messages = append(conversation.Messages, ConversationMessage{
		Role:      RoleUser,
		Text:      req.Message,
		CreatedAt: time.Now().Unix(),
	})

	requestData := map[string]interface{}{
		"model":    model.GetName(),
		"messages": conversation.ToRaw(),
	}

	chatResp := new(ChatCompletionResponse)
	reqsr := rest.NewRequester(CompletionsURL, chatResp)
	reqsr.WithBearer(h.cfg.APIKey)
	reqsr.WithPOST()
	reqsr.WithInput(requestData)

	err = reqsr.Request(ctx)
	if err != nil {
		return nil, err
	}

	messages := make([]string, 0, len(chatResp.Choices))
	for i := range chatResp.Choices {
		choice := chatResp.Choices[i]
		if choice.Message.Content == "" {
			continue
		}

		messages = append(messages, choice.Message.Content)
		conversation.Messages = append(conversation.Messages, ConversationMessage{
			Role:      RoleAssistant,
			Text:      choice.Message.Content,
			CreatedAt: chatResp.CreatedAt,
		})
	}

	if len(messages) == 0 {
		return &msg.Response{
			Message: "Didn't get any response from ChatGPT completion API",
			Type:    msg.Error,
		}, nil
	}

	err = h.db.Save(ctx, getConversationKey(req), conversation, defaultConversationValidity)
	if err != nil {
		log.Error(err)
	}

	return &msg.Response{
		Message: strings.Join(messages, "/n"),
		Type:    msg.Success,
	}, nil
}

func (h *ChatCompletionHandler) CanHandle(context.Context, *msg.Request) (bool, error) {
	return true, nil
}
