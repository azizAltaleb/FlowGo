package application

import (
	"context"
	"testing"
)

type recordingRepository struct {
	index string
}

func (r *recordingRepository) Upsert(_ context.Context, index, _ string, _ map[string]any) error {
	r.index = index
	return nil
}

func (r *recordingRepository) Delete(_ context.Context, index, _ string) error {
	r.index = index
	return nil
}

func (r *recordingRepository) UpdateWithScript(_ context.Context, index, _ string, _ string, _ map[string]any) error {
	r.index = index
	return nil
}

func TestSyncServiceDefaultsToCanonicalIndexPrefix(t *testing.T) {
	service := NewSyncService(&recordingRepository{}, "")

	if got := service.indexNameForTopic("artificialflow.public.process_instance"); got != "artificialflow-process_instance" {
		t.Fatalf("expected canonical index, got %q", got)
	}
	if got := service.indexNameForTopic("artificialflow.public.job"); got != "artificialflow-job" {
		t.Fatalf("source topic prefix must not control the write index, got %q", got)
	}
}

func TestSyncServicePreservesExplicitLegacyIndexOverride(t *testing.T) {
	service := NewSyncService(&recordingRepository{}, "artificialflow")

	if got := service.indexNameForTopic("artificialflow.public.process"); got != "artificialflow-process" {
		t.Fatalf("expected explicit legacy index override, got %q", got)
	}
}
