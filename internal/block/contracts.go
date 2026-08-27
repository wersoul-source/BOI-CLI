// Package block defines the stable architecture vocabulary shared by the six
// BOI Agent Suit blocks. It contains declarations only; behavior belongs to a
// concrete block and wiring belongs to the application composition root.
package block

import (
	"fmt"
	"strings"
)

// ID is the stable identity of one owner-defined BOI Agent Suit block.
type ID string

const (
	ServiceID     ID = "service"
	CoreID        ID = "core"
	EquipmentID   ID = "various-equipment"
	RuntimeID     ID = "runtime"
	AgentFolderID ID = "agent-folder"
	SubAgentID    ID = "subagent"
)

// Status distinguishes a package boundary from connected runtime behavior.
type Status string

const (
	StatusSkeleton Status = "skeleton"
	StatusPartial  Status = "partial"
	StatusActive   Status = "active"
	StatusDisabled Status = "disabled"
)

// Manifest records ownership without granting runtime authority.
type Manifest struct {
	ID         ID
	Name       string
	Purpose    string
	Owns       []string
	DoesNotOwn []string
	Status     Status
}

// Validate rejects incomplete or ambiguous block declarations.
func (m Manifest) Validate() error {
	if !KnownID(m.ID) {
		return fmt.Errorf("unknown BOI block ID %q", m.ID)
	}
	if strings.TrimSpace(m.Name) == "" {
		return fmt.Errorf("block %q has no name", m.ID)
	}
	if strings.TrimSpace(m.Purpose) == "" {
		return fmt.Errorf("block %q has no purpose", m.ID)
	}
	if len(m.Owns) == 0 {
		return fmt.Errorf("block %q has no ownership declaration", m.ID)
	}
	if len(m.DoesNotOwn) == 0 {
		return fmt.Errorf("block %q has no authority boundary", m.ID)
	}
	switch m.Status {
	case StatusSkeleton, StatusPartial, StatusActive, StatusDisabled:
		return nil
	default:
		return fmt.Errorf("block %q has invalid status %q", m.ID, m.Status)
	}
}

// KnownID reports whether an ID belongs to the owner-approved six blocks.
func KnownID(id ID) bool {
	switch id {
	case ServiceID, CoreID, EquipmentID, RuntimeID, AgentFolderID, SubAgentID:
		return true
	default:
		return false
	}
}
