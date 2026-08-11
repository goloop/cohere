package cohere

import (
	"fmt"

	"github.com/goloop/ai"
)

// checkHosted refuses a request that asks the provider to run a capability on
// its own side.
//
// This provider's search is orchestration around tools the caller supplies,
// which is what ai.Request.Tools already expresses. There is no capability the
// provider runs on its own side, so mapping Hosted onto it would mean handing
// back a ToolUse the caller has to answer, and that is the one thing Hosted
// promises never to do.
//
// Returning [ai.ErrNoHosted] before the request leaves is the documented
// behavior, not a placeholder: an answer produced without the search that was
// asked for looks exactly like one produced with it, so failing loudly is the
// only way a caller can tell the difference.
func checkHosted(req *ai.Request) error {
	if len(req.Hosted) == 0 {
		return nil
	}
	return fmt.Errorf("%w: this provider runs no capability on its own side", ai.ErrNoHosted)
}
