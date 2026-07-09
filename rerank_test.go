package cohere

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestRerank(t *testing.T) {
	c, done := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/rerank" {
			t.Errorf("path = %s", r.URL.Path)
		}
		var req RerankRequest
		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatal(err)
		}
		if req.Query != "capital" || len(req.Documents) != 3 || req.TopN != 2 {
			t.Errorf("req = %+v", req)
		}
		io.WriteString(w, `{"results":[{"index":2,"relevance_score":0.98},`+
			`{"index":0,"relevance_score":0.12}]}`)
	})
	defer done()

	res, err := c.Rerank(context.Background(), &RerankRequest{
		Model:     ModelRerankV35,
		Query:     "capital",
		Documents: []string{"a", "b", "c"},
		TopN:      2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 2 || res[0].Index != 2 || res[0].RelevanceScore != 0.98 {
		t.Errorf("results = %+v", res)
	}
}

func TestRerankError(t *testing.T) {
	c, done := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		io.WriteString(w, `{"message":"no documents"}`)
	})
	defer done()

	_, err := c.Rerank(context.Background(), &RerankRequest{Model: "m", Query: "x"})
	if err == nil {
		t.Fatal("want error")
	}
}
