package cohere

import "context"

// RerankRequest is the native v2 rerank request. Rerank scores each document by
// its relevance to the query and returns them ordered most relevant first.
type RerankRequest struct {
	Model     string   `json:"model"`
	Query     string   `json:"query"`
	Documents []string `json:"documents"`
	TopN      int      `json:"top_n,omitempty"`
}

// RerankResult is one scored document. Index is the document's position in the
// request's Documents slice; RelevanceScore is in the range 0..1.
type RerankResult struct {
	Index          int     `json:"index"`
	RelevanceScore float64 `json:"relevance_score"`
}

// Rerank scores the request's documents against its query and returns the
// results ordered most relevant first.
func (c *Client) Rerank(
	ctx context.Context,
	req *RerankRequest,
) ([]RerankResult, error) {
	var out struct {
		Results []RerankResult `json:"results"`
	}
	if err := c.postJSON(ctx, "/v2/rerank", req, &out); err != nil {
		return nil, err
	}
	return out.Results, nil
}
