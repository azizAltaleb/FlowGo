package messaging

import (
	"context"
	"time"
)

type EventConsumer interface {
	Start(ctx context.Context) error
}

type TopicStatsSnapshot struct {
	LastOffset      int64     `json:"lastOffset"`
	LastProcessedAt time.Time `json:"lastProcessedAt"`
	LastError       string    `json:"lastError,omitempty"`
}

type ConsumerStatsSnapshot struct {
	Processed       int64                         `json:"processed"`
	Succeeded       int64                         `json:"succeeded"`
	Failed          int64                         `json:"failed"`
	Retried         int64                         `json:"retried"`
	DLQPublished    int64                         `json:"dlqPublished"`
	LastProcessedAt time.Time                     `json:"lastProcessedAt"`
	Topics          map[string]TopicStatsSnapshot `json:"topics"`
}

type ConsumerStatsProvider interface {
	Snapshot() ConsumerStatsSnapshot
}
