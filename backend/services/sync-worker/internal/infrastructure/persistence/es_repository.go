package persistence

import (
	"context"
	"workflow-engine/backend/libs/elasticsearch"
	"workflow-engine/backend/services/sync-worker/internal/domain/repository"
)

type ESRepository struct {
	client *elasticsearch.Repository
}

// Ensure implementation
var _ repository.SyncRepository = &ESRepository{}

func NewESRepository(client *elasticsearch.Repository) *ESRepository {
	return &ESRepository{client: client}
}

func (r *ESRepository) Upsert(ctx context.Context, index string, id string, doc map[string]any) error {
	return r.client.Upsert(ctx, index, id, doc)
}

func (r *ESRepository) Delete(ctx context.Context, index string, id string) error {
	return r.client.Delete(ctx, index, id)
}

func (r *ESRepository) UpdateWithScript(ctx context.Context, index string, id string, script string, params map[string]any) error {
	return r.client.UpdateWithScript(ctx, index, id, script, params)
}
