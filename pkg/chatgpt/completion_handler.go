package chatgpt

import (
	"breathbathChatGPT/pkg/monitoring"
	"breathbathChatGPT/pkg/recommend"
	"breathbathChatGPT/pkg/utils"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httputil"
	"strconv"
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
	TranscriptionsURL             = URL + "/v1/audio/transcriptions"
	ModelsURL                     = URL + "/v1/models"
	ConversationTimeout           = time.Minute * 10
	MaxScopedConversationMessages = 20
	VoiceToTextModel              = "whisper-1"
	SystemMessage                 = `ты система рекомендации вин WinechefBot. Можно вести разговор только о вине. Если спросят, кто ты, отвечай WinechefBot (мужской род). Все ответы должны быть только про рекомендацию вин. Все рекомендации вин нужно делать путем вызова фукции find_wine. Вызывай ее если указан хотя бы один параметр выбора. На другие темы отвечай что ты не знаешь что ответить. Ценовые категории: бюджетный до 1000 руб, средний от 1000 до 1500 руб, премиум от 1500 до 2500 руб и люкс свыше 2500 руб. Классификация вин по крепости: низкая от 1 до 11.5%, средняя от 11,5 до 13,5%. средне высокая от 13,5 до 15%, высокая 15 и выше. Ты должен быть готов к тому, чтобы корректировать запросы пользователя, даже если они были сделаны с ошибкой. В случае, если название вина или винограда указано с ошибкой или не четко, нужно задавать скорректированное название при вызове функции find_wine.`
	NotFoundMessage               = `Извините, но наша система не нашла никаких вариантов вина, соответствующих вашему запросу. Пожалуйста, попробуйте изменить критерии для поиска, такие как уровень сахара, цвет или страна производства. Мы надеемся, что вы сможете найти подходящее вино!`
	PromptFiltersMessage          = `Запроси у пользователя дополнительную информацию для рекомендации по следующим фильтрам: %s`
)

var colors = []string{"Белое", "Розовое", "Красное", "Оранжевое"}
var sugars = []string{"полусладкое", "сухое", "полусухое", "сладкое", "экстра брют", "брют"}
var bodies = []string{"полнотелое", "неполнотелое"}
var types = []string{"вино", "игристое", "шампанское", "херес", "портвейн"}
var botLikeTexts = []string{
	"Я надеюсь, что тебе понравилось наше общение. Мы очень ценим твоё мнение! Пожалуйста, поставь оценку нашей работе: лайк или дислайк. Буду признателен за твою честную оценку!",
	"Прости, если отвлек тебя от чего-то важного. Но мне очень интересно узнать твоё мнение! Если у тебя есть возможность, буду благодарен, если ты поставишь оценку. Твоё мнение важно для меня!",
	"Прости, если путаю тебя своими вопросами. Но мне действительно интересно, что ты думаешь о моих рекомендациях. Пожалуйста, поставь оценку. Заранее благодарим за твоё мнение!",
	"Хей! Просто хотел напомнить тебе о возможности оценить мою работу. Если у тебя есть 1 секунда свободного времени, пожалуйста, нажми на одну из кнопок ниже. Спасибо большое!",
}

type ChatCompletionHandler struct {
	cfg            *Config
	settingsLoader *Loader
	cache          storage.Client
	isScopedMode   func() bool
	wineProvider   *recommend.WineProvider
	dbConn         *gorm.DB
	dialogHandler  *recommend.DialogHandler
}

func NewChatCompletionHandler(
	cfg *Config,
	cache storage.Client,
	loader *Loader,
	isScopedMode func() bool,
	wineProvider *recommend.WineProvider,
	dbConn *gorm.DB,
	dialogHandler *recommend.DialogHandler,
) (h *ChatCompletionHandler, err error) {
	e := cfg.Validate()
	if e.HasErrors() {
		return nil, e
	}

	return &ChatCompletionHandler{
		cfg:            cfg,
		cache:          cache,
		settingsLoader: loader,
		isScopedMode:   isScopedMode,
		wineProvider:   wineProvider,
		dbConn:         dbConn,
		dialogHandler:  dialogHandler,
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
	found, err := h.cache.Load(ctx, cacheKey, conversation)
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
	if h.isScopedMode() {
		return &Context{Message: SystemMessage}, nil
	}

	key := ""
	conversationContext := new(Context)
	found, err := h.cache.Load(ctx, key, conversationContext)
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

func (h *ChatCompletionHandler) convertVoiceToText(ctx context.Context, req *msg.Request) (string, error) {
	usageStats := &monitoring.UsageStats{
		UserId:       req.Sender.GetID(),
		SessionStart: time.Now().UTC(),
		GPTModel:     VoiceToTextModel,
		Type:         "voiceToText",
	}
	usageStats.SetTrackingID(ctx)
	log := logging.WithContext(ctx)

	outputFile, err := utils.ConvertAudioFileFromOggToMp3(req.File.FileReader)
	if err != nil {
		return "", err
	}
	log.Debugf("Converted file to mp3 format: %q", req.File)

	request, err := http.NewRequest("POST", TranscriptionsURL, nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("Authorization", "Bearer "+h.cfg.APIKey)
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	filePart, err := writer.CreateFormFile("file", req.File.FileID+".mp3")
	if err != nil {
		return "", err
	}

	_, err = io.Copy(filePart, outputFile)
	if err != nil {
		return "", err
	}

	err = writer.WriteField("model", VoiceToTextModel)
	if err != nil {
		return "", err
	}

	err = writer.Close()
	if err != nil {
		return "", err
	}

	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Body = io.NopCloser(body)

	log.Debugf("will do chatgpt request, url: %q, method: %s", request.URL.String(), request.Method)

	client := http.DefaultClient
	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	usageStats.SessionEnd = time.Now().UTC()

	dump, err := httputil.DumpResponse(response, true)
	if err != nil {
		log.Warnf("failed to dump response: %v", err)
	} else {
		log.Infof("response: %q", string(dump))
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", errors.New("bad response code from ChatGPT")
	}

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		log.Errorf("failed to read response body: %v", err)
		return "", errors.New("failed to read ChatGPT response")
	}

	textResp := new(AudioToTextResponse)
	err = json.Unmarshal(responseBody, textResp)
	if err != nil {
		log.Errorf("failed to pack response data into AudioToTextResponse model: %v", err)
		return "", errors.New("failed to interpret ChatGPT response")
	}

	usageStats.Input = textResp.Text
	usageStats.Save(ctx, h.dbConn)

	return textResp.Text, nil
}

func (h *ChatCompletionHandler) Handle(ctx context.Context, req *msg.Request) (*msg.Response, error) {
	log := logging.WithContext(ctx)

	var err error
	if req.File.Format == msg.FormatVoice {
		log.Infof("Got Voice input")
		req.Message, err = h.convertVoiceToText(ctx, req)
		if err != nil {
			return nil, err
		}
		log.Debugf("Converted voice to text: %q", req.Message)
	}

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

	usageStats := &monitoring.UsageStats{
		UserId:       req.Sender.GetID(),
		SessionStart: time.Now().UTC(),
		GPTModel:     model.GetName(),
		Type:         "recommendation",
	}
	usageStats.SetTrackingID(ctx)

	findWineFunction := map[string]interface{}{
		"name":        "find_wine",
		"description": "Find wine by provided parameters",
		"parameters": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"цвет": map[string]interface{}{
					"type": "string",
					"enum": colors,
				},
				"год": map[string]interface{}{
					"type": "number",
				},
				"сахар": map[string]interface{}{
					"type": "string",
					"enum": sugars,
				},
				"крепость": map[string]interface{}{
					"type": "array",
					"items": map[string]interface{}{
						"type": "number",
					},
				},
				"подходящие блюда": map[string]interface{}{
					"type": "array",
					"items": map[string]interface{}{
						"type": "string",
						"enum": []string{
							"аперитив", "баранина", "блюда", "вегетарианская", "говядина", "грибы", "десерт", "дичь", "закуски", "курица", "морепродукты", "мясные", "овощи", "оливки", "острые", "паста", "пернатая", "ракообразные", "рыба", "свинина", "суши", "сыр", "телятина", "фрукты", "фуа-гра", "ягнятина"},
					},
				},
				"тело": map[string]interface{}{
					"type": "string",
					"enum": bodies,
				},
				"название": map[string]interface{}{
					"description": "Название вина",
					"type":        "string",
				},
				"вкус и аромат": map[string]interface{}{
					"type": "string",
				},
				"страна": map[string]interface{}{
					"type": "string",
				},
				"регион": map[string]interface{}{
					"type": "string",
				},
				"виноград": map[string]interface{}{
					"description": "сорт винограда",
					"type":        "string",
				},
				"тип": map[string]interface{}{
					"type": "string",
					"enum": types,
				},
				"стиль": map[string]interface{}{
					"type": "array",
					"items": map[string]interface{}{
						"type": "string",
						"enum": recommend.StylesEnaum,
					},
				},
				"цена": map[string]interface{}{
					"type": "array",
					"items": map[string]interface{}{
						"type": "number",
					},
				},
			},
		},
	}
	requestData := map[string]interface{}{
		"model":    model.GetName(),
		"messages": conversation.ToRaw(),
		"tools": []map[string]interface{}{
			{
				"type":     "function",
				"function": findWineFunction,
			},
		},
	}

	chatResp := new(ChatCompletionResponse)
	reqsr := rest.NewRequester(CompletionsURL, chatResp)
	reqsr.WithBearer(h.cfg.APIKey)
	reqsr.WithPOST()
	reqsr.WithInput(requestData)

	recommendStats := &monitoring.Recommendation{
		UserID:         req.Sender.GetID(),
		RawModelInput:  utils.ConvToStr(requestData),
		RawModelOutput: utils.ConvToStr(chatResp),
		UserPrompt:     req.Message,
	}
	recommendStats.SetTrackingID(ctx)

	err = reqsr.Request(ctx)
	if err != nil {
		return nil, err
	}

	inputBytes, err := json.Marshal(requestData)
	if err != nil {
		inputBytes = []byte{}
	}
	usageStats.Input = string(inputBytes)

	usageStats.InputCompletionTokens = chatResp.Usage.CompletionTokens
	usageStats.InputPromptTokens = chatResp.Usage.PromptTokens
	usageStats.SessionEnd = time.Now().UTC()
	usageStats.Save(ctx, h.dbConn)

	messages := make([]string, 0, len(chatResp.Choices))
	var media *msg.Media
	var options *msg.Options
	for i := range chatResp.Choices {
		choice := chatResp.Choices[i]
		if choice.FinishReason == "tool_calls" {
			response, err := h.processToolCall(ctx, choice, &conversation.Messages, req, recommendStats)
			if err != nil {
				return nil, err
			}

			if response.Message != "" {
				messages = append(messages, response.Message)
			}
			if response.Media != nil {
				media = response.Media
			}
			if response.Options != nil {
				options = response.Options
			}
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
			Messages: []msg.ResponseMessage{
				{
					Message: "Didn't get any response from ChatGPT completion API",
					Type:    msg.Error,
				},
			},
		}, nil
	}

	err = h.cache.Save(ctx, getConversationKey(req), conversation, defaultConversationValidity)
	if err != nil {
		log.Error(err)
	}

	respMessages := []msg.ResponseMessage{
		{
			Message: strings.Join(messages, "/n"),
			Type:    msg.Success,
			Media:   media,
			Options: options,
		},
	}

	feedbackMessage, err := h.feedbackMessage(ctx, req, recommendStats)
	if err != nil {
		log.Errorf("failed to generate feedback message: %v", err)
	} else {
		if feedbackMessage != nil {
			respMessages = append(respMessages, *feedbackMessage)
		}
	}

	return &msg.Response{
		Messages: respMessages,
	}, nil
}

func (h *ChatCompletionHandler) feedbackMessage(
	ctx context.Context,
	req *msg.Request,
	recommendStats *monitoring.Recommendation,
) (*msg.ResponseMessage, error) {
	log := logging.WithContext(ctx)

	var userLike recommend.Like
	res := h.dbConn.First(&userLike, "user_login = ?", req.Sender.UserName)

	if res.Error != nil {
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			return h.createFeedbackResponse(ctx, recommendStats), nil
		}
		return nil, res.Error
	}

	currentTime := time.Now()

	timeDiff := currentTime.Sub(userLike.UpdatedAt)
	days := int(timeDiff.Hours() / 24)
	// Check if the number of days is a multiple of seven
	if days > 0 && days%7 == 0 {
		return h.createFeedbackResponse(ctx, recommendStats), nil
	}

	log.Debug("Skipping delayed like message since user already left a like before")
	return nil, nil
}

func (h *ChatCompletionHandler) createFeedbackResponse(
	ctx context.Context,
	recommendStats *monitoring.Recommendation,
) *msg.ResponseMessage {
	delayedOptions := &msg.Options{}
	delayedOptions.WithPredefinedResponse(msg.PredefinedResponse{
		Text: "❤️ " + "Нравится",
		Type: msg.PredefinedResponseInline,
		Data: fmt.Sprintf("%s %d", recommend.LikeCommand, recommendStats.ID),
	})
	delayedOptions.WithPredefinedResponse(msg.PredefinedResponse{
		Text: "🗣️️ " + "Отзыв",
		Type: msg.PredefinedResponseInline,
		Data: fmt.Sprintf("%s %d", recommend.DisLikeCommand, recommendStats.ID),
		Link: "https://t.me/ai_winechef",
	})
	return &msg.ResponseMessage{
		Message: utils.SelectRandomMessage(botLikeTexts),
		Type:    msg.Success,
		Options: delayedOptions,
		DelayedOptions: &msg.DelayedOptions{
			Timeout: time.Second * 30,
			Ctx:     ctx,
		},
	}
}

func (h *ChatCompletionHandler) processToolCall(
	ctx context.Context,
	choice ChatCompletionChoice,
	history *[]ConversationMessage,
	req *msg.Request,
	recommendStats *monitoring.Recommendation,
) (responseMessage *msg.ResponseMessage, err error) {
	log := logging.WithContext(ctx)

	if len(choice.Message.ToolCalls) == 0 {
		log.Errorf("Invalid function call, missing tool calls property: %+v", choice.Message)
		return responseMessage, errors.New("didn't get any response from ChatGPT completion API")
	}

	for i := range choice.Message.ToolCalls {
		toolCall := choice.Message.ToolCalls[i]
		if toolCall.Function.Name == "find_wine" {
			wineFilter, err := h.parseFilter(ctx, toolCall.Function.Arguments)
			if err != nil {
				return responseMessage, err
			}

			err = h.enrichFilter(ctx, wineFilter)
			if err != nil {
				return responseMessage, err
			}

			recommendStats.FunctionCall = string(toolCall.Function.Arguments)

			dialogAction, err := h.dialogHandler.DecideAction(ctx, wineFilter, req.Sender.GetID())
			if err != nil {
				return nil, err
			}

			if dialogAction.IsRecommendation() {
				return h.callFindWine(ctx, wineFilter, history, req, recommendStats)
			}

			filters := dialogAction.GetFilters()
			if len(filters) > 0 {
				respMessage, err := h.GenerateResponse(
					ctx,
					SystemMessage,
					fmt.Sprintf(PromptFiltersMessage, strings.Join(filters, ", ")),
					"recommendation_filters_prompt",
					req,
				)
				if err != nil {
					return nil, err
				}
				return &msg.ResponseMessage{
					Message: respMessage,
				}, nil
			}
			continue
		}
	}

	log.Errorf("Didn't find any matching function: %+v", choice.Message)

	return responseMessage, errors.New("didn't get any response from ChatGPT completion API")
}

func (h *ChatCompletionHandler) enrichFilter(ctx context.Context, f *recommend.WineFilter) error {
	log := logging.WithContext(ctx)
	if f.Region != "" && f.Country == "" {
		log.Debugf("going to find country by region %q", f.Region)
		c, err := recommend.FindCountryByRegion(h.dbConn, f.Region)
		if err != nil {
			return err
		}

		if c != "" {
			log.Debugf("found country %q by region %q", c, f.Region)
		} else {
			log.Debugf("didn't find any country by region %q", f.Region)
		}
		f.Country = c
	}

	return nil
}

func (h *ChatCompletionHandler) parseFilter(ctx context.Context, arguments json.RawMessage) (*recommend.WineFilter, error) {
	logging.Debugf("GPT Function call: %q", string(arguments))
	var data string
	err := json.Unmarshal(arguments, &data)
	if err != nil {
		return nil, err
	}

	var argumentsMap map[string]interface{}

	err = json.Unmarshal([]byte(data), &argumentsMap)
	if err != nil {
		normalisedData := utils.NormalizeJSON(ctx, data)
		logging.Debugf("JSON Normalization: %q", normalisedData)
		err = json.Unmarshal([]byte(normalisedData), &argumentsMap)
		if err != nil {
			logging.Errorf("Failed to parse arguments list %q: %v", string(arguments), err)
			return nil, nil
		}
	}

	wineFilter := &recommend.WineFilter{}

	if val, ok := argumentsMap["цвет"]; ok {
		wineFilter.Color = utils.ParseEnumStr(val, colors)
	}

	if val, ok := argumentsMap["сахар"]; ok {
		wineFilter.Sugar = utils.ParseEnumStr(val, sugars)
	}

	if val, ok := argumentsMap["страна"]; ok {
		wineFilter.Country = fmt.Sprint(val)
	}

	if val, ok := argumentsMap["регион"]; ok {
		wineFilter.Region = fmt.Sprint(val)
	}

	if val, ok := argumentsMap["виноград"]; ok {
		wineFilter.Grape = fmt.Sprint(val)
	}

	if wineFilter.Grape == "" {
		if val, ok := argumentsMap["сорт винограда"]; ok {
			wineFilter.Grape = fmt.Sprint(val)
		}
	}

	if val, ok := argumentsMap["сорт"]; ok {
		wineFilter.Grape = fmt.Sprint(val)
	}

	if val, ok := argumentsMap["год"]; ok {
		year, err := strconv.Atoi(fmt.Sprint(val))
		if err == nil {
			wineFilter.Year = year
		}
	}

	wineFilter.AlcoholPercentage = utils.ParseRangeFloat(argumentsMap, "крепость")

	wineFilter.MatchingDishes = utils.ParseArgumentsToStrings(argumentsMap, "подходящие блюда")

	if val, ok := argumentsMap["тело"]; ok {
		wineFilter.Body = utils.ParseEnumStr(val, bodies)
	}

	if val, ok := argumentsMap["тип"]; ok {
		wineFilter.Type = utils.ParseEnumStr(val, types)
	}

	if val, ok := argumentsMap["название"]; ok {
		wineFilter.Name = fmt.Sprint(val)
	}

	if val, ok := argumentsMap["вкус и аромат"]; ok {
		wineFilter.Taste = fmt.Sprint(val)
	}

	wineFilter.PriceRange = utils.ParseRangeFloat(argumentsMap, "цена")

	wineFilter.Style = utils.ParseArgumentsToStrings(argumentsMap, "стиль")

	return wineFilter, nil
}

func (h *ChatCompletionHandler) callFindWine(
	ctx context.Context,
	wineFilter *recommend.WineFilter,
	history *[]ConversationMessage,
	req *msg.Request,
	recommendStats *monitoring.Recommendation,
) (responseMessage *msg.ResponseMessage, err error) {
	log := logging.WithContext(ctx)

	found, wineFromDb, err := h.wineProvider.FindByCriteria(ctx, wineFilter, recommendStats)
	if err != nil {
		return responseMessage, err
	}

	if !found {
		*history = append(*history, ConversationMessage{
			Role:      RoleAssistant,
			Text:      "Ничего не найдено",
			CreatedAt: time.Now().Unix(),
		})
		recommendStats.Save(ctx, h.dbConn)
		return &msg.ResponseMessage{
			Message: NotFoundMessage,
		}, nil
	}

	log.Debugf("Found wine: %q", wineFromDb.String())

	text, err := h.generateWineAnswer(ctx, req, wineFromDb)
	if err != nil {
		return responseMessage, err
	}

	recommendStats.RecommendationText = text
	recommendStats.RecommendedWineID = wineFromDb.Article
	recommendStats.RecommendedWineSummary = wineFromDb.WineTextualSummaryStr()
	recommendStats.Save(ctx, h.dbConn)

	*history = append(*history, ConversationMessage{
		Role:      RoleAssistant,
		Text:      text,
		CreatedAt: time.Now().Unix(),
	})

	respMessage := &msg.ResponseMessage{
		Message: text,
	}
	if wineFromDb.Photo != "" {
		respMessage.Media = &msg.Media{
			Path:            wineFromDb.Photo,
			Type:            msg.MediaTypeImage,
			PathType:        msg.MediaPathTypeUrl,
			IsBeforeMessage: true,
		}
	}

	op := &msg.Options{}

	op.WithPredefinedResponse(msg.PredefinedResponse{
		Text: "📌️ " + "Запомнить",
		Type: msg.PredefinedResponseInline,
		Data: h.buildAddToFavoritesQuery(&wineFromDb, recommendStats),
	})
	op.WithPredefinedResponse(msg.PredefinedResponse{
		Text: "⭐ " + "Избранное",
		Type: msg.PredefinedResponseInline,
		Data: "/list_favorites",
	})

	respMessage.Options = op

	return respMessage, nil
}

func (h *ChatCompletionHandler) buildAddToFavoritesQuery(
	wineFromDb *recommend.Wine,
	recommendStats *monitoring.Recommendation,
) string {
	return fmt.Sprintf("%s %d %d", recommend.AddToFavoritesCommand, wineFromDb.ID, recommendStats.ID)
}

func (h *ChatCompletionHandler) generateWineAnswer(
	ctx context.Context,
	req *msg.Request,
	w recommend.Wine,
) (string, error) {
	respMessage, err := h.GenerateResponse(
		ctx,
		recommend.WineDescriptionContext,
		w.WineTextualSummaryStr(),
		"wine_card",
		req,
	)
	if err != nil {
		return "", err
	}

	if respMessage == "" {
		respMessage = w.String()
	} else {
		respMessage += fmt.Sprintf(" Цена %.f руб", w.Price)
	}

	return respMessage, nil
}

func (h *ChatCompletionHandler) GenerateResponse(
	ctx context.Context,
	contextMsg,
	message, typ string,
	req *msg.Request,
) (string, error) {
	usageStats := &monitoring.UsageStats{
		UserId:       req.Sender.GetID(),
		SessionStart: time.Now().UTC(),
		Type:         typ,
	}
	usageStats.SetTrackingID(ctx)

	log := logging.WithContext(ctx)
	model := h.settingsLoader.LoadModel(ctx, req)

	conversationContext := &Context{
		Message:            contextMsg,
		CreatedAtTimestamp: time.Now().Unix(),
	}

	conversation := &Conversation{
		ID:      req.GetConversationID(),
		Context: conversationContext,
		Messages: []ConversationMessage{
			{
				Role:      RoleUser,
				Text:      message,
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

	usageStats.Input = utils.ConvToStr(requestData)

	err := reqsr.Request(ctx)
	if err != nil {
		return "", err
	}

	usageStats.InputPromptTokens = chatResp.Usage.PromptTokens
	usageStats.InputCompletionTokens = chatResp.Usage.CompletionTokens
	usageStats.GPTModel = model.GetName()
	usageStats.SessionEnd = time.Now().UTC()
	usageStats.Save(ctx, h.dbConn)

	respMessage := ""
	for i := range chatResp.Choices {
		choice := chatResp.Choices[i]
		if choice.Message.Content == "" {
			continue
		}

		respMessage = choice.Message.Content
	}
	log.Debugf("Generated message by ChatGPT: %q", respMessage)

	return respMessage, nil
}

func (h *ChatCompletionHandler) CanHandle(context.Context, *msg.Request) (bool, error) {
	return true, nil
}
