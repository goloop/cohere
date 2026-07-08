package cohere

import (
	"encoding/json"

	"github.com/goloop/ai"
)

// parseError turns a non-success response body into an *ai.APIError. Cohere
// reports errors as {"message":"..."}.
func parseError(status int, body []byte) error {
	var w struct {
		Message string `json:"message"`
	}
	_ = json.Unmarshal(body, &w)

	return &ai.APIError{
		Status:  status,
		Message: w.Message,
		Raw:     append(json.RawMessage(nil), body...),
	}
}
