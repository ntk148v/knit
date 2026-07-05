package app

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/ntk148v/knit/internal/skills"
)

// ─── Message types ───────────────────────────────────────────────────

type loadedMsg struct {
	tab       Tab
	installed []skills.Skill
	discover  []skills.Skill
	sources   []skills.Source
	err       error
}
type sourceSkillsLoadedMsg struct {
	skills []skills.Skill
	err    error
}
type addSourceDoneMsg struct {
	err     error
	source  string
	message string
}
type actionResultMsg struct {
	action, command, output, message string
	err                              error
	refresh                          tea.Cmd
	nextMode                         mode
	nextTab                          Tab
	hasNextTab                       bool
}

type confirmResultMsg struct {
	message string
	err     error
	refresh tea.Cmd
}

type detailLoadedMsg struct {
	skill skills.Skill
	err   error
}

type clearLogsMsg struct{}

type debouncedSearchMsg struct {
	query string
	seq   int
}
