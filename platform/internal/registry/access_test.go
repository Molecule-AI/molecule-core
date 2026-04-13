package registry

import (
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/db"
)

func setupMockDB(t *testing.T) sqlmock.Sqlmock {
	t.Helper()
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	db.DB = mockDB
	t.Cleanup(func() { mockDB.Close() })
	return mock
}

func ptr(s string) *string { return &s }

func expectLookup(mock sqlmock.Sqlmock, id string, parentID *string) {
	row := mock.NewRows([]string{"id", "parent_id"})
	if parentID != nil {
		row.AddRow(id, *parentID)
	} else {
		row.AddRow(id, nil)
	}
	mock.ExpectQuery("SELECT id, parent_id FROM workspaces WHERE id").
		WithArgs(id).
		WillReturnRows(row)
}

func expectNotFound(mock sqlmock.Sqlmock, id string) {
	mock.ExpectQuery("SELECT id, parent_id FROM workspaces WHERE id").
		WithArgs(id).
		WillReturnError(sql.ErrNoRows)
}

// ---------- Tests ----------

func TestCanCommunicate_SameWorkspace(t *testing.T) {
	mock := setupMockDB(t)
	expectLookup(mock, "ws-1", nil)
	expectLookup(mock, "ws-1", nil)

	if !CanCommunicate("ws-1", "ws-1") {
		t.Error("same workspace should always communicate")
	}
}

func TestCanCommunicate_Siblings(t *testing.T) {
	mock := setupMockDB(t)
	// ws-a and ws-b are siblings (same parent ws-parent)
	expectLookup(mock, "ws-a", ptr("ws-parent"))
	expectLookup(mock, "ws-b", ptr("ws-parent"))

	if !CanCommunicate("ws-a", "ws-b") {
		t.Error("siblings should communicate")
	}
}

func TestCanCommunicate_RootSiblings(t *testing.T) {
	mock := setupMockDB(t)
	// Both at root level (no parent)
	expectLookup(mock, "ws-a", nil)
	expectLookup(mock, "ws-b", nil)

	if !CanCommunicate("ws-a", "ws-b") {
		t.Error("root-level siblings should communicate")
	}
}

func TestCanCommunicate_ParentToChild(t *testing.T) {
	mock := setupMockDB(t)
	// ws-parent talks to ws-child (whose parent is ws-parent)
	expectLookup(mock, "ws-parent", nil)
	expectLookup(mock, "ws-child", ptr("ws-parent"))

	if !CanCommunicate("ws-parent", "ws-child") {
		t.Error("parent should communicate with child")
	}
}

func TestCanCommunicate_ChildToParent(t *testing.T) {
	mock := setupMockDB(t)
	// ws-child talks up to ws-parent
	expectLookup(mock, "ws-child", ptr("ws-parent"))
	expectLookup(mock, "ws-parent", nil)

	if !CanCommunicate("ws-child", "ws-parent") {
		t.Error("child should communicate with parent")
	}
}

func TestCanCommunicate_Denied_DifferentParents(t *testing.T) {
	mock := setupMockDB(t)
	// ws-a (parent: p1) and ws-b (parent: p2) — not siblings
	expectLookup(mock, "ws-a", ptr("p1"))
	expectLookup(mock, "ws-b", ptr("p2"))

	if CanCommunicate("ws-a", "ws-b") {
		t.Error("workspaces with different parents should NOT communicate")
	}
}

func TestCanCommunicate_Denied_CousinToRoot(t *testing.T) {
	mock := setupMockDB(t)
	// ws-child (parent: ws-mid) and ws-root (no parent, NOT ws-mid)
	expectLookup(mock, "ws-child", ptr("ws-mid"))
	expectLookup(mock, "ws-root", nil)

	if CanCommunicate("ws-child", "ws-root") {
		t.Error("child should NOT communicate with unrelated root workspace")
	}
}

func TestCanCommunicate_Denied_CallerNotFound(t *testing.T) {
	mock := setupMockDB(t)
	expectNotFound(mock, "ws-missing")

	if CanCommunicate("ws-missing", "ws-target") {
		t.Error("nonexistent caller should be denied")
	}
}

func TestCanCommunicate_Denied_TargetNotFound(t *testing.T) {
	mock := setupMockDB(t)
	expectLookup(mock, "ws-caller", nil)
	expectNotFound(mock, "ws-missing")

	if CanCommunicate("ws-caller", "ws-missing") {
		t.Error("nonexistent target should be denied")
	}
}

func TestCanCommunicate_Denied_Grandchild(t *testing.T) {
	mock := setupMockDB(t)
	// ws-grandparent and ws-grandchild (parent: ws-mid, NOT ws-grandparent)
	expectLookup(mock, "ws-grandparent", nil)
	expectLookup(mock, "ws-grandchild", ptr("ws-mid"))

	if CanCommunicate("ws-grandparent", "ws-grandchild") {
		t.Error("grandparent should NOT communicate with grandchild directly")
	}
}
