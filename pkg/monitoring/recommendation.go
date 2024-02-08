package monitoring

import (
	"breathbathChatGPT/pkg/logging"
	"context"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"sync"
)

type Recommendation struct {
	gorm.Model
	TrackingID             string
	UserID                 string `gorm:"size:255"`
	FunctionCall           string
	DBQuery                string
	UserPrompt             string
	RecommendedWineSummary string
	RecommendedWineID      string
	RecommendationText     string
	RawModelInput          string
	RawModelOutput         string
}

func (us *Recommendation) SetUserID(userID string) {
	us.UserID = userID
	us.cache()
}

func (us *Recommendation) SetFunctionCall(function string) {
	us.FunctionCall = function
	us.cache()
}

func (us *Recommendation) SetDBQuery(query string) {
	us.DBQuery = query
	us.cache()
}

func (us *Recommendation) SetUserPrompt(prompt string) {
	us.UserPrompt = prompt
	us.cache()
}

func (us *Recommendation) SetRecommendedWineSummary(summary string) {
	us.RecommendedWineSummary = summary
	us.cache()
}

func (us *Recommendation) SetRecommendedWineID(id string) {
	us.RecommendedWineID = id
	us.cache()
}

func (us *Recommendation) SetRawModelInput(modelInput interface{}) {
	data, err := json.Marshal(modelInput)
	if err != nil {
		us.RawModelInput = fmt.Sprint(modelInput)
	} else {
		us.RawModelInput = string(data)
	}

	us.cache()
}

func (us *Recommendation) SetRawModelOutput(modelOutput interface{}) {
	data, err := json.Marshal(modelOutput)
	if err != nil {
		us.RawModelOutput = fmt.Sprint(modelOutput)
	} else {
		us.RawModelOutput = string(data)
	}

	us.cache()
}

func (us *Recommendation) SetRecommendationText(recommendationText string) {
	us.RecommendationText = recommendationText
	us.cache()
}

type RecommendationI interface {
	SetUserID(UserID string)
	SetFunctionCall(function string)
	SetDBQuery(query string)
	SetUserPrompt(prompt string)
	SetRecommendedWineSummary(summary string)
	SetRecommendedWineID(id string)
	SetRawModelInput(input interface{})
	SetRecommendationText(recommendationText string)
	SetRawModelOutput(modelOutput interface{})
	Flush(ctx context.Context, conn *gorm.DB)
}

func (r *Recommendation) cache() {
	if r.TrackingID != "" {
		recommendations.Store(r.TrackingID, r)
	}
}

func (r *Recommendation) Flush(ctx context.Context, conn *gorm.DB) {
	log := logrus.WithContext(ctx)

	if r.TrackingID == "" {
		log.Warnf("Will skip usage stat as its missing a tracking id")
		return
	}

	res := conn.Create(r)
	if res.Error != nil {
		log.Errorf("failed to save usage stats to db: %v", res.Error)
	}

	recommendations.Delete(r.TrackingID)
}

var recommendations = sync.Map{}

func TrackRecommend(ctx context.Context) RecommendationI {
	trackingIdI := ctx.Value(logging.TrackingIDKey)
	if trackingIdI == nil {
		return &Recommendation{}
	}
	trackingId := trackingIdI.(string)

	recI, found := recommendations.Load(trackingId)
	if !found {
		return &Recommendation{
			TrackingID: trackingId,
		}
	}

	return recI.(*Recommendation)
}
