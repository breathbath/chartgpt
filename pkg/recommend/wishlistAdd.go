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

const AddToFavoritesCommand = "/add_to_favorites"
const AddToFavoritesContextMessage = `Сообщи об успешном сохранении вина в избранном через нашего электронного сомелье WineChefBot. 
Дай короткий емкий эмоциональный текст, обращайся на вы. Сообщи, что юзер может показать избранное нажав кнопку "Избранное". Не нужно здороваться.`
const AlreadyExistsInFavoritesContextMessage = `Сообщи, что выбранное вино уже находится в избранном через нашего электронного сомелье WineChefBot. 
Дай короткий емкий эмоциональный текст. Не нужно здороваться.`

var AddToFavoritesFallbackMessages = []string{
	"Поздравляю! Ваше вино успешно сохранено в избранном. Отличный выбор! Теперь вы можете легко показать свои избранные вина через текстовое сообщение. Просто спросите меня о них!",
	"Ура! Ваше вино сохранено в избранном с помощью нашего электронного сомелье WineChefBot! Отличное решение! Теперь вы можете в любой момент показать свои избранные вина просто написав мне. Я всегда готов поделиться вашими лучшими находками!",
	"Молодец! Ты успешно сохранил своё любимое вино через нашего электронного сомелье WineChefBot. Теперь ты можешь легко поделиться своим избранным вином с помощью текстового сообщения. Показывай всем, что у тебя отличный вкус!",
	"Отлично! Ваше вино успешно добавлено в избранное с помощью нашего электронного сомелье WineChefBot. Это прекрасный выбор! Теперь вы можете показать свои избранные вина всего лишь одним текстовым сообщением. Гордитесь своими предпочтениями и делитесь ими с другими!",
	"Поздравляю! Ваше вино успешно сохранено в избранном с помощью WineChefBot. Вы сделали отличный выбор! Теперь вы можете без труда представить свои избранные вина всего одним текстовым сообщением. Так что не стесняйтесь показать свою коллекцию и поделиться своими винными находками!",
}

var AddToFavoritesErrorMessages = []string{
	"Извините, возникла техническая ошибка, и мы не можем добавить это вино в ваш избранный список. Пожалуйста, попробуйте позже.",
	"Технический глюк, алкоголиков у нас в избранном никак не принимаем! Пожалуйста, повторите попытку чуть позже.",
	"К сожалению, из-за технической ошибки мы временно не можем добавить выбранное вино в ваш избранный список. Просим прощения за неудобства.",
	"Ой-ой-ой, что-то не так с системой! Вино не может попасть в вашу избранную коллекцию. Попробуйте позже, мы уже занимаемся этим.",
	"Упс, видимо, кто-то в системе перепутал бутылочку. Вино не добавляется в избранное из-за технической шляпы. Подождите немного, мы это исправим!",
	"Произошла техническая ошибка, поэтому мы не можем добавить это вино в ваш избранный список в данный момент. Пожалуйста, попробуйте еще раз позже.",
	"Извините, похоже, произошла техническая ошибка, и вино не может быть добавлено в ваш избранный список. Мы работаем над решением проблемы.",
	"Ой, технические штучки-дрючки застопорили добавление вина в ваш избранный список. Мы делаем все возможное, чтобы починить эту шуточку! Зайдите попозже!",
	"К сожалению, из-за технической ошибки нам не удалось добавить вино в ваш избранный список. Мы приносим извинения за неудобства и рекомендуем попробовать позже.",
	"Вино пока придется оставить вне избранного, потому что где-то возникла техническая смута. Но не беспокойтесь, мы виномиры скоро это исправим!",
	"Ой, возникла техническая неполадка, и мы не можем добавить это вино в ваш избранный список в данный момент. Пожалуйста, попробуйте позже",
	"Ой, беда-пуча! Пока не получается добавить вино в вашу избранную коллекцию из-за технической белеберды. Попробуйте еще разок, и мы постараемся, чтобы всё шепотом пошло!",
	"К сожалению, произошла техническая ошибка, и мы временно не можем добавить выбранное вино в ваш избранный список. Пожалуйста, попробуйте еще раз через некоторое время.",
	" Космическое вино! К сожалению, из-за технического косяка оно не может улететь в ваш список избранного. Попробуйте позже, мы в команде инженеров работаем над этой галактической проблемой!",
	"Упс, вино запуталось в наших технических проволочках, поэтому пока не попадает в вашу избранную кружку. Но не отчаивайтесь, мы разгребаем этот электронный хлам!",
	"Ай, технический заскок! Вино категорически не хочет попадать в избранное, но не паникуйте, мы его скоро уговорим!",
	"Извините за неудобства! На данный момент у нас возникла техническая ошибка, и мы не можем добавить вино в ваш избранный список. Мы работаем над исправлением проблемы.",
	"Пива, водки, наливай, а вино в избранное льеться подпристанями. Из-за технической запарки пока не получается, но обещаем, скоро исправим эту винную катавасию!",
}

type ResponseGenerator interface {
	GenerateResponse(
		ctx context.Context,
		contextMsg,
		message, typ string,
		req *msg.Request,
	) (string, error)
}

type AddToFavoritesHandler struct {
	db      *gorm.DB
	respGen ResponseGenerator
}

func NewAddToFavoritesHandler(db *gorm.DB, respGen ResponseGenerator) *AddToFavoritesHandler {
	return &AddToFavoritesHandler{
		db:      db,
		respGen: respGen,
	}
}

func (afh *AddToFavoritesHandler) CanHandle(_ context.Context, req *msg.Request) (bool, error) {
	if !utils.MatchesCommand(req.Message, AddToFavoritesCommand) {
		return false, nil
	}

	return true, nil
}

func (afh *AddToFavoritesHandler) handleErrorCase(ctx context.Context) (*msg.Response, error) {
	log := logrus.WithContext(ctx)

	responseMessage := utils.SelectRandomMessage(AddToFavoritesErrorMessages)

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

func (afh *AddToFavoritesHandler) handleSuccessCase(
	ctx context.Context,
	req *msg.Request, w *Wine,
	alreadyExist bool,
) (*msg.Response, error) {
	log := logrus.WithContext(ctx)

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

	if !alreadyExist && w.WineTextualSummaryStr() != "" {
		responseFields = append(responseFields, fmt.Sprintf("Рекомендованное вино: %s", w.WineTextualSummaryStr()))
	}

	successMsg := AddToFavoritesContextMessage
	if alreadyExist {
		successMsg = AlreadyExistsInFavoritesContextMessage
	}
	responseMessage, err := afh.respGen.GenerateResponse(
		ctx,
		successMsg,
		strings.Join(responseFields, "."),
		"add_favorites_response",
		req,
	)
	if err != nil {
		log.Errorf("failed to generate add to favorites response message: %v", err)
		m := utils.SelectRandomMessage(AddToFavoritesFallbackMessages)
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

func (afh *AddToFavoritesHandler) Handle(ctx context.Context, req *msg.Request) (*msg.Response, error) {
	log := logrus.WithContext(ctx)
	log.Debugf("Will handle add to favorites for message %q", req.Message)

	wineArticle := utils.ExtractCommandValue(req.Message, AddToFavoritesCommand)
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
		return afh.handleSuccessCase(ctx, req, &wineFromDb, true)
	}
	if !errors.Is(res.Error, gorm.ErrRecordNotFound) {
		log.Errorf("failed to find a wine favorite: %v", res.Error)
		return afh.handleErrorCase(ctx)
	}

	log.Debugf("Didn't find a favorite for wine %d, user %s, id %d, will create a new one", wineFromDb.ID, usr.Login, wineFavorite.ID)

	wineFavorite = WineFavorite{
		Wine:      wineFromDb,
		UserLogin: usr.Login,
	}
	result := afh.db.Create(&wineFavorite)
	if err := result.Error; err != nil {
		log.Errorf("failed to create a wine favorite %q: %v", wineArticle, err)
		return afh.handleErrorCase(ctx)
	}

	log.Debugf("Created a wine favorite %d for wine %d, user %s, will create a new one", wineFavorite.ID, wineFromDb.ID, usr.Login)

	return afh.handleSuccessCase(ctx, req, &wineFromDb, false)
}
