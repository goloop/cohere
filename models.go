package cohere

import "context"

// Model describes a model reported by Cohere.
type Model struct {
	Name             string   `json:"name"`
	Endpoints        []string `json:"endpoints"`
	ContextLength    float64  `json:"context_length,omitempty"`
	TokenizerURL     string   `json:"tokenizer_url,omitempty"`
	Finetuned        bool     `json:"finetuned,omitempty"`
	SupportsVision   bool     `json:"supports_vision,omitempty"`
	DefaultEndpoints []string `json:"default_endpoints,omitempty"`
}

// Models lists the models available to the account.
func (c *Client) Models(ctx context.Context) ([]Model, error) {
	var out struct {
		Models []Model `json:"models"`
	}
	if err := c.getJSON(ctx, "/v1/models", &out); err != nil {
		return nil, err
	}
	return out.Models, nil
}

// GetModel retrieves one model by name.
func (c *Client) GetModel(ctx context.Context, name string) (*Model, error) {
	var m Model
	if err := c.getJSON(ctx, "/v1/models/"+name, &m); err != nil {
		return nil, err
	}
	return &m, nil
}
