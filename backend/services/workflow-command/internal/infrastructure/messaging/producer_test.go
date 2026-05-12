package messaging

import (
	workflowapi "github.com/azizAltaleb/flowgo/backend/api/v1/go"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestAggregateIDForEvent(t *testing.T) {
	tests := []struct {
		name  string
		event proto.Message
		want  string
	}{
		{
			name:  "process instance",
			event: &workflowapi.ProcessInstanceCreated{Key: 1001},
			want:  "process-instance:1001",
		},
		{
			name:  "job event",
			event: &workflowapi.JobActivated{Key: 2002},
			want:  "job:2002",
		},
		{
			name:  "variable with process instance",
			event: &workflowapi.VariableUpdated{ProcessInstanceKey: 3003, ScopeKey: 9},
			want:  "process-instance:3003",
		},
		{
			name:  "unknown event type",
			event: &emptypb.Empty{},
			want:  "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := aggregateIDForEvent(tc.event)
			if got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
	}
}
