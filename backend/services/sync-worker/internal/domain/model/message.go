package model

import "encoding/json"

// DebeziumMessage represents the CDC event structure
type DebeziumMessage struct {
	Before json.RawMessage `json:"before"`
	After  json.RawMessage `json:"after"`
	Op     string          `json:"op"` // c, u, d, r
	Source any             `json:"source"`
	TsMs   *int64          `json:"ts_ms"`
}
