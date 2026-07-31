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
)
