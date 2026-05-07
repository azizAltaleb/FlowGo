package search_test

import (
	"context"
	"encoding/json"
	"github.com/azizAltaleb/goflow/backend/libs/search"
	"testing"
)

// ComplianceTestSuite defines a reusable test suite for any implementation of search.Backend.
// Since we don't have a mock or real backend easily available in unit tests without docker,
// this serves as a contract compile-time check and a template for integration tests.
func RunComplianceTest(t *testing.T, backend search.Backend) {
	ctx := context.Background()
	index := "test-index"
	id := "test-id"
	doc := map[string]string{"foo": "bar"}

	// 1. Upsert
	if err := backend.Upsert(ctx, index, id, doc); err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	// 2. Search
	query := json.RawMessage(`{"query": {"match_all": {}}}`)
	res, err := backend.Search(ctx, index, query)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(res) == 0 {
		t.Error("Search returned empty response")
	}

	// 3. Delete
	if err := backend.Delete(ctx, index, id); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
}

func TestInterfaceCompliance(t *testing.T) {
	// this is just a compile-time check that the test suite signature is correct
	var _ = RunComplianceTest
}
