[![deps.dev](https://img.shields.io/badge/deps.dev-insights-4c8dbc)](https://deps.dev/go/github.com%2Fgoloop%2Fcohere) [![License](https://img.shields.io/badge/license-MIT-brightgreen)](https://github.com/goloop/cohere/blob/master/LICENSE) [![License](https://img.shields.io/badge/godoc-YES-green)](https://pkg.go.dev/github.com/goloop/cohere) [![Stay with Ukraine](https://img.shields.io/static/v1?label=Stay%20with&message=Ukraine%20♥&color=ffD700&labelColor=0057B8&style=flat)](https://u24.gov.ua/)


# cohere

`cohere` is a Go client for the Cohere API. It implements the
`github.com/goloop/ai` interface, so it looks and works like every other goloop
AI provider, and exposes Cohere's native v2 endpoints on top.

## Features

- Chat: `Generate` for a single response, `Stream` for token-by-token output
  through `iter.Seq2`.
- Tool use (function calling), multimodal image input and system prompts.
- Native `ChatCompletion` and `ChatStream` over the v2 chat API.
- Embeddings, reranking and model listing.
- Retries on 429 and 5xx with backoff; normalized, typed API errors.
- Depends only on `github.com/goloop/ai` and the standard library.
- Structured output: `ai.Format` maps onto the provider's `response_format`
  (JSON mode or a schema); read the reply with `resp.JSON(&v)`.

## Installation

```sh
go get github.com/goloop/cohere
```

## Quick start

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/goloop/ai"
	"github.com/goloop/cohere"
)

func main() {
	c := cohere.New(os.Getenv("COHERE_API_KEY"))

	resp, err := c.Generate(context.Background(), &ai.Request{
		Model:    cohere.ModelCommandA,
		Messages: []ai.Message{ai.UserText("Say hello in one word.")},
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(resp.Text())
}
```

## Streaming

```go
for chunk, err := range c.Stream(ctx, req) {
	if err != nil {
		break
	}
	fmt.Print(chunk.Text)
}
```

## Embeddings

```go
docs, err := c.Embed(ctx, cohere.ModelEmbedV3, "hello", "world")
// for a search query, pick the input type explicitly:
q, err := c.Embeddings(ctx, &cohere.EmbedRequest{
	Model: cohere.ModelEmbedV3, Texts: []string{"hello"}, InputType: "search_query",
})
```

## Rerank

Rerank scores each document against the query and returns them ordered most
relevant first. Each result's `Index` points back into the request documents.

```go
res, err := c.Rerank(ctx, &cohere.RerankRequest{
	Model:     cohere.ModelRerankV35,
	Query:     "What is the capital of Ukraine?",
	Documents: []string{"Kyiv is the capital.", "Bananas are yellow."},
	TopN:      1,
})
// res[0].Index == 0, res[0].RelevanceScore near 1.0
```

## Documentation

Full reference: **[DOC.md](DOC.md)** (Ukrainian: **[DOC.UK.md](DOC.UK.md)**).

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

MIT - see [LICENSE](LICENSE).
