package repository

import "context"

// SyncRepository defines the interface for syncing data to the read model (Elasticsearch)
type SyncRepository interface {
	Upsert(ctx context.Context, index string, id string, doc map[string]any) error
	Delete(ctx context.Context, index string, id string) error
	UpdateWithScript(ctx context.Context, index string, id string, script string, params map[string]any) error
}
