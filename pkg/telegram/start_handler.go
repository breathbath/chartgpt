package telegram

import (
	"context"

	"breathbathChatGPT/pkg/msg"
	"breathbathChatGPT/pkg/utils"
)

type StartHandler struct{}

func (sh *StartHandler) CanHandle(_ context.Context, req *msg.Request) (bool, error) {
	return utils.MatchesCommand(req.Message, "/start"), nil
}

func (sh *StartHandler) Handle(context.Context, *msg.Request) (*msg.Response, error) {
	return &msg.Response{
		Messages: []msg.ResponseMessage{
			{
				Message: `Как пользоваться ботом: 

1️⃣Отправьте голосовое сообщение: спросите у бота какое вино вам подойдет 

Например: 
«посоветуй вино для праздника» 

Или
 «подбери красное сухое для мамы» 

Или
 «игристое для хорошего настроения» 

Запрос может быть любым, бот найдет вино по вашему вкусу 


2️⃣Дождитесь ответа 
Бот подберет несколько вариантов вин подходящих именно для вас, опишет вкус и предоставит фото бутылки

3️⃣Купите вино по фото
Покажите фото бутылки консультанту или найдите ее на полке самостоятельно

4️⃣Наслаждайтесь вкусом 😊
`,
				Type: msg.Success,
			},
		},
	}, nil
}
