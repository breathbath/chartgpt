package logging

import (
	"context"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type loggingContextKey string

const trackingIDKey loggingContextKey = "trackingID"

func WithTrackingId(ctx context.Context) context.Context {
	trackingID := uuid.New().String()
	ctxWithTrackingId := context.WithValue(ctx, trackingIDKey, trackingID)
	return ctxWithTrackingId
}

type trackingIDFormatter struct {
	logrus.TextFormatter
}

func (f *trackingIDFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	if entry.Context == nil {
		return f.TextFormatter.Format(entry)
	}

	contextValueI := entry.Context.Value(trackingIDKey)
	if contextValueI == nil {
		return f.TextFormatter.Format(entry)
	}

	trackingIDField, _ := contextValueI.(string)
	entry.Data["trackingID"] = trackingIDField
	return f.TextFormatter.Format(entry)
}

func Init() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&trackingIDFormatter{})
}
