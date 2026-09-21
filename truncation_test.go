package cohere

import (
	"context"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/goloop/ai"
)

// streamOf serves the given SSE lines and runs Stream over them, returning
// the tool calls handed out, whether Done was reported and the final error.
func streamOf(t *testing.T, events []string) (calls []*ai.ToolUse, doneSeen bool, err error) {
	t.Helper()
	c, done := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		for _, line := range events {
			io.WriteString(w, line+"\n")
		}
	})
	defer done()

	for chunk, e := range c.Stream(context.Background(), &ai.Request{
		Model: "m", Messages: []ai.Message{ai.UserText("hi")},
	}) {
		if e != nil {
			return calls, doneSeen, e
		}
		if chunk.ToolCall != nil {
			calls = append(calls, chunk.ToolCall)
		}
		if chunk.Done {
			doneSeen = true
		}
	}
	return calls, doneSeen, nil
}

// A stream cut off before message-end is io.ErrUnexpectedEOF, and the tool
// call it left open is not handed out: a consumer that runs tools and stops
// at the first one would otherwise act on arguments the model never
// finished, and never see the error.
func TestStreamTruncatedKeepsOpenToolCall(t *testing.T) {
	calls, doneSeen, err := streamOf(t, []string{
		`data: {"type":"tool-call-start","index":0,"delta":{"message":{"tool_calls":` +
			`{"id":"call_1","type":"function","function":{"name":"lookup","arguments":""}}}}}`, ``,
		`data: {"type":"tool-call-delta","index":0,"delta":{"message":{"tool_calls":` +
			`{"function":{"arguments":"{\"q\":1}"}}}}}`, ``,
		// connection ends here: no tool-call-end, no message-end
	})
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("err = %v, want io.ErrUnexpectedEOF", err)
	}
	if len(calls) != 0 {
		t.Errorf("got %d tool calls from a cut-off stream, want none", len(calls))
	}
	if doneSeen {
		t.Error("a cut-off stream reported Done")
	}
}

// A tool call that did get its tool-call-end is handed out even when the
// stream is cut off afterwards: the call itself is complete.
func TestStreamTruncatedAfterToolCallEnd(t *testing.T) {
	calls, _, err := streamOf(t, []string{
		`data: {"type":"tool-call-start","index":0,"delta":{"message":{"tool_calls":` +
			`{"id":"call_1","type":"function","function":{"name":"lookup","arguments":""}}}}}`, ``,
		`data: {"type":"tool-call-delta","index":0,"delta":{"message":{"tool_calls":` +
			`{"function":{"arguments":"{\"q\":1}"}}}}}`, ``,
		`data: {"type":"tool-call-end","index":0}`, ``,
	})
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("err = %v, want io.ErrUnexpectedEOF", err)
	}
	if len(calls) != 1 || calls[0].Name != "lookup" || string(calls[0].Input) != `{"q":1}` {
		t.Errorf("calls = %+v, want the finished lookup call", calls)
	}
}

// Tool arguments that are not valid JSON are an error, not an Input that
// cannot be decoded.
func TestStreamInvalidToolArgumentsAreAnError(t *testing.T) {
	calls, _, err := streamOf(t, []string{
		`data: {"type":"tool-call-start","index":0,"delta":{"message":{"tool_calls":` +
			`{"id":"call_1","type":"function","function":{"name":"lookup","arguments":""}}}}}`, ``,
		`data: {"type":"tool-call-delta","index":0,"delta":{"message":{"tool_calls":` +
			`{"function":{"arguments":"{\"q\":"}}}}}`, ``,
		`data: {"type":"tool-call-end","index":0}`, ``,
		`data: {"type":"message-end","delta":{"finish_reason":"TOOL_CALL","usage":{"tokens":{"input_tokens":1,"output_tokens":1}}}}`, ``,
	})
	if len(calls) != 0 {
		t.Errorf("got %d tool calls with invalid arguments, want none", len(calls))
	}
	if err == nil {
		t.Error("invalid tool arguments were not reported")
	}
}

// An event that is not JSON is a broken stream, not something to skip.
func TestStreamBrokenEventIsAnError(t *testing.T) {
	_, doneSeen, err := streamOf(t, []string{
		`data: {broken`, ``,
		`data: {"type":"message-end","delta":{"finish_reason":"COMPLETE","usage":{"tokens":{"input_tokens":1,"output_tokens":1}}}}`, ``,
	})
	if err == nil {
		t.Error("a broken event was skipped without an error")
	}
	if doneSeen {
		t.Error("a stream with a broken event reported Done")
	}
}
