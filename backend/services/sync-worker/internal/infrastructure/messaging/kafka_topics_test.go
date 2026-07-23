package messaging

import (
	"errors"
	"testing"

	"github.com/segmentio/kafka-go"
)

func TestUniqueTopicsTrimsAndDeduplicates(t *testing.T) {
	got := uniqueTopics([]string{
		" artificialflow.public.process ",
		"workflow.events.v1",
		"artificialflow.public.process",
		"",
		" workflow.events.v1",
	})
	want := []string{"artificialflow.public.process", "workflow.events.v1"}
	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, got)
		}
	}
}

func TestIsIgnorableTopicCreateError(t *testing.T) {
	if !isIgnorableTopicCreateError(nil) {
		t.Fatal("nil error should be ignorable")
	}
	if !isIgnorableTopicCreateError(kafka.TopicAlreadyExists) {
		t.Fatal("TopicAlreadyExists should be ignorable")
	}
	if !isIgnorableTopicCreateError(errors.New("Topic 'x' already exists")) {
		t.Fatal("already-exists message should be ignorable")
	}
	if isIgnorableTopicCreateError(errors.New("broker not available")) {
		t.Fatal("unrelated errors must not be ignored")
	}
}

func TestIsUnknownTopicError(t *testing.T) {
	if !isUnknownTopicError(kafka.UnknownTopicOrPartition) {
		t.Fatal("UnknownTopicOrPartition should match")
	}
	if isUnknownTopicError(errors.New("connection reset")) {
		t.Fatal("unrelated errors must not match")
	}
}
