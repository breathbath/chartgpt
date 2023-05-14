package chatgpt

import (
	"breathbathChatGPT/pkg/msg"
	"breathbathChatGPT/pkg/rest"
	"breathbathChatGPT/pkg/storage"
	"context"
	logging "github.com/sirupsen/logrus"
	"strings"
	"time"
)

const URL = "https://api.openai.com"
const CompletionsURL = URL + "/v1/chat/completions"
const ModelsURL = URL + "/v1/models"
const maxConversationLength = 5

type ChatCompletionHandler struct {
	cfg            *Config
	settingsLoader *Loader
	db             storage.Client
}

func NewChatCompletionHandler(cfg *Config, db storage.Client, loader *Loader) (h *ChatCompletionHandler, err error) {
	e := cfg.Validate()
	if e.HasErrors() {
		return nil, e
	}

	return &ChatCompletionHandler{
		cfg:            cfg,
		db:             db,
		settingsLoader: loader,
	}, nil
}

func (h *ChatCompletionHandler) buildConversation(ctx context.Context, req *msg.Request) (*Conversation, error) {
	cacheKey := getConversationKey(req)
	conversation := new(Conversation)
	found, err := h.db.Load(ctx, cacheKey, conversation)
	if err != nil {
		return nil, err
	}

	if !found || len(conversation.Messages) > maxConversationLength {
		return &Conversation{ID: req.GetConversationId()}, nil
	}

	return conversation, nil
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
	reqsr.WithBearer(h.cfg.ApiKey)
	reqsr.WithPOST()
	reqsr.WithInput(requestData)

	err = reqsr.Request(ctx)
	if err != nil {
		return nil, err
	}

	messages := make([]string, 0, len(chatResp.Choices))
	for _, choice := range chatResp.Choices {
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
		Meta: map[string]interface{}{
			"created": chatResp.CreatedAt,
			"format":  "",
		},
	}, nil
}

func (h *ChatCompletionHandler) CanHandle(ctx context.Context, req *msg.Request) (bool, error) {
	return true, nil
}
