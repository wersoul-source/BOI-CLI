package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var boiLogoDOS = []string{
	"_______    _______   ______         ______   __       ______ ",
	"|       \\ /      \\ |      \\       /      \\ |  \\     |      \\",
	"| ▓▓▓▓▓▓▓\\  ▓▓▓▓▓▓\\\\▓▓▓▓▓▓      |  ▓▓▓▓▓▓\\ ▓▓      \\▓▓▓▓▓▓",
	"| ▓▓__/ ▓▓ ▓▓  | ▓▓ | ▓▓        | ▓▓   \\▓▓ ▓▓       | ▓▓  ",
	"| ▓▓    ▓▓ ▓▓  | ▓▓ | ▓▓        | ▓▓     | ▓▓       | ▓▓  ",
	"| ▓▓▓▓▓▓▓\\ ▓▓  | ▓▓ | ▓▓        | ▓▓   __| ▓▓       | ▓▓  ",
	"| ▓▓__/ ▓▓ ▓▓__/ ▓▓_| ▓▓_       | ▓▓__/  \\ ▓▓_____ _| ▓▓_ ",
	"| ▓▓    ▓▓\\▓▓    ▓▓   ▓▓ \\      \\▓▓    ▓▓ ▓▓     \\   ▓▓ \\",
	" \\▓▓▓▓▓▓▓  \\▓▓▓▓▓▓ \\▓▓▓▓▓▓       \\▓▓▓▓▓▓ \\▓▓▓▓▓▓▓▓\\▓▓▓▓▓▓",
}

var (
	splashLogoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A78BFA")).
			Bold(true)

	splashSubStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6C63FF"))

	splashVersionStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#4E44CE"))

	splashBorderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#6C63FF"))

	splashLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#A78BFA")).
				Bold(true)

	splashValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#E2E8F0"))

	splashDimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280"))

	splashPromptStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#6C63FF")).
				Bold(true)
)

type SplashModel struct {
	width         int
	height        int
	wd            string
	personaCount  int
	personaNames  string
	providerCount int
	memoryCount   int
	skillCount    int
	skillNames    string
}

func NewSplash(wd string, personaCount int, personaNames string, providerCount, memoryCount, skillCount int, skillNames string) *SplashModel {
	return &SplashModel{
		wd:            wd,
		personaCount:  personaCount,
		personaNames:  personaNames,
		providerCount: providerCount,
		memoryCount:   memoryCount,
		skillCount:    skillCount,
		skillNames:    skillNames,
	}
}

func (s *SplashModel) SetSize(w, h int) {
	s.width = w
	s.height = h
}

func (s *SplashModel) View() string {
	logoLines := boiLogoDOS

	logoWidth := 0
	for _, l := range logoLines {
		if len(l) > logoWidth {
			logoWidth = len(l)
		}
	}

	subtitle := "Chimera Architecture"
	version := "v0.1.0"

	contentWidth := logoWidth
	if len(subtitle) > contentWidth {
		contentWidth = len(subtitle)
	}
	if len(version) > contentWidth {
		contentWidth = len(version)
	}

	innerWidth := contentWidth + 6
	if innerWidth < 52 {
		innerWidth = 52
	}

	pad := func(s string, w int) string {
		return lipgloss.NewStyle().Width(w).Align(lipgloss.Center).Render(s)
	}

	var boxLines []string

	top := splashBorderStyle.Render("╔") + strings.Repeat("═", innerWidth) + splashBorderStyle.Render("╗")
	bot := splashBorderStyle.Render("╚") + strings.Repeat("═", innerWidth) + splashBorderStyle.Render("╝")
	edgeL := splashBorderStyle.Render("║")
	edgeR := splashBorderStyle.Render("║")
	empty := edgeL + strings.Repeat(" ", innerWidth) + edgeR

	boxLines = append(boxLines, top)
	boxLines = append(boxLines, empty)

	for _, line := range logoLines {
		styled := splashLogoStyle.Render(line)
		boxLines = append(boxLines, edgeL+pad(styled, innerWidth)+edgeR)
	}

	boxLines = append(boxLines, empty)
	boxLines = append(boxLines, edgeL+pad(splashSubStyle.Render(subtitle), innerWidth)+edgeR)
	boxLines = append(boxLines, edgeL+pad(splashVersionStyle.Render(version), innerWidth)+edgeR)
	boxLines = append(boxLines, empty)
	boxLines = append(boxLines, bot)

	var statusLines []string
	statusLines = append(statusLines, "")
	statusLines = append(statusLines, fmt.Sprintf("  %s %s", splashLabelStyle.Render("Workspace:"), splashValueStyle.Render(s.wd)))
	statusLines = append(statusLines, fmt.Sprintf("  %s %d loaded (%s)", splashLabelStyle.Render("Personas:"), s.personaCount, splashValueStyle.Render(s.personaNames)))

	if s.providerCount > 0 {
		provStatus := fmt.Sprintf("%d configured", s.providerCount)
		statusLines = append(statusLines, fmt.Sprintf("  %s %s", splashLabelStyle.Render("Providers:"), splashValueStyle.Render(provStatus)))
	} else {
		statusLines = append(statusLines, fmt.Sprintf("  %s %s  %s", splashLabelStyle.Render("Providers:"), splashDimStyle.Render("0 configured"), splashDimStyle.Render("[Set PSC_* in .env]")))
	}

	memStatus := "Phantom DB ready"
	if s.memoryCount > 0 {
		memStatus = fmt.Sprintf("Phantom DB (%d entries)", s.memoryCount)
	}
	statusLines = append(statusLines, fmt.Sprintf("  %s %s", splashLabelStyle.Render("Memory:"), splashValueStyle.Render(memStatus)))

	if s.skillCount > 0 {
		statusLines = append(statusLines, fmt.Sprintf("  %s %d loaded (%s)", splashLabelStyle.Render("Skills:"), s.skillCount, splashValueStyle.Render(s.skillNames)))
	} else {
		statusLines = append(statusLines, fmt.Sprintf("  %s %s", splashLabelStyle.Render("Skills:"), splashDimStyle.Render("none loaded")))
	}

	statusLines = append(statusLines, "")

	setupReady := s.providerCount > 0 && s.personaNames != ""
	if setupReady {
		statusLines = append(statusLines, fmt.Sprintf("  %s  %s", splashDimStyle.Render("Ready"), splashPromptStyle.Render("Press Enter to start...")))
	} else {
		statusLines = append(statusLines, fmt.Sprintf("  %s  %s", splashDimStyle.Render("Setup needed"), splashPromptStyle.Render("Press Enter to start anyway...")))
	}

	allLines := append(boxLines, statusLines...)
	content := strings.Join(allLines, "\n")

	if s.width > 0 && s.height > 0 {
		return lipgloss.Place(s.width, s.height,
			lipgloss.Center, lipgloss.Center,
			content,
		)
	}
	return content
}

func countPSCProviders() int {
	count := 0
	for i := 1; i <= 4; i++ {
		name := os.Getenv(fmt.Sprintf("PSC_%d_NAME", i))
		key := os.Getenv(fmt.Sprintf("PSC_%d_API_KEY", i))
		if name != "" && key != "" {
			count++
		}
	}
	return count
}

func countMemoryEntries(dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	count := 0
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			count++
		}
	}
	return count
}

func listSkills(dir string) ([]string, int) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, 0
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".skill.md") {
			name := strings.TrimSuffix(e.Name(), ".skill.md")
			names = append(names, name)
		}
	}
	return names, len(names)
}
