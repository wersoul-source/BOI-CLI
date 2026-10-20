package capability

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

const IndexSchemaVersion = 1
const MaxActiveSkills = 15
const MaxActiveTools = 15

var ErrAlreadyRegistered = errors.New("capability is already registered")

type Kind string

const (
	KindSkill Kind = "skill"
	KindTool  Kind = "tool"
)

type Index struct {
	SchemaVersion int     `json:"schema_version"`
	Kind          Kind    `json:"kind"`
	Entries       []Entry `json:"entries"`
}

type Entry struct {
	Name        string   `json:"name"`
	Source      string   `json:"source"`
	Summary     string   `json:"summary"`
	Enabled     bool     `json:"enabled"`
	Priority    int      `json:"priority,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Requires    []string `json:"requires,omitempty"`
	ContextCost int      `json:"context_cost,omitempty"`
}

type SelectionInput struct {
	Task                  string
	Installed             map[string]bool
	AvailableRequirements map[string]bool
	ProviderAllows        bool
	ContextBudget         int
}

type Decision struct {
	Name      string `json:"name"`
	Installed bool   `json:"installed"`
	Eligible  bool   `json:"eligible"`
	Selected  bool   `json:"selected"`
	Active    bool   `json:"active"`
	Disabled  bool   `json:"disabled"`
	Rejected  bool   `json:"rejected"`
	Reason    string `json:"reason"`
	Score     int    `json:"score"`
}

type Selection struct {
	Kind      Kind       `json:"kind"`
	Decisions []Decision `json:"decisions"`
	Active    []string   `json:"active"`
}

func (index Index) Validate() error {
	if index.SchemaVersion != IndexSchemaVersion {
		return fmt.Errorf("unsupported %s index schema %d", index.Kind, index.SchemaVersion)
	}
	if index.Kind != KindSkill && index.Kind != KindTool {
		return fmt.Errorf("invalid capability kind %q", index.Kind)
	}
	seen := map[string]bool{}
	for _, entry := range index.Entries {
		entry.Name = strings.TrimSpace(entry.Name)
		if entry.Name == "" || entry.Source == "" || entry.Summary == "" {
			return fmt.Errorf("capability name, source, and summary are required")
		}
		if seen[entry.Name] {
			return fmt.Errorf("duplicate %s entry %q", index.Kind, entry.Name)
		}
		if entry.ContextCost < 0 {
			return fmt.Errorf("negative context cost for %q", entry.Name)
		}
		seen[entry.Name] = true
	}
	return nil
}

func LoadIndex(path string, want Kind) (*Index, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s index: %w", want, err)
	}
	var index Index
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("parse %s index: %w", want, err)
	}
	if index.Kind != want {
		return nil, fmt.Errorf("index kind %q does not match %q", index.Kind, want)
	}
	if err := index.Validate(); err != nil {
		return nil, err
	}
	return &index, nil
}

func SaveIndex(path string, index *Index) error {
	if index == nil {
		return fmt.Errorf("capability index is required")
	}
	if err := index.Validate(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal capability index: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create registry directory: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("write capability index: %w", err)
	}
	return nil
}

func IndexPath(boiDir string, kind Kind) string {
	return filepath.Join(boiDir, "registry", string(kind)+"s.json")
}

func AddEntry(path string, kind Kind, entry Entry) error {
	index, err := LoadIndex(path, kind)
	if err != nil {
		return err
	}
	for _, existing := range index.Entries {
		if existing.Name == entry.Name {
			return fmt.Errorf("%w: %s %q", ErrAlreadyRegistered, kind, entry.Name)
		}
	}
	index.Entries = append(index.Entries, entry)
	sort.Slice(index.Entries, func(i, j int) bool { return index.Entries[i].Name < index.Entries[j].Name })
	return SaveIndex(path, index)
}

func Select(index Index, input SelectionInput) Selection {
	limit := MaxActiveTools
	if index.Kind == KindSkill {
		limit = MaxActiveSkills
	}
	taskWords := words(input.Task)
	type candidate struct {
		entry    Entry
		decision Decision
	}
	var candidates []candidate
	result := Selection{Kind: index.Kind}
	for _, entry := range index.Entries {
		d := Decision{Name: entry.Name, Installed: input.Installed[entry.Name]}
		switch {
		case !entry.Enabled:
			d.Disabled, d.Reason = true, "disabled by index policy"
		case !d.Installed:
			d.Rejected, d.Reason = true, "indexed source is not installed"
		case !input.ProviderAllows:
			d.Rejected, d.Reason = true, "Provider profile does not allow capability kind"
		case missingRequirement(entry.Requires, input.AvailableRequirements) != "":
			d.Rejected, d.Reason = true, "missing requirement: "+missingRequirement(entry.Requires, input.AvailableRequirements)
		default:
			relevance := matchScore(taskWords, entry)
			if index.Kind == KindSkill && len(taskWords) > 0 && relevance == 0 {
				d.Rejected, d.Reason = true, "not relevant to task"
			} else {
				d.Eligible = true
				d.Score = entry.Priority*100 + relevance
				candidates = append(candidates, candidate{entry, d})
			}
		}
		if !d.Eligible {
			result.Decisions = append(result.Decisions, d)
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].decision.Score != candidates[j].decision.Score {
			return candidates[i].decision.Score > candidates[j].decision.Score
		}
		return candidates[i].entry.Name < candidates[j].entry.Name
	})
	remaining := input.ContextBudget
	for position, item := range candidates {
		d := item.decision
		d.Selected = true
		if position >= limit {
			d.Rejected, d.Reason = true, "active limit reached"
		} else if item.entry.ContextCost > remaining && input.ContextBudget > 0 {
			d.Rejected, d.Reason = true, "Context budget exceeded"
		} else {
			d.Active, d.Reason = true, "selected deterministically"
			remaining -= item.entry.ContextCost
			result.Active = append(result.Active, item.entry.Name)
		}
		result.Decisions = append(result.Decisions, d)
	}
	sort.Slice(result.Decisions, func(i, j int) bool { return result.Decisions[i].Name < result.Decisions[j].Name })
	return result
}

func words(text string) map[string]bool {
	result := map[string]bool{}
	for _, word := range strings.FieldsFunc(strings.ToLower(text), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsNumber(r) }) {
		result[word] = true
	}
	return result
}
func matchScore(task map[string]bool, entry Entry) int {
	score := 0
	values := strings.Join(entry.Tags, " ") + " " + entry.Name + " " + entry.Summary
	for value := range words(values) {
		if task[value] {
			score++
		}
	}
	return score
}
func missingRequirement(requirements []string, available map[string]bool) string {
	for _, requirement := range requirements {
		if !available[requirement] {
			return requirement
		}
	}
	return ""
}
