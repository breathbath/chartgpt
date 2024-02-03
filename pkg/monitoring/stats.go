package monitoring

import (
	logging "breathbathChatGPT/pkg/logging"
	"context"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"sync"
	"time"
)

type UsageStats struct {
	gorm.Model
	LastName              string `gorm:"size:255"`
	UserId                string `gorm:"size:255"`
	FirstName             string `gorm:"size:255"`
	SessionStart          time.Time
	SessionEnd            time.Time
	InputPromptTokens     int    //the amount of tokens provided by user to chatgpt model
	InputCompletionTokens int    //the amount of additional tokens (conversation cache + system + function)
	GPTModel              string `gorm:"size:255"`
	TrackingID            string `gorm:"size:255"`
	Error                 string
	IsVoiceInput          bool
	VoiceToTextModel      string `gorm:"size:255"`
	Input                 string
	GenPromptTokens       int //the amount of tokens used for generation of the final wine recommendation as prompt tokens
	GenCompletionTokens   int //the amount of tokens used for generation of the final wine recommendation as completion tokens
}

type UsageStatsI interface {
	SetLastName(lastName string)
	SetUserId(userId string)
	SetFirstName(firstName string)
	SetSessionStart(sessionStart time.Time)
	SetSessionEnd(sessionEnd time.Time)
	SetInputPromptTokens(inputPromptTokens int)
	SetInputCompletionTokens(inputCompletionTokens int)
	SetGPTModel(gPTModel string)
	SetTrackingID(trackingID string)
	SetError(err error)
	SetIsVoiceInput(isVoiceInput bool)
	SetInput(input string)
	SetGenPromptTokens(genPromptTokens int)
	SetGenCompletionTokens(genCompletionTokens int)
	SetVoiceToTextModel(model string)
	Flush(ctx context.Context, conn *gorm.DB)
}

func (us *UsageStats) SetLastName(lastName string) {
	us.LastName = lastName
	us.cache()
}

func (us *UsageStats) SetUserId(userId string) {
	us.UserId = userId
	us.cache()
}

func (us *UsageStats) SetFirstName(firstName string) {
	us.FirstName = firstName
	us.cache()
}

func (us *UsageStats) SetSessionStart(sessionStart time.Time) {
	us.SessionStart = sessionStart
	us.cache()
}

func (us *UsageStats) SetSessionEnd(sessionEnd time.Time) {
	us.SessionEnd = sessionEnd
	us.cache()
}

func (us *UsageStats) SetInputPromptTokens(inputPromptTokens int) {
	us.InputPromptTokens = inputPromptTokens
	us.cache()
}

func (us *UsageStats) SetInputCompletionTokens(inputCompletionTokens int) {
	us.InputCompletionTokens = inputCompletionTokens
	us.cache()
}

func (us *UsageStats) SetGPTModel(gPTModel string) {
	us.GPTModel = gPTModel
	us.cache()
}

func (us *UsageStats) SetTrackingID(trackingID string) {
	us.TrackingID = trackingID
	us.cache()
}

func (us *UsageStats) SetError(err error) {
	us.Error = err.Error()
	us.cache()
}

func (us *UsageStats) SetIsVoiceInput(isVoiceInput bool) {
	us.IsVoiceInput = isVoiceInput
	us.cache()
}

func (us *UsageStats) SetInput(input string) {
	us.Input = input
	us.cache()
}

func (us *UsageStats) SetGenPromptTokens(genPromptTokens int) {
	us.GenPromptTokens = genPromptTokens
	us.cache()
}

func (us *UsageStats) SetGenCompletionTokens(genCompletionTokens int) {
	us.GenCompletionTokens = genCompletionTokens
	us.cache()
}

func (us *UsageStats) SetVoiceToTextModel(model string) {
	us.VoiceToTextModel = model
	us.cache()
}

func (us *UsageStats) cache() {
	if us.TrackingID != "" {
		stats.Store(us.TrackingID, us)
	}
}

func (us *UsageStats) Flush(ctx context.Context, conn *gorm.DB) {
	log := logrus.WithContext(ctx)

	if us.TrackingID == "" {
		log.Warnf("Will skip usage stat as its missing a tracking id")
		return
	}

	res := conn.Create(us)
	if res.Error != nil {
		log.Errorf("failed to save usage stats to db: %v", res.Error)
	}

	stats.Delete(us.TrackingID)
}

var stats = sync.Map{}

func Usage(ctx context.Context) UsageStatsI {
	trackingIdI := ctx.Value(logging.TrackingIDKey)
	if trackingIdI == nil {
		return &UsageStats{}
	}
	trackingId := trackingIdI.(string)

	statI, found := stats.Load(trackingId)
	if !found {
		return &UsageStats{
			TrackingID: trackingId,
		}
	}

	return statI.(*UsageStats)
}
