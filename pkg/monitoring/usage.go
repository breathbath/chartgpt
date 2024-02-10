package monitoring

import (
	"breathbathChatGPT/pkg/logging"
	"context"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"time"
)

type UsageStats struct {
	gorm.Model
	UserId                string `gorm:"size:255"`
	SessionStart          time.Time
	SessionEnd            time.Time
	InputPromptTokens     int    //the amount of tokens provided by user to chatgpt model
	InputCompletionTokens int    //the amount of additional tokens (conversation cache + system + function)
	GPTModel              string `gorm:"size:255"`
	TrackingID            string `gorm:"size:255"`
	Type                  string `gorm:"size:255"`
	Input                 string
}

func (us *UsageStats) SetTrackingID(ctx context.Context) {
	trackingIdI := ctx.Value(logging.TrackingIDKey)
	if trackingIdI == nil {
		return
	}
	us.TrackingID = trackingIdI.(string)
}

func (us *UsageStats) Save(ctx context.Context, db *gorm.DB) {
	log := logrus.WithContext(ctx)

	res := db.Create(us)
	if res.Error != nil {
		log.Errorf("failed to save usage stats to db: %v", res.Error)
	}
}
