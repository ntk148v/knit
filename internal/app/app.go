package app

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ntk148v/knit/internal/skills"
)

var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)

func stripANSI(s string) string { return ansiRE.ReplaceAllString(s, "") }

type Tab int

const (
	TabInstalled Tab = iota
	TabDiscover
	TabSources
	TabLogs
)

type mode int

const (
	modeNormal mode = iota
	modeHelp
	modeDetail
	modeAction
	modeAddSource
	modeConfirm
	modeSourceDetail
	modeInstallScope
	modeLogDetail
)

// focusArea tracks whether keyboard input goes to the search box or to the
// result list. Start in search; navigation keys switch to list; "/" switches
// back to search.
type focusArea int

const (
	focusSearch focusArea = iota
	focusList
)

type model struct {
	client skills.Client
	ctx    context.Context

	width, height int
	tab           Tab
	mode          mode
	focus         focusArea
	message       string
	confirm       string
	confirmDo     func() tea.Cmd

	style styles

	installed []skills.Skill
	discover  []skills.Skill
	sources   []skills.Source
	logs      []skills.LogEntry

	installedSel int
	discoverSel  int
	sourcesSel   int
	logsSel      int

	installedSearch string
	discoverSearch  string
	sourcesSearch   string

	detailSel       int
	detail          skills.Skill
	preview         viewport.Model
	previewContent  string

	actions     []string
	actionSel   int
	addSourceIn textinput.Model

	detailBack mode

	// Source detail page
	sourceDetail   skills.Source
	sourceSkills   []skills.Skill
	sourceSkillSel int

	// Install scope chooser
	pendingInstall      skills.Skill
	pendingInstallGlobal bool

	logDetail         skills.LogEntry
}

func New(client skills.Client) tea.Model {
	in := textinput.New()
	in.Placeholder = "owner/repo or URL"
	in.Prompt = ""
	in.CharLimit = 256
	in.Width = 60
	m := &model{
		client:      client,
		ctx:         context.Background(),
		addSourceIn: in,
		preview:     viewport.New(0, 0),
		style:       newStyles(),
		focus:       focusSearch,
	}
	return m
}

func (m *model) Init() tea.Cmd {
	return tea.Batch(m.refreshInstalledCmd(), m.refreshSourcesCmd())
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.preview.Width = m.contentWidth() - 4
		m.preview.Height = max(5, m.contentHeight()-10)
		// Force preview re-layout when window resizes.
		if m.previewContent != "" {
			m.preview.SetContent(m.previewContent)
		}
		return m, nil
	case tea.KeyMsg:
		if cmd := m.handleKey(msg); cmd != nil {
			return m, cmd
		}
	case loadedMsg:
		m.applyLoaded(msg)
		return m, nil
	case actionResultMsg:
		m.message = msg.message
		if msg.err != nil {
			m.message = msg.err.Error()
		}
		m.logs = append([]skills.LogEntry{{
			At:      time.Now(),
			Action:  msg.action,
			Command: msg.command,
			Output:  msg.output,
			Err:     errString(msg.err),
		}}, m.logs...)
		// Return to relevant list after action completion.
		// nextMode == 0 means modeNormal (iota), always applies when hasNextTab.
		if msg.hasNextTab {
			m.mode = msg.nextMode
			m.tab = msg.nextTab
		}
		if msg.refresh != nil {
			return m, msg.refresh
		}
		return m, nil
	case sourceSkillsLoadedMsg:
		m.applySourceSkills(msg)
		return m, nil
	case addSourceDoneMsg:
		if msg.err != nil {
			m.message = msg.err.Error()
			// Stay in modal on error so user can fix input.
			return m, nil
		}
		m.mode = modeNormal
		m.addSourceIn.SetValue("")
		m.message = msg.message
		if msg.source != "" {
			return m, m.refreshSourcesCmd()
		}
		return m, nil
	case confirmResultMsg:
		m.mode = modeNormal
		m.confirm = ""
		if msg.err != nil {
			m.message = msg.err.Error()
		} else {
			m.message = msg.message
		}
		if msg.refresh != nil {
			return m, msg.refresh
		}
		return m, nil
	}
	var cmd tea.Cmd
	m.preview, cmd = m.preview.Update(msg)
	return m, cmd
}

func (m *model) View() string {
	if m.width == 0 {
		return "loading..."
	}
	switch m.mode {
	case modeHelp:
		return m.frame("Help", helpBody(m.tab))
	case modeAddSource:
		return m.addSourceView()
	case modeAction:
		return m.actionView()
	case modeConfirm:
		return m.frame("Confirm", m.confirm+"\n\n[y] yes   [n] no")
	case modeDetail:
		return m.detailView()
	case modeSourceDetail:
		return m.sourceDetailView()
	case modeInstallScope:
		return m.installScopeView()
	case modeLogDetail:
		return m.logDetailView()
	default:
		return m.rootView()
	}
}

// handleKey dispatches key events. The routing order is:
//  1. Ctrl+C – quit
//  2. Modal overlays (help, add-source, action, confirm, detail)
//  3. "?" – help
//  4. Global keys (tab, esc-back-to-normal)
//  5. "/" – focus search
//  6. When focus is on search: printable + backspace are consumed by search
//  7. Tab-specific handlers (navigation + shortcuts)
func (m *model) handleKey(k tea.KeyMsg) tea.Cmd {
	if k.String() == "ctrl+c" {
		return tea.Quit
	}

	// ── Modal overlays ──────────────────────────────────────────────
	if m.mode == modeHelp {
		if k.String() == "esc" || k.String() == "?" {
			m.mode = modeNormal
		}
		return nil
	}
	if m.mode == modeAddSource {
		switch k.String() {
		case "esc":
			m.mode = modeNormal
			return nil
		case "enter":
			val := strings.TrimSpace(m.addSourceIn.Value())
			if val == "" {
				m.message = "source required"
				return nil
			}
			// Validate and persist locally, staying in modal on failure.
			return func() tea.Msg {
				err := m.client.AddSource(m.ctx, val)
				if err != nil {
					return addSourceDoneMsg{err: err}
				}
				return addSourceDoneMsg{message: "Loaded source " + val, source: val}
			}
		}
		var cmd tea.Cmd
		m.addSourceIn, cmd = m.addSourceIn.Update(k)
		return cmd
	}
	if m.mode == modeAction {
		actions := m.actions
		switch k.String() {
		case "esc":
			m.mode = modeNormal
			return nil
		case "j", "down":
			m.actionSel = min(m.actionSel+1, len(actions)-1)
			return nil
		case "k", "up":
			m.actionSel = max(m.actionSel-1, 0)
			return nil
		case "enter":
			return m.runSelectedAction()
		}
		return nil
	}
	if m.mode == modeConfirm {
		switch k.String() {
		case "y", "enter":
			if m.confirmDo != nil {
				return m.confirmDo()
			}
		case "n", "esc":
			m.mode = modeNormal
			m.confirm = ""
			m.confirmDo = nil
		}
		return nil
	}
	if m.mode == modeSourceDetail {
		return m.handleSourceDetailKey(k)
	}
	if m.mode == modeLogDetail {
		if k.String() == "esc" || k.String() == "q" {
			m.mode = modeNormal
		}
		return nil
	}
	if m.mode == modeInstallScope {
		switch k.String() {
		case "esc":
			m.mode = modeNormal
			m.pendingInstall = skills.Skill{}
			m.pendingInstallGlobal = false
			return nil
		case "j", "down":
			m.pendingInstallGlobal = true
			return nil
		case "k", "up":
			m.pendingInstallGlobal = false
			return nil
		case "enter":
			item := m.pendingInstall
			if item.Name == "" {
				return nil
			}
			return m.runAction("install", installCommand(item, m.pendingInstallGlobal), actionOKMessage("install", item),
				func() error { return m.client.InstallSkill(m.ctx, item, m.pendingInstallGlobal) },
				m.refreshInstalledCmd())
		case "p":
			m.pendingInstallGlobal = false
			return nil
		case "g":
			m.pendingInstallGlobal = true
			return nil
		}
		return nil
	}
	if m.mode == modeDetail {
		switch k.String() {
		case "esc":
			back := m.detailBack
			if back == 0 {
				back = modeNormal
			}
			m.mode = back
			m.detailBack = 0
			return nil
		case "h", "left":
			m.detailSel = 0
			return nil
		case "l", "right":
			m.detailSel = 1
			return nil
		case "c":
			m.mode = modeAction
			m.actionSel = 0
			m.actions = detailActions(m.tab)
			return nil
		case "j", "down":
			m.preview.LineDown(1)
			return nil
		case "k", "up":
			m.preview.LineUp(1)
			return nil
		case "ctrl+d", "pgdown":
			m.preview.LineDown(max(1, m.preview.Height/2))
			return nil
		case "ctrl+u", "pgup":
			m.preview.LineUp(max(1, m.preview.Height/2))
			return nil
		}
		return nil
	}

	// ── Non-modal keys ──────────────────────────────────────────────
	// q quits from normal/detail/help modes (not from modals like confirm/add-source).
	if k.String() == "q" && m.mode != modeAddSource && m.mode != modeConfirm && m.mode != modeInstallScope && m.mode != modeAction && m.mode != modeLogDetail {
		return tea.Quit
	}
	if k.String() == "?" {
		m.mode = modeHelp
		return nil
	}
	if m.handleAlwaysGlobal(k) {
		return nil
	}

	// "/" focuses the search box.
	if k.String() == "/" {
		m.focus = focusSearch
		return nil
	}

	// When search is focused, printable characters and backspace edit the
	// search string. Navigation keys fall through to the tab handler,
	// which will also switch focus to the list.
	if m.focus == focusSearch {
		if k.String() == "backspace" || k.String() == "delete" {
			return m.backspaceSearch()
		}
		if printable(k.String()) {
			return m.typeSearch(k.String())
		}
	}

	// Numeric tab shortcuts only fire when search is NOT focused.
	if m.handleListGlobal(k) {
		return nil
	}

	// Tab-specific handlers (navigation + shortcuts).
	switch m.tab {
	case TabInstalled:
		return m.handleInstalled(k)
	case TabDiscover:
		return m.handleDiscover(k)
	case TabSources:
		return m.handleSources(k)
	case TabLogs:
		return m.handleLogs(k)
	}
	return nil
}

func (m *model) handleAlwaysGlobal(k tea.KeyMsg) bool {
	switch k.String() {
	case "tab", "shift+tab":
		m.switchTab(k.String())
		return true
	case "esc":
		if m.hasSearch() {
			m.clearSearch()
			return true
		}
	}
	return false
}

func (m *model) handleListGlobal(k tea.KeyMsg) bool {
	switch k.String() {
	case "1", "2", "3", "4":
		m.switchTab(k.String())
		return true
	}
	return false
}

func (m *model) handleInstalled(k tea.KeyMsg) tea.Cmd {
	switch k.String() {
	case "j", "down":
		if m.focus == focusSearch {
			m.installedSel = 0
			m.focus = focusList
			return nil
		}
		m.installedSel = min(m.installedSel+1, len(m.filteredInstalled())-1)
		m.focus = focusList
	case "k", "up":
		if m.installedSel == 0 && m.focus == focusList {
			m.focus = focusSearch
			return nil
		}
		m.installedSel = max(m.installedSel-1, 0)
		m.focus = focusList
	case "g", "home":
		m.installedSel = 0
		m.focus = focusList
	case "G", "end":
		m.installedSel = max(0, len(m.filteredInstalled())-1)
		m.focus = focusList
	case "r":
		return m.refreshInstalledCmd()
	case "enter":
		item := m.currentInstalled()
		if item.Name == "" {
			return nil
		}
		m.focus = focusList
		m.detailBack = modeNormal
		return m.openDetail(item)
	case "c":
		if m.currentInstalled().Name == "" {
			return nil
		}
		m.mode = modeAction
		m.actionSel = 0
		m.actions = []string{"Update", "Uninstall", "Back to plugin list"}
	case "u":
		item := m.currentInstalled()
		if item.Name == "" {
			return nil
		}
		return m.runAction("update", updateCommand(item), actionOKMessage("update", item),
			func() error { return m.client.UpdateSkill(m.ctx, item) },
			m.refreshInstalledCmd())
	case "d":
		item := m.currentInstalled()
		if item.Name == "" {
			return nil
		}
		m.mode = modeConfirm
		m.confirm = "Uninstall " + item.Name + "?"
		m.confirmDo = func() tea.Cmd {
			return m.runAction("uninstall", removeCommand(item), actionOKMessage("uninstall", item),
				func() error { return m.client.UninstallSkill(m.ctx, item) },
				m.refreshInstalledCmd())
		}
	case "p":
		m.mode = modeConfirm
		m.confirm = "Prune orphaned locks?"
		m.confirmDo = func() tea.Cmd {
			return m.runAction("prune", "npx skills prune", "Pruned locks",
				func() error { return m.client.PruneLocks(m.ctx) }, nil)
		}
	}
	return nil
}

func (m *model) handleDiscover(k tea.KeyMsg) tea.Cmd {
	switch k.String() {
	case "j", "down":
		if m.focus == focusSearch {
			m.discoverSel = 0
			m.focus = focusList
			return nil
		}
		m.discoverSel = min(m.discoverSel+1, len(m.filteredDiscover())-1)
		m.focus = focusList
	case "k", "up":
		if m.discoverSel == 0 && m.focus == focusList {
			m.focus = focusSearch
			return nil
		}
		m.discoverSel = max(m.discoverSel-1, 0)
		m.focus = focusList
	case "g", "home":
		m.discoverSel = 0
		m.focus = focusList
	case "G", "end":
		m.discoverSel = max(0, len(m.filteredDiscover())-1)
		m.focus = focusList
	case "r":
		return m.refreshDiscoverCmd()
	case "enter":
		item := m.currentDiscover()
		if item.Name == "" {
			return nil
		}
		m.focus = focusList
		m.detailBack = modeNormal
		return m.openDetail(item)
	case "i":
		item := m.currentDiscover()
		if item.Name == "" {
			return nil
		}
		m.pendingInstall = item
		m.pendingInstallGlobal = false
		m.mode = modeInstallScope
		return nil
	case "s":
		m.mode = modeAddSource
		m.addSourceIn.Focus()
		return nil
	}
	return nil
}

// handleSources manages the Sources tab.
// Row 0 is search, row 1 is "+ Add source".
// Real sources start at row 2, so sourcesSel==0=search, 1=Add source,
// 2+= sources[sourcesSel-2].
func (m *model) handleSources(k tea.KeyMsg) tea.Cmd {
	switch k.String() {
	case "a":
		m.mode = modeAddSource
		m.addSourceIn.Focus()
		return nil
	case "S":
		m.message = "Use: knit sync -f <skills-lock.json> [-g]"
		return nil
	case "j", "down":
		if m.focus == focusSearch {
			m.sourcesSel = 1
			m.focus = focusList
			return nil
		}
		m.sourcesSel = min(m.sourcesSel+1, len(m.filteredSources())+1)
		m.focus = focusList
	case "k", "up":
		if m.sourcesSel == 0 && m.focus == focusList {
			m.focus = focusSearch
			return nil
		}
		m.sourcesSel = max(m.sourcesSel-1, 0)
		m.focus = focusList
	case "r":
		return m.refreshSourcesCmd()
	case "u":
		s := m.currentSource()
		if s.Name == "" {
			return nil
		}
		return m.runAction("update-source", "refresh source", "Updated source "+s.Name,
			func() error { return m.client.UpdateSource(m.ctx, s.Name) },
			m.refreshSourcesCmd())
	case "d":
		s := m.currentSource()
		if s.Name == "" {
			return nil
		}
		m.mode = modeConfirm
		m.confirm = "Remove source " + s.Name + "?"
		m.confirmDo = func() tea.Cmd {
			return m.runAction("remove-source", "Removed source "+s.Name, "Removed source "+s.Name,
				func() error { return m.client.RemoveSource(m.ctx, s.Name) },
				m.refreshSourcesCmd())
		}
	case "enter":
		if m.sourcesSel <= 1 {
			m.mode = modeAddSource
			m.addSourceIn.Focus()
			return nil
		}
		// Enter on real source → open source detail.
		s := m.currentSource()
		if s.Name == "" {
			return nil
		}
		return m.openSourceDetail(s)
	}
	return nil
}

func (m *model) handleLogs(k tea.KeyMsg) tea.Cmd {
	switch k.String() {
	case "j", "down":
		if m.focus == focusSearch {
			m.logsSel = 0
			m.focus = focusList
			return nil
		}
		m.logsSel = min(m.logsSel+1, max(0, len(m.logs)-1))
		m.focus = focusList
	case "k", "up":
		if m.logsSel == 0 && m.focus == focusList {
			m.focus = focusSearch
			return nil
		}
		m.logsSel = max(m.logsSel-1, 0)
		m.focus = focusList
	case "r":
		return nil
	case "c":
		m.mode = modeConfirm
		m.confirm = "Clear logs?"
		m.confirmDo = func() tea.Cmd {
			return func() tea.Msg { m.logs = nil; return confirmResultMsg{message: "logs cleared"} }
		}
	case "enter":
		if m.logsSel >= 0 && m.logsSel < len(m.logs) {
			m.logDetail = m.logs[m.logsSel]
			m.mode = modeLogDetail
		}
		return nil
	}
	return nil
}

func (m *model) switchTab(k string) {
	switch k {
	case "tab":
		m.tab = (m.tab + 1) % 4
	case "shift+tab":
		m.tab = (m.tab + 3) % 4
	case "1":
		m.tab = TabInstalled
	case "2":
		m.tab = TabDiscover
	case "3":
		m.tab = TabSources
	case "4":
		m.tab = TabLogs
	}
	m.mode = modeNormal
	m.focus = focusSearch
}

func (m *model) typeSearch(s string) tea.Cmd {
	switch m.tab {
	case TabInstalled:
		m.installedSearch += s
		m.installedSel = 0
		return nil
	case TabDiscover:
		m.discoverSearch += s
		m.discoverSel = 0
		return m.refreshDiscoverCmd()
	case TabSources:
		m.sourcesSearch += s
		m.sourcesSel = 0
		return nil
	case TabLogs:
		return nil
	}
	return nil
}

func (m *model) backspaceSearch() tea.Cmd {
	switch m.tab {
	case TabInstalled:
		if len(m.installedSearch) > 0 {
			m.installedSearch = m.installedSearch[:len(m.installedSearch)-1]
		}
	case TabDiscover:
		if len(m.discoverSearch) > 0 {
			m.discoverSearch = m.discoverSearch[:len(m.discoverSearch)-1]
		}
		return m.refreshDiscoverCmd()
	case TabSources:
		if len(m.sourcesSearch) > 0 {
			m.sourcesSearch = m.sourcesSearch[:len(m.sourcesSearch)-1]
		}
	}
	return nil
}

func (m *model) clearSearch() {
	switch m.tab {
	case TabInstalled:
		m.installedSearch = ""
	case TabDiscover:
		m.discoverSearch = ""
	case TabSources:
		m.sourcesSearch = ""
	}
}

func (m *model) hasSearch() bool { return m.searchValue() != "" }
func (m *model) searchValue() string {
	switch m.tab {
	case TabInstalled:
		return m.installedSearch
	case TabDiscover:
		return m.discoverSearch
	case TabSources:
		return m.sourcesSearch
	default:
		return ""
	}
}

// ─── Root view ──────────────────────────────────────────────────────

func (m *model) rootView() string {
	var b strings.Builder
	b.WriteString(m.header())
	b.WriteString("\n")
	switch m.tab {
	case TabInstalled:
		b.WriteString(m.renderInstalled())
	case TabDiscover:
		b.WriteString(m.renderDiscover())
	case TabSources:
		b.WriteString(m.renderSources())
	case TabLogs:
		b.WriteString(m.renderLogs())
	}
	b.WriteString("\n")
	b.WriteString(m.footer())
	if m.message != "" {
		b.WriteString("\n")
		b.WriteString(m.style.dim.Render(m.message))
	}

	content := m.style.appFrame.
		Width(m.contentWidth()).
		Height(m.contentHeight()).
		Render(b.String())

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}

func (m *model) header() string {
	tabs := []string{"Installed", "Discover", "Sources", "Logs"}
	styled := make([]string, len(tabs))
	for i, t := range tabs {
		if i == int(m.tab) {
			styled[i] = m.style.activeTab.Padding(0, 1).Render(t)
		} else {
			styled[i] = m.style.tab.Padding(0, 1).Render(t)
		}
	}
	return m.style.logo.Render(" Knit ") + "  " + strings.Join(styled, "  ")
}

func (m *model) renderSearchRow() string {
	value := m.searchValue()
	if value == "" {
		value = "Search…"
	}
	cur := "  "
	s := m.style.searchRow.Width(max(20, m.contentWidth()-4))
	if m.focus == focusSearch {
		cur = "❯ "
		s = m.style.searchRowFocused.Width(max(20, m.contentWidth()-4))
	}
	return s.Render(cur + "⌕ " + value) + "\n"
}

func (m *model) footer() string {
	return m.style.footer.Render(map[Tab]string{
		TabInstalled: "/ search · Enter view · u update · d uninstall · c actions",
		TabDiscover:  "/ search · Enter view · i install · s add source",
		TabSources:   "/ search · Enter view skills · a add · u update · d remove",
		TabLogs:      "c clear · Enter detail",
	}[m.tab])
}

// ─── List renderers ──────────────────────────────────────────────────

func (m *model) renderInstalled() string {
	var b strings.Builder
	b.WriteString(m.renderSearchRow())
	items := m.filteredInstalled()
	if len(items) == 0 {
		if m.installedSearch != "" {
			return b.String() + m.style.dim.Render(fmt.Sprintf(`no results for "%s"`, m.installedSearch))
		}
		return b.String() + m.style.dim.Render("no installed skills")
	}
	for i, s := range items {
		selected := i == m.installedSel && m.focus == focusList
		sty := rowStyle(m.style, selected)
		status := m.style.danger.Render("✖ disabled")
		if s.Enabled {
			status = m.style.success.Render("✔ enabled")
		}
		b.WriteString(renderListLine(m.contentWidth(), selected,
			rowCell{Text: s.Name, Width: 24, Style: sty},
			rowCell{Text: scopeBadge(m.style, s.Scope), Width: 4, Style: sty},
			rowCell{Text: s.Source, Style: m.style.muted},
			rowCell{Text: status, Width: 12, Style: m.style.muted}))
		b.WriteString("\n")
		if s.Description != "" {
			b.WriteString(m.style.dim.Render("  "+s.Description) + "\n")
		}
	}
	return b.String()
}

func (m *model) renderDiscover() string {
	var b strings.Builder
	b.WriteString(m.renderSearchRow())
	items := m.filteredDiscover()
	if len(items) == 0 {
		if m.discoverSearch != "" {
			return b.String() + m.style.dim.Render(fmt.Sprintf(`no results for "%s"`, m.discoverSearch))
		}
		return b.String() + m.style.dim.Render("no results")
	}
	for i, s := range items {
		selected := i == m.discoverSel && m.focus == focusList
		sty := rowStyle(m.style, selected)
		installs := m.style.dim.Render(fmt.Sprintf("%d installs", s.Installs))
		b.WriteString(renderListLine(m.contentWidth(), selected,
			rowCell{Text: s.Name, Width: 24, Style: sty},
			rowCell{Text: s.Source, Style: m.style.muted},
			rowCell{Text: installs, Width: 12, Style: m.style.dim}))
		b.WriteString("\n")
		if s.Description != "" {
			b.WriteString(m.style.dim.Render("  "+s.Description) + "\n")
		}
	}
	return b.String()
}

func (m *model) renderSources() string {
	var b strings.Builder
	b.WriteString(m.renderSearchRow())

	items := m.filteredSources()
	// Row 1: Add source.
	addSelected := m.sourcesSel == 1 && m.focus == focusList
	b.WriteString(renderListLine(m.contentWidth(), addSelected,
		rowCell{Text: "+ Add source", Style: rowStyle(m.style, addSelected)}))
	b.WriteString("\n")

	for i, s := range items {
		updated := "unknown"
		if !s.Updated.IsZero() {
			updated = s.Updated.Format("1/2/2006")
		} else if s.RawUpdated != "" {
			updated = s.RawUpdated
		}
		available := "skills unknown"
		if s.Available > 0 {
			available = fmt.Sprintf("%d available", s.Available)
		}
		selected := m.sourcesSel == i+2 && m.focus == focusList
		sty := rowStyle(m.style, selected)
		b.WriteString(renderListLine(m.contentWidth(), selected,
			rowCell{Text: s.Name, Width: 24, Style: sty},
			rowCell{Text: s.Repo, Style: m.style.muted},
			rowCell{Text: available, Width: 14, Style: m.style.dim},
			rowCell{Text: fmt.Sprintf("%d installed", s.Installed), Width: 14, Style: m.style.dim},
			rowCell{Text: "Updated " + updated, Width: 18, Style: m.style.dim}))
		b.WriteString("\n")
	}
	return b.String()
}

func (m *model) renderLogs() string {
	var b strings.Builder
	b.WriteString(m.renderSearchRow())
	if len(m.logs) == 0 {
		return b.String() + m.style.dim.Render("no logs")
	}
	for i, l := range m.logs {
		selected := i == m.logsSel && m.focus == focusList
		sty := rowStyle(m.style, selected)
		status := l.Command
		if l.Err != "" {
			status = l.Err
		}
		b.WriteString(renderListLine(m.contentWidth(), selected,
			rowCell{Text: l.At.Format(time.Kitchen), Width: 10, Style: m.style.dim},
			rowCell{Text: l.Action, Width: 16, Style: sty},
			rowCell{Text: status, Style: m.style.muted}))
		b.WriteString("\n")
	}
	return b.String()
}

// ─── Detail / sub-views ──────────────────────────────────────────────

func (m *model) detailView() string {
	agents := ""
	if len(m.detail.Agents) > 0 {
		agents = "\nAgents: " + strings.Join(m.detail.Agents, ", ")
	}
	head := fmt.Sprintf("Scope: [%s]\nStatus: %s\nSource: %s\nPath: %s%s",
		detailScopeText(m.detail.Scope),
		detailStatusText(m.detail),
		emptyDash(m.detail.Source),
		emptyDash(m.detail.Path),
		agents)
	if strings.TrimSpace(m.detail.Description) != "" {
		head += "\nDescription: " + strings.TrimSpace(m.detail.Description)
	}
	if m.detailSel == 0 {
		return m.frame("Metadata",
			head+"\n\n"+m.style.dim.Render("h/l switch pane · c actions · Esc back"))
	}
	content := m.detail.Preview
	if content == "" {
		content = m.detail.Description
	}
	m.ensurePreviewContent(content)
	return m.frame("Preview",
		m.preview.View()+"\n\n"+m.style.dim.Render("h/l switch pane · c actions · Esc back"))
}

func (m *model) actionView() string {
	if len(m.actions) == 0 {
		return m.frame("Actions", "Back")
	}
	var b strings.Builder
	for i, a := range m.actions {
		b.WriteString(renderListLine(m.contentWidth(), i == m.actionSel,
			rowCell{Text: a, Style: rowStyle(m.style, i == m.actionSel)}))
		b.WriteString("\n")
	}
	return m.frame("Actions", b.String())
}

func (m *model) addSourceView() string {
	return m.frame("Add source", strings.Join([]string{
		m.style.panelTitle.Render("Enter source:"),
		"",
		m.addSourceIn.View(),
		"",
		m.style.panelTitle.Render("Examples:"),
		m.style.dim.Render("  · owner/repo (GitHub)"),
		m.style.dim.Render("  · git@github.com:owner/repo.git (SSH)"),
		m.style.dim.Render("  · https://example.com/marketplace.json"),
		m.style.dim.Render("  · ./path/to/marketplace"),
	}, "\n"))
}

func (m *model) installScopeView() string {
	items := []struct {
		label  string
		global bool
	}{
		{"Project  (no flag, default)", false},
		{"Global   (-g)", true},
	}
	var b strings.Builder
	for _, item := range items {
		selected := m.pendingInstallGlobal == item.global
		b.WriteString(renderListLine(m.contentWidth(), selected,
			rowCell{Text: item.label, Style: rowStyle(m.style, selected)}))
		b.WriteString("\n")
	}
	return m.frame("Install scope",
		b.String()+"\n"+
			m.style.dim.Render("j/k select · Enter confirm · Esc back"))
}

// frame wraps title + body in a consistent panel with a border.
func (m *model) frame(title, body string) string {
	return m.style.border.
		Width(max(20, m.width-4)).
		Render(m.style.panelTitle.Render(title) + "\n\n" + body)
}

func (m *model) logDetailView() string {
	l := m.logDetail
	var body string
	if l.Err != "" {
		body = fmt.Sprintf("Action: %s\nTime: %s\nCommand: %s\n\nOutput:\n%s\n\nError: %s",
			l.Action, l.At.Format(time.RFC3339), l.Command, emptyDash(l.Output), l.Err)
	} else {
		body = fmt.Sprintf("Action: %s\nTime: %s\nCommand: %s\n\nOutput:\n%s",
			l.Action, l.At.Format(time.RFC3339), l.Command, emptyDash(l.Output))
	}
	body += "\n\n" + m.style.dim.Render("Esc back")
	return m.frame("Log Detail", body)
}

// ensurePreviewContent updates the viewport content only when the source
// content actually changes, so the scroll position is preserved between
// renders.
func (m *model) ensurePreviewContent(content string) {
	if content == "" || content == m.previewContent {
		return
	}
	m.previewContent = content
	term := ""
	if m.tab == TabInstalled {
		term = m.installedSearch
	} else if m.tab == TabDiscover {
		term = m.discoverSearch
	}
	m.preview.SetContent(renderPreview(content, m.contentWidth()-6, m.style, term))
}

func (m *model) openDetail(s skills.Skill) tea.Cmd {
	if m.detailBack == 0 {
		m.detailBack = modeNormal
	}
	m.detail = s
	m.detailSel = 0
	m.previewContent = "" // force re-render when detail data arrives
	m.mode = modeDetail
	return func() tea.Msg {
		if m.client == nil {
			return loadedMsg{}
		}
		d, err := m.client.SkillDetail(m.ctx, s)
		if err == nil {
			if d.Name != "" {
				m.detail = mergeDetailSkill(m.detail, d)
			}
		}
		return loadedMsg{}
	}
}

func mergeDetailSkill(base, detail skills.Skill) skills.Skill {
	if detail.Name != "" {
		base.Name = detail.Name
	}
	if detail.Source != "" {
		base.Source = detail.Source
	}
	if detail.Scope != "" {
		base.Scope = detail.Scope
	}
	if detail.Status != "" {
		base.Status = detail.Status
	}
	if detail.Path != "" {
		base.Path = detail.Path
	}
	if detail.Folder != "" {
		base.Folder = detail.Folder
	}
	if len(detail.Agents) > 0 {
		base.Agents = detail.Agents
	}
	if detail.Description != "" {
		base.Description = detail.Description
	}
	if detail.Preview != "" {
		base.Preview = detail.Preview
	}
	if len(detail.Warnings) > 0 {
		base.Warnings = detail.Warnings
	}
	if detail.Installs != 0 {
		base.Installs = detail.Installs
	}
	return base
}

// ─── Actions ─────────────────────────────────────────────────────────

func (m *model) runSelectedAction() tea.Cmd {
	item := m.currentActionItem()
	switch m.tab {
	case TabInstalled:
		switch m.actionSel {
		case 0:
			return m.runAction("update", updateCommand(item), actionOKMessage("update", item),
				func() error { return m.client.UpdateSkill(m.ctx, item) },
				m.refreshInstalledCmd())
		case 1:
			m.mode = modeConfirm
			m.confirm = "Uninstall " + item.Name + "?"
			m.confirmDo = func() tea.Cmd {
				return m.runAction("uninstall", removeCommand(item), actionOKMessage("uninstall", item),
					func() error { return m.client.UninstallSkill(m.ctx, item) },
					m.refreshInstalledCmd())
			}
		case 2:
			m.mode = modeNormal
		}
	case TabDiscover:
		switch m.actionSel {
		case 0:
			item := m.currentDiscover()
			if item.Name == "" {
				return nil
			}
			m.pendingInstall = item
			m.pendingInstallGlobal = false
			m.mode = modeInstallScope
			return nil
		case 1:
			m.mode = modeNormal
		}
	}
	return nil
}

func (m *model) currentActionItem() skills.Skill {
	if m.tab == TabInstalled {
		return m.currentInstalled()
	}
	return m.currentDiscover()
}

func (m *model) runAction(action, command, okMessage string, fn func() error, refresh tea.Cmd) tea.Cmd {
	return func() tea.Msg {
		err := fn()
		msg := actionResultMsg{
			action:  action,
			command: command,
			refresh: refresh,
			message: okMessage,
		}
		if msg.message == "" {
			msg.message = action + " ok"
		}
		if err != nil {
			msg.err = err
		}
		// Default routing by action type.
		switch action {
		case "install", "uninstall", "update", "prune":
			msg.nextMode = modeNormal
			msg.nextTab = TabInstalled
			msg.hasNextTab = true
		case "update-source", "remove-source":
			msg.nextMode = modeNormal
			msg.nextTab = TabSources
			msg.hasNextTab = true
		}
		return msg
	}
}

// ─── Refresh commands ────────────────────────────────────────────────

func (m *model) refreshInstalledCmd() tea.Cmd {
	return func() tea.Msg {
		items, err := m.client.ListInstalled(m.ctx)
		return loadedMsg{tab: TabInstalled, installed: items, err: err}
	}
}
func (m *model) refreshDiscoverCmd() tea.Cmd {
	return func() tea.Msg {
		q := strings.TrimSpace(m.discoverSearch)
		if len([]rune(q)) < 2 {
			return loadedMsg{tab: TabDiscover, discover: nil, err: nil}
		}
		items, err := m.client.Find(m.ctx, q)
		return loadedMsg{tab: TabDiscover, discover: items, err: err}
	}
}
func (m *model) refreshSourcesCmd() tea.Cmd {
	return func() tea.Msg {
		lockItems, err := m.client.ListSources(m.ctx)
		if err != nil && lockItems == nil {
			return loadedMsg{tab: TabSources, err: err}
		}
		return loadedMsg{tab: TabSources, sources: lockItems}
	}
}

func (m *model) applyLoaded(msg loadedMsg) {
	if msg.err != nil {
		m.message = msg.err.Error()
		return
	}
	switch msg.tab {
	case TabInstalled:
		if len(msg.installed) == 0 && len(m.installed) > 0 {
			return
		}
		m.installed = msg.installed
		m.installedSel = clampIndex(m.installedSel, len(m.filteredInstalled()))
	case TabDiscover:
		if len(msg.discover) == 0 && len(m.discover) > 0 {
			return
		}
		m.discover = msg.discover
		m.discoverSel = clampIndex(m.discoverSel, len(m.filteredDiscover()))
	case TabSources:
		if len(msg.sources) == 0 && len(m.sources) > 0 {
			return
		}
		m.sources = msg.sources
		m.sourcesSel = clampIndex(m.sourcesSel, len(m.filteredSources())+2)
	}
}

func (m *model) applySourceSkills(msg sourceSkillsLoadedMsg) {
	if msg.err != nil {
		m.message = stripANSI(msg.err.Error())
		m.sourceSkills = []skills.Skill{} // non-nil, stops "loading"
		return
	}
	m.sourceSkills = msg.skills
	if m.sourceSkills == nil {
		m.sourceSkills = []skills.Skill{}
	}
}

// ─── Filters and helpers ─────────────────────────────────────────────

func (m *model) filteredInstalled() []skills.Skill {
	return filterSkills(m.installed, m.installedSearch)
}
func (m *model) filteredDiscover() []skills.Skill {
	return filterSkills(m.discover, m.discoverSearch)
}
func (m *model) filteredSources() []skills.Source {
	return filterSources(m.sources, m.sourcesSearch)
}

func (m *model) currentInstalled() skills.Skill {
	items := m.filteredInstalled()
	if len(items) == 0 || m.installedSel >= len(items) {
		return skills.Skill{}
	}
	return items[m.installedSel]
}
func (m *model) currentDiscover() skills.Skill {
	items := m.filteredDiscover()
	if len(items) == 0 || m.discoverSel >= len(items) {
		return skills.Skill{}
	}
	return items[m.discoverSel]
}

// currentSource returns empty Source when search or Add-source row selected.
// sourcesSel: 0=search, 1=Add source, 2+=sources[sourcesSel-2].
func (m *model) currentSource() skills.Source {
	if m.sourcesSel <= 1 {
		return skills.Source{}
	}
	items := m.filteredSources()
	idx := m.sourcesSel - 2
	if idx < 0 || idx >= len(items) {
		return skills.Source{}
	}
	return items[idx]
}

// ─── Source detail mode ────────────────────────────────────────────

func (m *model) openSourceDetail(s skills.Source) tea.Cmd {
	m.sourceDetail = s
	m.sourceSkills = nil
	m.sourceSkillSel = 0
	m.mode = modeSourceDetail
	return m.loadSourceSkillsCmd()
}

func (m *model) loadSourceSkillsCmd() tea.Cmd {
	return func() tea.Msg {
		items, err := m.client.ListSourceSkills(m.ctx, m.sourceDetail.Name)
		return sourceSkillsLoadedMsg{skills: items, err: err}
	}
}

func (m *model) handleSourceDetailKey(k tea.KeyMsg) tea.Cmd {
	switch k.String() {
	case "esc":
		m.mode = modeNormal
		return nil
	case "j", "down":
		m.sourceSkillSel = min(m.sourceSkillSel+1, len(m.sourceSkills)-1)
	case "k", "up":
		m.sourceSkillSel = max(m.sourceSkillSel-1, 0)
	case "enter":
		item := m.currentSourceSkill()
		if item.Name == "" {
			return nil
		}
		m.detailBack = modeSourceDetail
		return m.openDetail(m.enrichSourceSkillForDetail(item))
	case "i":
		item := m.currentSourceSkill()
		if item.Name == "" {
			return nil
		}
		if item.Source == "" {
			item.Source = m.sourceDetail.Name
		}
		m.pendingInstall = item
		m.pendingInstallGlobal = false
		m.mode = modeInstallScope
		return nil
	case "u":
		return m.runAction("update-source", "refresh source", "Updated source "+m.sourceDetail.Name,
			func() error { return m.client.UpdateSource(m.ctx, m.sourceDetail.Name) },
			m.loadSourceSkillsCmd())
	case "d":
		m.mode = modeConfirm
		m.confirm = "Remove source " + m.sourceDetail.Name + "?"
		m.confirmDo = func() tea.Cmd {
			return m.runAction("remove-source", "remove source from ~/.config/knit/knit.json", "Removed source "+m.sourceDetail.Name,
				func() error { return m.client.RemoveSource(m.ctx, m.sourceDetail.Name) },
				m.refreshSourcesCmd())
		}
	}
	return nil
}

func (m *model) currentSourceSkill() skills.Skill {
	if m.sourceSkillSel < 0 || m.sourceSkillSel >= len(m.sourceSkills) {
		return skills.Skill{}
	}
	return m.sourceSkills[m.sourceSkillSel]
}

func (m *model) enrichSourceSkillForDetail(item skills.Skill) skills.Skill {
	if item.Source == "" {
		item.Source = m.sourceDetail.Name
	}
	if item.ID == "" && item.Source != "" && item.Name != "" {
		item.ID = item.Source + "/" + item.Name
	}
	for _, installed := range m.installed {
		if sameSourceSkill(installed, item) {
			return mergeDetailSkill(item, installed)
		}
	}
	item.Status = skills.SkillStatus("available")
	item.Scope = ""
	return item
}

func sameSourceSkill(a, b skills.Skill) bool {
	if !strings.EqualFold(strings.TrimSpace(a.Name), strings.TrimSpace(b.Name)) {
		return false
	}
	as := strings.Trim(strings.ToLower(strings.TrimSpace(a.Source)), "/")
	bs := strings.Trim(strings.ToLower(strings.TrimSpace(b.Source)), "/")
	return as == "" || bs == "" || as == bs
}

func (m *model) sourceDetailView() string {
	src := m.sourceDetail
	header := fmt.Sprintf("Source: %s\n%s\n", m.style.accent.Render(src.Name), m.style.dim.Render(src.Repo))
	if m.sourceSkills == nil {
		return m.frame("Source Detail", header+m.style.dim.Render("loading skills…"))
	}
	var b strings.Builder
	b.WriteString(header)
	if len(m.sourceSkills) == 0 {
		b.WriteString("\n" + m.style.dim.Render("no skills found") + "\n")
		return m.frame("Source Detail", b.String())
	}
	b.WriteString(fmt.Sprintf("\n%s\n\n", m.style.dim.Render(fmt.Sprintf("%d skills", len(m.sourceSkills)))))
	for i, p := range m.sourceSkills {
		selected := i == m.sourceSkillSel
		sty := rowStyle(m.style, selected)
		b.WriteString(renderListLine(m.contentWidth(), selected,
			rowCell{Text: p.Name, Style: sty}))
		b.WriteString("\n")
	}
	b.WriteString("\n" + m.style.dim.Render("Enter preview/install · i install · u update · d remove · Esc back"))
	return m.frame("Source Detail", b.String())
}

func detailActions(tab Tab) []string {
	if tab == TabInstalled {
		return []string{"Update", "Uninstall", "Back to plugin list"}
	}
	return []string{"Install", "Back to plugin list"}
}

func helpBody(tab Tab) string {
	base := []string{"Tab/Shift+Tab switch tabs", "1-4 jump tabs", "?/Esc help", "q/Ctrl+C quit", "/ search"}
	specific := map[Tab][]string{
		TabInstalled: {"j/k move", "u update", "d uninstall", "c actions"},
		TabDiscover:  {"j/k move", "i install", "s add source", "c actions"},
		TabSources:   {"a add", "u update", "d remove"},
		TabLogs:      {"c clear"},
	}[tab]
	return strings.Join(append(append(base, ""), specific...), "\n")
}

func filterSkills(items []skills.Skill, q string) []skills.Skill {
	if q == "" {
		return items
	}
	q = strings.ToLower(q)
	var out []skills.Skill
	for _, s := range items {
		if strings.Contains(strings.ToLower(s.Name), q) ||
			strings.Contains(strings.ToLower(s.Source), q) ||
			strings.Contains(strings.ToLower(s.Description), q) {
			out = append(out, s)
		}
	}
	return out
}

func filterSources(items []skills.Source, q string) []skills.Source {
	if q == "" {
		return items
	}
	q = strings.ToLower(q)
	var out []skills.Source
	for _, s := range items {
		if strings.Contains(strings.ToLower(s.Name), q) || strings.Contains(strings.ToLower(s.Repo), q) {
			out = append(out, s)
		}
	}
	return out
}

func printable(s string) bool {
	return len(s) == 1 && s[0] >= 32 && s[0] != 127 && !unicode.IsControl(rune(s[0]))
}
func statusText(s skills.SkillStatus) string {
	if s == "" {
		return "unknown"
	}
	return string(s)
}

func detailStatusText(s skills.Skill) string {
	if s.Status == skills.SkillStatus("available") {
		return "Available"
	}
	if s.Scope != "" {
		return "Installed"
	}
	return statusText(s.Status)
}

func detailScopeText(scope skills.Scope) string {
	if scope == "" {
		return "-"
	}
	return strings.Title(string(scope))
}
func emptyDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func scopeFlag(global bool) string {
	if global {
		return "-g"
	}
	return ""
}

func actionOKMessage(action string, skill skills.Skill) string {
	name := strings.TrimSpace(skill.Name)
	if name == "" {
		return action + " ok"
	}
	switch action {
	case "install":
		return "Installed " + name
	case "update":
		return "Updated " + name
	case "uninstall":
		return "Removed " + name
	default:
		return action + " ok"
	}
}

func installCommand(item skills.Skill, global bool) string {
	if global {
		return fmt.Sprintf("npx skills add %s --skill %s -g -y", item.Source, item.Name)
	}
	return fmt.Sprintf("npx skills add %s --skill %s -y", item.Source, item.Name)
}

func updateCommand(item skills.Skill) string {
	cmd := fmt.Sprintf("npx skills update %s -y", item.Name)
	if item.Scope == skills.ScopeGlobal || item.Scope == skills.ScopeUser {
		return cmd + " -g"
	}
	if item.Scope == skills.ScopeProject {
		return cmd + " -p"
	}
	return cmd
}

func removeCommand(item skills.Skill) string {
	cmd := fmt.Sprintf("npx skills remove %s -y", item.Name)
	if item.Scope == skills.ScopeGlobal || item.Scope == skills.ScopeUser {
		return cmd + " -g"
	}
	if item.Scope == skills.ScopeProject {
		return cmd + " -p"
	}
	return cmd
}

func scopeBadge(style styles, scope skills.Scope) string {
	switch scope {
	case skills.ScopeGlobal, skills.ScopeUser:
		return style.scopeGlobal.Render("G")
	case skills.ScopeProject:
		return style.scopeProject.Render("P")
	default:
		return style.dim.Render("-")
	}
}

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
	err                            error
	refresh                         tea.Cmd
	nextMode                        mode
	nextTab                         Tab
	hasNextTab                      bool
}

type confirmResultMsg struct {
	message string
	err     error
	refresh tea.Cmd
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
// warn and dim remain as package-level helpers for non-model code.
// ponytail: warn is a no-op; kept for callers that may pass user strings.
// Remove when all callers migrate to m.style.warning.
func warn(s string) string { return s }

func dim(s string) string { return lipgloss.NewStyle().Faint(true).Render(s) }

func (m *model) contentWidth() int {
	return max(20, m.width-8)
}

func (m *model) contentHeight() int {
	return max(8, m.height-4)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
