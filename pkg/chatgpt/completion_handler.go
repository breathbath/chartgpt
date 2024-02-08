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
	SystemMessage                 = `ты система рекомендации вин WinechefBot. Можно вести разговор только о вине. Если спросят, кто ты, отвечай WinechefBot. Все ответы должны быть только про рекомендацию вин. На другие темы отвечай что ты не знаешь что ответить. запрашивай информацию о цвете и сахаре. Ценовые категории: бюджетный до 1000 руб, средний от 1000 до 1500 руб, премиум от 1500 до 2500 руб и люкс свыше 2500 руб. Если не указан год выпуска, то пропускай упоминание о годе. Низкая 5-10%. Классификация вин по крепости: низкая от 1 до 11.5%, средняя от 11,5 до 13,5%. средне высокая от 13,5 до 15%, высокая 15 и выше. Если цена не указана в диалоге, то не завай ее диапазон в функции find_wine.`
)

var colors = []string{"Белое", "Розовое", "Красное", "Оранжевое"}
var sugars = []string{"полусладкое", "сухое", "полусухое", "сладкое", "экстра брют", "брют"}
var bodies = []string{"полнотелое", "неполнотелое"}
var types = []string{"вино", "игристое", "шампанское", "херес", "портвейн"}

type ChatCompletionHandler struct {
	cfg            *Config
	settingsLoader *Loader
	cache          storage.Client
	isScopedMode   func() bool
	wineProvider   *recommend.WineProvider
	dbConn         *gorm.DB
}

func NewChatCompletionHandler(
	cfg *Config,
	cache storage.Client,
	loader *Loader,
	isScopedMode func() bool,
	wineProvider *recommend.WineProvider,
	dbConn *gorm.DB,
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
	monitoring.Usage(ctx).SetIsVoiceInput(true)

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
	monitoring.Usage(ctx).SetVoiceToTextModel(VoiceToTextModel)

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

	monitoring.Usage(ctx).SetInput(req.Message)
	monitoring.TrackRecommend(ctx).SetUserPrompt(req.Message)

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
					},
				},
				"тело": map[string]interface{}{
					"type": "string",
					"enum": bodies,
				},
				"название": map[string]interface{}{
					"type": "string",
				},
				"страна": map[string]interface{}{
					"type": "string",
				},
				"регион": map[string]interface{}{
					"type": "string",
				},
				"виноград": map[string]interface{}{
					"type": "string",
				},
				"тип": map[string]interface{}{
					"type": "string",
					"enum": types,
				},
				"стиль": map[string]interface{}{
					"type": "array",
					"items": map[string]interface{}{
						"type": "string",
						"enum": []string{"минеральные", "травянистые", "пряные", "пикантные", "ароматные", "фруктовые", "освежающие", "десертные", "выдержанные", "бархатистые"},
					},
				},
				"цена": map[string]interface{}{
					"type": "array",
					"items": map[string]interface{}{
						"type": "number",
					},
				},
			},
			"required": []string{"цвет", "сахар"},
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

	monitoring.TrackRecommend(ctx).SetRawModelInput(requestData)

	err = reqsr.Request(ctx)
	if err != nil {
		return nil, err
	}

	monitoring.TrackRecommend(ctx).SetRawModelOutput(chatResp)

	monitoring.Usage(ctx).SetInputPromptTokens(chatResp.Usage.PromptTokens)
	monitoring.Usage(ctx).SetInputCompletionTokens(chatResp.Usage.CompletionTokens)
	monitoring.Usage(ctx).SetGPTModel(model.GetName())

	messages := make([]string, 0, len(chatResp.Choices))
	var media *msg.Media
	for i := range chatResp.Choices {
		choice := chatResp.Choices[i]
		if choice.FinishReason == "tool_calls" {
			response, err := h.processToolCall(ctx, choice, &conversation.Messages, req, model)
			if err != nil {
				return nil, err
			}

			if response.Message != "" {
				messages = append(messages, response.Message)
			}
			if response.Media != nil {
				media = response.Media
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
			Message: "Didn't get any response from ChatGPT completion API",
			Type:    msg.Error,
		}, nil
	}

	err = h.cache.Save(ctx, getConversationKey(req), conversation, defaultConversationValidity)
	if err != nil {
		log.Error(err)
	}

	return &msg.Response{
		Message: strings.Join(messages, "/n"),
		Type:    msg.Success,
		Media:   media,
	}, nil
}

func (h *ChatCompletionHandler) processToolCall(
	ctx context.Context,
	choice ChatCompletionChoice,
	history *[]ConversationMessage,
	req *msg.Request,
	model *ConfiguredModel,
) (responseMessage *msg.Response, err error) {
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

const DescriptionContext = `ты формулируешь описания вин для сайта. Начинай описание так: <цвет вина> <сахар>  вино <название> <год> года, <страна> и дальше текст описания. Избегай повторов в выдаваемом тексте.`
const NotFoundMessage = `Извините, но наша система не нашла никаких вариантов вина, соответствующих вашему запросу. Пожалуйста, попробуйте изменить критерии для поиска, такие как уровень сахара, цвет или страна производства. Мы надеемся, что вы сможете найти подходящее вино!`

func (h *ChatCompletionHandler) parseFilter(ctx context.Context, arguments json.RawMessage) (*recommend.WineFilter, error) {
	var data string
	err := json.Unmarshal(arguments, &data)
	if err != nil {
		return nil, err
	}

	data = utils.NormalizeStringArrays(data)
	var argumentsMap map[string]interface{}

	err = json.Unmarshal([]byte(data), &argumentsMap)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to parse arguments list")
	}

	monitoring.TrackRecommend(ctx).SetFunctionCall(data)

	wineFilter := &recommend.WineFilter{}

	logging.Debugf("FunctionCall call: %q", string(arguments))

	if argumentsMap["цвет"] != nil {
		wineFilter.Color = utils.ParseEnumStr(argumentsMap["цвет"], colors)
	}

	if argumentsMap["сахар"] != nil {
		wineFilter.Sugar = utils.ParseEnumStr(argumentsMap["сахар"], sugars)
	}

	if argumentsMap["страна"] != nil {
		wineFilter.Country = fmt.Sprint(argumentsMap["страна"])
	}

	if argumentsMap["регион"] != nil {
		wineFilter.Region = fmt.Sprint(argumentsMap["регион"])
	}

	if argumentsMap["виноград"] != nil {
		wineFilter.Grape = fmt.Sprint(argumentsMap["виноград"])
	}

	if argumentsMap["год"] != nil {
		year, err := strconv.Atoi(fmt.Sprint(argumentsMap["год"]))
		if err == nil {
			wineFilter.Year = year
		}
	}

	if argumentsMap["крепость"] != nil {
		rawRange, ok := argumentsMap["крепость"].([]interface{})
		if ok {
			wineFilter.AlcoholPercentage = utils.ParseRangeFloat(rawRange)
		}
	}

	if argumentsMap["подходящие блюда"] != nil {
		rawList, ok := argumentsMap["подходящие блюда"].([]interface{})
		if ok {
			wineFilter.MatchingDishes = utils.ParseStrings(rawList)
		}
	}

	if argumentsMap["тело"] != nil {
		wineFilter.Body = utils.ParseEnumStr(argumentsMap["тело"], bodies)
	}

	if argumentsMap["тип"] != nil {
		wineFilter.Type = utils.ParseEnumStr(argumentsMap["тип"], types)
	}

	if argumentsMap["название"] != nil {
		wineFilter.Name = fmt.Sprint(argumentsMap["название"])
	}

	if argumentsMap["цена"] != nil {
		rawRange, ok := argumentsMap["цена"].([]interface{})
		if ok {
			wineFilter.PriceRange = utils.ParseRangeFloat(rawRange)
		}
	}

	if argumentsMap["стиль"] != nil {
		rawList, ok := argumentsMap["стиль"].([]interface{})
		if ok {
			wineFilter.Style = utils.ParseStrings(rawList)
		}
	}

	return wineFilter, nil
}

func (h *ChatCompletionHandler) callFindWine(
	ctx context.Context,
	arguments json.RawMessage,
	history *[]ConversationMessage,
	req *msg.Request,
	model *ConfiguredModel,
) (responseMessage *msg.Response, err error) {
	log := logging.WithContext(ctx)

	wineFilter, err := h.parseFilter(ctx, arguments)
	if err != nil {
		return responseMessage, err
	}

	found, wineFromDb, err := h.wineProvider.FindByCriteria(ctx, wineFilter)
	if err != nil {
		return responseMessage, err
	}

	if !found {
		*history = append(*history, ConversationMessage{
			Role:      RoleAssistant,
			Text:      "Ничего не найдено",
			CreatedAt: time.Now().Unix(),
		})
		monitoring.TrackRecommend(ctx).Flush(ctx, h.dbConn)
		return &msg.Response{Message: NotFoundMessage}, nil
	}

	log.Debugf("Found wine: %q", wineFromDb.String())

	text, err := h.generateWineAnswer(ctx, req, wineFromDb, model)
	if err != nil {
		return responseMessage, err
	}

	monitoring.TrackRecommend(ctx).SetRecommendationText(text)
	monitoring.TrackRecommend(ctx).Flush(ctx, h.dbConn)

	*history = append(*history, ConversationMessage{
		Role:      RoleAssistant,
		Text:      text,
		CreatedAt: time.Now().Unix(),
	})

	respMessage := &msg.Response{
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

	return respMessage, nil
}

func (h *ChatCompletionHandler) generateWineAnswer(
	ctx context.Context,
	req *msg.Request,
	w recommend.Wine,
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
				Text:      w.WineTextualSummaryStr(),
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

	monitoring.Usage(ctx).SetGenPromptTokens(chatResp.Usage.PromptTokens)
	monitoring.Usage(ctx).SetGenCompletionTokens(chatResp.Usage.CompletionTokens)

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
	} else {
		respMessage += fmt.Sprintf(" Цена %.f руб", w.Price)
	}

	return respMessage, nil
}

func (h *ChatCompletionHandler) CanHandle(context.Context, *msg.Request) (bool, error) {
	return true, nil
}
