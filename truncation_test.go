package cohere

import (
	"context"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/goloop/ai"
)

// TestStreamTruncated verifies a stream cut off before message-end surfaces
// io.ErrUnexpectedEOF, after salvaging any partial tool call as a best effort.
func TestStreamTruncated(t *testing.T) {
	events := []string{
		`data: {"type":"tool-call-start","index":0,"delta":{"message":{"tool_calls":` +
			`{"id":"call_1","type":"function","function":{"name":"lookup","arguments":""}}}}}`, ``,
		`data: {"type":"tool-call-delta","index":0,"delta":{"message":{"tool_calls":` +
			`{"function":{"arguments":"{\"q\":1}"}}}}}`, ``,
		// connection ends here: no tool-call-end, no message-end
	}
	c, done := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		for _, line := range events {
			io.WriteString(w, line+"\n")
		}
	})
	defer done()

	var salvaged *ai.ToolUse
	var gotErr error
	for chunk, err := range c.Stream(context.Background(), &ai.Request{
		Model: "m", Messages: []ai.Message{ai.UserText("hi")},
	}) {
		if err != nil {
			gotErr = err
			break
		}
		if chunk.ToolCall != nil {
			salvaged = chunk.ToolCall
		}
	}
	if !errors.Is(gotErr, io.ErrUnexpectedEOF) {
		t.Fatalf("err = %v, want io.ErrUnexpectedEOF", gotErr)
	}
	if salvaged == nil || salvaged.Name != "lookup" {
		t.Errorf("partial tool call not salvaged: %+v", salvaged)
	}
}
