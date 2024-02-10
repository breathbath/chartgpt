package recommend

import (
	"breathbathChatGPT/pkg/auth"
	"breathbathChatGPT/pkg/msg"
	"breathbathChatGPT/pkg/utils"
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"strings"
)

const DeleteFromFavoritesCommand = "/delete_from_favorites"
const DeleteFromFavoritesContextMessage = `Сообщи об успешном удалении вина из избранного через нашего электронного сомелье WineChefBot. 
Дай короткий емкий эмоциональный текст, обращайся на вы.`

var DeleteFromFavoritesFallbackMessages = []string{
	"Рад сообщить, что успешно удалили вино из избранного через нашего электронного сомелье WineChefBot! Теперь ваш список избранных вин стал ещё лучше и отражает только самые изысканные и восхитительные вина. Продолжайте наслаждаться искусством виноделия с нами!",
	"Сообщаю, что успешно удалено вино из вашего списка избранных через нашего электронного сомелье WineChefBot. Теперь ваше избранное отражает только те вина, которые вам действительно интересны. Продолжайте наслаждаться вином с нами!",
	"Готово! Ваш избранный напиток успешно удален из списка в WineChefBot 🍷. Мы надеемся, что ваше путешествие по миру вин продолжится с новыми вкусными открытиями. Наши электронные рекомендации всегда к вашим услугам!",
	"Ваше вино успешно удалено из избранного в WineChefBot! 🍇 Надеемся, это место скоро займет новое удивительное вино. Погружайтесь в волнующий мир вин с нами! 🍷✨",
	"Вино исчезло из вашего избранного в WineChefBot! 🌠 Мы верим, это только освободит место для новых захватывающих вкусов. С нетерпением ждем, чтобы помочь вам открыть их! 🍷💖",
}

var DeleteFromFavoritesErrorMessages = []string{
	"Ой, кажется, что-то пошло не так и вино осталось в избранном 🤔. Давайте попробуем ещё раз! Если проблема сохранится, мы здесь, чтобы помочь. 🍷✨",
	"Упс, произошла ошибка, и вино не удалилось из избранного в WineChefBot 🐛. Попробуйте ещё разок? Если всё ещё не получится, мы всегда на связи, чтобы разобраться! 🍷🚀",
	"Ух ты, кажется, удаление не сработало 🙈. Не волнуйтесь, давайте предпримем еще одну попытку. Если всё еще возникают проблемы, мы тут, чтобы помочь! 🍇👨‍🍳",
	"Ой-ой, кажется, удалить вино не вышло 🍷. Давай попробуем еще раз? Если не пойдет, обращайтесь, поможем!",
	"Неа, винишко всё ещё с нами 🤷‍♂️. Еще разок, а? Если вдруг что, мы здесь, чтобы все исправить!",
	"Ай, вино упорно не хочет уходить из избранного 😅. Попытаемся снова? Если возникнут сложности, мы к вашим услугам!",
	"Вот это нежданчик, удаление не удалось 🎈. Как насчет еще одной попытки? Если есть трудности, мы всегда рядом!",
}

type DeleteFromFavoritesHandler struct {
	db      *gorm.DB
	respGen ResponseGenerator
}

func NewDeleteFromFavoritesHandler(db *gorm.DB, respGen ResponseGenerator) *DeleteFromFavoritesHandler {
	return &DeleteFromFavoritesHandler{
		db:      db,
		respGen: respGen,
	}
}

func (afh *DeleteFromFavoritesHandler) CanHandle(_ context.Context, req *msg.Request) (bool, error) {
	if !utils.MatchesCommand(req.Message, DeleteFromFavoritesCommand) {
		return false, nil
	}

	return true, nil
}

func (afh *DeleteFromFavoritesHandler) handleErrorCase(ctx context.Context) (*msg.Response, error) {
	log := logrus.WithContext(ctx)

	responseMessage := utils.SelectRandomMessage(DeleteFromFavoritesErrorMessages)

	log.Debugf("Selected a random message for delete from favorites failure : %q", responseMessage)

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

func (afh *DeleteFromFavoritesHandler) handleSuccessCase(ctx context.Context, req *msg.Request, w *Wine) (*msg.Response, error) {
	log := logrus.WithContext(ctx)

	responseMessage := utils.SelectRandomMessage(DeleteFromFavoritesErrorMessages)

	log.Debugf("Selected a random message for add to favorites failure : %q", responseMessage)

	userFields := []string{}
	responseFields := []string{}
	if req.Sender.FirstName != "" {
		userFields = append(userFields, "Имя: "+req.Sender.FirstName)
	}
	if req.Sender.LastName != "" {
		userFields = append(userFields, "Фамилия: "+req.Sender.LastName)
	}

	if len(userFields) > 0 {
		responseFields = append(responseFields, strings.Join(userFields, ", "))
	}

	if w.WineTextualSummaryStr() != "" {
		responseFields = append(responseFields, fmt.Sprintf("Рекомендованное вино: %s", w.WineTextualSummaryStr()))
	}

	responseMessage, err := afh.respGen.GenerateResponse(
		ctx,
		DeleteFromFavoritesContextMessage,
		strings.Join(responseFields, "."),
		"delete_from_favorites_response",
		req,
	)
	if err != nil {
		log.Errorf("failed to delete from favorites response message: %v", err)
		m := utils.SelectRandomMessage(DeleteFromFavoritesFallbackMessages)
		return &msg.Response{
			Messages: []msg.ResponseMessage{
				{
					Message: m,
					Type:    msg.Success,
					Options: &msg.Options{},
				},
			},
		}, nil
	}

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

func (afh *DeleteFromFavoritesHandler) Handle(ctx context.Context, req *msg.Request) (*msg.Response, error) {
	log := logrus.WithContext(ctx)
	log.Debugf("Will handle delete from favorites for message %q", req.Message)

	wineArticle := utils.ExtractCommandValue(req.Message, DeleteFromFavoritesCommand)
	usr := auth.GetUserFromReq(req)
	if usr == nil {
		log.Error("Failed to find user data in the current request")
		return afh.handleErrorCase(ctx)
	}

	log.Debugf("Going to find a wine by article %q", wineArticle)
	var wineFromDb Wine
	res := afh.db.Where("article = ?", wineArticle).First(&wineFromDb)
	if err := res.Error; err != nil {
		log.Errorf("failed to find wine by article %q: %v", wineArticle, err)
		return afh.handleErrorCase(ctx)
	}

	log.Debugf("Going to find a favorite wine %d and user %q", wineFromDb.ID, usr.Login)
	var wineFavorite WineFavorite
	res = afh.db.Where("wine_id = ?", wineFromDb.ID).Where("user_login = ?", usr.Login).First(&wineFavorite)

	if res.Error == nil {
		log.Debugf("Found a favorite for wine %d, user %s, id %d", wineFromDb.ID, usr.Login, wineFavorite.ID)
		result := afh.db.Unscoped().Delete(&wineFavorite)
		if err := result.Error; err != nil {
			log.Errorf("failed to delete a wine from favorites %q: %v", wineArticle, err)
			return afh.handleErrorCase(ctx)
		}

		log.Debugf("Deleted a wine favorite %d for wine %d, user %s", wineFavorite.ID, wineFromDb.ID, usr.Login)

		return afh.handleSuccessCase(ctx, req, &wineFromDb)
	}

	if !errors.Is(res.Error, gorm.ErrRecordNotFound) {
		log.Errorf("failed to find a wine favorite: %v", res.Error)
		return afh.handleErrorCase(ctx)
	}

	return afh.handleSuccessCase(ctx, req, &wineFromDb)
}
