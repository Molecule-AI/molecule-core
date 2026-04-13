package wsauth

import (
	"context"
	"database/sql"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func setupMock(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db, mock
}

// ------------------------------------------------------------
// IssueToken
// ------------------------------------------------------------

func TestIssueToken_PersistsHashNotPlaintext(t *testing.T) {
	db, mock := setupMock(t)

	mock.ExpectExec(`INSERT INTO workspace_auth_tokens`).
		WithArgs(
			"ws-abc",
			sqlmock.AnyArg(), // hash (bytea)
			sqlmock.AnyArg(), // prefix
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	tok, err := IssueToken(context.Background(), db, "ws-abc")
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}
	if len(tok) < 40 {
		t.Errorf("token looks too short to be 256-bit: len=%d", len(tok))
	}
	// Standard base64url-no-padding alphabet only.
	if !regexp.MustCompile(`^[A-Za-z0-9_-]+$`).MatchString(tok) {
		t.Errorf("token contains non-urlsafe chars: %q", tok)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestIssueToken_ReturnsDifferentTokensEachCall(t *testing.T) {
	db, mock := setupMock(t)
	mock.ExpectExec(`INSERT INTO workspace_auth_tokens`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO workspace_auth_tokens`).WillReturnResult(sqlmock.NewResult(1, 1))

	a, _ := IssueToken(context.Background(), db, "ws-1")
	b, _ := IssueToken(context.Background(), db, "ws-1")
	if a == b {
		t.Errorf("expected unique tokens across calls, got %q twice", a)
	}
}

// ------------------------------------------------------------
// ValidateToken
// ------------------------------------------------------------

func TestValidateToken_HappyPath(t *testing.T) {
	db, mock := setupMock(t)

	// First insert a token we can validate.
	mock.ExpectExec(`INSERT INTO workspace_auth_tokens`).WillReturnResult(sqlmock.NewResult(1, 1))
	tok, err := IssueToken(context.Background(), db, "ws-xyz")
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}

	// Validate: lookup by hash returns matching workspace.
	mock.ExpectQuery(`SELECT id, workspace_id FROM workspace_auth_tokens`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "workspace_id"}).AddRow("tok-id-1", "ws-xyz"))
	// Best-effort last_used_at update.
	mock.ExpectExec(`UPDATE workspace_auth_tokens SET last_used_at`).
		WithArgs("tok-id-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := ValidateToken(context.Background(), db, "ws-xyz", tok); err != nil {
		t.Errorf("expected valid token, got error: %v", err)
	}
}

func TestValidateToken_WrongWorkspaceRejected(t *testing.T) {
	db, mock := setupMock(t)

	// Token belongs to ws-owner; caller claims to be ws-attacker.
	mock.ExpectQuery(`SELECT id, workspace_id FROM workspace_auth_tokens`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "workspace_id"}).AddRow("tok-id-2", "ws-owner"))

	err := ValidateToken(context.Background(), db, "ws-attacker", "some-token")
	if err != ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken, got %v", err)
	}
}

func TestValidateToken_RejectsEmptyInputs(t *testing.T) {
	db, _ := setupMock(t)
	if err := ValidateToken(context.Background(), db, "", "x"); err != ErrInvalidToken {
		t.Errorf("empty workspace id: got %v, want ErrInvalidToken", err)
	}
	if err := ValidateToken(context.Background(), db, "ws-x", ""); err != ErrInvalidToken {
		t.Errorf("empty token: got %v, want ErrInvalidToken", err)
	}
}

func TestValidateToken_UnknownTokenRejected(t *testing.T) {
	db, mock := setupMock(t)
	mock.ExpectQuery(`SELECT id, workspace_id FROM workspace_auth_tokens`).
		WillReturnError(sql.ErrNoRows)

	if err := ValidateToken(context.Background(), db, "ws-a", "not-a-real-token"); err != ErrInvalidToken {
		t.Errorf("got %v, want ErrInvalidToken", err)
	}
}

// ------------------------------------------------------------
// HasAnyLiveToken
// ------------------------------------------------------------

func TestHasAnyLiveToken(t *testing.T) {
	cases := []struct {
		name  string
		count int
		want  bool
	}{
		{"no tokens", 0, false},
		{"one token", 1, true},
		{"many tokens", 7, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock := setupMock(t)
			mock.ExpectQuery(`SELECT COUNT\(\*\) FROM workspace_auth_tokens`).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(tc.count))

			got, err := HasAnyLiveToken(context.Background(), db, "ws-x")
			if err != nil {
				t.Fatalf("err: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

// ------------------------------------------------------------
// RevokeAllForWorkspace
// ------------------------------------------------------------

func TestRevokeAllForWorkspace(t *testing.T) {
	db, mock := setupMock(t)
	mock.ExpectExec(`UPDATE workspace_auth_tokens\s+SET revoked_at`).
		WithArgs("ws-delete-me").
		WillReturnResult(sqlmock.NewResult(0, 3))

	if err := RevokeAllForWorkspace(context.Background(), db, "ws-delete-me"); err != nil {
		t.Fatalf("err: %v", err)
	}
}

// ------------------------------------------------------------
// BearerTokenFromHeader
// ------------------------------------------------------------

func TestBearerTokenFromHeader(t *testing.T) {
	cases := map[string]string{
		"":                          "",
		"xyz":                       "", // no Bearer prefix
		"bearer lowercase-no-match": "", // case-sensitive
		"Bearer ":                   "",
		"Bearer abc123":             "abc123",
		"Bearer   spaced  ":         "spaced", // TrimSpace
		"Bearer token-with-dashes":  "token-with-dashes",
	}
	for in, want := range cases {
		got := BearerTokenFromHeader(in)
		if got != want {
			t.Errorf("BearerTokenFromHeader(%q) = %q, want %q", in, got, want)
		}
	}
}
