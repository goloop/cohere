package cohere

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/goloop/ai"
)

// ChatCompletion sends a native v2 chat request and returns the whole response.
// Use it for provider-specific options; use Generate for the shared,
// provider-agnostic path.
func (c *Client) ChatCompletion(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	out, _, err := c.chatCompletion(ctx, req)
	return out, err
}

func (c *Client) chatCompletion(ctx context.Context, req *ChatRequest) (*ChatResponse, []byte, error) {
	r := *req // do not mutate the caller's request
	r.Stream = false
	body, err := json.Marshal(&r)
	if err != nil {
		return nil, nil, err
	}
	data, status, err := c.send(ctx, http.MethodPost, "/v2/chat", body)
	if err != nil {
		return nil, nil, err
	}
	if status != http.StatusOK {
		return nil, data, parseError(status, data)
	}
	var out ChatResponse
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, data, err
	}
	return &out, data, nil
}

// Generate implements [ai.Client] over the v2 chat API.
func (c *Client) Generate(ctx context.Context, req *ai.Request) (*ai.Response, error) {
	cr, err := c.chatRequest(req, false)
	if err != nil {
		return nil, err
	}
	out, raw, err := c.chatCompletion(ctx, cr)
	if err != nil {
		return nil, err
	}
	resp := cohereToResponse(out, raw)
	resp.Model = req.Model
	resp.Format = formatMode(req.Format)
	return resp, nil
}

// chatRequest converts an ai.Request into a native ChatRequest.
func (c *Client) chatRequest(req *ai.Request, stream bool) (*ChatRequest, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if err := checkHosted(req); err != nil {
		return nil, err
	}

	cr := &ChatRequest{
		Model:         req.Model,
		Messages:      chatMessages(req),
		Temperature:   req.Temperature,
		P:             req.TopP,
		MaxTokens:     req.MaxTokens,
		StopSequences: req.Stop,
		Stream:        stream,
	}
	for _, t := range req.Tools {
		schema := t.Schema
		if len(schema) == 0 {
			schema = json.RawMessage(`{"type":"object"}`)
		}
		cr.Tools = append(cr.Tools, Tool{
			Type: "function",
			Function: ToolFunction{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  schema,
			},
		})
	}
	if len(cr.Tools) > 0 {
		switch req.ToolChoice {
		case ai.ToolRequired:
			cr.ToolChoice = "REQUIRED"
		case ai.ToolNone:
			cr.ToolChoice = "NONE"
		}
	}

	format, err := responseFormat(req.Format)
	if err != nil {
		return nil, err
	}
	cr.ResponseFormat = format
	return cr, nil
}

// responseFormat renders an [ai.Format] as this provider's response_format
// value, which carries the schema inside the same object rather than in a
// nested one. It returns nil when nothing was asked for.
func responseFormat(f *ai.Format) (json.RawMessage, error) {
	if f == nil || f.Type == ai.FormatText {
		return nil, nil
	}

	switch f.Type {
	case ai.FormatJSON:
		return json.RawMessage(`{"type":"json_object"}`), nil
	case ai.FormatJSONSchema:
		var body struct {
			Type   string          `json:"type"`
			Schema json.RawMessage `json:"json_schema"`
		}
		body.Type = "json_object"
		body.Schema = f.Schema
		return json.Marshal(body)
	default:
		return nil, ai.ErrBadFormat
	}
}

// formatMode reports how the request's format was satisfied. Both shapes this
// driver accepts are enforced by the provider itself.
func formatMode(f *ai.Format) ai.FormatMode {
	if f == nil || f.Type == ai.FormatText {
		return ai.FormatNone
	}
	return ai.FormatNative
}

func chatMessages(req *ai.Request) []ChatMessage {
	var out []ChatMessage
	if req.System != "" {
		out = append(out, ChatMessage{Role: "system", Content: req.System})
	}
	for _, m := range req.Messages {
		switch m.Role {
		case ai.RoleSystem:
			out = append(out, ChatMessage{Role: "system", Content: partsText(m.Parts)})
		case ai.RoleTool:
			for _, p := range m.Parts {
				if tr, ok := p.(ai.ToolResult); ok {
					out = append(out, ChatMessage{
						Role:       "tool",
						ToolCallID: tr.ID,
						Content:    tr.Content,
					})
				}
			}
		case ai.RoleAssistant:
			msg := ChatMessage{Role: "assistant"}
			var text strings.Builder
			for _, p := range m.Parts {
				switch v := p.(type) {
				case ai.Text:
					text.WriteString(v.Text)
				case ai.ToolUse:
					msg.ToolCalls = append(msg.ToolCalls, ToolCall{
						ID:   v.ID,
						Type: "function",
						Function: ToolCallFunction{
							Name:      v.Name,
							Arguments: string(v.Input),
						},
					})
				}
			}
			if text.Len() > 0 {
				msg.Content = text.String()
			}
			out = append(out, msg)
		default:
			out = append(out, ChatMessage{Role: "user", Content: userContent(m.Parts)})
		}
	}
	return out
}

// userContent returns a plain string when there are no images, or a slice of
// content blocks when there are.
func userContent(parts []ai.Part) any {
	hasImage := false
	for _, p := range parts {
		if _, ok := p.(ai.Image); ok {
			hasImage = true
			break
		}
	}
	if !hasImage {
		return partsText(parts)
	}

	var blocks []ContentBlock
	for _, p := range parts {
		switch v := p.(type) {
		case ai.Text:
			blocks = append(blocks, ContentBlock{Type: "text", Text: v.Text})
		case ai.Image:
			url := v.URL
			if len(v.Data) > 0 {
				url = "data:" + v.MIME + ";base64," +
					base64.StdEncoding.EncodeToString(v.Data)
			}
			blocks = append(blocks, ContentBlock{
				Type: "image_url", ImageURL: &ImageURL{URL: url},
			})
		}
	}
	return blocks
}

func partsText(parts []ai.Part) string {
	var b strings.Builder
	for _, p := range parts {
		if t, ok := p.(ai.Text); ok {
			b.WriteString(t.Text)
		}
	}
	return b.String()
}

func cohereToResponse(cr *ChatResponse, raw []byte) *ai.Response {
	resp := &ai.Response{
		StopReason: cr.FinishReason,
		Usage: ai.Usage{
			InputTokens:  int(cr.Usage.Tokens.InputTokens),
			OutputTokens: int(cr.Usage.Tokens.OutputTokens),
		},
		Raw: append(json.RawMessage(nil), raw...),
	}
	for _, b := range cr.Message.Content {
		if b.Type == "text" && b.Text != "" {
			resp.Parts = append(resp.Parts, ai.Text{Text: b.Text})
		}
	}
	for _, tc := range cr.Message.ToolCalls {
		resp.Parts = append(resp.Parts, ai.ToolUse{
			ID:    tc.ID,
			Name:  tc.Function.Name,
			Input: json.RawMessage(tc.Function.Arguments),
		})
	}
	return resp
}
