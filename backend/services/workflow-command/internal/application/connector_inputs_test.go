package application

import "testing"

func TestConnectorInputsFromProperties(t *testing.T) {
	got := connectorInputsFromProperties(map[string]any{
		"url":          "https://api.example.com",
		"method":       "GET",
		"unrelated":    "x",
		"emailSubject": "   ",
		"kafkaTopic":   "orders",
	})
	if got["url"] != "https://api.example.com" {
		t.Fatalf("url: %v", got["url"])
	}
	if _, ok := got["unrelated"]; ok {
		t.Fatal("unrelated should be skipped")
	}
	if _, ok := got["emailSubject"]; ok {
		t.Fatal("blank emailSubject should be skipped")
	}
	if got["kafkaTopic"] != "orders" {
		t.Fatalf("kafkaTopic: %v", got["kafkaTopic"])
	}
}
