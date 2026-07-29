package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/artificialflow/artificialflow/backend/libs/dmn"
	"github.com/artificialflow/artificialflow/backend/libs/model"
)

func (e *Engine) DeployDecision(ctx context.Context, decisionID, name string, resource []byte) (*model.DecisionDefinition, error) {
	if e.decisions == nil {
		return nil, fmt.Errorf("decision store is not available")
	}
	decisionID = strings.TrimSpace(decisionID)
	if decisionID == "" {
		return nil, fmt.Errorf("decision id is required")
	}
	table, err := dmn.Parse(resource)
	if err != nil {
		return nil, err
	}
	if table.ID != decisionID {
		return nil, fmt.Errorf("decision id mismatch: resource id %q != %q", table.ID, decisionID)
	}
	def := &model.DecisionDefinition{
		DecisionID: decisionID,
		Name:       firstNonEmpty(name, decisionID),
		Resource:   resource,
	}
	if err := e.decisions.UpsertDecision(ctx, def); err != nil {
		return nil, err
	}
	return e.decisions.GetDecisionByDecisionID(ctx, decisionID)
}

func (e *Engine) ListDecisions(ctx context.Context) ([]model.DecisionDefinition, error) {
	if e.decisions == nil {
		return nil, fmt.Errorf("decision store is not available")
	}
	return e.decisions.ListDecisions(ctx)
}

func (e *Engine) EvaluateDecision(ctx context.Context, decisionID string, inputs map[string]any) (map[string]any, error) {
	if e.decisions == nil {
		return nil, fmt.Errorf("decision store is not available")
	}
	def, err := e.decisions.GetDecisionByDecisionID(ctx, strings.TrimSpace(decisionID))
	if err != nil {
		return nil, err
	}
	table, err := dmn.Parse(def.Resource)
	if err != nil {
		return nil, err
	}
	return dmn.Evaluate(table, inputs)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
