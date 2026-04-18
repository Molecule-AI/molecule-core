package embeddings

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// makeEmbedding constructs a fake 1536-dimensional embedding slice.
func makeEmbedding(dim int) []float32 {
	v := make([]float32, dim)
	for i := range v {
		v[i] = float32(i) * 0.001
	}
	return v
}

// mockOpenAIServer returns a test server that writes the given status and body.
func mockOpenAIServer(t *testing.T, statusCode int, respBody interface{}) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request looks like an embedding call.
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
			t.Errorf("missing Authorization header")
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected application/json Content-Type")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(respBody)
	}))
	return srv
}

func TestNewOpenAIEmbeddingFunc_MissingKey_ReturnsNil(t *testing.T) {
	// When OPENAI_API_KEY is absent, the constructor must return nil so
	// handlers.MemoriesHandler.WithEmbedding(nil) is a no-op and the
	// handler stays on the FTS/ILIKE path with zero log noise.
	t.Setenv("OPENAI_API_KEY", "")

	fn := NewOpenAIEmbeddingFunc()
	if fn != nil {
		t.Fatal("expected nil func when OPENAI_API_KEY is empty")
	}
}

func TestNewOpenAIEmbeddingFunc_Success(t *testing.T) {
	embedding := makeEmbedding(1536)
	respBody := map[string]interface{}{
		"data": []map[string]interface{}{
			{"embedding": embedding, "index": 0},
		},
	}

	srv := mockOpenAIServer(t, http.StatusOK, respBody)
	defer srv.Close()

	// Redirect the package-level URL to our mock server.
	old := openAIEmbedURL
	openAIEmbedURL = srv.URL
	defer func() { openAIEmbedURL = old }()

	t.Setenv("OPENAI_API_KEY", "sk-test-key")
	t.Setenv("OPENAI_EMBEDDING_MODEL", "text-embedding-3-small")

	fn := NewOpenAIEmbeddingFuncWithClient(srv.Client())
	vec, err := fn(context.Background(), "remember this fact")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vec) != 1536 {
		t.Errorf("expected 1536-dim vector, got %d", len(vec))
	}
	// Spot-check a value.
	if vec[1] != embedding[1] {
		t.Errorf("vec[1] mismatch: got %v, want %v", vec[1], embedding[1])
	}
}

func TestNewOpenAIEmbeddingFunc_DefaultModel(t *testing.T) {
	embedding := makeEmbedding(1536)
	var receivedModel string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req openAIEmbedRequest
		json.NewDecoder(r.Body).Decode(&req)
		receivedModel = req.Model
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{"embedding": embedding, "index": 0},
			},
		})
	}))
	defer srv.Close()

	old := openAIEmbedURL
	openAIEmbedURL = srv.URL
	defer func() { openAIEmbedURL = old }()

	t.Setenv("OPENAI_API_KEY", "sk-test-key")
	t.Setenv("OPENAI_EMBEDDING_MODEL", "")

	fn := NewOpenAIEmbeddingFuncWithClient(srv.Client())
	_, err := fn(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedModel != "text-embedding-3-small" {
		t.Errorf("expected default model text-embedding-3-small, got %s", receivedModel)
	}
}

func TestNewOpenAIEmbeddingFunc_HTTPError(t *testing.T) {
	srv := mockOpenAIServer(t, http.StatusInternalServerError, map[string]interface{}{
		"error": map[string]string{
			"message": "internal server error",
			"type":    "server_error",
		},
	})
	defer srv.Close()

	old := openAIEmbedURL
	openAIEmbedURL = srv.URL
	defer func() { openAIEmbedURL = old }()

	t.Setenv("OPENAI_API_KEY", "sk-test-key")

	fn := NewOpenAIEmbeddingFuncWithClient(srv.Client())
	_, err := fn(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error on HTTP 500")
	}
	if !strings.Contains(err.Error(), "server_error") {
		t.Errorf("error should contain error type, got: %v", err)
	}
}

func TestNewOpenAIEmbeddingFunc_APIError(t *testing.T) {
	// API returns 200 but with an error object (e.g. invalid model).
	srv := mockOpenAIServer(t, http.StatusOK, map[string]interface{}{
		"error": map[string]string{
			"message": "model does not exist",
			"type":    "invalid_request_error",
		},
	})
	defer srv.Close()

	old := openAIEmbedURL
	openAIEmbedURL = srv.URL
	defer func() { openAIEmbedURL = old }()

	t.Setenv("OPENAI_API_KEY", "sk-test-key")

	fn := NewOpenAIEmbeddingFuncWithClient(srv.Client())
	_, err := fn(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error when API returns error object")
	}
	if !strings.Contains(err.Error(), "model does not exist") {
		t.Errorf("error message not propagated, got: %v", err)
	}
}

func TestNewOpenAIEmbeddingFunc_WrongDimension(t *testing.T) {
	// Server returns a 512-dim vector — should fail the dimension check.
	embedding := makeEmbedding(512)
	srv := mockOpenAIServer(t, http.StatusOK, map[string]interface{}{
		"data": []map[string]interface{}{
			{"embedding": embedding, "index": 0},
		},
	})
	defer srv.Close()

	old := openAIEmbedURL
	openAIEmbedURL = srv.URL
	defer func() { openAIEmbedURL = old }()

	t.Setenv("OPENAI_API_KEY", "sk-test-key")

	fn := NewOpenAIEmbeddingFuncWithClient(srv.Client())
	_, err := fn(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error on dimension mismatch")
	}
	if !strings.Contains(err.Error(), "dimension mismatch") {
		t.Errorf("error should mention dimension mismatch, got: %v", err)
	}
}

func TestNewOpenAIEmbeddingFunc_EmptyEmbedding(t *testing.T) {
	srv := mockOpenAIServer(t, http.StatusOK, map[string]interface{}{
		"data": []map[string]interface{}{},
	})
	defer srv.Close()

	old := openAIEmbedURL
	openAIEmbedURL = srv.URL
	defer func() { openAIEmbedURL = old }()

	t.Setenv("OPENAI_API_KEY", "sk-test-key")

	fn := NewOpenAIEmbeddingFuncWithClient(srv.Client())
	_, err := fn(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error on empty embedding data")
	}
	if !strings.Contains(err.Error(), "empty embedding") {
		t.Errorf("error should mention empty embedding, got: %v", err)
	}
}

func TestNewOpenAIEmbeddingFunc_MalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not json {{{"))
	}))
	defer srv.Close()

	old := openAIEmbedURL
	openAIEmbedURL = srv.URL
	defer func() { openAIEmbedURL = old }()

	t.Setenv("OPENAI_API_KEY", "sk-test-key")

	fn := NewOpenAIEmbeddingFuncWithClient(srv.Client())
	_, err := fn(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error on malformed JSON")
	}
}

func TestNewOpenAIEmbeddingFunc_ContextCancellation(t *testing.T) {
	// Server that hangs until the request context is cancelled.
	done := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
		close(done)
	}))
	defer srv.Close()

	old := openAIEmbedURL
	openAIEmbedURL = srv.URL
	defer func() { openAIEmbedURL = old }()

	t.Setenv("OPENAI_API_KEY", "sk-test-key")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	fn := NewOpenAIEmbeddingFuncWithClient(srv.Client())
	_, err := fn(ctx, "test")
	if err == nil {
		t.Fatal("expected error when context is cancelled")
	}
}
