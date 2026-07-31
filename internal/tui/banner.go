package tui

import "github.com/charmbracelet/lipgloss"

var (
	nameStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A78BFA")).
			Bold(true).
			Padding(0, 1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6C63FF"))

	versionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#4E44CE"))

	borderStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6C63FF"))

	separatorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#3D3596"))
)

func Render(name, subtitle, version string) string {
	maxWidth := 0
	for _, s := range []string{name, subtitle, version} {
		if len(s) > maxWidth {
			maxWidth = len(s)
		}
	}
	boxWidth := maxWidth + 15
	if boxWidth < 44 {
		boxWidth = 44
	}

	pad := func(s string, w int) string {
		return lipgloss.NewStyle().Width(w).Align(lipgloss.Center).Render(s)
	}

	nameStyled := nameStyle.Render(name)
	subtStyled := subtitleStyle.Render(subtitle)
	verStyled := versionStyle.Render(version)

	top := borderStyle.Render("╔") +
		separatorStyle.Render(hline(boxWidth-2)) +
		borderStyle.Render("╗")

	bottom := borderStyle.Render("╚") +
		separatorStyle.Render(hline(boxWidth-2)) +
		borderStyle.Render("╝")

	edgeL := borderStyle.Render("║")
	edgeR := borderStyle.Render("║")

	emptyLine := edgeL + lipgloss.NewStyle().Width(boxWidth-2).Render("") + edgeR

	lines := []string{
		top,
		emptyLine,
		edgeL + pad(nameStyled, boxWidth-2) + edgeR,
		emptyLine,
		edgeL + pad(subtStyled, boxWidth-2) + edgeR,
		edgeL + pad(verStyled, boxWidth-2) + edgeR,
		emptyLine,
		bottom,
	}

	result := ""
	for _, line := range lines {
		result += line + "\n"
	}
	return result
}

func hline(n int) string {
	line := ""
	for i := 0; i < n; i++ {
		line += "═"
	}
	return line
}
