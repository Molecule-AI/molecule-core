// Package embeddings provides handlers.EmbeddingFunc implementations for
// semantic search.  All providers are constructed lazily — a nil func is
// returned when the required credentials are absent, causing the handler to
// degrade gracefully to its existing FTS/ILIKE path.
package embeddings

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/Molecule-AI/molecule-monorepo/platform/internal/handlers"
)

// openAIEmbedHTTPTimeout bounds each embedding call.  10 s is generous for a
// single-document embed; the handler's own request context will cancel sooner
// if the upstream request is abandoned.
const openAIEmbedHTTPTimeout = 10 * time.Second

// openAIEmbedResponseCap limits the bytes we read from the OpenAI response.
// A 1536-float32 embedding encodes to ~24 KiB JSON; 64 KiB leaves headroom for
// metadata without admitting unbounded reads.
const openAIEmbedResponseCap = 64 * 1024

// openAIEmbedURL is a var so tests can redirect to a local mock server.
var openAIEmbedURL = "https://api.openai.com/v1/embeddings"

// openAIEmbedHTTPClient is the default client for production.  Tests inject
// their own via NewOpenAIEmbeddingFuncWithClient.
var openAIEmbedHTTPClient = &http.Client{Timeout: openAIEmbedHTTPTimeout}

// openAIEmbedRequest is the JSON body sent to the embeddings endpoint.
type openAIEmbedRequest struct {
	Input          string `json:"input"`
	Model          string `json:"model"`
	EncodingFormat string `json:"encoding_format"` // "float"
}

// openAIEmbedResponse is the successful JSON response from OpenAI.
type openAIEmbedResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

// NewOpenAIEmbeddingFunc returns a handlers.EmbeddingFunc backed by the OpenAI
// embeddings API, or nil when the required credentials are absent.
//
// Environment variables read at construction time:
//
//	OPENAI_API_KEY         – required; returns nil when absent so
//	                         handlers.MemoriesHandler.WithEmbedding(nil) is a
//	                         no-op and the handler falls back to FTS with zero
//	                         log noise (issue #576 chunk 1).
//	OPENAI_EMBEDDING_MODEL – optional; defaults to "text-embedding-3-small"
//	                         which produces 1536-dim vectors matching the
//	                         platform/migrations/031_memories_pgvector.up.sql
//	                         column definition.
//
// The returned func is safe for concurrent use.  The API key and model are
// captured at construction time — a platform restart is required to pick up
// secret rotations, consistent with how all other platform credentials work.
func NewOpenAIEmbeddingFunc() handlers.EmbeddingFunc {
	return newOpenAIEmbeddingFuncWithClient(openAIEmbedHTTPClient)
}

// NewOpenAIEmbeddingFuncWithClient is the same as NewOpenAIEmbeddingFunc but
// uses the supplied *http.Client.  Intended for tests that need a mock server.
func NewOpenAIEmbeddingFuncWithClient(client *http.Client) handlers.EmbeddingFunc {
	return newOpenAIEmbeddingFuncWithClient(client)
}

func newOpenAIEmbeddingFuncWithClient(client *http.Client) handlers.EmbeddingFunc {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		// No key configured — return nil so WithEmbedding(nil) is a no-op.
		// The MemoriesHandler will use its FTS/ILIKE path with no log noise.
		return nil
	}

	model := os.Getenv("OPENAI_EMBEDDING_MODEL")
	if model == "" {
		model = "text-embedding-3-small"
	}

	return func(ctx context.Context, text string) ([]float32, error) {

		payload := openAIEmbedRequest{
			Input:          text,
			Model:          model,
			EncodingFormat: "float",
		}
		body, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("marshal embed request: %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, openAIEmbedURL,
			bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("build embed request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("POST embeddings: %w", err)
		}
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, openAIEmbedResponseCap))
		resp.Body.Close()

		var parsed openAIEmbedResponse
		if err := json.Unmarshal(respBody, &parsed); err != nil {
			return nil, fmt.Errorf("parse embed response (status=%d): %w", resp.StatusCode, err)
		}
		if parsed.Error != nil {
			return nil, fmt.Errorf("OpenAI error %s: %s", parsed.Error.Type, parsed.Error.Message)
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("OpenAI HTTP %d: %s", resp.StatusCode, string(respBody))
		}
		if len(parsed.Data) == 0 || len(parsed.Data[0].Embedding) == 0 {
			return nil, fmt.Errorf("OpenAI returned empty embedding")
		}

		vec := parsed.Data[0].Embedding

		// Migration 031 creates a vector(1536) column.  Reject unexpected
		// dimensions to surface misconfigured model choices early rather than
		// letting pgvector reject the INSERT at query time.
		const expectedDim = 1536
		if len(vec) != expectedDim {
			return nil, fmt.Errorf("OpenAI embedding dimension mismatch: got %d, want %d (check OPENAI_EMBEDDING_MODEL)",
				len(vec), expectedDim)
		}

		return vec, nil
	}
}
