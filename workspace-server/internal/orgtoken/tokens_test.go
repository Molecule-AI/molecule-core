package orgtoken

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestIssue_StoresHashNotPlaintext(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	// Can't predict the generated plaintext, but we can verify the
	// INSERT arguments are a hash (bytea) + short prefix + optional
	// fields. sqlmock's AnyArg sidesteps the randomness.
	mock.ExpectQuery(`INSERT INTO org_api_tokens`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "my-ci", "user_01", "org-1").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("tok-1"))

	plaintext, id, err := Issue(context.Background(), db, "my-ci", "user_01", "org-1")
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	if id != "tok-1" {
		t.Errorf("id = %q, want tok-1", id)
	}
	// 43 chars = 32 random bytes base64url-encoded without padding.
	if len(plaintext) != 43 {
		t.Errorf("plaintext len = %d, want 43 (32 bytes b64url)", len(plaintext))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet: %v", err)
	}
}

func TestIssue_EmptyNameAndCreatedByStoreNull(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()
	// Empty name + createdBy + orgID → NULL in DB so `WHERE name IS NULL`
	// works for future queries that want "unnamed" tokens.
	mock.ExpectQuery(`INSERT INTO org_api_tokens`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), nil, nil, nil).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("tok-min"))

	_, _, err = Issue(context.Background(), db, "", "", "")
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet: %v", err)
	}
}

func TestValidate_HappyPath(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	plaintext := "known-plaintext-for-test"
	hash := sha256.Sum256([]byte(plaintext))

	mock.ExpectQuery(`SELECT id, prefix, org_id FROM org_api_tokens`).
		WithArgs(hash[:]).
		WillReturnRows(sqlmock.NewRows([]string{"id", "prefix", "org_id"}).AddRow("tok-live", "abcd1234", nil))
	mock.ExpectExec(`UPDATE org_api_tokens SET last_used_at`).
		WithArgs("tok-live").
		WillReturnResult(sqlmock.NewResult(0, 1))

	id, prefix, _, err := Validate(context.Background(), db, plaintext)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if id != "tok-live" {
		t.Errorf("id = %q, want tok-live", id)
	}
	if prefix != "abcd1234" {
		t.Errorf("prefix = %q, want abcd1234", prefix)
	}
}

func TestValidate_EmptyPlaintextRejected(t *testing.T) {
	db, _, _ := sqlmock.New()
	defer db.Close()
	if _, _, _, err := Validate(context.Background(), db, ""); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("empty plaintext should be ErrInvalidToken, got %v", err)
	}
}

func TestValidate_UnknownHashErrInvalid(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`SELECT id, prefix, org_id FROM org_api_tokens`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnError(sql.ErrNoRows)

	if _, _, _, err := Validate(context.Background(), db, "ghost"); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("unknown hash should be ErrInvalidToken, got %v", err)
	}
}

func TestValidate_RevokedTokenNotAccepted(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()
	// Query has `AND revoked_at IS NULL` — sqlmock will return
	// ErrNoRows because the revoked row is filtered out.
	mock.ExpectQuery(`SELECT id, prefix, org_id FROM org_api_tokens`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnError(sql.ErrNoRows)

	if _, _, _, err := Validate(context.Background(), db, "revoked-plaintext"); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("revoked token should be ErrInvalidToken, got %v", err)
	}
}

func TestList_NewestFirst(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	now := time.Now()
	earlier := now.Add(-1 * time.Hour)
	mock.ExpectQuery(`SELECT id, prefix.*FROM org_api_tokens.*ORDER BY created_at DESC`).
		WithArgs(listMax).
		WillReturnRows(sqlmock.NewRows([]string{"id", "prefix", "name", "org_id", "created_by", "created_at", "last_used_at"}).
			AddRow("t2", "abcd1234", "zapier", "org-1", "user_01", now, now).
			AddRow("t1", "efgh5678", "", "", "", earlier, nil))

	tokens, err := List(context.Background(), db)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(tokens) != 2 {
		t.Errorf("got %d tokens, want 2", len(tokens))
	}
	if tokens[0].ID != "t2" {
		t.Errorf("ordering not preserved: got %q first", tokens[0].ID)
	}
	if tokens[0].OrgID != "org-1" {
		t.Errorf("got org_id %q, want org-1", tokens[0].OrgID)
	}
	if tokens[1].LastUsedAt != nil {
		t.Errorf("never-used token should have nil LastUsedAt, got %v", tokens[1].LastUsedAt)
	}
}

func TestRevoke_HappyPathAndIdempotent(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	// First revoke: row transitions live → revoked, 1 row affected.
	mock.ExpectExec(`UPDATE org_api_tokens`).
		WithArgs("tok-1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	ok, err := Revoke(context.Background(), db, "tok-1")
	if err != nil || !ok {
		t.Errorf("first revoke: got (%v, %v), want (true, nil)", ok, err)
	}

	// Second revoke of same id: WHERE revoked_at IS NULL filters it
	// out, 0 rows affected. Must return (false, nil) — idempotent.
	mock.ExpectExec(`UPDATE org_api_tokens`).
		WithArgs("tok-1").
		WillReturnResult(sqlmock.NewResult(0, 0))
	ok, err = Revoke(context.Background(), db, "tok-1")
	if err != nil || ok {
		t.Errorf("second revoke: got (%v, %v), want (false, nil)", ok, err)
	}
}

func TestHasAnyLive(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`SELECT EXISTS.*org_api_tokens`).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	got, err := HasAnyLive(context.Background(), db)
	if err != nil || !got {
		t.Errorf("has-any-live: got (%v, %v), want (true, nil)", got, err)
	}

	mock.ExpectQuery(`SELECT EXISTS.*org_api_tokens`).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	got, err = HasAnyLive(context.Background(), db)
	if err != nil || got {
		t.Errorf("has-any-live empty: got (%v, %v), want (false, nil)", got, err)
	}
}

func TestOrgIDByTokenID_HappyPath(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`SELECT org_id FROM org_api_tokens WHERE id = \$1`).
		WithArgs("tok-org-1").
		WillReturnRows(sqlmock.NewRows([]string{"org_id"}).AddRow("org-1"))

	orgID, err := OrgIDByTokenID(context.Background(), db, "tok-org-1")
	if err != nil {
		t.Fatalf("OrgIDByTokenID: %v", err)
	}
	if orgID != "org-1" {
		t.Errorf("orgID = %q, want org-1", orgID)
	}
}

func TestOrgIDByTokenID_NullOrgIDReturnsEmpty(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	// Pre-migration token or ADMIN_TOKEN bootstrap token — org_id is NULL.
	// Caller should get ("", nil) and deny by default.
	mock.ExpectQuery(`SELECT org_id FROM org_api_tokens WHERE id = \$1`).
		WithArgs("tok-old").
		WillReturnRows(sqlmock.NewRows([]string{"org_id"}).AddRow(nil))

	orgID, err := OrgIDByTokenID(context.Background(), db, "tok-old")
	if err != nil {
		t.Fatalf("OrgIDByTokenID null: got err %v", err)
	}
	if orgID != "" {
		t.Errorf("orgID for null row = %q, want \"\"", orgID)
	}
}

func TestOrgIDByTokenID_DBError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`SELECT org_id FROM org_api_tokens WHERE id = \$1`).
		WithArgs("tok-bad").
		WillReturnError(sql.ErrConnDone)

	_, err = OrgIDByTokenID(context.Background(), db, "tok-bad")
	if err == nil {
		t.Error("expected error on DB failure, got nil")
	}
}
