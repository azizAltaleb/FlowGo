package search

import (
	"context"
	"encoding/json"
)

// Backend defines the contract for search engine implementations (Elasticsearch, OpenSearch, etc.).
type Backend interface {
	// Search executes a raw JSON query against the specified index.
	Search(ctx context.Context, index string, query json.RawMessage) (json.RawMessage, error)

	// Upsert indexes a document by ID, replacing it if it exists.
	Upsert(ctx context.Context, index string, id string, doc any) error

	// Delete removes a document by ID.
	Delete(ctx context.Context, index string, id string) error
}
