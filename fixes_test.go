package cohere

import (
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/goloop/ai"
)

// Tool calls must be flushed even when the stream ends without a
// tool-call-end event (truncated stream).
func TestFixStreamToolCallWithoutEnd(t *testing.T) {
	events := []string{
		`data: {"type":"tool-call-start","index":0,"delta":{"message":{"tool_calls":` +
			`{"id":"call_1","type":"function","function":{"name":"lookup","arguments":""}}}}}`, ``,
		`data: {"type":"tool-call-delta","index":0,"delta":{"message":{"tool_calls":` +
			`{"function":{"arguments":"{\"q\":42}"}}}}}`, ``,
		// no tool-call-end, but the stream still completes with message-end
		`data: {"type":"message-end","delta":{"usage":{"tokens":{"input_tokens":1,"output_tokens":1}}}}`, ``,
	}
	c, done := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		for _, line := range events {
			io.WriteString(w, line+"\n")
		}
	})
	defer done()

	var call *ai.ToolUse
	for chunk, err := range c.Stream(context.Background(), &ai.Request{
		Model: "m", Messages: []ai.Message{ai.UserText("hi")},
	}) {
		if err != nil {
			t.Fatal(err)
		}
		if chunk.ToolCall != nil {
			call = chunk.ToolCall
		}
	}
	if call == nil || call.Name != "lookup" || string(call.Input) != `{"q":42}` {
		t.Fatalf("tool call lost: %+v", call)
	}
}

// A tool call closed by tool-call-end must be emitted exactly once (no
// duplicate from the post-loop flush).
func TestFixStreamToolCallNoDuplicate(t *testing.T) {
	events := []string{
		`data: {"type":"tool-call-start","index":0,"delta":{"message":{"tool_calls":` +
			`{"id":"call_1","type":"function","function":{"name":"lookup","arguments":""}}}}}`, ``,
		`data: {"type":"tool-call-delta","index":0,"delta":{"message":{"tool_calls":` +
			`{"function":{"arguments":"{}"}}}}}`, ``,
		`data: {"type":"tool-call-end","index":0}`, ``,
		`data: {"type":"message-end","delta":{"usage":{"tokens":{"input_tokens":1,"output_tokens":1}}}}`, ``,
	}
	c, done := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		for _, line := range events {
			io.WriteString(w, line+"\n")
		}
	})
	defer done()

	n := 0
	for chunk, err := range c.Stream(context.Background(), &ai.Request{
		Model: "m", Messages: []ai.Message{ai.UserText("hi")},
	}) {
		if err != nil {
			t.Fatal(err)
		}
		if chunk.ToolCall != nil {
			n++
		}
	}
	if n != 1 {
		t.Errorf("tool call emitted %d times, want 1", n)
	}
}

// Native ChatCompletion must not mutate the caller's request.
func TestFixChatRequestNotMutated(t *testing.T) {
	c, done := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"id":"1","finish_reason":"COMPLETE","message":{"role":"assistant",`+
			`"content":[{"type":"text","text":"hi"}]}}`)
	})
	defer done()

	req := &ChatRequest{Model: "m", Messages: []ChatMessage{{Role: "user", Content: "hi"}}}
	if _, err := c.ChatCompletion(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	if req.Stream {
		t.Error("ChatCompletion mutated req.Stream")
	}
}
