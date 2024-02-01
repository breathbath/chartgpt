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
		Message: `Голосовой бот-помощник для выбора вина 🍷

👉 Подберет вино, которое вам понравится
👉 Для любого повода
👉 С учетом вашего бюджета
👉В магазине или ресторане  

Как пользоваться ботом: 

1️⃣Отправьте голосовое сообщение: кратко опишите какое вино вы хотите купить: 

⚪️ белое, красное или игристое,
🟠 каким оно должно быть на вкус, 
🟢 для какого блюда
🔵 по какому поводу

2️⃣Дождитесь ответа 

Искусственный интеллект подберет несколько вариантов вин подходящих именно для вас, опишет вкус максимально достоверно и предоставит фото бутылки

3️⃣Купите вино по фото

Покажите фото нужной бутылки консультанту или найдите ее на полке самостоятельно

4️⃣Наслаждайтесь вкусом 😊
`,
		Type: msg.Success,
	}, nil
}
