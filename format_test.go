package cohere

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/goloop/ai"
)

func askFor(f *ai.Format) *ai.Request {
	return &ai.Request{
		Model:    "the-model",
		Messages: []ai.Message{ai.UserText("Describe this article.")},
		Format:   f,
	}
}

// TestResponseFormat pins this provider's shape: the schema rides inside the
// same object as the type, not in a nested one.
func TestResponseFormat(t *testing.T) {
	schema := json.RawMessage(`{"type":"object","properties":{"a":{"type":"string"}}}`)

	cr, err := (&Client{}).chatRequest(askFor(&ai.Format{Type: ai.FormatJSON}), false)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(cr.ResponseFormat); got != `{"type":"json_object"}` {
		t.Errorf("response_format = %s", got)
	}

	cr, err = (&Client{}).chatRequest(askFor(&ai.Format{
		Type:   ai.FormatJSONSchema,
		Schema: schema,
	}), false)
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Type   string          `json:"type"`
		Schema json.RawMessage `json:"json_schema"`
	}
	if err := json.Unmarshal(cr.ResponseFormat, &got); err != nil {
		t.Fatalf("response_format is not valid JSON: %v", err)
	}
	if got.Type != "json_object" {
		t.Errorf("type = %q, want json_object", got.Type)
	}
	if string(got.Schema) != string(schema) {
		t.Errorf("json_schema = %s, want %s", got.Schema, schema)
	}

	for _, f := range []*ai.Format{nil, {Type: ai.FormatText}} {
		cr, err := (&Client{}).chatRequest(askFor(f), false)
		if err != nil {
			t.Fatal(err)
		}
		if cr.ResponseFormat != nil {
			t.Errorf("response_format = %s, want it absent", cr.ResponseFormat)
		}
		if got := formatMode(f); got != ai.FormatNone {
			t.Errorf("formatMode = %s, want none", got)
		}
	}

	if got := formatMode(&ai.Format{Type: ai.FormatJSON}); got != ai.FormatNative {
		t.Errorf("formatMode = %s, want native", got)
	}
}

// TestResponseFormatRejectsUnknown checks a format this driver cannot render
// fails here, not as a puzzling error from the provider.
func TestResponseFormatRejectsUnknown(t *testing.T) {
	_, err := (&Client{}).chatRequest(askFor(&ai.Format{Type: ai.FormatType(99)}), false)
	if !errors.Is(err, ai.ErrBadFormat) {
		t.Errorf("err = %v, want %v", err, ai.ErrBadFormat)
	}
}
