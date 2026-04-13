package registry

import (
	"database/sql"
	"log"

	"github.com/Molecule-AI/molecule-monorepo/platform/internal/db"
)

type workspaceRef struct {
	ID       string
	ParentID *string
}

func getWorkspaceRef(id string) (*workspaceRef, error) {
	var ws workspaceRef
	var parentID sql.NullString
	err := db.DB.QueryRow(`SELECT id, parent_id FROM workspaces WHERE id = $1`, id).
		Scan(&ws.ID, &parentID)
	if err != nil {
		return nil, err
	}
	if parentID.Valid {
		ws.ParentID = &parentID.String
	}
	return &ws, nil
}

// CanCommunicate checks if two workspaces can talk to each other
// based on the hierarchy rules: siblings, parent-child, root-level siblings.
func CanCommunicate(callerID, targetID string) bool {
	if callerID == targetID {
		return true
	}

	caller, err := getWorkspaceRef(callerID)
	if err != nil {
		log.Printf("CanCommunicate: lookup caller %s: %v", callerID, err)
		return false
	}
	target, err := getWorkspaceRef(targetID)
	if err != nil {
		log.Printf("CanCommunicate: lookup target %s: %v", targetID, err)
		return false
	}

	// Siblings — same parent (including root-level where both have no parent)
	if caller.ParentID != nil && target.ParentID != nil &&
		*caller.ParentID == *target.ParentID {
		return true
	}
	// Root-level siblings — both have no parent
	if caller.ParentID == nil && target.ParentID == nil {
		return true
	}

	// Parent talking to child
	if target.ParentID != nil && caller.ID == *target.ParentID {
		return true
	}

	// Child talking up to parent
	if caller.ParentID != nil && target.ID == *caller.ParentID {
		return true
	}

	return false
}
