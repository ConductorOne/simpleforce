package simpleforce

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newTestClient returns a logged-in client pointed at a test server that
// records the headers of every request it receives.
func newTestClient(t *testing.T) (*Client, *[]http.Header) {
	t.Helper()
	var received []http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = append(received, r.Header.Clone())
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/query"):
			_, _ = w.Write([]byte(`{"totalSize":1,"done":true,"records":[{"attributes":{"type":"User"},"Id":"005X"}]}`))
		case strings.Contains(r.URL.Path, "/sobjects/User/"):
			_, _ = w.Write([]byte(`{"attributes":{"type":"User"},"Id":"005X","Email":"a@example.com"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	client, err := NewClient(context.Background(), server.URL, DefaultClientID, DefaultAPIVersion)
	if err != nil {
		t.Fatal(err)
	}
	client.SetSidLoc("session", server.URL)
	return client, &received
}

// cacheControlValues returns the Cache-Control header of each received request.
func cacheControlValues(received []http.Header) []string {
	values := make([]string, 0, len(received))
	for _, h := range received {
		values = append(values, h.Get("Cache-Control"))
	}
	return values
}

func TestClient_QueryWithNoCache(t *testing.T) {
	ctx := context.Background()
	client, received := newTestClient(t)

	result, err := client.Query(ctx, "SELECT Id FROM User WHERE Email = 'nocache@example.com'", WithNoCache())
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Records) != 1 || result.Records[0].ID() != "005X" {
		t.Fatalf("unexpected records: %v", result.Records)
	}
	if got := cacheControlValues(*received); len(got) != 1 || got[0] != "no-cache" {
		t.Fatalf("expected one request with Cache-Control: no-cache, got %q", got)
	}
}

func TestClient_QueryWithoutOptions(t *testing.T) {
	ctx := context.Background()
	client, received := newTestClient(t)

	_, err := client.Query(ctx, "SELECT Id FROM User WHERE Email = 'default@example.com'")
	if err != nil {
		t.Fatal(err)
	}
	if got := cacheControlValues(*received); len(got) != 1 || got[0] != "" {
		t.Fatalf("expected one request without Cache-Control, got %q", got)
	}
}

func TestSObject_GetWithOptions(t *testing.T) {
	ctx := context.Background()
	client, received := newTestClient(t)

	obj, err := client.SObject("User").GetWithOptions(ctx, "005X", WithNoCache())
	if err != nil {
		t.Fatal(err)
	}
	if obj == nil || obj.StringField("Email") != "a@example.com" {
		t.Fatalf("unexpected object: %v", obj)
	}
	if got := cacheControlValues(*received); len(got) != 1 || got[0] != "no-cache" {
		t.Fatalf("expected one request with Cache-Control: no-cache, got %q", got)
	}
}

func TestSObject_GetUsesOwnID(t *testing.T) {
	ctx := context.Background()
	client, _ := newTestClient(t)

	obj, err := client.SObject("User").Set("Id", "005X").Get(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if obj == nil || obj.StringField("Email") != "a@example.com" {
		t.Fatalf("unexpected object: %v", obj)
	}
}

func TestClient_QueryWithHeader(t *testing.T) {
	ctx := context.Background()
	client, received := newTestClient(t)

	_, err := client.Query(ctx, "SELECT Id FROM User WHERE Email = 'header@example.com'",
		WithHeader("Sforce-Query-Options", "batchSize=200"))
	if err != nil {
		t.Fatal(err)
	}
	if len(*received) != 1 {
		t.Fatalf("expected one request, got %d", len(*received))
	}
	headers := (*received)[0]
	if got := headers.Get("Sforce-Query-Options"); got != "batchSize=200" {
		t.Fatalf("expected Sforce-Query-Options: batchSize=200, got %q", got)
	}
	if got := headers.Get("Authorization"); got != "Bearer session" {
		t.Fatalf("expected the default Authorization header to be kept, got %q", got)
	}
}
