package core

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestIdentityRoundTrip(t *testing.T) {
	now := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	identity, err := NewIdentity("  แก้ว  ", now)
	if err != nil {
		t.Fatalf("NewIdentity: %v", err)
	}
	path := filepath.Join(t.TempDir(), IdentityFilename)
	if err := SaveIdentity(path, identity); err != nil {
		t.Fatalf("SaveIdentity: %v", err)
	}
	loaded, err := LoadIdentity(path)
	if err != nil {
		t.Fatalf("LoadIdentity: %v", err)
	}
	if loaded.Name != "แก้ว" || loaded.CorePersona != CorePersonaName || loaded.ID != identity.ID {
		t.Fatalf("loaded identity = %#v", loaded)
	}
}

func TestIdentityRejectsInvalidNamesAndPersona(t *testing.T) {
	if _, err := NewIdentity("\n", time.Now()); err == nil {
		t.Fatal("expected blank name rejection")
	}
	if _, err := NewIdentity(strings.Repeat("ก", 65), time.Now()); err == nil {
		t.Fatal("expected long name rejection")
	}
	identity, err := NewIdentity("Boi", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	identity.CorePersona = "legacy"
	if err := identity.Validate(); err == nil {
		t.Fatal("expected non-Core Persona rejection")
	}
}
