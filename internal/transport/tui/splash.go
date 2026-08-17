package tui

import (
	"fmt"
	"os"
	"strings"

	llmfactory "github.com/boi-family/boi-cli/internal/provider/factory"
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
	version       string
	personaCount  int
	personaNames  string
	providerCount int
	memoryCount   int
	skillCount    int
	skillNames    string
}

func NewSplash(wd string, personaCount int, personaNames string, providerCount, memoryCount, skillCount int, skillNames string, version string) *SplashModel {
	return &SplashModel{
		wd:            wd,
		version:       version,
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

	contentWidth := logoWidth
	if len(subtitle) > contentWidth {
		contentWidth = len(subtitle)
	}
	if len(s.version) > contentWidth {
		contentWidth = len(s.version)
	}

	innerWidth := contentWidth + 4
	if innerWidth < 50 {
		innerWidth = 50
	}
	// Cap frame at half terminal width for compact look
	if s.width > 0 && innerWidth > s.width/2 {
		innerWidth = s.width / 2
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
	boxLines = append(boxLines, edgeL+pad(splashVersionStyle.Render(s.version), innerWidth)+edgeR)
	boxLines = append(boxLines, empty)
	boxLines = append(boxLines, bot)

	var statusLines []string
	statusLines = append(statusLines, "")
	statusLines = append(statusLines, fmt.Sprintf("  %s %s", splashLabelStyle.Render("◈"), splashValueStyle.Render(s.wd)))
	statusLines = append(statusLines, fmt.Sprintf("  %s %d loaded  %s", splashLabelStyle.Render("⚡"), s.personaCount, splashValueStyle.Render(s.personaNames)))

	if s.providerCount > 0 {
		provStatus := fmt.Sprintf("%d configured", s.providerCount)
		statusLines = append(statusLines, fmt.Sprintf("  %s %s", splashLabelStyle.Render("⛭"), splashValueStyle.Render(provStatus)))
	} else {
		statusLines = append(statusLines, fmt.Sprintf("  %s %s  %s", splashLabelStyle.Render("⛭"), splashDimStyle.Render("0 configured"), splashDimStyle.Render("[Set PSC_* in .env]")))
	}

	memStatus := "Phantom DB ready"
	if s.memoryCount > 0 {
		memStatus = fmt.Sprintf("Phantom DB (%d entries)", s.memoryCount)
	}
	statusLines = append(statusLines, fmt.Sprintf("  %s %s", splashLabelStyle.Render("⟡"), splashValueStyle.Render(memStatus)))

	if s.skillCount > 0 {
		statusLines = append(statusLines, fmt.Sprintf("  %s %d loaded  %s", splashLabelStyle.Render("◎"), s.skillCount, splashValueStyle.Render(s.skillNames)))
	} else {
		statusLines = append(statusLines, fmt.Sprintf("  %s %s", splashLabelStyle.Render("◎"), splashDimStyle.Render("none loaded")))
	}

	statusLines = append(statusLines, "")

	setupReady := s.providerCount > 0 && s.personaNames != ""
	if setupReady {
		statusLines = append(statusLines, fmt.Sprintf("  %s  %s", splashDimStyle.Render("▶"), splashPromptStyle.Render("Press Enter to start...")))
	} else {
		statusLines = append(statusLines, fmt.Sprintf("  %s  %s", splashDimStyle.Render("▶"), splashPromptStyle.Render("Press Enter to start anyway...")))
	}

	allLines := append(boxLines, statusLines...)
	content := strings.Join(allLines, "\n")

	if s.width > 0 && s.height > 0 {
		return lipgloss.Place(s.width, s.height,
			lipgloss.Left, lipgloss.Center,
			content,
		)
	}
	return content
}

func countPSCProviders() int {
	return llmfactory.CountProvidersFromEnv()
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
