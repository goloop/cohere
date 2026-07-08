package cohere

import "encoding/json"

// ContentBlock is one piece of a message's content. For text, Type is "text";
// for images, Type is "image_url".
type ContentBlock struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
}

// ImageURL holds an image reference, typically a base64 data URI.
type ImageURL struct {
	URL string `json:"url"`
}

// ToolCall is a tool call produced by the model.
type ToolCall struct {
	ID       string           `json:"id,omitempty"`
	Type     string           `json:"type,omitempty"`
	Function ToolCallFunction `json:"function"`
}

// ToolCallFunction is the function name and JSON-encoded arguments of a call.
type ToolCallFunction struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

// Tool declares a callable function.
type Tool struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

// ToolFunction is a function's name, description and JSON Schema parameters.
type ToolFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
}

// ChatMessage is one message in a chat request. Content is a string or a slice
// of content blocks (for images).
type ChatMessage struct {
	Role       string     `json:"role"`
	Content    any        `json:"content,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	ToolPlan   string     `json:"tool_plan,omitempty"`
}

// ChatRequest is the native v2 chat request body.
type ChatRequest struct {
	Model          string          `json:"model"`
	Messages       []ChatMessage   `json:"messages"`
	Tools          []Tool          `json:"tools,omitempty"`
	ToolChoice     string          `json:"tool_choice,omitempty"`
	Temperature    *float64        `json:"temperature,omitempty"`
	P              *float64        `json:"p,omitempty"`
	MaxTokens      int             `json:"max_tokens,omitempty"`
	StopSequences  []string        `json:"stop_sequences,omitempty"`
	ResponseFormat json.RawMessage `json:"response_format,omitempty"`
	Stream         bool            `json:"stream,omitempty"`
}

// Usage reports token counts for a chat request.
type Usage struct {
	Tokens struct {
		InputTokens  float64 `json:"input_tokens"`
		OutputTokens float64 `json:"output_tokens"`
	} `json:"tokens"`
}

// ResponseMessage is the assistant message in a chat response.
type ResponseMessage struct {
	Role      string         `json:"role"`
	Content   []ContentBlock `json:"content"`
	ToolCalls []ToolCall     `json:"tool_calls"`
	ToolPlan  string         `json:"tool_plan"`
}

// ChatResponse is the native v2 chat response.
type ChatResponse struct {
	ID           string          `json:"id"`
	FinishReason string          `json:"finish_reason"`
	Message      ResponseMessage `json:"message"`
	Usage        Usage           `json:"usage"`
}
