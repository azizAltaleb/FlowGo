package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/elastic/go-elasticsearch/v8/esapi"
)

type Repository struct {
	client *Client
}

func NewRepository(client *Client) *Repository {
	return &Repository{client: client}
}

// Upsert indexes a document by ID. This is the simplest and most robust approach for CDC-based projections.
func (r *Repository) Upsert(ctx context.Context, index string, id string, doc any) error {
	body, err := json.Marshal(doc)
	if err != nil {
		return err
	}

	req := esapi.IndexRequest{
		Index:      index,
		DocumentID: id,
		Body:       bytes.NewReader(body),
		Refresh:    "true",
	}

	res, err := req.Do(ctx, r.client.Raw())
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("elasticsearch upsert failed: index=%s id=%s status=%s", index, id, res.Status())
	}

	return nil
}

func (r *Repository) Delete(ctx context.Context, index string, id string) error {
	req := esapi.DeleteRequest{
		Index:      index,
		DocumentID: id,
		Refresh:    "true",
	}

	res, err := req.Do(ctx, r.client.Raw())
	if err != nil {
		return err
	}
	defer res.Body.Close()

	// 404 is OK (idempotent)
	if res.StatusCode == 404 {
		return nil
	}

	if res.IsError() {
		return fmt.Errorf("elasticsearch delete failed: index=%s id=%s status=%s", index, id, res.Status())
	}

	return nil
}

// Search provides a generic query-side method. Domain-specific query repositories can wrap this.
func (r *Repository) Search(ctx context.Context, index string, query json.RawMessage) (json.RawMessage, error) {
	req := esapi.SearchRequest{
		Index: []string{index},
		Body:  bytes.NewReader(query),
	}

	res, err := req.Do(ctx, r.client.Raw())
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("elasticsearch search failed: index=%s status=%s", index, res.Status())
	}

	var out json.RawMessage
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return nil, err
	}

	return out, nil
}

// UpdateWithScript updates a document using a Painless script.
func (r *Repository) UpdateWithScript(ctx context.Context, index string, id string, script string, params map[string]any) error {
	body := map[string]any{
		"script": map[string]any{
			"source": script,
			"lang":   "painless",
			"params": params,
		},
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req := esapi.UpdateRequest{
		Index:      index,
		DocumentID: id,
		Body:       bytes.NewReader(bodyBytes),
		Refresh:    "true",
	}

	res, err := req.Do(ctx, r.client.Raw())
	if err != nil {
		return err
	}
	defer res.Body.Close()

	// 404 is NOT okay here, we expect the parent document to exist.
	// But if it doesn't, we might want to handle it (upsert?).
	// For now, if it fails, it fails.
	if res.IsError() {
		return fmt.Errorf("elasticsearch update_with_script failed: index=%s id=%s status=%s", index, id, res.Status())
	}

	return nil
}
