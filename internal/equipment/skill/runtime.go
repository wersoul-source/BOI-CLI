package skill

import (
	"fmt"
	"strings"
)

// ApplyContext injects a skill into the system prompt context
func ApplyContext(s *Skill, context map[string]string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("## Skill: %s\n", s.Name))
	if s.Description != "" {
		sb.WriteString(fmt.Sprintf("> %s\n", s.Description))
	}
	sb.WriteString("\n")

	prompt := s.Prompt
	for key, val := range context {
		prompt = strings.ReplaceAll(prompt, "{{"+key+"}}", val)
	}
	sb.WriteString(prompt)
	sb.WriteString("\n")

	if len(s.Requires) > 0 {
		sb.WriteString("\n**Required capabilities:** ")
		sb.WriteString(strings.Join(s.Requires, ", "))
		sb.WriteString("\n")
	}

	return sb.String()
}

// MatchRequirements checks if available tools satisfy skill requirements
func MatchRequirements(s *Skill, available []string) []string {
	var missing []string
	for _, req := range s.Requires {
		found := false
		for _, avail := range available {
			if req == avail {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, req)
		}
	}
	return missing
}

// InjectSkills adds all skill contexts to a system prompt
func InjectSkills(skills []*Skill, context map[string]string) string {
	if len(skills) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("# Available Skills\n\n")
	sb.WriteString("The following skills are available. Use them when relevant:\n\n")

	for _, s := range skills {
		sb.WriteString(fmt.Sprintf("### %s\n", s.Name))
		sb.WriteString(fmt.Sprintf("%s\n\n", s.Description))
	}

	return sb.String()
}
