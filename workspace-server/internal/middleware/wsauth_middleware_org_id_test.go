package middleware

import (
	"crypto/sha256"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
)

// orgTokenValidateQueryV1 is matched for orgtoken.Validate().
// Validate() scans id, prefix, org_id (sql.NullString) — 3 columns,
// no ::text cast needed.
const orgTokenValidateQueryV1 = "SELECT id, prefix, org_id FROM org_api_tokens"

// orgTokenLastUsedQuery is matched for the best-effort last_used_at UPDATE
// inside orgtoken.Validate (called after the SELECT scan succeeds).
const orgTokenLastUsedQuery = "UPDATE org_api_tokens SET last_used_at"

// TestAdminAuth_OrgToken_SetsOrgIDContext verifies that AdminAuth's org-token
// tier reads the org_id column returned directly by Validate() and sets it
// in the gin context so that requireCallerOwnsOrg / orgCallerID can use it
// downstream. No secondary org_id lookup is needed — Validate() now returns
// org_id inline.
func TestAdminAuth_OrgToken_SetsOrgIDContext(t *testing.T) {
	tests := []struct {
		name          string
		orgIDFromDB   interface{} // sqlmock row value: nil, "", or "ws-org-1"
		wantOrgIDCtx  bool
		wantOrgIDVal  string
	}{
		{
			name:         "token has org_id — context key set",
			orgIDFromDB:  "ws-org-1",
			wantOrgIDCtx: true,
			wantOrgIDVal: "ws-org-1",
		},
		{
			name:         "pre-migration token (org_id=NULL) — no org_id context key",
			orgIDFromDB:  nil,
			wantOrgIDCtx: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mockDB, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("sqlmock.New: %v", err)
			}
			defer mockDB.Close()

			orgBearer := "valid-org-token"
			orgTokenHash := sha256.Sum256([]byte(orgBearer))

			// HasAnyLiveTokenGlobal: at least one workspace token exists globally.
			mock.ExpectQuery(hasAnyLiveTokenGlobalQuery).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

			// orgtoken.Validate: 3-column scan — id, prefix, org_id (sql.NullString).
			orgIDRow := sqlmock.NewRows([]string{"id", "prefix", "org_id"})
			if tt.orgIDFromDB == nil {
				orgIDRow = orgIDRow.AddRow("tok-org-1", "tok-org-1", nil)
			} else {
				orgIDRow = orgIDRow.AddRow("tok-org-1", "tok-org-1", tt.orgIDFromDB)
			}
			mock.ExpectQuery(orgTokenValidateQueryV1).
				WithArgs(orgTokenHash[:]).
				WillReturnRows(orgIDRow)

			// Best-effort last_used_at bump after Validate succeeds.
			mock.ExpectExec(orgTokenLastUsedQuery).
				WithArgs("tok-org-1").
				WillReturnResult(sqlmock.NewResult(0, 1))

			// orgtoken.Validate: 3-column scan — id, prefix, org_id (sql.NullString).
			orgIDRow := sqlmock.NewRows([]string{"id", "prefix", "org_id"})
			if tt.orgIDFromDB == nil {
				orgIDRow = orgIDRow.AddRow("tok-org-1", "tok-org-1", nil)
			} else {
				orgIDRow = orgIDRow.AddRow("tok-org-1", "tok-org-1", tt.orgIDFromDB)
			}
			mock.ExpectQuery(orgTokenValidateQueryV1).
				WithArgs(orgTokenHash[:]).
				WillReturnRows(orgIDRow)

			// Best-effort last_used_at bump after Validate succeeds.
			mock.ExpectExec(orgTokenLastUsedQuery).
				WithArgs("tok-org-1").
				WillReturnResult(sqlmock.NewResult(0, 1))

			r := gin.New()
			var gotOrgID string
			var haveOrgID bool
			r.GET("/admin/org/tokens", AdminAuth(mockDB), func(c *gin.Context) {
				if v, ok := c.Get("org_id"); ok {
					if s, ok := v.(string); ok {
						gotOrgID = s
						haveOrgID = true
					}
				}
				c.JSON(http.StatusOK, gin.H{"ok": true})
			})

			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/admin/org/tokens", nil)
			req.Header.Set("Authorization", "Bearer "+orgBearer)
			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
			}
			if haveOrgID != tt.wantOrgIDCtx {
				t.Errorf("c.Get(org_id) present = %v, want %v", haveOrgID, tt.wantOrgIDCtx)
			}
			if tt.wantOrgIDCtx && gotOrgID != tt.wantOrgIDVal {
				t.Errorf("org_id = %q, want %q", gotOrgID, tt.wantOrgIDVal)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet sqlmock expectations: %v", err)
			}
		})
	}
}

	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()

	orgToken := "tok_full_context_token"
	tokenHash := sha256.Sum256([]byte(orgToken))
	expectedOrgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"

	// Validate() 3-column scan with org_id set.
	mock.ExpectQuery(orgTokenValidateQueryV1).
		WithArgs(tokenHash[:]).
		WillReturnRows(sqlmock.NewRows([]string{"id", "prefix", "org_id"}).
			AddRow("tok-full", "tok_fu_", expectedOrgID))
		// Best-effort last_used_at bump.
		mock.ExpectExec(orgTokenLastUsedQuery).
			WithArgs("tok-full").
			WillReturnResult(sqlmock.NewResult(0, 1))

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