package cohere

import "context"

// EmbedRequest is the native v2 embed request.
type EmbedRequest struct {
	Model          string   `json:"model"`
	Texts          []string `json:"texts"`
	InputType      string   `json:"input_type"`
	EmbeddingTypes []string `json:"embedding_types,omitempty"`
}

// Embeddings sends a native v2 embed request and returns the float vectors.
func (c *Client) Embeddings(
	ctx context.Context,
	req *EmbedRequest,
) ([][]float64, error) {
	if len(req.EmbeddingTypes) == 0 {
		req.EmbeddingTypes = []string{"float"}
	}
	var out struct {
		Embeddings struct {
			Float [][]float64 `json:"float"`
		} `json:"embeddings"`
	}
	if err := c.postJSON(ctx, "/v2/embed", req, &out); err != nil {
		return nil, err
	}
	return out.Embeddings.Float, nil
}

// Embed embeds one or more texts as documents and returns their vectors in
// order. For queries, use Embeddings with InputType "search_query".
func (c *Client) Embed(
	ctx context.Context,
	model string,
	texts ...string,
) ([][]float64, error) {
	return c.Embeddings(ctx, &EmbedRequest{
		Model:     model,
		Texts:     texts,
		InputType: "search_document",
	})
}
