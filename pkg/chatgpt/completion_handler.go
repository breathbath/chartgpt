package chatgpt

import (
	"breathbathChatGPT/pkg/db"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jmoiron/sqlx"
	"strings"
	"time"

	"breathbathChatGPT/pkg/msg"
	"breathbathChatGPT/pkg/rest"
	"breathbathChatGPT/pkg/storage"

	logging "github.com/sirupsen/logrus"

	_ "github.com/go-sql-driver/mysql"
)

const (
	URL                           = "https://api.openai.com"
	CompletionsURL                = URL + "/v1/chat/completions"
	ModelsURL                     = URL + "/v1/models"
	ConversationTimeout           = time.Minute * 10
	MaxScopedConversationMessages = 20
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

	if !found {
		log.Debug("the conversation is not found or outdated, will start a new conversation")
		return &Conversation{ID: req.GetConversationID(), Context: conversationContext}, nil
	}

	if h.isConversationOutdated(conversation, ConversationTimeout) {
		log.Debug("the conversation is not found or outdated, will start a new conversation")
		return &Conversation{ID: req.GetConversationID(), Context: conversationContext}, nil
	}

	if len(conversation.Messages) > MaxScopedConversationMessages {
		conversation.Messages = conversation.Messages[len(conversation.Messages)-MaxScopedConversationMessages:]
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

func (h *ChatCompletionHandler) isConversationOutdated(conv *Conversation, timeout time.Duration) bool {
	// for the case when we started a conversation with a context but didn't send any messages yet
	if len(conv.Messages) == 0 && conv.Context.GetMessage() != "" {
		contextCreatedAt := time.Unix(conv.Context.GetCreatedAt(), 0)
		return contextCreatedAt.Add(timeout).Before(time.Now())
	}

	lastMessageTime := h.getLastMessageTime(conv.Messages)
	return lastMessageTime.Add(timeout).Before(time.Now())
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
		"tools": []map[string]interface{}{
			{
				"type": "function",
				"function": map[string]interface{}{
					"name":        "find_wine",
					"description": "Find wine by provided parameters",
					"parameters": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"цвет": map[string]interface{}{
								"type": "string",
								"enum": []string{"Белое", "Розовое", "Красное", "Оранжевое"},
							},
							"год": map[string]interface{}{
								"type": "number",
							},
							"сахар": map[string]interface{}{
								"type": "string",
								"enum": []string{"полусладкое", "сухое", "полусухое", "сладкое", "экстра брют", "брют"},
							},
							"крепость": map[string]interface{}{
								"type": "string",
								"enum": []string{"крепкое", "легкое", "среднекрепкое"},
							},
							"подходящие блюда": map[string]interface{}{
								"type": "array",
								"items": map[string]interface{}{
									"type": "string",
								},
							},
							"тело": map[string]interface{}{
								"type": "string",
								"enum": []string{"среднее", "легкое", "полнотелое"},
							},
							"название": map[string]interface{}{
								"type": "string",
							},
							"страна": map[string]interface{}{
								"type": "string",
							},
							"цена": map[string]interface{}{
								"type": "string",
								"enum": []string{"массовое", "бюджетное", "премиальное", "коллекционное"},
							},
						},
						"required": []string{"цвет", "сахар"},
					},
				},
			},
		},
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
		if choice.FinishReason == "tool_calls" {
			responseTxt, err := h.processToolCall(ctx, choice, &conversation.Messages, req, model)
			if err != nil {
				return nil, err
			}
			messages = append(messages, responseTxt)
		} else {
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

func (h *ChatCompletionHandler) processToolCall(
	ctx context.Context,
	choice ChatCompletionChoice,
	history *[]ConversationMessage,
	req *msg.Request,
	model *ConfiguredModel,
) (responseMessage string, err error) {
	log := logging.WithContext(ctx)

	if len(choice.Message.ToolCalls) == 0 {
		log.Errorf("Invalid function call, missing tool calls property: %+v", choice.Message)
		return responseMessage, errors.New("didn't get any response from ChatGPT completion API")
	}

	for i := range choice.Message.ToolCalls {
		toolCall := choice.Message.ToolCalls[i]
		if toolCall.Function.Name == "find_wine" {
			return h.callFindWine(ctx, toolCall.Function.Arguments, history, req, model)
		}
	}

	log.Errorf("Didn't find any matching function: %+v", choice.Message)

	return responseMessage, errors.New("didn't get any response from ChatGPT completion API")
}

const DescriptionContext = `ты формулируешь описания вин для сайта. Начинай описание так: <цвет вина> <сахар>  вино <название> <год> года и дальше текст описания, в конце выдавай информацию о цене. Не повторяй название вина больше одного раза.`

func (h *ChatCompletionHandler) callFindWine(
	ctx context.Context,
	arguments json.RawMessage,
	history *[]ConversationMessage,
	req *msg.Request,
	model *ConfiguredModel,
) (responseMessage string, err error) {
	var data string
	err = json.Unmarshal(arguments, &data)
	if err != nil {
		return responseMessage, err
	}

	var argumentsMap map[string]interface{}

	err = json.Unmarshal([]byte(data), &argumentsMap)
	if err != nil {
		return responseMessage, err
	}

	logging.Debugf("Function call: %q", string(arguments))

	w, err := h.findByCriteria(
		fmt.Sprint(argumentsMap["цвет"]),
		fmt.Sprint(argumentsMap["сахар"]),
		fmt.Sprint(argumentsMap["страна"]),
	)

	if err != nil {
		return responseMessage, err
	}

	text, err := h.generateWineAnswer(ctx, req, w, model)
	if err != nil {
		return responseMessage, err
	}

	*history = append(*history, ConversationMessage{
		Role:      RoleAssistant,
		Text:      text,
		CreatedAt: time.Now().Unix(),
	})

	return text, nil
}

func (h *ChatCompletionHandler) generateWineAnswer(
	ctx context.Context,
	req *msg.Request,
	w Wine,
	model *ConfiguredModel,
) (string, error) {
	conversationContext := &Context{
		Message:            DescriptionContext,
		CreatedAtTimestamp: time.Now().Unix(),
	}

	conversation := &Conversation{
		ID:      req.GetConversationID(),
		Context: conversationContext,
		Messages: []ConversationMessage{
			{
				Role:      RoleUser,
				Text:      w.String(),
				CreatedAt: time.Now().Unix(),
			},
		},
	}

	requestData := map[string]interface{}{
		"model":    model.GetName(),
		"messages": conversation.ToRaw(),
	}

	chatResp := new(ChatCompletionResponse)
	reqsr := rest.NewRequester(CompletionsURL, chatResp)
	reqsr.WithBearer(h.cfg.APIKey)
	reqsr.WithPOST()
	reqsr.WithInput(requestData)

	err := reqsr.Request(ctx)
	if err != nil {
		return "", err
	}

	respMessage := ""
	for i := range chatResp.Choices {
		choice := chatResp.Choices[i]
		if choice.Message.Content == "" {
			continue
		}

		respMessage = choice.Message.Content
	}

	if respMessage == "" {
		respMessage = w.String()
	}

	return respMessage, nil
}

func (h *ChatCompletionHandler) CanHandle(context.Context, *msg.Request) (bool, error) {
	return true, nil
}

func (h *ChatCompletionHandler) findByCriteria(color string, sugar string, country string) (w Wine, err error) {
	config, err := db.LoadConfig()
	if err != nil {
		return w, err
	}

	conn, err := sqlx.Open("mysql", config.ConnString)
	if err != nil {
		return w, err
	}

	defer conn.Close()

	params := map[string]interface{}{}
	filters := []string{}

	if color != "" {
		params["color"] = color
		filters = append(filters, "color=:color")
	}

	if country != "" {
		params["country"] = country
		filters = append(filters, "country=:country")
	}

	if sugar != "" {
		params["sugar"] = sugar
		filters = append(filters, "sugar=:sugar")
	}

	where := ""
	if len(filters) > 0 {
		where = fmt.Sprintf("WHERE %s", strings.Join(filters, " AND "))
	}

	const query = "SELECT * FROM winechef_wines %s order by RAND() limit 1"
	q := fmt.Sprintf(query, where)

	results, err := conn.NamedQuery(q, params)
	if err != nil {
		return w, err
	}

	for results.Next() {
		var w Wine
		err = results.StructScan(&w)
		if err != nil {
			return w, err
		}

		return w, nil
	}

	q = fmt.Sprintf(query, "")
	rows, err := conn.Queryx(q)
	for rows.Next() {
		var w Wine
		err := rows.StructScan(&w)
		if err != nil {
			return w, err
		}

		return w, nil
	}

	return w, errors.New("no wines found")
}

type Wine struct {
	Color            string  `db:"color"`
	Sugar            string  `db:"sugar"`
	Strength         string  `db:"strength"`
	Photo            string  `db:"photo"`
	Name             string  `db:"name"`
	Article          string  `db:"article"`
	RealName         string  `db:"real_name"`
	Year             string  `db:"year"`
	Country          string  `db:"country"`
	Region           string  `db:"region"`
	Manufacturer     string  `db:"manufacturer"`
	Grape            string  `db:"grape"`
	Price            float64 `db:"price"`
	Body             string  `db:"body"`
	SmellDescription string  `db:"smell_description"`
	TasteDescription string  `db:"taste_description"`
	FoodDescription  string  `db:"food_description"`
	Style            string  `db:"style"`
	Recommend        string  `db:"recommend"`
	Id               int     `db:"int"`
}

func (w Wine) String() string {
	textParts := []string{}
	if w.Color != "" {
		textParts = append(textParts, fmt.Sprintf("Цвет вина: %s", w.Color))
	}
	if w.Sugar != "" {
		textParts = append(textParts, fmt.Sprintf("Сахар: %s", w.Sugar))
	}
	if w.Strength != "" {
		textParts = append(textParts, fmt.Sprintf("Крепость: %s", w.Strength))
	}
	if w.RealName != "" {
		textParts = append(textParts, fmt.Sprintf("Название вина: %s", w.RealName))
	} else if w.Name != "" {
		textParts = append(textParts, fmt.Sprintf("Название вина: %s", w.Name))
	}

	if w.Year != "" {
		textParts = append(textParts, fmt.Sprintf("Год: %s", w.Year))
	}

	if w.Country != "" {
		textParts = append(textParts, fmt.Sprintf("Страна происхождения: %s", w.Country))
	}

	if w.Price > 0 {
		textParts = append(textParts, fmt.Sprintf("Цена: %.0f руб.", w.Price))
	}

	if w.Body != "" {
		textParts = append(textParts, fmt.Sprintf("Тело вина: %s", w.Body))
	}

	if w.SmellDescription != "" {
		textParts = append(textParts, fmt.Sprintf("Аромат: %s", w.SmellDescription))
	}

	if w.TasteDescription != "" {
		textParts = append(textParts, fmt.Sprintf("Вкус: %s", w.TasteDescription))
	}

	if w.FoodDescription != "" {
		textParts = append(textParts, fmt.Sprintf("Сочетаемость с блюдами: %s", w.FoodDescription))
	} else if w.Style != "" {
		textParts = append(textParts, fmt.Sprintf("Сочетаемость с блюдами: %s", w.Style))
	}

	return strings.Join(textParts, ", ")
}
