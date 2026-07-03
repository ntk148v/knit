package app

import (
	"context"
	"errors"
	"fmt"
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
func (f fakeClient) PruneLocks(context.Context) error                 { return nil }
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

// ─── Test helpers for alignment tests ──────────────────────────────

func testLineContaining(t *testing.T, out, needle string) string {
	t.Helper()
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, needle) {
			return line
		}
	}
	t.Fatalf("missing line containing %q:\n%s", needle, out)
	return ""
}

func testGapAtMost(t *testing.T, line, left, right string, maxGap int) {
	t.Helper()
	leftIdx := strings.Index(line, left)
	rightIdx := strings.Index(line, right)
	if leftIdx < 0 || rightIdx < 0 || rightIdx <= leftIdx {
		t.Fatalf("line does not contain %q before %q:\n%s", left, right, line)
	}
	gap := rightIdx - (leftIdx + len(left))
	if gap > maxGap {
		t.Fatalf("gap between %q and %q = %d, want <= %d:\n%s", left, right, gap, maxGap, line)
	}
}

func testStartsAfter(t *testing.T, line, needle string, minIndex int) {
	t.Helper()
	idx := strings.Index(line, needle)
	if idx < 0 {
		t.Fatalf("missing %q in line:\n%s", needle, line)
	}
	if idx < minIndex {
		t.Fatalf("%q starts at column %d, want >= %d:\n%s", needle, idx, minIndex, line)
	}
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

// ─── Alignment tests ────────────────────────────────────────────────

func TestInstalledRootViewRowsAreLeftAlignedInFrame(t *testing.T) {
	m := newTestModel()
	m.width = 180
	m.height = 24
	m.tab = TabInstalled
	m.focus = focusList
	m.installed = []skills.Skill{{Name: "caveman", Source: "ntk148v/skills", Scope: skills.ScopeProject, Enabled: true}}

	line := testLineContaining(t, m.rootView(), "caveman")
	idx := strings.Index(line, "caveman")
	if idx > 12 {
		t.Fatalf("installed row appears centered, name starts at column %d:\n%s", idx, line)
	}
}

func TestInstalledRowsUseFullWidthLikeSources(t *testing.T) {
	m := newTestModel()
	m.width = 180
	m.tab = TabInstalled
	m.focus = focusList
	m.installed = []skills.Skill{{Name: "caveman", Source: "ntk148v/skills", Scope: skills.ScopeProject, Enabled: true}}

	line := testLineContaining(t, m.renderInstalled(), "caveman")
	// Name, scope, source, status should all appear on one line.
	if !strings.Contains(line, "P") || !strings.Contains(line, "ntk148v/skills") || !strings.Contains(line, "✔ enabled") {
		t.Fatalf("installed row missing columns:\n%s", line)
	}
	// No fixed widths = status can appear anywhere leftwards; just verify not truncated.
	idx := strings.Index(line, "✔ enabled")
	if idx < 20 {
		t.Fatalf("status starts at col %d, looks crowded:\n%s", idx, line)
	}
}

func TestDiscoverRowsUseFullWidthLikeSources(t *testing.T) {
	m := newTestModel()
	m.width = 180
	m.tab = TabDiscover
	m.focus = focusList
	m.discover = []skills.Skill{{Name: "caveman", Source: "ntk148v/skills", Installs: 42}}

	line := testLineContaining(t, m.renderDiscover(), "caveman")
	if !strings.Contains(line, "ntk148v/skills") || !strings.Contains(line, "42 installs") {
		t.Fatalf("discover row missing columns:\n%s", line)
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
	for _, want := range []string{"Agents:", "claude ✓", "codex ✓", "cursor -", "gemini -"} {
		if !strings.Contains(view, want) {
			t.Fatalf("agent matrix missing %q:\n%s", want, view)
		}
	}
}

func TestInstalledBulkSelectionAndUpdate(t *testing.T) {
	client := &bulkFakeClient{}
	m := New(client).(*model)
	m.width = 80
	m.height = 24
	m.tab = TabInstalled
	m.focus = focusList
	m.installed = []skills.Skill{{Name: "one"}, {Name: "two"}}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = updated.(*model)
	m.installedSel = 1
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = updated.(*model)
	if len(m.selectedInstalled) != 2 {
		t.Fatalf("selected=%#v", m.selectedInstalled)
	}

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("U")})
	if cmd == nil {
		t.Fatal("bulk update should return a command")
	}
	msg := cmd()
	updated, _ = m.Update(msg)
	m = updated.(*model)
	if strings.Join(client.updated, ",") != "one,two" {
		t.Fatalf("updated=%v", client.updated)
	}
	if len(m.selectedInstalled) != 0 {
		t.Fatalf("bulk success should clear selection: %#v", m.selectedInstalled)
	}
}

type bulkFakeClient struct {
	fakeClient
	updated []string
}

func (f *bulkFakeClient) UpdateSkill(_ context.Context, s skills.Skill) error {
	f.updated = append(f.updated, s.Name)
	return nil
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

func TestRenderPreviewStripsFrontmatter(t *testing.T) {
	out := renderPreview("---\nname: demo\ndescription: hidden\n---\n# Demo", 80, newStyles(), "")
	if strings.Contains(out, "---") || strings.Contains(out, "description: hidden") || !strings.Contains(out, "# Demo") {
		t.Fatalf("frontmatter not stripped cleanly:\n%s", out)
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

func TestMergeDetailSkillClearsStaleWarnings(t *testing.T) {
	got := mergeDetailSkill(
		skills.Skill{Name: "demo", Warnings: []string{"missing description"}},
		skills.Skill{Name: "demo", Description: "exists"},
	)
	if len(got.Warnings) != 0 {
		t.Fatalf("stale warnings were not cleared: %#v", got.Warnings)
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

// ─── Scroll tests ─────────────────────────────────────────────────

func TestInstalledLongListClipsOutput(t *testing.T) {
	m := newTestModel()
	m.tab = TabInstalled
	m.focus = focusList
	m.installed = make([]skills.Skill, 30)
	for i := range m.installed {
		m.installed[i] = skills.Skill{
			Name:    fmt.Sprintf("skill-%d", i),
			Source:  "test-source",
			Scope:   skills.ScopeProject,
			Enabled: true,
		}
	}
	m.installedSel = 0
	view := m.renderInstalled()
	// First item should be visible.
	if !strings.Contains(view, "skill-0") {
		t.Fatalf("first item missing from view:\n%s", view)
	}
	// Last item should NOT be visible (clipped).
	if strings.Contains(view, "skill-29") {
		t.Fatalf("last item unexpectedly visible, window too large:\n%s", view)
	}
	// Down-scroll indicator should be present.
	if !strings.Contains(view, "↓ more") {
		t.Fatalf("down scroll indicator missing:\n%s", view)
	}
}

func TestInstalledDownScrollsIntoView(t *testing.T) {
	m := newTestModel()
	m.tab = TabInstalled
	m.focus = focusList
	m.installed = make([]skills.Skill, 30)
	for i := range m.installed {
		m.installed[i] = skills.Skill{
			Name:    fmt.Sprintf("skill-%d", i),
			Source:  "test-source",
			Scope:   skills.ScopeProject,
			Enabled: true,
		}
	}
	// Move selection far down.
	m.installedSel = 25
	view := m.renderInstalled()
	if !strings.Contains(view, "skill-25") {
		t.Fatalf("selected item skill-25 not visible:\n%s", view)
	}
	// First item should have scrolled out.
	if strings.Contains(view, "skill-0") {
		t.Fatalf("first item still visible after scrolling down:\n%s", view)
	}
	// Up-scroll indicator should be present.
	if !strings.Contains(view, "↑ more") {
		t.Fatalf("up scroll indicator missing:\n%s", view)
	}
}

func TestInstalledEndJumpsToBottom(t *testing.T) {
	m := newTestModel()
	m.tab = TabInstalled
	m.focus = focusList
	m.installed = make([]skills.Skill, 30)
	for i := range m.installed {
		m.installed[i] = skills.Skill{
			Name:    fmt.Sprintf("skill-%d", i),
			Source:  "test-source",
			Scope:   skills.ScopeProject,
			Enabled: true,
		}
	}
	// Simulate pressing G: set selection to last item.
	m.installedSel = 29
	view := m.renderInstalled()
	if !strings.Contains(view, "skill-29") {
		t.Fatalf("last item skill-29 not visible after G:\n%s", view)
	}
	// Up indicator should be present.
	if !strings.Contains(view, "↑ more") {
		t.Fatalf("up scroll indicator missing after G:\n%s", view)
	}
	// Down indicator should NOT be present at the bottom.
	if strings.Contains(view, "↓ more") {
		t.Fatalf("down scroll indicator present at bottom:\n%s", view)
	}
}

func TestInstalledSearchResetsOffset(t *testing.T) {
	m := newTestModel()
	m.tab = TabInstalled
	m.focus = focusList
	m.installed = make([]skills.Skill, 30)
	for i := range m.installed {
		m.installed[i] = skills.Skill{
			Name:    fmt.Sprintf("skill-%d", i),
			Source:  "test-source",
			Scope:   skills.ScopeProject,
			Enabled: true,
		}
	}
	// Scroll down and verify offset moved.
	m.installedSel = 25
	m.renderInstalled() // triggers offset adjustment
	if m.installedOffset == 0 {
		t.Fatal("expected non-zero offset after scrolling down")
	}
	// Type a search character — offset should reset.
	m.typeSearch("test")
	if m.installedOffset != 0 {
		t.Fatalf("offset should be 0 after search, got %d", m.installedOffset)
	}
}

func TestInstalledBackspaceResetsOffset(t *testing.T) {
	m := newTestModel()
	m.tab = TabInstalled
	m.focus = focusList
	m.installed = make([]skills.Skill, 30)
	for i := range m.installed {
		m.installed[i] = skills.Skill{
			Name:    fmt.Sprintf("skill-%d", i),
			Source:  "test-source",
			Scope:   skills.ScopeProject,
			Enabled: true,
		}
	}
	// Scroll down first.
	m.installedSel = 25
	m.renderInstalled()
	prevOffset := m.installedOffset
	// Simulate backspace (deleting from an existing search).
	m.installedSearch = "x"
	m.backspaceSearch()
	// Offset should reset.
	if m.installedOffset != 0 {
		t.Fatalf("offset should be 0 after backspace, got %d (was %d)", m.installedOffset, prevOffset)
	}
}

func TestDiscoverScrollClipsOutput(t *testing.T) {
	m := newTestModel()
	m.tab = TabDiscover
	m.focus = focusList
	m.discover = make([]skills.Skill, 30)
	for i := range m.discover {
		m.discover[i] = skills.Skill{
			Name:   fmt.Sprintf("discover-%d", i),
			Source: "test-source",
		}
	}
	m.discoverSel = 0
	view := m.renderDiscover()
	if !strings.Contains(view, "discover-0") {
		t.Fatalf("first item missing:\n%s", view)
	}
	if strings.Contains(view, "discover-29") {
		t.Fatalf("last item unexpectedly visible:\n%s", view)
	}
	if !strings.Contains(view, "↓ more") {
		t.Fatalf("down scroll indicator missing:\n%s", view)
	}
}

func TestLogsScrollClipsOutput(t *testing.T) {
	m := newTestModel()
	m.tab = TabLogs
	m.focus = focusList
	m.logs = make([]skills.LogEntry, 30)
	for i := range m.logs {
		m.logs[i] = skills.LogEntry{
			At:      time.Unix(int64(i), 0),
			Action:  "install",
			Command: fmt.Sprintf("npx install %d", i),
		}
	}
	m.logsSel = 0
	view := m.renderLogs()
	if !strings.Contains(view, "install") {
		t.Fatalf("first log missing:\n%s", view)
	}
	if !strings.Contains(view, "↓ more") {
		t.Fatalf("down scroll indicator missing:\n%s", view)
	}
}

func TestSourcesSuccessfulEmptyRefreshClearsExistingList(t *testing.T) {
	m := newTestModel()
	m.sources = []skills.Source{{Name: "ntk148v/skills", Repo: "ntk148v/skills"}}
	m.sourcesSel = 2

	m.applyLoaded(loadedMsg{tab: TabSources, sources: []skills.Source{}})

	if len(m.sources) != 0 {
		t.Fatalf("sources list stayed stale after empty refresh: %#v", m.sources)
	}
	// Selection may be 0 (search) or 1 (Add source row) — both are valid
	// as long as it isn't pointing to a non-existent source.
	if m.sourcesSel > 1 {
		t.Fatalf("sourcesSel=%d, want 0 or 1 after empty refresh", m.sourcesSel)
	}
}

func TestRemoveSourceResultReturnsToSourcesAndRefreshes(t *testing.T) {
	m := newTestModel()
	m.mode = modeConfirm
	m.tab = TabSources
	msg := actionResultMsg{
		action:     "remove-source",
		command:    "Removed source ntk148v/skills",
		message:    "Removed source ntk148v/skills",
		nextMode:   modeNormal,
		nextTab:    TabSources,
		hasNextTab: true,
		refresh:    m.refreshSourcesCmd(),
	}

	updated, cmd := m.Update(msg)
	m = updated.(*model)

	if m.mode != modeNormal || m.tab != TabSources {
		t.Fatalf("mode/tab=%v/%v, want normal/sources", m.mode, m.tab)
	}
	if cmd == nil {
		t.Fatal("expected refresh command")
	}
	if m.message != "Removed source ntk148v/skills" {
		t.Fatalf("message=%q", m.message)
	}
}
