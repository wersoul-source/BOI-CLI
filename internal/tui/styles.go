package tui

import "github.com/charmbracelet/lipgloss"

var (
	boiTeal     = lipgloss.Color("37")
	boiPurple   = lipgloss.Color("99")
	boiGreen    = lipgloss.Color("78")
	boiOrange   = lipgloss.Color("208")
	boiRed      = lipgloss.Color("197")
	bgDark      = lipgloss.Color("234")
	textPrimary = lipgloss.Color("252")
	textDim     = lipgloss.Color("245")
	borderColor = lipgloss.Color("62")

	UserStyle = lipgloss.NewStyle().
			Foreground(boiGreen).
			Bold(true)

	AgentStyle = lipgloss.NewStyle().
			Foreground(boiPurple).
			Bold(true)

	StatusBarStyle = lipgloss.NewStyle().
			Background(boiTeal).
			Foreground(lipgloss.Color("255")).
			Padding(0, 1)

	HelpStyle = lipgloss.NewStyle().
			Foreground(textDim).
			Padding(0, 1)

	ChatBorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColor)

	InputBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(borderColor)

	SpinnerStyle = lipgloss.NewStyle().
			Foreground(boiOrange)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(boiRed).
			Bold(true)

	InfoStyle = lipgloss.NewStyle().
			Foreground(textDim)

	HighlightStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("78")).
			Bold(true)

	SuggestionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("141"))

	DimStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	SystemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	timeStampStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))
)

// bubbleStyle holds ASCII box-drawing characters and styled sub-elements.
type bubbleStyle struct {
	topLeft  string
	topRight string
	botLeft  string
	botRight string
	vert     string
	horiz    string
	header   lipgloss.Style
	meta     lipgloss.Style
}

var (
	UserBubbleStyle = bubbleStyle{
		topLeft:  lipgloss.NewStyle().Foreground(lipgloss.Color("78")).Render("╭─"),
		topRight: lipgloss.NewStyle().Foreground(lipgloss.Color("78")).Render("─╮"),
		botLeft:  lipgloss.NewStyle().Foreground(lipgloss.Color("78")).Render("╰─"),
		botRight: lipgloss.NewStyle().Foreground(lipgloss.Color("78")).Render("─╯"),
		vert:     lipgloss.NewStyle().Foreground(lipgloss.Color("78")).Render("│"),
		horiz:    "─",
		header:   UserStyle.Copy().Foreground(lipgloss.Color("78")),
		meta:     lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
	}

	AgentBubbleStyle = bubbleStyle{
		topLeft:  lipgloss.NewStyle().Foreground(lipgloss.Color("99")).Render("╭─"),
		topRight: lipgloss.NewStyle().Foreground(lipgloss.Color("99")).Render("─╮"),
		botLeft:  lipgloss.NewStyle().Foreground(lipgloss.Color("99")).Render("╰─"),
		botRight: lipgloss.NewStyle().Foreground(lipgloss.Color("99")).Render("─╯"),
		vert:     lipgloss.NewStyle().Foreground(lipgloss.Color("99")).Render("│"),
		horiz:    "─",
		header:   AgentStyle.Copy().Foreground(lipgloss.Color("99")),
		meta:     lipgloss.NewStyle().Foreground(lipgloss.Color("141")),
	}
)
