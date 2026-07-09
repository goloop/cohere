//go:build integration

// Integration smoke tests hit the live Cohere API. They are excluded from the
// normal build and run only with the "integration" tag and a real key:
//
//	COHERE_API_KEY=... go test -tags integration -run Integration ./...
package cohere_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/goloop/ai"
	"github.com/goloop/cohere"
)

const integrationModel = cohere.ModelCommandR

func integrationClient(t *testing.T) *cohere.Client {
	t.Helper()
	key := os.Getenv("COHERE_API_KEY")
	if key == "" {
		t.Skip("set COHERE_API_KEY to run integration tests")
	}
	return cohere.New(key)
}

func TestIntegrationGenerate(t *testing.T) {
	c := integrationClient(t)
	resp, err := c.Generate(context.Background(), &ai.Request{
		Model:     integrationModel,
		MaxTokens: 16,
		Messages:  []ai.Message{ai.UserText("Reply with exactly one word: pong")},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Text() == "" {
		t.Fatal("empty text")
	}
	t.Logf("generate: %q (in=%d out=%d)", resp.Text(), resp.Usage.InputTokens, resp.Usage.OutputTokens)
}

func TestIntegrationStream(t *testing.T) {
	c := integrationClient(t)
	var text string
	var done bool
	for chunk, err := range c.Stream(context.Background(), &ai.Request{
		Model:     integrationModel,
		MaxTokens: 32,
		Messages:  []ai.Message{ai.UserText("Count from 1 to 5.")},
	}) {
		if err != nil {
			t.Fatal(err)
		}
		text += chunk.Text
		if chunk.Done {
			done = true
		}
	}
	if text == "" || !done {
		t.Fatalf("text=%q done=%v", text, done)
	}
	t.Logf("stream: %q done=%v", text, done)
}

func TestIntegrationTools(t *testing.T) {
	c := integrationClient(t)
	resp, err := c.Generate(context.Background(), &ai.Request{
		Model:     integrationModel,
		MaxTokens: 128,
		Messages:  []ai.Message{ai.UserText("What is the weather in Kyiv? Use the tool.")},
		Tools: []ai.Tool{{
			Name:        "get_weather",
			Description: "Get the current weather for a city.",
			Schema:      json.RawMessage(`{"type":"object","properties":{"city":{"type":"string"}},"required":["city"]}`),
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Text() == "" && len(resp.ToolCalls()) == 0 {
		t.Fatal("neither text nor tool call")
	}
	t.Logf("tools: text=%q calls=%d", resp.Text(), len(resp.ToolCalls()))
}

func TestIntegrationEmbed(t *testing.T) {
	c := integrationClient(t)
	vecs, err := c.Embed(context.Background(), cohere.ModelEmbedV3, "hello", "world")
	if err != nil {
		t.Fatal(err)
	}
	if len(vecs) != 2 || len(vecs[0]) == 0 {
		t.Fatalf("vectors = %d x %d", len(vecs), len(vecs[0]))
	}
	t.Logf("embed: %d vectors of dim %d", len(vecs), len(vecs[0]))
}

func TestIntegrationRerank(t *testing.T) {
	c := integrationClient(t)
	res, err := c.Rerank(context.Background(), &cohere.RerankRequest{
		Model:     cohere.ModelRerankV35,
		Query:     "What is the capital of Ukraine?",
		Documents: []string{"Bananas are yellow.", "Kyiv is the capital of Ukraine."},
		TopN:      2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res) == 0 {
		t.Fatal("no rerank results")
	}
	// The Kyiv document (index 1) should rank first.
	if res[0].Index != 1 {
		t.Errorf("top result index = %d, want 1", res[0].Index)
	}
	t.Logf("rerank: top index=%d score=%.3f", res[0].Index, res[0].RelevanceScore)
}
