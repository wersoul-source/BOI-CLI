package tui

type HelpModel struct {
	width int
	show  bool
}

func NewHelp() HelpModel {
	return HelpModel{show: true}
}

func (h *HelpModel) SetWidth(w int) {
	h.width = w
}

func (h *HelpModel) Toggle() {
	h.show = !h.show
}

func (h *HelpModel) View() string {
	if !h.show {
		return ""
	}
	help := "Enter:send  Tab:persona  Ctrl+Q:quit  /:commands  Ctrl+L:clear"
	if h.width > 0 {
		return HelpStyle.Width(h.width).Render(help)
	}
	return HelpStyle.Render(help)
}
