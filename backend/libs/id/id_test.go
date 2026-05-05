package id_test

import (
	"testing"
	"time"
	"workflow-engine/backend/libs/id"
)

func TestSnowflakeGeneration(t *testing.T) {
	// Initialize default node
	id.InitDefaultNode(1)

	// Generate a few IDs
	id1 := id.GenerateSnowflake()
	id2 := id.GenerateSnowflake()

	if id1 == 0 {
		t.Error("Generated ID should not be 0")
	}
	if id2 == 0 {
		t.Error("Generated ID should not be 0")
	}
	if id1 >= id2 {
		t.Errorf("IDs should be increasing: %d >= %d", id1, id2)
	}

	// Verify uniqueness (simple check)
	count := 1000
	ids := make(map[int64]bool)
	for i := 0; i < count; i++ {
		uid := id.GenerateSnowflake()
		if ids[uid] {
			t.Errorf("Duplicate ID generated: %d", uid)
		}
		ids[uid] = true
	}
}

func TestUUIDv7Generation(t *testing.T) {
	uuid1 := id.GenerateUUIDv7()
	if uuid1 == "" {
		t.Error("Generated UUID should not be empty")
	}

	// Verify it looks like a UUID (length check)
	if len(uuid1) != 36 {
		t.Errorf("Invalid UUID length: %d, expected 36", len(uuid1))
	}

	// Wait a bit to ensure time difference if v7 is time-ordered
	time.Sleep(2 * time.Millisecond)

	uuid2 := id.GenerateUUIDv7()
	if uuid1 == uuid2 {
		t.Error("Generated UUIDs should be unique")
	}

	// Lexicographical comparison for v7 should roughly work for ordering
	if uuid1 >= uuid2 {
		t.Logf("Warning: UUIDv7 might not be strictly ordered strings if fallback v4 used, or clock issues: %s >= %s", uuid1, uuid2)
	}
}
