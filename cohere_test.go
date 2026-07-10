package cohere

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/goloop/ai"
)

func newTestClient(t *testing.T, h http.HandlerFunc) (*Client, func()) {
	t.Helper()
	srv := httptest.NewServer(h)
	c := New("key", WithBaseURL(srv.URL), WithMaxRetries(0))
	return c, srv.Close
}

func TestGenerateText(t *testing.T) {
	c, done := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer key" {
			t.Errorf("auth = %q", r.Header.Get("Authorization"))
		}
		if !strings.HasSuffix(r.URL.Path, "/v2/chat") {
			t.Errorf("path = %q", r.URL.Path)
		}
		io.WriteString(w, `{"id":"1","finish_reason":"COMPLETE","message":{"role":"assistant",`+
			`"content":[{"type":"text","text":"hello"}]},`+
			`"usage":{"tokens":{"input_tokens":3,"output_tokens":2}}}`)
	})
	defer done()

	resp, err := c.Generate(context.Background(), &ai.Request{
		Model:    "m",
		Messages: []ai.Message{ai.UserText("hi")},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Text() != "hello" {
		t.Errorf("text = %q", resp.Text())
	}
	if resp.Usage.InputTokens != 3 || resp.Usage.OutputTokens != 2 {
		t.Errorf("usage = %+v", resp.Usage)
	}
	if resp.Model != "m" || resp.StopReason != "COMPLETE" {
		t.Errorf("model/stop = %q %q", resp.Model, resp.StopReason)
	}
}

func TestGenerateToolUse(t *testing.T) {
	c, done := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		var req ChatRequest
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &req)
		if req.ToolChoice != "REQUIRED" {
			t.Errorf("tool_choice = %q", req.ToolChoice)
		}
		io.WriteString(w, `{"id":"1","finish_reason":"TOOL_CALL","message":{"role":"assistant",`+
			`"tool_calls":[{"id":"call_1","type":"function","function":{"name":"get_weather",`+
			`"arguments":"{\"city\":\"Kyiv\"}"}}]}}`)
	})
	defer done()

	resp, err := c.Generate(context.Background(), &ai.Request{
		Model:      "m",
		Messages:   []ai.Message{ai.UserText("weather?")},
		Tools:      []ai.Tool{{Name: "get_weather"}},
		ToolChoice: ai.ToolRequired,
	})
	if err != nil {
		t.Fatal(err)
	}
	calls := resp.ToolCalls()
	if len(calls) != 1 || calls[0].Name != "get_weather" || calls[0].ID != "call_1" {
		t.Fatalf("calls = %+v", calls)
	}
	if string(calls[0].Input) != `{"city":"Kyiv"}` {
		t.Errorf("input = %s", calls[0].Input)
	}
}

func TestStream(t *testing.T) {
	events := []string{
		`data: {"type":"content-delta","index":0,"delta":{"message":{"content":{"text":"Hel"}}}}`, ``,
		`data: {"type":"content-delta","index":0,"delta":{"message":{"content":{"text":"lo"}}}}`, ``,
		`data: {"type":"message-end","delta":{"finish_reason":"COMPLETE",` +
			`"usage":{"tokens":{"input_tokens":5,"output_tokens":2}}}}`, ``,
	}
	c, done := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		for _, line := range events {
			io.WriteString(w, line+"\n")
		}
	})
	defer done()

	var text strings.Builder
	var usage ai.Usage
	var doneSeen bool
	for chunk, err := range c.Stream(context.Background(), &ai.Request{
		Model: "m", Messages: []ai.Message{ai.UserText("hi")},
	}) {
		if err != nil {
			t.Fatal(err)
		}
		text.WriteString(chunk.Text)
		if chunk.Done {
			doneSeen = true
			if chunk.Usage != nil {
				usage = *chunk.Usage
			}
		}
	}
	if text.String() != "Hello" || !doneSeen {
		t.Errorf("text = %q done = %v", text.String(), doneSeen)
	}
	if usage.InputTokens != 5 || usage.OutputTokens != 2 {
		t.Errorf("usage = %+v", usage)
	}
}

func TestStreamToolCall(t *testing.T) {
	events := []string{
		`data: {"type":"tool-call-start","index":0,"delta":{"message":{"tool_calls":` +
			`{"id":"call_1","type":"function","function":{"name":"lookup","arguments":""}}}}}`, ``,
		`data: {"type":"tool-call-delta","index":0,"delta":{"message":{"tool_calls":` +
			`{"function":{"arguments":"{\"q\":42}"}}}}}`, ``,
		`data: {"type":"tool-call-end","index":0}`, ``,
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
	if call == nil || call.Name != "lookup" || call.ID != "call_1" ||
		string(call.Input) != `{"q":42}` {
		t.Fatalf("call = %+v", call)
	}
}

func TestError(t *testing.T) {
	c, done := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		io.WriteString(w, `{"message":"invalid api token"}`)
	})
	defer done()

	_, err := c.Generate(context.Background(), &ai.Request{
		Model: "m", Messages: []ai.Message{ai.UserText("hi")},
	})
	var apiErr *ai.APIError
	if !errors.As(err, &apiErr) || apiErr.Status != 401 ||
		apiErr.Message != "invalid api token" {
		t.Fatalf("err = %v", err)
	}
}

func TestChatMessagesMapping(t *testing.T) {
	req := &ai.Request{
		System: "sys",
		Messages: []ai.Message{
			{Role: ai.RoleUser, Parts: []ai.Part{
				ai.Text{Text: "look"},
				ai.Image{MIME: "image/png", Data: []byte{1, 2, 3}},
			}},
			{Role: ai.RoleAssistant, Parts: []ai.Part{
				ai.ToolUse{ID: "call_1", Name: "t", Input: json.RawMessage(`{}`)},
			}},
			{Role: ai.RoleTool, Parts: []ai.Part{
				ai.ToolResult{ID: "call_1", Content: "42"},
			}},
		},
	}
	msgs := chatMessages(req)
	if msgs[0].Role != "system" || msgs[0].Content != "sys" {
		t.Errorf("system = %+v", msgs[0])
	}
	blocks, ok := msgs[1].Content.([]ContentBlock)
	if !ok || len(blocks) != 2 || blocks[1].ImageURL == nil {
		t.Fatalf("user content = %+v", msgs[1].Content)
	}
	if !strings.HasPrefix(blocks[1].ImageURL.URL, "data:image/png;base64,") {
		t.Errorf("image url = %q", blocks[1].ImageURL.URL)
	}
	if len(msgs[2].ToolCalls) != 1 || msgs[2].ToolCalls[0].ID != "call_1" {
		t.Errorf("assistant = %+v", msgs[2])
	}
	if msgs[3].Role != "tool" || msgs[3].ToolCallID != "call_1" {
		t.Errorf("tool = %+v", msgs[3])
	}
}

func TestValidate(t *testing.T) {
	c := New("key")
	_, err := c.Generate(context.Background(), &ai.Request{Model: "m"})
	if !errors.Is(err, ai.ErrNoMessages) {
		t.Errorf("want ErrNoMessages, got %v", err)
	}
}
