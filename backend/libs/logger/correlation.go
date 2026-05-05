package logger

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// GenerateCorrelationID generates a unique correlation ID
// Format: <timestamp_hex>-<random_hex>
func GenerateCorrelationID() string {
	timestamp := time.Now().UnixNano()
	randomBytes := make([]byte, 4)
	_, _ = rand.Read(randomBytes)
	return fmt.Sprintf("%x-%s", timestamp, hex.EncodeToString(randomBytes))
}

// GenerateMessageCorrelationID generates a correlation ID for a Kafka message
// Format: <topic_short>-<partition>-<offset>-<random>
func GenerateMessageCorrelationID(topic string, partition int, offset int64) string {
	// Extract table name from topic (last part after dots)
	shortTopic := topic
	for i := len(topic) - 1; i >= 0; i-- {
		if topic[i] == '.' {
			shortTopic = topic[i+1:]
			break
		}
	}

	randomBytes := make([]byte, 2)
	_, _ = rand.Read(randomBytes)
	return fmt.Sprintf("%s-p%d-o%d-%s", shortTopic, partition, offset, hex.EncodeToString(randomBytes))
}
