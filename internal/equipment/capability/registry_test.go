package capability

import (
	"fmt"
	"path/filepath"
	"testing"
)

func testIndex(kind Kind, count int) Index {
	index := Index{SchemaVersion: IndexSchemaVersion, Kind: kind}
	for i := 1; i <= count; i++ {
		name := fmt.Sprintf("cap-%02d", i)
		index.Entries = append(index.Entries, Entry{Name: name, Source: name, Summary: "test capability", Enabled: true, Priority: count - i})
	}
	return index
}

func TestSixteenthCapabilityIsRejectedDeterministically(t *testing.T) {
	for _, kind := range []Kind{KindSkill, KindTool} {
		index := testIndex(kind, 16)
		installed := map[string]bool{}
		for _, entry := range index.Entries {
			installed[entry.Name] = true
		}
		first := Select(index, SelectionInput{Installed: installed, ProviderAllows: true})
		second := Select(index, SelectionInput{Installed: installed, ProviderAllows: true})
		if len(first.Active) != 15 || fmt.Sprint(first) != fmt.Sprint(second) {
			t.Fatalf("%s selection is not stable: %#v", kind, first)
		}
		for _, decision := range first.Decisions {
			if decision.Name == "cap-16" && (!decision.Rejected || decision.Reason != "active limit reached") {
				t.Fatalf("sixteenth %s decision = %#v", kind, decision)
			}
		}
	}
}

func TestUnindexedInstalledCapabilityIsNeverSelected(t *testing.T) {
	index := testIndex(KindSkill, 1)
	selection := Select(index, SelectionInput{Installed: map[string]bool{"cap-01": true, "loose-file": true}, ProviderAllows: true})
	if len(selection.Active) != 1 || selection.Active[0] != "cap-01" {
		t.Fatalf("unindexed capability leaked: %#v", selection)
	}
}

func TestSelectionStatesAndContextBudget(t *testing.T) {
	index := Index{SchemaVersion: 1, Kind: KindSkill, Entries: []Entry{
		{Name: "active", Source: "a", Summary: "git helper", Enabled: true, ContextCost: 5},
		{Name: "disabled", Source: "b", Summary: "off", Enabled: false},
		{Name: "missing", Source: "c", Summary: "missing", Enabled: true},
		{Name: "expensive", Source: "d", Summary: "large", Enabled: true, ContextCost: 20},
	}}
	selection := Select(index, SelectionInput{Task: "git", Installed: map[string]bool{"active": true, "disabled": true, "expensive": true}, ProviderAllows: true, ContextBudget: 10})
	if len(selection.Active) != 1 || selection.Active[0] != "active" {
		t.Fatalf("unexpected active set: %#v", selection)
	}
}

func TestIndexRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "skills.json")
	index := testIndex(KindSkill, 2)
	if err := SaveIndex(path, &index); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadIndex(path, KindSkill)
	if err != nil || len(loaded.Entries) != 2 {
		t.Fatalf("round trip: %#v %v", loaded, err)
	}
}
