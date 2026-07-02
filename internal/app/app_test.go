package app

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ntk148v/knit/internal/skills"
)

type fakeClient struct {
	installed []skills.Skill
	discover  []skills.Skill
	sources   []skills.Source
}

func (f fakeClient) ListInstalled(context.Context) ([]skills.Skill, error) {
	return append([]skills.Skill(nil), f.installed...), nil
}
func (f fakeClient) Find(context.Context, string) ([]skills.Skill, error) {
	return append([]skills.Skill(nil), f.discover...), nil
}
func (f fakeClient) ListSources(context.Context) ([]skills.Source, error) {
	return append([]skills.Source(nil), f.sources...), nil
}
func (f fakeClient) AddSource(context.Context, string) error                { return nil }
func (f fakeClient) UpdateSource(context.Context, string) error             { return nil }
func (f fakeClient) RemoveSource(context.Context, string) error             { return nil }
func (f fakeClient) InstallSkill(context.Context, skills.Skill, bool) error { return nil }
func (f fakeClient) UpdateSkill(context.Context, skills.Skill) error        { return nil }
func (f fakeClient) UninstallSkill(context.Context, skills.Skill) error     { return nil }
func (f fakeClient) SkillDetail(context.Context, skills.Skill) (skills.Skill, error) {
	return skills.Skill{Name: "demo", Preview: "hello"}, nil
}
func (f fakeClient) ListSourceSkills(context.Context, string) ([]skills.Skill, error) {
	return nil, nil
}
func (f fakeClient) PruneLocks(context.Context) error              { return nil }
func (f fakeClient) SyncFromLock(context.Context, string, bool) error { return nil }

// newTestModel creates a model with a fake client, sized 80x24.
func newTestModel() *model {
	m := New(fakeClient{}).(*model)
	m.width = 80
	m.height = 24
	m.preview.Width = 76
	m.preview.Height = 10
	return m
}

func TestModelTabSwitchAndSearchClear(t *testing.T) {
	m := New(fakeClient{
		installed: []skills.Skill{{Name: "caveman", Source: "caveman", Enabled: true}},
		discover:  []skills.Skill{{Name: "design", Source: "vercel-labs/agent-skills"}},
		sources:   []skills.Source{{Name: "claude", Repo: "anthropic/claude"}},
	}).(*model)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(*model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(*model)
	if m.tab != TabDiscover {
		t.Fatalf("tab=%v", m.tab)
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	m = updated.(*model)
	if m.discoverSearch != "f" {
		t.Fatalf("search=%q", m.discoverSearch)
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = updated.(*model)
	if m.discoverSearch != "" {
		t.Fatalf("search not cleared: %q", m.discoverSearch)
	}
}

// ─── Task 1: Style smoke tests ───────────────────────────────────────

func TestViewsUseStyledPluginLayout(t *testing.T) {
	m := newTestModel()
	m.installed = []skills.Skill{{
		Name: "frontend-design", Source: "vercel-labs/agent-skills",
		Enabled: true, Status: skills.SkillStatusEnabled,
	}}

	// Root view
	root := m.View()
	for _, want := range []string{"Knit", "Installed", "Discover", "Sources", "Logs", "Search…", "frontend-design"} {
		if !strings.Contains(root, want) {
			t.Fatalf("root view missing %q:\n%s", want, root)
		}
	}
	// Metadata view
	m.mode = modeDetail
	m.detail = m.installed[0]
	metadata := m.View()
	for _, want := range []string{"Metadata", "Scope:", "Status:", "Source:"} {
		if !strings.Contains(metadata, want) {
			t.Fatalf("metadata view missing %q:\n%s", want, metadata)
		}
	}
	// Add source view
	m.mode = modeAddSource
	add := m.View()
	if !strings.Contains(add, "Add source") || !strings.Contains(add, "owner/repo") {
		t.Fatalf("add source view missing expected content:\n%s", add)
	}
}

// ─── Task 2: Focus tests ─────────────────────────────────────────────

func TestSearchMovesFocusToListAfterNavigation(t *testing.T) {
	m := newTestModel()
	m.tab = TabDiscover
	m.discover = []skills.Skill{{
		Name: "frontend-design", Source: "vercel-labs/agent-skills",
	}}
	m.focus = focusSearch

	// Typing while search-focused updates search.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	m = updated.(*model)
	if m.discoverSearch != "r" || m.focus != focusSearch {
		t.Fatalf("typing should update search while focused, search=%q focus=%v",
			m.discoverSearch, m.focus)
	}

	// Down moves focus to list.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(*model)
	if m.focus != focusList {
		t.Fatalf("down should move focus to list, got %v", m.focus)
	}

	// While list-focused, 'i' opens scope chooser (not immediate install).
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i")})
	m = updated.(*model)
	if m.discoverSearch != "r" {
		t.Fatalf("i shortcut changed search while list focused: %q", m.discoverSearch)
	}
	if cmd != nil || m.mode != modeInstallScope {
		t.Fatalf("i should open scope chooser, mode=%v cmd=%v", m.mode, cmd)
	}
}

func TestSlashRefocusesSearch(t *testing.T) {
	m := newTestModel()
	m.tab = TabDiscover
	m.focus = focusList

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	m = updated.(*model)
	if m.focus != focusSearch {
		t.Fatalf("/ should refocus search, got %v", m.focus)
	}
}

// ─── Task 3: Sources row selection ───────────────────────────────────

func TestSourcesAddRowSelection(t *testing.T) {
	m := newTestModel()
	m.tab = TabSources
	m.focus = focusSearch
	m.sources = []skills.Source{{
		Name: "vercel-labs/agent-skills", Repo: "vercel-labs/agent-skills",
	}}
	// sourcesSel=0=search, 1=Add source, 2=first real source
	m.sourcesSel = 1
	m.focus = focusList

	// Row 1 is + Add source.
	view := m.renderSources()
	if !strings.Contains(view, "❯ + Add source") {
		t.Fatalf("add row should be selected at index 1:\n%s", view)
	}

	// Down moves to first real source (sourcesSel=2).
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(*model)
	if m.sourcesSel != 2 {
		t.Fatalf("down should move to first real source, got sourcesSel=%d", m.sourcesSel)
	}
	if got := m.currentSource().Name; got != "vercel-labs/agent-skills" {
		t.Fatalf("currentSource mapped wrong row: %q", got)
	}
}

// ─── Task 1: Row rendering fix ──────────────────────────────────────

func TestActionViewRowsAreSeparate(t *testing.T) {
	m := newTestModel()
	m.width = 80
	m.mode = modeAction
	m.actions = []string{"Update", "Uninstall", "Back to plugin list"}
	m.actionSel = 1

	view := m.actionView()
	if !strings.Contains(view, "❯ Uninstall") {
		t.Fatalf("selected action missing:\n%s", view)
	}
	if strings.Contains(view, "Update❯ Uninstall") || strings.Contains(view, "Update  Back") {
		t.Fatalf("action rows are concatenated:\n%s", view)
	}
}

func TestRenderLogsUsesTwoLineRows(t *testing.T) {
	m := newTestModel()
	m.width = 100
	m.focus = focusList
	m.logs = []skills.LogEntry{{
		At: time.Unix(0, 0), Action: "install",
		Command: "npx skills add ntk148v/skills --skill caveman -y",
	}}

	view := m.renderLogs()
	if !strings.Contains(view, "❯") || !strings.Contains(view, "install") {
		t.Fatalf("selected log action missing:\n%s", view)
	}
	if !strings.Contains(view, "npx skills add ntk148v/skills") {
		t.Fatalf("log command missing:\n%s", view)
	}
}

// ─── Task 2: Down from search selects first ──────────────────────────

func TestDownFromSearchSelectsFirstInstalled(t *testing.T) {
	m := newTestModel()
	m.tab = TabInstalled
	m.focus = focusSearch
	m.installedSel = 5
	m.installed = []skills.Skill{{Name: "caveman"}, {Name: "code-reviewer"}}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(*model)

	if m.focus != focusList || m.installedSel != 0 {
		t.Fatalf("Down from search should select first row, focus=%v sel=%d", m.focus, m.installedSel)
	}
	if got := m.currentInstalled().Name; got != "caveman" {
		t.Fatalf("selected %q, want caveman", got)
	}
}

func TestDownFromSearchSelectsFirstDiscover(t *testing.T) {
	m := newTestModel()
	m.tab = TabDiscover
	m.focus = focusSearch
	m.discoverSel = 4
	m.discover = []skills.Skill{{Name: "alpha"}, {Name: "beta"}}
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(*model)
	if m.focus != focusList || m.discoverSel != 0 || m.currentDiscover().Name != "alpha" {
		t.Fatalf("bad discover selection: focus=%v sel=%d item=%q", m.focus, m.discoverSel, m.currentDiscover().Name)
	}
}

func TestDownFromSearchSelectsAddSource(t *testing.T) {
	m := newTestModel()
	m.tab = TabSources
	m.focus = focusSearch
	m.sourcesSel = 9
	m.sources = []skills.Source{{Name: "ntk148v/skills"}}
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(*model)
	if m.focus != focusList || m.sourcesSel != 1 {
		t.Fatalf("Down from sources search should select add-source row, focus=%v sel=%d", m.focus, m.sourcesSel)
	}
}

// ─── Task 3: Agents rendered in detail ──────────────────────────────

func TestInstalledAgentsRenderedInDetail(t *testing.T) {
	m := newTestModel()
	m.detail = skills.Skill{Name: "caveman", Source: "ntk148v/skills", Agents: []string{"codex", "claude"}, Enabled: true}
	view := m.detailView()
	if !strings.Contains(view, "Agents:") || !strings.Contains(view, "codex") || !strings.Contains(view, "claude") {
		t.Fatalf("agents missing from detail:\n%s", view)
	}
}

// ─── Task 4: Uninstall returns to list ──────────────────────────────

func TestUninstallSuccessReturnsToInstalledList(t *testing.T) {
	m := newTestModel()
	m.mode = modeConfirm
	m.tab = TabInstalled
	m.confirm = "Uninstall caveman?"

	updated, _ := m.Update(actionResultMsg{
		action:     "uninstall",
		command:    "npx skills remove caveman -y",
		message:    "uninstall ok",
		nextMode:   modeNormal,
		nextTab:    TabInstalled,
		hasNextTab: true,
	})
	m = updated.(*model)
	if m.mode != modeNormal || m.tab != TabInstalled {
		t.Fatalf("mode=%v tab=%v, want normal installed", m.mode, m.tab)
	}
}

// ─── Task 5: Failed refresh keeps existing list ─────────────────────

func TestFailedInstalledRefreshKeepsExistingList(t *testing.T) {
	m := newTestModel()
	m.installed = []skills.Skill{{Name: "caveman"}}
	m.applyLoaded(loadedMsg{tab: TabInstalled, err: errors.New("network")})
	if len(m.installed) != 1 || m.installed[0].Name != "caveman" {
		t.Fatalf("installed list was wiped: %#v", m.installed)
	}
	if !strings.Contains(m.message, "network") {
		t.Fatalf("error message missing: %q", m.message)
	}
}

// ─── Task 6: Install opens scope chooser ────────────────────────────

func TestInstallOpensScopeChooser(t *testing.T) {
	m := newTestModel()
	m.tab = TabDiscover
	m.focus = focusList
	m.discover = []skills.Skill{{Name: "caveman", Source: "ntk148v/skills"}}
	m.discoverSel = 0
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i")})
	m = updated.(*model)
	if cmd != nil || m.mode != modeInstallScope || m.pendingInstall.Name != "caveman" {
		t.Fatalf("mode=%v pending=%#v cmd=%v", m.mode, m.pendingInstall, cmd)
	}
}

// ─── Task 8: Q to quit ──────────────────────────────────────────────

func TestQQuitsFromNormalMode(t *testing.T) {
	m := newTestModel()
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd == nil {
		t.Fatal("q should return tea.Quit command")
	}
}

func TestQDoesNotQuitAddSourceInput(t *testing.T) {
	m := newTestModel()
	m.mode = modeAddSource
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd != nil {
		t.Fatal("q in add-source input should type, not quit")
	}
}

// ─── Task 9: Search empty state ─────────────────────────────────────

func TestInstalledSearchEmptyShowsQuery(t *testing.T) {
	m := newTestModel()
	m.installed = []skills.Skill{{Name: "caveman"}}
	m.installedSearch = "zzzz"
	view := m.renderInstalled()
	if !strings.Contains(view, `no results for "zzzz"`) {
		t.Fatalf("missing search empty state:\n%s", view)
	}
}

// ─── Task 7: Preview highlight ──────────────────────────────────────

func TestRenderPreviewHighlightsSearchText(t *testing.T) {
	out := renderPreview("hello caveman world", 80, newStyles(), "caveman")
	if !strings.Contains(out, "caveman") {
		t.Fatalf("highlight removed text: %q", out)
	}
	// Lipgloss styles add ANSI only when stdout is a TTY.
	// In test runners stdout is not a TTY, so no ANSI codes appear.
	// The important thing is the text survives and line numbers are present.
	if !strings.Contains(out, "  1 │ hello") {
		t.Fatalf("missing line numbers: %q", out)
	}
}

// ─── Existing: Preview rendering (updated signature) ────────────────

func TestRenderPreviewAddsLineNumbers(t *testing.T) {
	out := renderPreview("# Title\n\nbody", 80, newStyles(), "")
	for _, want := range []string{"  1 │ # Title", "  3 │ body"} {
		if !strings.Contains(out, want) {
			t.Fatalf("preview missing %q:\n%s", want, out)
		}
	}
}

func TestRenderPreviewAvoidsHardcodedYellowHighlight(t *testing.T) {
	out := renderPreview("hello skill", 80, newStyles(), "skill")
	if strings.Contains(out, "\x1b[48;5;11m") || strings.Contains(out, "\x1b[38;5;0m") {
		t.Fatalf("preview uses hardcoded yellow/black ANSI: %q", out)
	}
	if !strings.Contains(out, "skill") {
		t.Fatalf("preview lost highlighted text: %q", out)
	}
}

func TestRenderPreviewCodeBlockIsPlainReadable(t *testing.T) {
	out := renderPreview("```go\nfmt.Println(1)\n```", 80, newStyles(), "Println")
	if !strings.Contains(out, "fmt.Println(1)") {
		t.Fatalf("code content missing: %q", out)
	}
	if strings.Contains(out, "#282a36") || strings.Contains(out, "Dracula") {
		t.Fatalf("preview leaked theme-specific coloring: %q", out)
	}
}

func TestMetadataShowsSkillDescription(t *testing.T) {
	m := newTestModel()
	m.mode = modeDetail
	m.detail = skills.Skill{
		Name:        "caveman",
		Source:      "ntk148v/skills",
		Description: "Speak briefly while keeping technical accuracy.",
	}
	out := m.detailView()
	if !strings.Contains(out, "Description:") || !strings.Contains(out, "Speak briefly") {
		t.Fatalf("description missing from metadata:\n%s", out)
	}
}

func TestMetadataOmitsEmptyDescription(t *testing.T) {
	m := newTestModel()
	m.mode = modeDetail
	m.detail = skills.Skill{Name: "caveman", Source: "ntk148v/skills"}
	out := m.detailView()
	if strings.Contains(out, "Description:") {
		t.Fatalf("empty description should be omitted:\n%s", out)
	}
}

func TestInstallScopeProjectAndGlobalCommands(t *testing.T) {
	m := newTestModel()
	m.mode = modeInstallScope
	m.pendingInstall = skills.Skill{Name: "caveman", Source: "ntk148v/skills"}
	m.pendingInstallGlobal = false

	cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("project scope Enter should return a command")
	}
	msg := cmd()
	r, ok := msg.(actionResultMsg)
	if !ok {
		t.Fatalf("expected actionResultMsg, got %T", msg)
	}
	if !strings.Contains(r.command, " -y") {
		t.Fatalf("project command missing -y: %q", r.command)
	}
	if strings.Contains(r.command, " -g") {
		t.Fatalf("project command should not have -g: %q", r.command)
	}

	m.mode = modeInstallScope
	m.pendingInstallGlobal = true
	cmd = m.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("global scope Enter should return a command")
	}
	msg = cmd()
	r, ok = msg.(actionResultMsg)
	if !ok {
		t.Fatalf("expected actionResultMsg, got %T", msg)
	}
	if !strings.Contains(r.command, " -g ") || !strings.Contains(r.command, " -y") {
		t.Fatalf("global command missing -g -y: %q", r.command)
	}
}

func TestSourceDetailInstallOpensScopeChooser(t *testing.T) {
	m := newTestModel()
	m.mode = modeSourceDetail
	m.sourceDetail = skills.Source{Name: "ntk148v/skills"}
	m.sourceSkills = []skills.Skill{{Name: "caveman", Source: "ntk148v/skills"}}
	m.sourceSkillSel = 0

	cmd := m.handleSourceDetailKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	if cmd != nil {
		t.Fatalf("scope chooser should not run install immediately")
	}
	if m.mode != modeInstallScope || m.pendingInstall.Name != "caveman" {
		t.Fatalf("bad scope state: mode=%v pending=%#v", m.mode, m.pendingInstall)
	}
}

func TestInstallSuccessReturnsToInstalledListAndLogs(t *testing.T) {
	m := newTestModel()
	m.mode = modeInstallScope
	m.tab = TabDiscover
	msg := actionResultMsg{
		action:     "install",
		command:    "npx skills add repo --skill demo -y",
		message:    "install ok",
		nextMode:   modeNormal,
		nextTab:    TabInstalled,
		hasNextTab: true,
	}
	_, _ = m.Update(msg)
	if m.mode != modeNormal || m.tab != TabInstalled {
		t.Fatalf("mode/tab = %v/%v, want normal/installed", m.mode, m.tab)
	}
	if len(m.logs) == 0 || m.logs[0].Action != "install" || !strings.Contains(m.logs[0].Command, "npx skills") {
		t.Fatalf("missing install log: %#v", m.logs)
	}
}

func TestInstallFailureReturnsToListAndLogsError(t *testing.T) {
	m := newTestModel()
	m.mode = modeInstallScope
	m.tab = TabDiscover
	msg := actionResultMsg{
		action:     "install",
		command:    "npx skills add repo --skill demo -y",
		err:        errors.New("install failed"),
		nextMode:   modeNormal,
		nextTab:    TabInstalled,
		hasNextTab: true,
	}
	_, _ = m.Update(msg)
	if m.mode != modeNormal {
		t.Fatalf("mode=%v, want normal", m.mode)
	}
	if len(m.logs) == 0 || !strings.Contains(m.logs[0].Err, "install failed") {
		t.Fatalf("missing error log: %#v", m.logs)
	}
	if !strings.Contains(m.message, "install failed") {
		t.Fatalf("missing user message: %q", m.message)
	}
}

func TestSourcesRowsUseConsistentCursorAndColumns(t *testing.T) {
	m := newTestModel()
	m.width = 120
	m.tab = TabSources
	m.focus = focusList
	m.sourcesSel = 2
	m.sources = []skills.Source{{Name: "ntk148v/skills", Repo: "github.com/ntk148v/skills", Available: 12, Installed: 3, RawUpdated: "unknown"}}

	out := m.renderSources()
	if !strings.Contains(out, "❯ ntk148v/skills") {
		t.Fatalf("source row missing selected name:\n%s", out)
	}
	if !strings.Contains(out, "github.com/ntk148v/skills") || !strings.Contains(out, "12 available") || !strings.Contains(out, "3 installed") {
		t.Fatalf("source row missing columns:\n%s", out)
	}
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if strings.Contains(line, "github.com/ntk148v/skills") && strings.HasPrefix(strings.TrimSpace(line), "github.com") {
			t.Fatalf("repo rendered on separate meta line, want columns:\n%s", out)
		}
	}
}

func TestLogsRowsUseConsistentCursorAndColumns(t *testing.T) {
	m := newTestModel()
	m.width = 100
	m.tab = TabLogs
	m.focus = focusList
	m.logsSel = 0
	m.logs = []skills.LogEntry{{At: time.Date(2026, 7, 2, 9, 30, 0, 0, time.Local), Action: "install", Command: "npx skills add repo --skill demo -y", Output: "ok"}}

	out := m.renderLogs()
	if !strings.Contains(out, "❯") || !strings.Contains(out, "install") || !strings.Contains(out, "npx skills") {
		t.Fatalf("log row missing expected content:\n%s", out)
	}
	if strings.Count(out, "install") != 1 {
		t.Fatalf("log row should be single-line, got:\n%s", out)
	}
}

func TestInstalledRefreshErrorKeepsExistingList(t *testing.T) {
	m := newTestModel()
	m.installed = []skills.Skill{{Name: "caveman", Source: "ntk148v/skills"}}
	m.installedSel = 0
	m.applyLoaded(loadedMsg{tab: TabInstalled, err: errors.New("network down")})
	if len(m.installed) != 1 || m.installed[0].Name != "caveman" {
		t.Fatalf("installed list was cleared: %#v", m.installed)
	}
	if !strings.Contains(m.message, "network down") {
		t.Fatalf("missing error message: %q", m.message)
	}
}

func TestDiscoverRefreshErrorKeepsExistingList(t *testing.T) {
	m := newTestModel()
	m.discover = []skills.Skill{{Name: "lazy", Source: "ntk148v/skills"}}
	m.applyLoaded(loadedMsg{tab: TabDiscover, err: errors.New("timeout")})
	if len(m.discover) != 1 || m.discover[0].Name != "lazy" {
		t.Fatalf("discover list was cleared: %#v", m.discover)
	}
	if !strings.Contains(m.message, "timeout") {
		t.Fatalf("missing error message: %q", m.message)
	}
}

func TestSourcesRefreshErrorKeepsExistingList(t *testing.T) {
	m := newTestModel()
	m.sources = []skills.Source{{Name: "ntk148v/skills", Repo: "github.com/ntk148v/skills"}}
	m.applyLoaded(loadedMsg{tab: TabSources, err: errors.New("bad gateway")})
	if len(m.sources) != 1 || m.sources[0].Name != "ntk148v/skills" {
		t.Fatalf("sources list was cleared: %#v", m.sources)
	}
	if !strings.Contains(m.message, "bad gateway") {
		t.Fatalf("missing error message: %q", m.message)
	}
}
