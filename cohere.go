package cohere

import "github.com/goloop/ai"

// DefaultBaseURL is the Cohere API base URL.
const DefaultBaseURL = "https://api.cohere.com"

// Convenience model identifiers. Any model string is accepted; use Models to
// discover what the account can call.
const (
	ModelCommandA     = "command-a-03-2025"
	ModelCommandRPlus = "command-r-plus-08-2024"
	ModelCommandR     = "command-r-08-2024"
	ModelEmbedV3      = "embed-english-v3.0"
)

// Client is a Cohere API client. It implements [ai.Client] and adds the
// provider's native endpoints (the v2 chat, embed and models APIs).
type Client struct {
	opts ai.Options
}

var _ ai.Client = (*Client)(nil)

// New returns a Client for the given API key. Shared options (WithBaseURL,
// WithHTTPClient, WithTimeout, WithMaxRetries, WithHeader) configure it.
func New(apiKey string, opts ...Option) *Client {
	s := settings{}
	for _, o := range opts {
		o(&s)
	}

	o := ai.NewOptions(apiKey, s.aiOpts...)
	if o.BaseURL == "" {
		o.BaseURL = DefaultBaseURL
	}

	return &Client{opts: o}
}
