package recommend

import (
	"breathbathChatGPT/pkg/auth"
	"breathbathChatGPT/pkg/monitoring"
	"breathbathChatGPT/pkg/msg"
	"breathbathChatGPT/pkg/utils"
	"context"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"strings"
)

const LikeCommand = "/like"
const DisLikeCommand = "/dislike"
const LikeContextMessage = "Поблагодари нашего пользователя за оценку нашего электронного сомелье WineChefBot. Дай короткий емкий эмоциональный текст и учитывай поставленную оценку."

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
	LikeType         string
	LikeValue        string
	UserLogin        string
	RecommendationID *uint
	Recommendation   *monitoring.Recommendation `gorm:"constraint:OnDelete:SET NULL;"`
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
	recommendationId := ""
	if utils.MatchesCommand(req.Message, LikeCommand) {
		like.LikeValue = "like"
		recommendationId = utils.ExtractCommandValue(req.Message, LikeCommand)
		userResponseMessages = append(userResponseMessages, "Оценка: понравилось")
	} else if utils.MatchesCommand(req.Message, DisLikeCommand) {
		like.LikeValue = "dislike"
		recommendationId = utils.ExtractCommandValue(req.Message, DisLikeCommand)
		userResponseMessages = append(userResponseMessages, "Оценка: не понравилось")
	}

	usr := auth.GetUserFromReq(req)
	if usr == nil {
		log.Error("Failed to find user data in the current request")
		return lh.handleErrorCase(ctx)
	}

	like.UserLogin = usr.Login

	var reco *monitoring.Recommendation

	if recommendationId != "" {
		log.Debugf("Going to find recommendation tracking for recommendation %s", recommendationId)
		reco = &monitoring.Recommendation{}
		res := lh.db.First(reco, recommendationId)
		if err := res.Error; err != nil {
			log.Errorf("failed to query recommendation %s: %v", recommendationId, err)
		} else {
			like.Recommendation = reco
		}
	}

	if req.Sender.FirstName != "" {
		userResponseMessages = append(userResponseMessages, "Имя пользователя: "+req.Sender.FirstName)
	}
	if req.Sender.LastName != "" {
		userResponseMessages = append(userResponseMessages, "Фамилия: "+req.Sender.LastName)
	}

	query := lh.db.Model(&Like{})
	lh.db.Where("user_login=?", like.UserLogin)
	if like.Recommendation != nil {
		lh.db.Where("recommendation_id=?", like.Recommendation.ID)
	}

	var existingLike Like
	res := query.Take(&existingLike)
	if res.Error == nil {
		if existingLike.LikeValue != like.LikeValue {
			res := lh.db.Model(&Like{}).Where("id=?", existingLike.ID).Update("like_value", like.LikeValue)
			if err := res.Error; err != nil {
				log.Errorf("failed to update like with id %d: %v", existingLike.ID, err)
			}
		}

	} else if errors.Is(res.Error, gorm.ErrRecordNotFound) {
		res := lh.db.Create(like)
		if err := res.Error; err != nil {
			log.Errorf("failed to save like from user %q: %v", usr.Login, err)
		}

	} else if res.Error != nil {
		log.Errorf("failed to find like from user %q: %v", usr.Login, res.Error)
	}

	responseMessage, err := lh.respGen.GenerateResponse(
		ctx,
		LikeContextMessage,
		strings.Join(userResponseMessages, "."),
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
