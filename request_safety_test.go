package cohere

import (
	"context"
	"errors"
	"fmt"
	"github.com/goloop/ai"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNilRequests(t *testing.T) {
	c := New("")
	if _, err := c.ChatCompletion(context.Background(), nil); !errors.Is(err, ai.ErrNoRequest) {
		t.Fatalf("chat: %v", err)
	}
	if _, err := c.Embeddings(context.Background(), nil); !errors.Is(err, ai.ErrNoRequest) {
		t.Fatalf("embed: %v", err)
	}
	for _, err := range c.ChatStream(context.Background(), nil) {
		if !errors.Is(err, ai.ErrNoRequest) {
			t.Fatalf("stream: %v", err)
		}
	}
}

func TestEmbeddingsKeepsRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, `{"embeddings":{"float":[[1,2]]}}`) }))
	defer srv.Close()
	req := &EmbedRequest{Model: "test", Texts: []string{"hello"}}
	if _, err := New("", WithBaseURL(srv.URL)).Embeddings(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	if req.EmbeddingTypes != nil {
		t.Fatal("request was modified")
	}
}

func TestChatStreamRejectsMalformedEvent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "data: {broken\n\n") }))
	defer srv.Close()
	var got error
	for _, err := range New("", WithBaseURL(srv.URL)).ChatStream(context.Background(), &ChatRequest{Model: "test"}) {
		got = err
	}
	if got == nil {
		t.Fatal("malformed event was silently discarded")
	}
}
