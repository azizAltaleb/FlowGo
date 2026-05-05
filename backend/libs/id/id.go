package id

import (
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Snowflake Generator Constants
const (
	nodeBits        = 10
	stepBits        = 12
	nodeMax         = -1 ^ (-1 << nodeBits)
	stepMax         = -1 ^ (-1 << stepBits)
	timeShift       = nodeBits + stepBits
	nodeShift       = stepBits
	epoch     int64 = 1704067200000 // 2024-01-01 00:00:00 UTC
)

type Node struct {
	mu        sync.Mutex
	timestamp int64
	node      int64
	step      int64
}

var defaultNode *Node
var once sync.Once

// NewNode creates a new Snowflake node
func NewNode(node int64) (*Node, error) {
	if node < 0 || node > nodeMax {
		return nil, errors.New("node number must be between 0 and 1023")
	}

	return &Node{
		timestamp: 0,
		node:      node,
		step:      0,
	}, nil
}

// Generate creates a new unique int64 ID
func (n *Node) Generate() int64 {
	n.mu.Lock()
	defer n.mu.Unlock()

	now := time.Now().UnixMilli()

	if now < n.timestamp {
		// Clock regressed, refuse to generate ID (or wait)
		// For simplicity in this implementation, we just use the last timestamp
		now = n.timestamp
	}

	if now == n.timestamp {
		n.step = (n.step + 1) & stepMax
		if n.step == 0 {
			// Sequence exhausted, wait for next millisecond
			for now <= n.timestamp {
				now = time.Now().UnixMilli()
			}
		}
	} else {
		n.step = 0
	}

	n.timestamp = now

	return ((now - epoch) << timeShift) | (n.node << nodeShift) | n.step
}

// InitDefaultNode initializes the global default node (node ID 0 by default)
func InitDefaultNode(nodeID int64) {
	var err error
	defaultNode, err = NewNode(nodeID)
	if err != nil {
		panic(err)
	}
}

// GenerateSnowflake generates an int64 ID using the default node
func GenerateSnowflake() int64 {
	once.Do(func() {
		if defaultNode == nil {
			InitDefaultNode(1) // Default to node 1
		}
	})
	return defaultNode.Generate()
}

// GenerateUUIDv7 generates a new UUIDv7 string
func GenerateUUIDv7() string {
	if id, err := uuid.NewV7(); err == nil {
		return id.String()
	}
	return uuid.NewString() // Fallback to v4
}
