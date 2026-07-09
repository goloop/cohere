package cohere_test

import (
	"encoding/json"
	"fmt"

	"github.com/goloop/ai"
	"github.com/goloop/cohere"
)

func ExampleNew() {
	c := cohere.New("...")
	_ = c // use c.Generate, c.Stream, c.ChatCompletion, ...
	fmt.Println(cohere.ModelCommandA)
	// Output: command-a-03-2025
}

// ExampleClient_Generate builds a request. Sending it needs a real API key, so
// this example only shows the shape.
func ExampleClient_Generate() {
	req := &ai.Request{
		Model: cohere.ModelCommandA,
		Messages: []ai.Message{
			ai.UserText("Name the capital of France."),
		},
	}
	fmt.Println(req.Model, len(req.Messages))
	// Output: command-a-03-2025 1
}

// ExampleClient_Rerank shows a rerank request shape. Rerank scores each
// document against the query and returns them ordered most relevant first.
func ExampleClient_Rerank() {
	req := &cohere.RerankRequest{
		Model:     cohere.ModelRerankV35,
		Query:     "What is the capital of Ukraine?",
		Documents: []string{"Kyiv is the capital.", "Bananas are yellow."},
		TopN:      1,
	}
	fmt.Println(req.Model, len(req.Documents))
	// Output: rerank-v3.5 2
}

// ExampleTool shows a tool definition passed with a request.
func ExampleTool() {
	tool := ai.Tool{
		Name:        "get_weather",
		Description: "Get the current weather for a city.",
		Schema:      json.RawMessage(`{"type":"object","properties":{"city":{"type":"string"}}}`),
	}
	fmt.Println(tool.Name)
	// Output: get_weather
}
