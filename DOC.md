# cohere - reference

The full reference for the `cohere` package: the client, the shared `goloop/ai`
model, chat (interface and native), streaming, embeddings and models.

Ukrainian version: **[DOC.UK.md](DOC.UK.md)**.

## Contents

- [Mental model](#mental-model)
- [Creating a client](#creating-a-client)
- [Generate and Stream](#generate-and-stream)
- [Native chat](#native-chat)
- [Tools, images and system prompts](#tools-images-and-system-prompts)
- [Embeddings](#embeddings)
- [Models](#models)
- [Options and errors](#options-and-errors)

## Mental model

`cohere.Client` implements `ai.Client`, the provider-agnostic contract from
`github.com/goloop/ai`. The shared `Generate` and `Stream` cover the common
ground - chat with tools, images and streaming - so code written against the
interface runs on any provider.

Native power lives in the v2 chat API (`ChatCompletion`/`ChatStream`),
embeddings and model listing.

```go
import (
	"github.com/goloop/ai"
	"github.com/goloop/cohere"
)
```

## Creating a client

```go
c := cohere.New(os.Getenv("COHERE_API_KEY"))

c = cohere.New(apiKey, cohere.WithTimeout(30*time.Second))
```

The base URL defaults to `https://api.cohere.com`.

## Generate and Stream

```go
resp, err := c.Generate(ctx, &ai.Request{
	Model:    cohere.ModelCommandA,
	System:   "You are concise.",
	Messages: []ai.Message{ai.UserText("Name three primary colors.")},
})
resp.Text()
resp.ToolCalls()
resp.Usage
```

`Stream` returns `iter.Seq2[ai.Chunk, error]`: text deltas as chunks with
`Text`, a finished tool call as a chunk with `ToolCall`, and a final chunk with
`Done` and `Usage`.

```go
for chunk, err := range c.Stream(ctx, req) {
	if err != nil {
		return err
	}
	fmt.Print(chunk.Text)
}
```

If the stream is cut off before its `message-end` event, `Stream` yields
`io.ErrUnexpectedEOF` (after salvaging any partial tool call) rather than
silently reporting a completed response.

## Native chat

For provider-only options build a `ChatRequest` and call `ChatCompletion` or
`ChatStream`. Cohere streams typed Server-Sent Events (`content-delta`,
`tool-call-start`, `message-end`, ...); the native `ChatStream` yields them as
`StreamEvent` values.

```go
resp, err := c.ChatCompletion(ctx, &cohere.ChatRequest{
	Model:          cohere.ModelCommandA,
	Messages:       []cohere.ChatMessage{{Role: "user", Content: "as JSON"}},
	ResponseFormat: json.RawMessage(`{"type":"json_object"}`),
})
```

The `P` field maps to Cohere's nucleus sampling; `MaxTokens` and
`StopSequences` map to `max_tokens` and `stop_sequences`.

## Tools, images and system prompts

Tool use, images and system prompts use the shared `ai` types: `ai.Tool`,
`ai.Image`, `ai.ToolResult` and a `RoleSystem` message or the `System` field.
Tool results are sent back as `RoleTool` messages whose `ai.ToolResult.ID`
matches the `ai.ToolUse.ID`. Inline image bytes are sent as a base64 data URI
(use a vision-capable model).

## Embeddings

```go
docs, err := c.Embed(ctx, cohere.ModelEmbedV3, "hello", "world") // documents
q, err := c.Embeddings(ctx, &cohere.EmbedRequest{
	Model: cohere.ModelEmbedV3, Texts: []string{"hello"}, InputType: "search_query",
})
```

## Models

```go
models, err := c.Models(ctx)
m, err := c.GetModel(ctx, cohere.ModelCommandA)
m.Endpoints // e.g. ["chat"]
```

## Options and errors

Options: `WithBaseURL`, `WithHTTPClient`, `WithTimeout`, `WithMaxRetries`,
`WithHeader`.

A non-success response becomes an `*ai.APIError` with `Status`, `Message` and
the raw body:

```go
var apiErr *ai.APIError
if errors.As(err, &apiErr) && apiErr.Status == http.StatusTooManyRequests {
	// back off
}
```

Requests missing a model or messages fail before the network with
`ai.ErrNoModel` or `ai.ErrNoMessages`.
