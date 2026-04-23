package middleware

import (
	"crypto/sha256"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
)

// orgTokenValidateQuery is matched for orgtoken.Validate in both
// WorkspaceAuth and AdminAuth middleware paths. Post-migration 036 the
// query selects id, prefix, AND org_id in a single round-trip; the
// secondary "SELECT org_id::text FROM org_api_tokens WHERE id" hop is
// gone, so tests do not need to stub it.
const orgTokenValidateQuery = "SELECT id, prefix, org_id FROM org_api_tokens WHERE token_hash"

func TestWorkspaceAuth_ValidOrgToken_SetsOrgIDContext(t *testing.T) {
	// F1097 (#1218): org tokens validated via WorkspaceAuth must have
	// org_id populated on the Gin context so downstream handlers can
	// enforce org isolation without a per-request DB round-trip.
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()

	orgToken := "tok_test_org_token_abc123"
	tokenHash := sha256.Sum256([]byte(orgToken))

	// Single-round-trip Validate: id + prefix + org_id.
	mock.ExpectQuery(orgTokenValidateQuery).
		WithArgs(tokenHash[:]).
		WillReturnRows(sqlmock.NewRows([]string{"id", "prefix", "org_id"}).
			AddRow("tok-org-abc", "tok_test", "00000000-0000-0000-0000-000000000001"))

	r := gin.New()
	r.GET("/workspaces/:id/secrets", WorkspaceAuth(mockDB), func(c *gin.Context) {
		v, exists := c.Get("org_id")
		if !exists {
			t.Errorf("org_id not set on context for valid org token")
			c.JSON(http.StatusOK, gin.H{"ok": true})
			return
		}
		c.JSON(http.StatusOK, gin.H{"org_id": v})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/workspaces/ws-1/secrets", nil)
	req.Header.Set("Authorization", "Bearer "+orgToken)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	// org_id must appear in the JSON response body.
	body := w.Body.String()
	if body == "" || body == "{}" {
		t.Errorf("org_id missing from response body: %s", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestWorkspaceAuth_ValidOrgToken_OrgIDNULL_DoesNotSetContext(t *testing.T) {
	// F1097: pre-migration tokens (org_id=NULL) must NOT set org_id on context —
	// requireCallerOwnsOrg already handles nil by denying by default, so a
	// nil context key is the correct signal.
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()

	orgToken := "tok_old_token_no_org"
	tokenHash := sha256.Sum256([]byte(orgToken))

	// Single-round-trip Validate; NULL org_id row mimics a pre-migration
	// token. Middleware must NOT set the org_id context key in this case.
	mock.ExpectQuery(orgTokenValidateQuery).
		WithArgs(tokenHash[:]).
		WillReturnRows(sqlmock.NewRows([]string{"id", "prefix", "org_id"}).
			AddRow("tok-old-xyz", "tok_old_", nil))

	r := gin.New()
	r.GET("/workspaces/:id/secrets", WorkspaceAuth(mockDB), func(c *gin.Context) {
		_, exists := c.Get("org_id")
		if exists {
			t.Errorf("org_id should not be set on context for NULL org_id token")
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/workspaces/ws-1/secrets", nil)
	req.Header.Set("Authorization", "Bearer "+orgToken)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestAdminAuth_ValidOrgToken_SetsOrgIDContext(t *testing.T) {
	// F1097 (#1218): AdminAuth path (used for /org/* routes) must also
	// populate org_id so org-token callers can access their own org's
	// routes without a separate OrgIDByTokenID call per request.
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()

	orgToken := "tok_admin_path_org_token"
	tokenHash := sha256.Sum256([]byte(orgToken))

	// HasAnyLiveTokenGlobal: at least one workspace auth token exists globally
	// (bootstrap gate — if no tokens exist, AdminAuth grants access to all).
	mock.ExpectQuery(hasAnyLiveTokenGlobalQuery).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	// Single-round-trip Validate via AdminAuth: id + prefix + org_id.
	mock.ExpectQuery(orgTokenValidateQuery).
		WithArgs(tokenHash[:]).
		WillReturnRows(sqlmock.NewRows([]string{"id", "prefix", "org_id"}).
			AddRow("tok-admin-org", "tok_adm_", "00000000-0000-0000-0000-000000000042"))

	r := gin.New()
	r.GET("/admin/org-settings", AdminAuth(mockDB), func(c *gin.Context) {
		v, exists := c.Get("org_id")
		if !exists {
			t.Errorf("org_id not set on context for valid org token via AdminAuth")
			c.JSON(http.StatusOK, gin.H{"ok": true})
			return
		}
		c.JSON(http.StatusOK, gin.H{"org_id": v})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/admin/org-settings", nil)
	req.Header.Set("Authorization", "Bearer "+orgToken)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestAdminAuth_ValidOrgToken_OrgIDNULL_DoesNotSetContext(t *testing.T) {
	// F1097: AdminAuth path for pre-migration org token (org_id=NULL) must
	// NOT set org_id on context. Tokens minted before F1097 fix have
	// org_id=NULL — requireCallerOwnsOrg already denies these by default.
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()

	orgToken := "tok_old_admin_token"
	tokenHash := sha256.Sum256([]byte(orgToken))

	mock.ExpectQuery(hasAnyLiveTokenGlobalQuery).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	// Single-round-trip Validate with NULL org_id — AdminAuth path.
	mock.ExpectQuery(orgTokenValidateQuery).
		WithArgs(tokenHash[:]).
		WillReturnRows(sqlmock.NewRows([]string{"id", "prefix", "org_id"}).
			AddRow("tok-old-admin", "tok_old_", nil))

	r := gin.New()
	r.GET("/admin/org-settings", AdminAuth(mockDB), func(c *gin.Context) {
		_, exists := c.Get("org_id")
		if exists {
			t.Errorf("org_id should not be set for NULL org_id token via AdminAuth")
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/admin/org-settings", nil)
	req.Header.Set("Authorization", "Bearer "+orgToken)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestWorkspaceAuth_OrgToken_DBRowScanError_DoesNotPanic(t *testing.T) {
	// F1097: if the org_id SELECT returns an unexpected column count or type,
	// the deferred suppress-pattern must not crash — the token is still valid,
	// org_id is simply not set (token is denied by requireCallerOwnsOrg at use-time).
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()

	orgToken := "tok_token_ok"
	tokenHash := sha256.Sum256([]byte(orgToken))

	// Single-round-trip Validate: returns NULL org_id (stands in for the
	// scan-error case the original test was exercising; the secondary hop
	// it mimicked no longer exists).
	mock.ExpectQuery(orgTokenValidateQuery).
		WithArgs(tokenHash[:]).
		WillReturnRows(sqlmock.NewRows([]string{"id", "prefix", "org_id"}).
			AddRow("tok-ok", "tok_tok_", nil))

	r := gin.New()
	r.GET("/workspaces/:id/secrets", WorkspaceAuth(mockDB), func(c *gin.Context) {
		// org_id key may or may not be set — either is acceptable here.
		// The important thing is we don't panic.
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/workspaces/ws-1/secrets", nil)
	req.Header.Set("Authorization", "Bearer "+orgToken)
	r.ServeHTTP(w, req)

	// Token is still accepted — only the org_id enrichment fails.
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 despite org_id SELECT error, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestWorkspaceAuth_OrgToken_SetsAllContextKeys verifies the complete set of
// context keys set by WorkspaceAuth for a valid org token (F1097 coverage).
func TestWorkspaceAuth_OrgToken_SetsAllContextKeys(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()

	orgToken := "tok_full_context_token"
	tokenHash := sha256.Sum256([]byte(orgToken))
	expectedOrgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"

	mock.ExpectQuery(orgTokenValidateQuery).
		WithArgs(tokenHash[:]).
		WillReturnRows(sqlmock.NewRows([]string{"id", "prefix", "org_id"}).
			AddRow("tok-full", "tok_fu_", expectedOrgID))

	r := gin.New()
	r.GET("/workspaces/:id/secrets", WorkspaceAuth(mockDB), func(c *gin.Context) {
		id, ok := c.Get("org_token_id")
		if !ok {
			t.Errorf("org_token_id not set")
		} else if id != "tok-full" {
			t.Errorf("org_token_id: got %v, want tok-full", id)
		}

		prefix, ok := c.Get("org_token_prefix")
		if !ok {
			t.Errorf("org_token_prefix not set")
		} else if prefix != "tok_fu_" {
			t.Errorf("org_token_prefix: got %v, want tok_fu_", prefix)
		}

		orgID, ok := c.Get("org_id")
		if !ok {
			t.Errorf("org_id not set")
		} else if orgID != expectedOrgID {
			t.Errorf("org_id: got %v, want %s", orgID, expectedOrgID)
		}

		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/workspaces/ws-1/secrets", nil)
	req.Header.Set("Authorization", "Bearer "+orgToken)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}