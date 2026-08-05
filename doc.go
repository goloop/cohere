// Package cohere is a client for the Cohere API, built on the goloop/ai
// interface.
//
// The Client implements ai.Client, so Generate and Stream work the same as
// with any other goloop AI provider. On top of that it exposes Cohere's native
// v2 endpoints: chat (with tool use and image input), embeddings, reranking
// and model listing. Streaming uses Server-Sent Events with typed events; the
// driver hides that.
//
//	c := cohere.New(os.Getenv("COHERE_API_KEY"))
//	resp, err := c.Generate(ctx, &ai.Request{
//	    Model:    cohere.ModelCommandA,
//	    Messages: []ai.Message{ai.UserText("Say hello in one word.")},
//	})
//
// # Structured output
//
// ai.Request.Format maps onto the provider's response_format, so a request for
// JSON is enforced rather than merely asked for, and ai.Response.JSON decodes
// the reply. ai.FormatJSONSchema sends the schema inside that same object,
// which is the shape this provider expects.
//
// It depends only on goloop/ai and the standard library.
package cohere
