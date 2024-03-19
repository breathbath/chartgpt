package recommend

import (
	"breathbathChatGPT/pkg/msg"
	"breathbathChatGPT/pkg/utils"
	"context"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"strings"
	"time"
)

const LikeCommand = "/like"
const DisLikeCommand = "/dislike"
const LikeContextMessage = "Ты голосовой помощник, действующий как сомелье на базе искусственного интеллекта WineChefBot. Поблагодари пользователя за оценку твоей работы и предложи закрепить себя поверх сообщений, предложи подписаться на свой инфо канал https://t.me/ai_winechef из которого пользователь узнает все новости проекта. Дай короткий емкий эмоциональный текст, в шутливой, неформальной форме и учитывай поставленную оценку."

var FallbackResponseMessages = []string{
	"Супер, спасибо за лайк нашей системе рекомендаций вин! Это круто, что тебе понравилось. Мы старались сделать ее полезной и интересной, и твой лайк подтверждает, что у нас получилось! Очень ценим твою поддержку и благодарим тебя за нее. Если у тебя есть еще какие-то предложения или пожелания, дай нам знать. Мы всегда рады помочь в выборе вина и делиться с тобой своими рекомендациями.",
	"Спасибо за лайк. Мы очень ценим ваше мнение!",
	"Ух ты, спасибо за лайк нашей системе рекомендации вин! Это просто огромное вдохновение для нас!",
	"Огромное спасибо за лайк. Вы делаете нас счастливыми!",
	"Благодарю тебя из глубины души за твой лайк! Твоя поддержка очень важна для нас, и мы ценим каждый твой жест.",
	"Спасибо, что поддерживаете нас своим лайком! Ваша оценка нашей работы очень важна для нас",
	"Спасибо, ваша поддержка значит много для нас!",
}

type Like struct {
	gorm.Model
	LikeType  string
	LikeValue string
	UserLogin string
}

type LikeHandler struct {
	db      *gorm.DB
	respGen ResponseGenerator
}

func NewLikeHandler(db *gorm.DB, respGen ResponseGenerator) *LikeHandler {
	return &LikeHandler{
		db:      db,
		respGen: respGen,
	}
}

func (lh *LikeHandler) CanHandle(_ context.Context, req *msg.Request) (bool, error) {
	if !utils.MatchesCommands(req.Message, []string{LikeCommand, DisLikeCommand}) {
		return false, nil
	}

	return true, nil
}

func (lh *LikeHandler) handleErrorCase(ctx context.Context) (*msg.Response, error) {
	log := logrus.WithContext(ctx)

	responseMessage := utils.SelectRandomMessage(FallbackResponseMessages)

	log.Debugf("Selected a random like thank you response: %q", responseMessage)

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

func (lh *LikeHandler) Handle(ctx context.Context, req *msg.Request) (*msg.Response, error) {
	log := logrus.WithContext(ctx)
	log.Debugf("Will handle like for message %q", req.Message)

	like := &Like{
		LikeType: "button_like",
	}

	userResponseMessages := []string{}
	if utils.MatchesCommand(req.Message, LikeCommand) {
		like.LikeValue = "like"
		userResponseMessages = append(userResponseMessages, "Оценка: понравилось")
	} else if utils.MatchesCommand(req.Message, DisLikeCommand) {
		like.LikeValue = "dislike"
		userResponseMessages = append(userResponseMessages, "Оценка: не понравилось")
	}

	if req.Sender == nil {
		log.Error("Failed to find user data in the current request")
		return lh.handleErrorCase(ctx)
	}

	like.UserLogin = req.Sender.UserName

	if req.Sender.FirstName != "" {
		userResponseMessages = append(userResponseMessages, "Имя пользователя: "+req.Sender.FirstName)
	}
	if req.Sender.LastName != "" {
		userResponseMessages = append(userResponseMessages, "Фамилия: "+req.Sender.LastName)
	}

	var existingLike Like
	query := lh.db.Model(&Like{})
	res := query.Where("user_login=?", like.UserLogin).Take(&existingLike)
	if res.Error == nil {
		res := lh.db.Model(&Like{}).Where("id=?", existingLike.ID).Update("updated_at", time.Now().UTC())
		if err := res.Error; err != nil {
			log.Errorf("failed to update like with id %d: %v", existingLike.ID, err)
		}

	} else if errors.Is(res.Error, gorm.ErrRecordNotFound) {
		res := lh.db.Create(like)
		if err := res.Error; err != nil {
			log.Errorf("failed to save like from user %q: %v", req.Sender.UserName, err)
		}

	} else if res.Error != nil {
		log.Errorf("failed to find like from user %q: %v", req.Sender.UserName, res.Error)
	}

	responseMessage, err := lh.respGen.Generate(
		ctx,
		LikeContextMessage,
		strings.Join(userResponseMessages, "."),
		"like_response",
		"like_response",
		req,
	)
	if err != nil {
		log.Errorf("failed to generate like response message: %v", err)
		return lh.handleErrorCase(ctx)
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
