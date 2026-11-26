package cohere

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"net/http"

	"github.com/goloop/ai"
)

// StreamEvent is one streamed v2 chat event. The Type field selects which
// nested fields are populated ("content-delta", "tool-call-start",
// "tool-call-delta", "tool-call-end", "message-end", ...).
type StreamEvent struct {
	Type  string `json:"type"`
	Index int    `json:"index"`
	Delta struct {
		FinishReason string `json:"finish_reason"`
		Message      struct {
			Content struct {
				Text string `json:"text"`
			} `json:"content"`
			ToolCalls struct {
				ID       string `json:"id"`
				Type     string `json:"type"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"message"`
		Usage Usage `json:"usage"`
	} `json:"delta"`
}

type toolAcc struct {
	id, name string
	args     []byte
}

// Stream implements [ai.Client] over the streaming v2 chat API.
func (c *Client) Stream(
	ctx context.Context,
	req *ai.Request,
) iter.Seq2[ai.Chunk, error] {
	return func(yield func(ai.Chunk, error) bool) {
		cr, err := c.chatRequest(req, true)
		if err != nil {
			yield(ai.Chunk{}, err)
			return
		}
		resp, err := c.openStream(ctx, cr)
		if err != nil {
			yield(ai.Chunk{}, wrapUnsupportedCapability(req, err))
			return
		}
		defer resp.Body.Close()

		var usage ai.Usage
		var sawEnd bool
		tools := map[int]*toolAcc{}
		var order []int

		emit := func(acc *toolAcc, raw json.RawMessage) bool {
			args := acc.args
			if len(args) == 0 {
				args = []byte("{}")
			}
			if !json.Valid(args) {
				yield(ai.Chunk{}, fmt.Errorf(
					"cohere: tool call %q has invalid JSON arguments", acc.name))
				return false
			}
			call := ai.ToolUse{ID: acc.id, Name: acc.name, Input: json.RawMessage(args)}
			return yield(ai.Chunk{ToolCall: &call, Raw: raw}, nil)
		}

		for data, err := range ai.SSEEvents(resp.Body) {
			if err != nil {
				yield(ai.Chunk{}, err)
				return
			}
			var ev StreamEvent
			if e := json.Unmarshal([]byte(data), &ev); e != nil {
				// A broken event is a broken stream: skipping it would
				// quietly drop part of the answer.
				yield(ai.Chunk{}, e)
				return
			}
			switch ev.Type {
			case "content-delta":
				if t := ev.Delta.Message.Content.Text; t != "" {
					if !yield(ai.Chunk{Text: t, Raw: json.RawMessage(data)}, nil) {
						return
					}
				}
			case "tool-call-start":
				if _, ok := tools[ev.Index]; !ok {
					order = append(order, ev.Index)
				}
				tools[ev.Index] = &toolAcc{
					id:   ev.Delta.Message.ToolCalls.ID,
					name: ev.Delta.Message.ToolCalls.Function.Name,
				}
			case "tool-call-delta":
				if acc := tools[ev.Index]; acc != nil {
					acc.args = append(acc.args,
						ev.Delta.Message.ToolCalls.Function.Arguments...)
				}
			case "tool-call-end":
				if acc := tools[ev.Index]; acc != nil {
					delete(tools, ev.Index)
					if !emit(acc, json.RawMessage(data)) {
						return
					}
				}
			case "message-end":
				sawEnd = true
				usage = ai.Usage{
					InputTokens:  int(ev.Delta.Usage.Tokens.InputTokens),
					OutputTokens: int(ev.Delta.Usage.Tokens.OutputTokens),
				}
			}
		}

		// A missing message-end means the stream was truncated. The tool
		// calls it left open are not handed out: their arguments may be cut
		// off too, and a tool loop must not run an action the model never
		// finished. The error comes before anything a consumer could act on.
		if !sawEnd {
			yield(ai.Chunk{}, io.ErrUnexpectedEOF)
			return
		}

		// The message ended properly; flush any tool call it started but
		// did not close with a tool-call-end, in the order they started.
		for _, idx := range order {
			if acc := tools[idx]; acc != nil {
				if !emit(acc, nil) {
					return
				}
			}
		}

		final := usage
		yield(ai.Chunk{Done: true, Usage: &final}, nil)
	}
}

// ChatStream sends a native streaming v2 chat request and yields each event as
// it arrives.
func (c *Client) ChatStream(
	ctx context.Context,
	req *ChatRequest,
) iter.Seq2[StreamEvent, error] {
	return func(yield func(StreamEvent, error) bool) {
		resp, err := c.openStream(ctx, req)
		if err != nil {
			yield(StreamEvent{}, err)
			return
		}
		defer resp.Body.Close()

		for data, err := range ai.SSEEvents(resp.Body) {
			if err != nil {
				yield(StreamEvent{}, err)
				return
			}
			var ev StreamEvent
			if err := json.Unmarshal([]byte(data), &ev); err != nil {
				yield(StreamEvent{}, err)
				return
			}
			if !yield(ev, nil) || ev.Type == "message-end" {
				return
			}
		}
		yield(StreamEvent{}, io.ErrUnexpectedEOF)
	}
}

// openStream opens the streaming /v2/chat connection.
func (c *Client) openStream(ctx context.Context, req *ChatRequest) (*http.Response, error) {
	if req == nil {
		return nil, ai.ErrNoRequest
	}
	r := *req // do not mutate the caller's request
	r.Stream = true
	body, err := json.Marshal(&r)
	if err != nil {
		return nil, err
	}
	h := c.headers()
	h.Set("accept", "text/event-stream")
	resp, err := c.opts.DoStream(ctx, http.MethodPost, c.opts.BaseURL+"/v2/chat", body, h)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		data, readErr := ai.ReadResponse(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return nil, readErr
		}
		return nil, parseError(resp.StatusCode, data)
	}
	return resp, nil
}
