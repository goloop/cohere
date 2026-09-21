# cohere - reference

The full reference for the `cohere` package: the client, the shared `goloop/ai`
model, chat (interface and native), streaming, embeddings and models.

Ukrainian version: **[DOC.UK.md](DOC.UK.md)**.

## Contents

- [Mental model](#mental-model)
- [Creating a client](#creating-a-client)
- [Generate and Stream](#generate-and-stream)
- [Structured output](#structured-output)
- [Hosted web search](#hosted-web-search)
- [Capabilities and model-level refusals](#capabilities-and-model-level-refusals)
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
`io.ErrUnexpectedEOF` rather than silently reporting a completed response. A
tool call the stream left open is not handed out, since its arguments may be
cut off too; one that got its `tool-call-end` is. An event that is not valid
JSON, or tool arguments that are not, end the stream with an error as well.

## Structured output

`ai.Request.Format` maps onto the provider's own `response_format`, so a request for JSON
is enforced by the provider rather than merely asked for:

```go
resp, err := c.Generate(ctx, &ai.Request{
	Model:    "the-model",
	Messages: []ai.Message{ai.UserText("Draft SEO fields for this article.")},
	Format: &ai.Format{
		Type:   ai.FormatJSONSchema,
		Name:   "seo",
		Schema: schema,
	},
})

var seo SEO
err = resp.JSON(&seo)
```

`ai.FormatJSON` goes out as `{"type":"json_object"}`; `ai.FormatJSONSchema`
sends the schema inside that same object, as `json_schema`, which is the shape
this provider expects.


`ai.Response.Format` is `ai.FormatNative`: this provider enforces every shape
it accepts. Which models support schema mode is the provider's business - there
is no capability table here, so an unsupported pairing is reported by the
provider itself.

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

## Hosted web search

`ai.Request.Hosted` is answered with `ai.ErrNoHosted` before the request
leaves.

This provider's search is orchestration around tools you supply, which is what
`ai.Request.Tools` already expresses. Nothing runs on the provider's own side,
so mapping `Hosted` onto it would hand back a `ToolUse` you have to answer -
the one thing `Hosted` promises never to do.

The refusal is the documented behavior, not a gap left in silence: an answer
produced without the search that was asked for looks exactly like one produced
with it, so failing loudly is the only way you can tell them apart. If you would
rather have the answer anyway, ask again without `Hosted`.

## Capabilities and model-level refusals

This driver implements `ai.Capable` and reports no hosted capability - the
same answer `ai.Request.Hosted` already gets here, now available before the
call instead of only as an error after it. A conformance test pins the two
together.

A refusal the provider reports only as a 400 - "this model cannot produce that
format" - now arrives wrapped in `ai.ErrNoFormat`, so one `errors.Is` replaces
matching English prose in an error message; the provider's own `ai.APIError`
stays reachable with `errors.As`. The wrapping is deliberately narrow: only a
400, only a format the request actually asked for, only an error naming that
exact field.

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
