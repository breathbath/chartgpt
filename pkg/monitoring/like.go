package monitoring

import (
	"breathbathChatGPT/pkg/auth"
	"breathbathChatGPT/pkg/msg"
	"breathbathChatGPT/pkg/utils"
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"strings"
)

const LikeCommand = "/like"
const LikeContextMessage = "Поблагодари нашего пользователя за лайк который он поставил вину через нашего электронного сомелье WineChefBot. Дай короткий емкий эмоциональный текст, обращайся на вы."

var FallbackResponseMessages = []string{
	"Супер, спасибо за лайк нашей системе рекомендаций вин! Это круто, что тебе понравилось. Мы старались сделать ее полезной и интересной, и твой лайк подтверждает, что у нас получилось! Очень ценим твою поддержку и благодарим тебя за нее. Если у тебя есть еще какие-то предложения или пожелания, дай нам знать. Мы всегда рады помочь в выборе вина и делиться с тобой своими рекомендациями.",
	"Спасибо за лайк. Мы очень ценим ваше мнение!",
	"Ух ты, спасибо за лайк нашей системе рекомендации вин! Это просто огромное вдохновение для нас!",
	"Огромное спасибо за лайк. Вы делаете нас счастливыми!",
	"Благодарю тебя из глубины души за твой лайк! Твоя поддержка очень важна для нас, и мы ценим каждый твой жест.",
	"Спасибо, что поддерживаете нас своим лайком! Ваша оценка нашей работы очень важна для нас",
	"Спасибо, ваша поддержка значит много для нас!",
}

type ResponseGenerator interface {
	GenerateResponse(
		ctx context.Context,
		contextMsg,
		message, typ string,
		req *msg.Request,
	) (string, error)
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
	if !utils.MatchesCommand(req.Message, LikeCommand) {
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

	trackingID := utils.ExtractCommandValue(req.Message, LikeCommand)
	usr := auth.GetUserFromReq(req)
	if usr == nil {
		log.Error("Failed to find user data in the current request")
		return lh.handleErrorCase(ctx)
	}

	log.Debugf("Going to find recommendation tracking for trackingID %q and user %q", trackingID, usr.Login)
	var result Recommendation
	res := lh.db.Where("tracking_id = ?", trackingID).Where("user_id = ?", usr.Login).First(&result)
	if err := res.Error; err != nil {
		log.Errorf("failed to query recommendation: %v", err)
		return lh.handleErrorCase(ctx)
	}

	res = lh.db.Model(&result).Update("likes_count", result.LikesCount+1)
	if res.Error != nil {
		log.Errorf("failed to save like for a recommendation: %v", res.Error)
	} else {
		log.Debugf("Saved a like for recommendation %d", result.ID)
	}

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

	if result.RecommendedWineSummary != "" {
		responseFields = append(responseFields, fmt.Sprintf("Рекомендованное вино: %s", result.RecommendedWineSummary))
	}

	responseMessage, err := lh.respGen.GenerateResponse(
		ctx,
		LikeContextMessage,
		strings.Join(responseFields, "."),
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
