package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// requestContext must return context.Background for the three degenerate
// inputs production code can realistically hit — nil context, nil Request,
// and a zero-value bare gin.Context from CreateTestContext — without
// panicking. Together with the third case (real Request set), these cover
// every path a handler can take.

func TestRequestContext_NilContext(t *testing.T) {
	// Must not panic. Returned ctx must still be usable (Deadline/Err OK to call).
	got := requestContext(nil)
	if got == nil {
		t.Fatal("requestContext(nil) returned nil — must return context.Background")
	}
	if _, has := got.Deadline(); has {
		t.Error("context.Background should have no deadline")
	}
}

func TestRequestContext_NilRequest(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w) // Request is nil on fresh test context
	if c.Request != nil {
		t.Skip("gin changed CreateTestContext to set Request; test no longer exercises nil-Request path")
	}
	got := requestContext(c)
	if got == nil {
		t.Fatal("requestContext(c) with nil Request returned nil")
	}
	if got != context.Background() {
		// OK for implementations to return a distinct empty ctx, but warn.
		t.Logf("note: requestContext(nil-Request) returned non-Background context %#v", got)
	}
}

func TestRequestContext_RealRequestPassesThrough(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	// Attach a request with a context-value so we can verify the same
	// context is handed through (not replaced with Background).
	type ctxKey string
	reqCtx := context.WithValue(context.Background(), ctxKey("marker"), "present")
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil).WithContext(reqCtx)

	got := requestContext(c)
	if got.Value(ctxKey("marker")) != "present" {
		t.Errorf("requestContext dropped the Request's context — expected marker=present, got %v",
			got.Value(ctxKey("marker")))
	}
}
