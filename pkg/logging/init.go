package logging

import (
	"context"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type loggingContextKey string

const TrackingIDKey loggingContextKey = "trackingID"

func WithTrackingId(ctx context.Context) context.Context {
	trackingID := uuid.New().String()
	ctxWithTrackingId := context.WithValue(ctx, TrackingIDKey, trackingID)
	return ctxWithTrackingId
}

type trackingIDFormatter struct {
	logrus.TextFormatter
}

func (f *trackingIDFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	if entry.Context == nil {
		return f.TextFormatter.Format(entry)
	}

	contextValueI := entry.Context.Value(TrackingIDKey)
	if contextValueI == nil {
		return f.TextFormatter.Format(entry)
	}

	trackingIDField, _ := contextValueI.(string)
	entry.Data["trackingID"] = trackingIDField

	return f.TextFormatter.Format(entry)
}

func Init() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&trackingIDFormatter{
		TextFormatter: logrus.TextFormatter{
			TimestampFormat: "2006-01-02 15:04:05",
			FullTimestamp:   true,
		},
	})
}
