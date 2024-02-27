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
				Message: `🌴 Алоха, Винные Искатели Приключений! Ваш Алкобот на Стартовой Линии! 🚀

Соберитесь, будет так же просто, как налить вино в бокал! 

📣  Говорите или Пишите – Я Во Всём С Вами!  "Хочу вино, чтобы сердце пело" – и я в деле! Может случиться, что спрошу уточнить детали, но ваша идеальная винная рекомендация уже в пути!

🤹‍♂️ Без Церемоний: Говорите открыто, без украшений. Мы тут за вином, а не за словами. Даже Влада Лесниченко аплодировала бы вашему выбору!

👭 Расскажите Друзьям! Наверняка они тоже хотят немного винного волшебства в своей жизни. А чтобы я не затерялся среди других сообщений, 📌 закрепите меня вверху , и веселье не закончится никогда.

🔍 Скоро... Я научусь подсказывать, где купить вино мечты и по лучшей цене! И, кто знает, может приглашу на дегустацию!

🎙 Стартуем! Не стесняйтесь, нажмите на микрофон и дайте мне знать ваши винные мечты. Поехали! 🚀
`,
				Type: msg.Success,
			},
		},
	}, nil
}
