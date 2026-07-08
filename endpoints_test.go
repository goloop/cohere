package cohere

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestChatCompletionNative(t *testing.T) {
	c, done := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"id":"1","finish_reason":"COMPLETE","message":{"role":"assistant",`+
			`"content":[{"type":"text","text":"hi"}]}}`)
	})
	defer done()

	resp, err := c.ChatCompletion(context.Background(), &ChatRequest{
		Model:    "m",
		Messages: []ChatMessage{{Role: "user", Content: "hi"}},
	})
	if err != nil || len(resp.Message.Content) != 1 || resp.Message.Content[0].Text != "hi" {
		t.Fatalf("chat: %v %+v", err, resp)
	}
}

func TestChatStreamNative(t *testing.T) {
	events := []string{
		`data: {"type":"content-delta","index":0,"delta":{"message":{"content":{"text":"a"}}}}`, ``,
		`data: {"type":"content-delta","index":0,"delta":{"message":{"content":{"text":"b"}}}}`, ``,
	}
	c, done := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		for _, line := range events {
			io.WriteString(w, line+"\n")
		}
	})
	defer done()

	var text strings.Builder
	for ev, err := range c.ChatStream(context.Background(), &ChatRequest{
		Model: "m", Messages: []ChatMessage{{Role: "user", Content: "hi"}},
	}) {
		if err != nil {
			t.Fatal(err)
		}
		if ev.Type == "content-delta" {
			text.WriteString(ev.Delta.Message.Content.Text)
		}
	}
	if text.String() != "ab" {
		t.Errorf("text = %q", text.String())
	}
}

func TestEmbed(t *testing.T) {
	c, done := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/v2/embed") {
			t.Errorf("path = %q", r.URL.Path)
		}
		io.WriteString(w, `{"embeddings":{"float":[[0.1,0.2],[0.3,0.4]]}}`)
	})
	defer done()

	vecs, err := c.Embed(context.Background(), ModelEmbedV3, "a", "b")
	if err != nil || len(vecs) != 2 || vecs[1][1] != 0.4 {
		t.Fatalf("embed: %v %+v", err, vecs)
	}
}

func TestModels(t *testing.T) {
	c, done := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/v1/models/command-a-03-2025"):
			io.WriteString(w, `{"name":"command-a-03-2025","endpoints":["chat"]}`)
		default:
			io.WriteString(w, `{"models":[{"name":"command-a-03-2025","endpoints":["chat"]}]}`)
		}
	})
	defer done()

	ctx := context.Background()
	if models, err := c.Models(ctx); err != nil || len(models) != 1 {
		t.Fatalf("models: %v %+v", err, models)
	}
	m, err := c.GetModel(ctx, "command-a-03-2025")
	if err != nil || m.Name != "command-a-03-2025" {
		t.Fatalf("get model: %v %+v", err, m)
	}
}
