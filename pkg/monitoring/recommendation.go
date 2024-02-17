package monitoring

import (
	"breathbathChatGPT/pkg/logging"
	"context"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Recommendation struct {
	gorm.Model
	logging.TrackingIDModel `gorm:"embedded"`
	UserID                  string `gorm:"size:255"`
	FunctionCall            string
	DBQuery                 string
	UserPrompt              string
	RecommendedWineSummary  string
	RecommendedWineID       string
	RecommendationText      string
	RawModelInput           string
	RawModelOutput          string
}

func (r *Recommendation) Save(ctx context.Context, db *gorm.DB) {
	log := logrus.WithContext(ctx)

	res := db.Create(r)
	if res.Error != nil {
		log.Errorf("failed to save recommendation tracking to db: %v", res.Error)
	}
}
