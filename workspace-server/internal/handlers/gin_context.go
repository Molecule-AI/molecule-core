package handlers

// gin_context.go — shared gin.Context helpers.
//
// These exist because a number of code paths read c.Request.Context() and
// pass it straight into DB / HTTP calls. When a handler is invoked with a
// gin.Context that has no Request (usually a bare gin.CreateTestContext
// in a unit test, rarely a misconfigured middleware chain in prod),
// c.Request is nil and the dereference panics. The 2026-04-23 incident
// (PR #1755) had one such panic kill the test binary mid-run and mask
// ~25 pre-existing failures downstream.

import (
	"context"

	"github.com/gin-gonic/gin"
)

// requestContext returns c.Request.Context() when Request is set, and
// context.Background() otherwise. Use this everywhere instead of a raw
// c.Request.Context() so a nil Request hardens into a benign empty
// context instead of a nil-deref panic.
//
// Intentionally forgiving: production Gin always sets Request, so the
// branch is dead code at runtime. The guard exists to keep the handler
// robust against test contexts and any future middleware that might
// reset the Request field — a safer default than crashing the process.
func requestContext(c *gin.Context) context.Context {
	if c == nil || c.Request == nil {
		return context.Background()
	}
	return c.Request.Context()
}
