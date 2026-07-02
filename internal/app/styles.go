package app

import "github.com/charmbracelet/lipgloss"

// Claude-plugin inspired palette using AdaptiveColor so it works on any
// terminal background.
type styles struct {
	app         lipgloss.Style
	appFrame    lipgloss.Style
	logo        lipgloss.Style
	tab         lipgloss.Style
	activeTab   lipgloss.Style
	pill        lipgloss.Style

	dim             lipgloss.Style
	muted           lipgloss.Style
	accent          lipgloss.Style
	selected        lipgloss.Style
	searchHighlight lipgloss.Style
	warning         lipgloss.Style
	success         lipgloss.Style
	danger          lipgloss.Style

	border      lipgloss.Style
	panelTitle  lipgloss.Style
	searchRow   lipgloss.Style
	searchRowFocused lipgloss.Style
	footer      lipgloss.Style

	rowSelected  lipgloss.Style
	rowMuted     lipgloss.Style
	scopeProject lipgloss.Style
	scopeGlobal  lipgloss.Style
}

func newStyles() styles {
	// Semantic terminal-adaptive colors
	accentColor    := lipgloss.AdaptiveColor{Light: "#7C3AED", Dark: "#A78BFA"} // purple
	mutedColor     := lipgloss.AdaptiveColor{Light: "#9CA3AF", Dark: "#6B7280"} // gray
	successColor   := lipgloss.AdaptiveColor{Light: "#059669", Dark: "#34D399"} // green
	dangerColor    := lipgloss.AdaptiveColor{Light: "#DC2626", Dark: "#F87171"} // red
	warningColor   := lipgloss.AdaptiveColor{Light: "#D97706", Dark: "#FBBF24"} // amber
	selectedBg     := lipgloss.AdaptiveColor{Light: "#EDE9FE", Dark: "#2E1065"} // purple bg

	return styles{
		app:   lipgloss.NewStyle().Padding(0, 2),
		appFrame: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(mutedColor).
			Padding(0, 1).
			Align(lipgloss.Left),
		logo:  lipgloss.NewStyle().Bold(true).Background(accentColor).Foreground(selectedBg).Padding(0, 1),
		tab:   lipgloss.NewStyle().Padding(0, 1).Foreground(mutedColor),
		activeTab: lipgloss.NewStyle().Bold(true).Padding(0, 1).
			Foreground(lipgloss.AdaptiveColor{Light: "#7C3AED", Dark: "#A78BFA"}),
		pill: lipgloss.NewStyle().Padding(0, 1).
			Foreground(lipgloss.AdaptiveColor{Light: "#FFF", Dark: "#000"}).
			Background(accentColor),

		dim:      lipgloss.NewStyle().Faint(true).Foreground(mutedColor),
		muted:    lipgloss.NewStyle().Foreground(mutedColor),
		accent:   lipgloss.NewStyle().Bold(true),
		selected: lipgloss.NewStyle().Bold(true).Background(selectedBg).Padding(0, 1),
		searchHighlight: lipgloss.NewStyle().Bold(true).Background(selectedBg),
		warning:  lipgloss.NewStyle().Bold(true).Foreground(warningColor),
		success:  lipgloss.NewStyle().Foreground(successColor),
		danger:   lipgloss.NewStyle().Foreground(dangerColor),

		border:     lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(mutedColor).Padding(0, 1),
		panelTitle: lipgloss.NewStyle().Bold(true).Foreground(accentColor),
		searchRow: lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(mutedColor).Padding(0, 1),
		searchRowFocused: lipgloss.NewStyle().Border(lipgloss.DoubleBorder()).BorderForeground(accentColor).Padding(0, 1),
		footer:   lipgloss.NewStyle().Foreground(mutedColor),

		rowSelected: lipgloss.NewStyle().Bold(true).
			Foreground(lipgloss.AdaptiveColor{Light: "#000000", Dark: "#FFFFFF"}),
		rowMuted: lipgloss.NewStyle().Foreground(mutedColor),
		scopeProject: lipgloss.NewStyle().Bold(true).Padding(0, 1).
			Foreground(lipgloss.AdaptiveColor{Light: "#111827", Dark: "#111827"}).
			Background(lipgloss.AdaptiveColor{Light: "#BFDBFE", Dark: "#93C5FD"}),
		scopeGlobal: lipgloss.NewStyle().Bold(true).Padding(0, 1).
			Foreground(lipgloss.AdaptiveColor{Light: "#111827", Dark: "#111827"}).
			Background(lipgloss.AdaptiveColor{Light: "#BBF7D0", Dark: "#86EFAC"}),
	}
}
