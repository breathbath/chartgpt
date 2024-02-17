package recommend

import (
	"breathbathChatGPT/pkg/msg"
	"breathbathChatGPT/pkg/utils"
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

const ListFavoritesCommand = "/list_favorites"
const WineDescriptionContext = `ты формулируешь описания вин для сайта. Избегай повторов в выдаваемом тексте. Выдавай вначале название сахар, цвет, название, страну, год вина. Выдавай эмоциональный, красивый, продающий текст как бы это делал сомелье.`

var NoFavoritesFoundMessages = []string{
	"Эй, привет! Похоже, что твой список избранных вин пустой. Ничего страшного, это легко исправить! Я готов предложить тебе некоторые великолепные вина, чтобы ты мог добавить их в список.",
	"Ух, у тебя список избранных вин пустой? Ничего страшного, с такими рекомендациями, как у меня, ты точно не останешься без хорошего вина!",
	"Не расстраивайся из-за пустого списка избранных вин! С моей помощью ты сможешь выбрать несколько волшебных бутылочек. 🌟🍷",
	"Мои рекомендации вина помогут тебе заполнить пустоту в списке избранных! Приготовься к удовольствию! 🍷🔝",
	"Чего это у тебя список избранных вин пустой? Не беда, я знаю, как его заполнить восхитительными вариантами! 🍷😉",
	"Опа! Я посмотрел в твою винную коллекцию и не обнаружил ни одного избранного вина. Ничего, не все так плохо! Я готов поделиться сочными рекомендациями, чтобы ты смог насладиться вкусом лучших вин. Погнали!",
	"Ой, а у тебя в списке избранных вин все еще пусто! Не беда, я здесь, чтобы помочь тебе с выбором. Давай я подкину еще одну порцию винных рекомендаций прямиком из моей виртуальной погребушки",
}

var ReadingFavoritesListErrors = []string{
	"⚠️Ой-ой, кажется я попал в небольшую переделку! Не могу прочитать список избранных вин, который ты сохранил. Может быть, я наткнулся на непредвиденную техническую проблему. Приношу извинения за неудобства! Тем не менее я готов помочь тебе порекомендовать подходящее вино.",
	"🐍 Ой, что-то пошло не так! Возникла неприятная ошибка при чтении списка твоих избранных вин. Извини за неудобства, я делаю все возможное, чтобы исправить эту проблему!",
	"☠️ Ох-ох, простите за небольшую накладку! При попытке прочитать список избранных вин юзера у меня возникла ошибка.",
	"🛑Eй, прости, но у меня возникла небольшая заминка! Я никак не могу прочесть список вин, которые ты добавил в избранное. Наши технические гении уже трудятся, чтобы исправить эту ошибку. Но не беспокойся, я здесь, чтобы помочь тебе с рекомендациями вин! Расскажи мне о своих предпочтениях, и я подберу для тебя некоторые удивительные варианты.",
}

type ListFavoritesHandler struct {
	db      *gorm.DB
	respGen ResponseGenerator
}

func NewListFavoritesHandler(db *gorm.DB, respGen ResponseGenerator) *ListFavoritesHandler {
	return &ListFavoritesHandler{
		db:      db,
		respGen: respGen,
	}
}

func (afh *ListFavoritesHandler) CanHandle(_ context.Context, req *msg.Request) (bool, error) {
	if !utils.MatchesCommand(req.Message, ListFavoritesCommand) {
		return false, nil
	}

	return true, nil
}

func (afh *ListFavoritesHandler) handleErrorCase(ctx context.Context) (*msg.Response, error) {
	log := logrus.WithContext(ctx)

	responseMessage := utils.SelectRandomMessage(ReadingFavoritesListErrors)

	log.Debugf("Selected a random message for add to favorites failure : %q", responseMessage)

	return &msg.Response{
		Messages: []msg.ResponseMessage{
			{
				Message: responseMessage,
				Type:    msg.Error,
				Options: &msg.Options{},
			},
		},
	}, nil
}

func (afh *ListFavoritesHandler) handleListEmpty(ctx context.Context) (*msg.Response, error) {
	log := logrus.WithContext(ctx)

	responseMessage := utils.SelectRandomMessage(NoFavoritesFoundMessages)

	log.Debugf("Selected a random message for add to favorites failure : %q", responseMessage)

	return &msg.Response{
		Messages: []msg.ResponseMessage{
			{
				Message: responseMessage,
				Type:    msg.Success,
				Options: &msg.Options{},
			},
		},
	}, nil
}

func (afh *ListFavoritesHandler) generateWineDescription(
	ctx context.Context,
	req *msg.Request,
	w Wine,
) (string, error) {
	respMessage, err := afh.respGen.GenerateResponse(
		ctx,
		WineDescriptionContext,
		w.WineTextualSummaryStr(),
		"wine_card_favorite",
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

func (afh *ListFavoritesHandler) handleSuccessCase(ctx context.Context, req *msg.Request, favWines []WineFavorite) (*msg.Response, error) {
	log := logrus.WithContext(ctx)

	if len(favWines) == 0 {
		return afh.handleListEmpty(ctx)
	}

	resp := msg.Response{
		Messages: []msg.ResponseMessage{
			{
				Message: "🍷🍷🍷🍷 ----- ИЗБРАННОЕ ----- 🍷🍷🍷🍷",
				Type:    msg.Success,
				Options: &msg.Options{},
			},
		},
	}

	for _, w := range favWines {
		op := &msg.Options{}
		op.WithPredefinedResponse(msg.PredefinedResponse{
			Text: "❌️ " + "Удалить из избранного",
			Type: msg.PredefinedResponseInline,
			Data: DeleteFromFavoritesCommand + " " + w.Wine.Article,
		})
		op.WithPredefinedResponse(msg.PredefinedResponse{
			Text: "⭐ " + "Избранное",
			Type: msg.PredefinedResponseInline,
			Data: "/list_favorites",
		})

		var media *msg.Media
		if w.Wine.Photo != "" {
			media = &msg.Media{
				Path:            w.Wine.Photo,
				Type:            msg.MediaTypeImage,
				PathType:        msg.MediaPathTypeUrl,
				IsBeforeMessage: true,
			}
		}

		wineDescription, err := afh.generateWineDescription(ctx, req, w.Wine)
		if err != nil {
			log.Errorf("failed to generate wine description: %v", err)
			return afh.handleErrorCase(ctx)
		}

		resp.Messages = append(resp.Messages, msg.ResponseMessage{
			Message: wineDescription,
			Type:    msg.Success,
			Options: op,
			Media:   media,
		})
	}

	return &resp, nil
}

func (afh *ListFavoritesHandler) Handle(ctx context.Context, req *msg.Request) (*msg.Response, error) {
	log := logrus.WithContext(ctx)
	log.Debugf("Will handle list favorites for message %q", req.Message)

	if req.Sender == nil {
		log.Error("Failed to find user data in the current request")
		return afh.handleErrorCase(ctx)
	}

	var wineFavorites []WineFavorite
	res := afh.db.Preload("Wine").Where("user_login = ?", req.Sender.UserName).Find(&wineFavorites)
	if err := res.Error; err != nil {
		log.Errorf("failed to find favorites for user %q: %v", req.Sender.UserName, err)
		return afh.handleErrorCase(ctx)
	}

	log.Debugf("Found %d favorites for user %q", req.Sender.UserName)

	return afh.handleSuccessCase(ctx, req, wineFavorites)
}
