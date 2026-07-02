package app

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ntk148v/knit/internal/skills"
)

func TestSourceDetailShowsLoadedSkills(t *testing.T) {
	m := newTestModel()
	m.mode = modeSourceDetail
	m.sourceDetail = skills.Source{Name: "ntk148v/skills", Repo: "github.com/ntk148v/skills"}
	m.applySourceSkills(sourceSkillsLoadedMsg{skills: []skills.Skill{{
		Name:   "grill-with-doc",
		Source: "ntk148v/skills",
	}}})

	out := m.sourceDetailView()
	for _, want := range []string{"1 skills", "grill-with-doc"} {
		if !strings.Contains(out, want) {
			t.Fatalf("source detail missing %q:\n%s", want, out)
		}
	}
	// Source repo should not repeat on each row; it's in the header.
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "grill-with-doc") && strings.Contains(line, "ntk148v/skills") {
			if !strings.Contains(line, "Source:") && !strings.Contains(line, "github.com") {
				t.Fatalf("source repeated on row, want name only:\n%s", out)
			}
		}
	}
	if strings.Contains(out, "loading") || strings.Contains(out, "plugins") {
		t.Fatalf("source detail has stale loading/plugin copy:\n%s", out)
	}
}

func TestSourceDetailErrorStopsLoadingAndShowsSanitizedMessage(t *testing.T) {
	m := newTestModel()
	m.mode = modeSourceDetail
	m.sourceDetail = skills.Source{Name: "bad/source", Repo: "bad/source"}
	m.sourceSkills = nil

	m.applySourceSkills(sourceSkillsLoadedMsg{err: errors.New("\x1b[31mfailed to clone\x1b[0m")})

	out := m.sourceDetailView()
	if strings.Contains(out, "loading skills") {
		t.Fatalf("source detail should stop loading after error:\n%s", out)
	}
	if !strings.Contains(out, "no skills found") {
		t.Fatalf("source detail should show empty state after error:\n%s", out)
	}
	if strings.Contains(m.message, "\x1b[") {
		t.Fatalf("message contains raw ANSI: %q", m.message)
	}
}

func TestActionResultShowsBriefFooterAndDetailedLog(t *testing.T) {
	m := newTestModel()
	msg := actionResultMsg{
		action:  "install",
		command: "npx skills add ntk148v/skills --skill grill-with-doc -g -y",
		output:  "installed grill-with-doc for claude-code",
		message: "Installed grill-with-doc",
	}

	_, _ = m.Update(msg)

	if m.message != "Installed grill-with-doc" {
		t.Fatalf("footer message=%q", m.message)
	}
	if len(m.logs) != 1 {
		t.Fatalf("logs len=%d", len(m.logs))
	}
	if m.logs[0].Command != msg.command || m.logs[0].Output != msg.output || m.logs[0].Action != "install" {
		t.Fatalf("log entry mismatch: %#v", m.logs[0])
	}
}

func TestActionResultErrorShowsBriefFooterAndDetailedErrorLog(t *testing.T) {
	m := newTestModel()
	msg := actionResultMsg{
		action:  "update",
		command: "npx skills update grill-with-doc -y",
		err:     errors.New("failed to update grill-with-doc"),
	}

	_, _ = m.Update(msg)

	if !strings.Contains(m.message, "failed to update") {
		t.Fatalf("footer message=%q", m.message)
	}
	if len(m.logs) != 1 || !strings.Contains(m.logs[0].Err, "failed to update") {
		t.Fatalf("error log missing: %#v", m.logs)
	}
}

func TestInstallResultReturnsToInstalledAndRefreshes(t *testing.T) {
	m := newTestModel()
	m.mode = modeInstallScope
	m.tab = TabDiscover
	msg := actionResultMsg{
		action:     "install",
		command:    "npx skills add ntk148v/skills --skill grill-with-doc -g -y",
		message:    "Installed grill-with-doc",
		nextMode:   modeNormal,
		nextTab:    TabInstalled,
		hasNextTab: true,
		refresh:    m.refreshInstalledCmd(),
	}

	updated, cmd := m.Update(msg)
	m = updated.(*model)

	if m.mode != modeNormal || m.tab != TabInstalled {
		t.Fatalf("mode/tab=%v/%v, want normal/installed", m.mode, m.tab)
	}
	if cmd == nil {
		t.Fatal("expected refresh command")
	}
}

func TestUpdateResultReturnsToInstalledAndRefreshes(t *testing.T) {
	m := newTestModel()
	m.mode = modeAction
	m.tab = TabInstalled
	msg := actionResultMsg{
		action:     "update",
		command:    "npx skills update grill-with-doc -y",
		message:    "Updated grill-with-doc",
		nextMode:   modeNormal,
		nextTab:    TabInstalled,
		hasNextTab: true,
		refresh:    m.refreshInstalledCmd(),
	}

	updated, cmd := m.Update(msg)
	m = updated.(*model)

	if m.mode != modeNormal || m.tab != TabInstalled {
		t.Fatalf("mode/tab=%v/%v, want normal/installed", m.mode, m.tab)
	}
	if cmd == nil {
		t.Fatal("expected refresh command")
	}
}

func TestUninstallResultReturnsToInstalledAndRefreshes(t *testing.T) {
	m := newTestModel()
	m.mode = modeConfirm
	m.tab = TabInstalled
	msg := actionResultMsg{
		action:     "uninstall",
		command:    "npx skills remove grill-with-doc -y",
		message:    "Removed grill-with-doc",
		nextMode:   modeNormal,
		nextTab:    TabInstalled,
		hasNextTab: true,
		refresh:    m.refreshInstalledCmd(),
	}

	updated, cmd := m.Update(msg)
	m = updated.(*model)

	if m.mode != modeNormal || m.tab != TabInstalled {
		t.Fatalf("mode/tab=%v/%v, want normal/installed", m.mode, m.tab)
	}
	if cmd == nil {
		t.Fatal("expected refresh command")
	}
}

func TestInstallScopeUsesConsistentSelectedAndMutedRows(t *testing.T) {
	m := newTestModel()
	m.mode = modeInstallScope
	m.pendingInstall = skills.Skill{Name: "grill-with-doc", Source: "ntk148v/skills"}
	m.pendingInstallGlobal = false

	out := m.installScopeView()
	if !strings.Contains(out, "❯ Project") {
		t.Fatalf("project row should be selected:\n%s", out)
	}
	if !strings.Contains(out, "  Global") {
		t.Fatalf("global row should be unselected:\n%s", out)
	}
}

func TestSourceDetailSkillsUseConsistentSelectedRows(t *testing.T) {
	m := newTestModel()
	m.mode = modeSourceDetail
	m.sourceDetail = skills.Source{Name: "ntk148v/skills", Repo: "github.com/ntk148v/skills"}
	m.sourceSkills = []skills.Skill{{Name: "grill-with-doc"}, {Name: "code-reviewer"}}
	m.sourceSkillSel = 1

	out := m.sourceDetailView()
	if !strings.Contains(out, "❯ code-reviewer") {
		t.Fatalf("selected source skill missing cursor:\n%s", out)
	}
	if !strings.Contains(out, "  grill-with-doc") {
		t.Fatalf("unselected source skill missing normal prefix:\n%s", out)
	}
}

func TestSourceDetailEnterOpensDetail(t *testing.T) {
	m := newTestModel()
	m.mode = modeSourceDetail
	m.sourceDetail = skills.Source{Name: "ntk148v/skills", Repo: "github.com/ntk148v/skills"}
	m.sourceSkills = []skills.Skill{{Name: "caveman", Source: "ntk148v/skills", ID: "ntk148v/skills/caveman"}}
	m.sourceSkillSel = 0

	cmd := m.handleSourceDetailKey(tea.KeyMsg{Type: tea.KeyEnter})
	// openDetail returns a command, so m.detail is not set yet (it's set via messages)
	if m.mode != modeDetail {
		t.Fatalf("expected modeDetail after Enter, got mode %d", m.mode)
	}
	if m.detail.Name != "caveman" {
		t.Fatalf("expected detail.Name caveman, got %q", m.detail.Name)
	}
	if cmd == nil {
		t.Fatal("expected non-nil command from openDetail")
	}
}

func TestSourceDetailEscReturnsToSourceSkillList(t *testing.T) {
	m := newTestModel()
	m.mode = modeSourceDetail
	m.sourceDetail = skills.Source{Name: "ntk148v/skills", Repo: "github.com/ntk148v/skills"}
	m.sourceSkills = []skills.Skill{{Name: "caveman", Source: "ntk148v/skills", ID: "ntk148v/skills/caveman"}}
	m.sourceSkillSel = 0

	cmd := m.handleSourceDetailKey(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected detail load command")
	}
	if m.mode != modeDetail {
		t.Fatalf("expected modeDetail after Enter, got %d", m.mode)
	}

	m.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
	if m.mode != modeSourceDetail {
		t.Fatalf("expected Esc from source-opened detail to return to modeSourceDetail, got %d", m.mode)
	}
	if m.sourceDetail.Name != "ntk148v/skills" || m.sourceSkillSel != 0 {
		t.Fatalf("source detail selection not preserved: source=%#v sel=%d", m.sourceDetail, m.sourceSkillSel)
	}
}

func TestSourceDetailAvailableSkillMetadata(t *testing.T) {
	m := newTestModel()
	m.mode = modeSourceDetail
	m.sourceDetail = skills.Source{Name: "ntk148v/skills", Repo: "github.com/ntk148v/skills"}
	m.sourceSkills = []skills.Skill{{Name: "caveman", Source: "ntk148v/skills", ID: "ntk148v/skills/caveman"}}

	m.handleSourceDetailKey(tea.KeyMsg{Type: tea.KeyEnter})
	out := m.detailView()
	for _, want := range []string{"Status: Available", "Scope: [-]", "Source: ntk148v/skills"} {
		if !strings.Contains(out, want) {
			t.Fatalf("metadata missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "Status: unknown") || strings.Contains(out, "Scope: []") {
		t.Fatalf("metadata still shows empty/unknown state:\n%s", out)
	}
}

func TestSourceDetailInstalledSkillMetadataUsesInstalledScope(t *testing.T) {
	m := newTestModel()
	m.mode = modeSourceDetail
	m.sourceDetail = skills.Source{Name: "ntk148v/skills", Repo: "github.com/ntk148v/skills"}
	m.installed = []skills.Skill{{
		Name:   "caveman",
		Source: "ntk148v/skills",
		Scope:  skills.ScopeProject,
		Status: skills.SkillStatusEnabled,
		Path:   ".agents/skills/caveman/SKILL.md",
	}}
	m.sourceSkills = []skills.Skill{{Name: "caveman", Source: "ntk148v/skills", ID: "ntk148v/skills/caveman"}}

	m.handleSourceDetailKey(tea.KeyMsg{Type: tea.KeyEnter})
	out := m.detailView()
	for _, want := range []string{"Status: Installed", "Scope: [Project]", "Source: ntk148v/skills", "Path: .agents/skills/caveman/SKILL.md"} {
		if !strings.Contains(out, want) {
			t.Fatalf("metadata missing %q:\n%s", want, out)
		}
	}
}

func TestInstalledDetailEscStillReturnsToNormalMode(t *testing.T) {
	m := newTestModel()
	m.mode = modeNormal
	m.tab = TabInstalled
	m.installed = []skills.Skill{{Name: "caveman", Source: "ntk148v/skills", Scope: skills.ScopeProject, Status: skills.SkillStatusEnabled}}

	m.openDetail(m.installed[0])
	if m.mode != modeDetail {
		t.Fatalf("expected modeDetail, got %d", m.mode)
	}
	m.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
	if m.mode != modeNormal {
		t.Fatalf("expected modeNormal after Esc from installed detail, got %d", m.mode)
	}
}

func TestInstalledRowsShowScopeBadges(t *testing.T) {
	m := newTestModel()
	m.installed = []skills.Skill{
		{Name: "global-skill", Source: "ntk148v/skills", Scope: skills.ScopeGlobal, Enabled: true},
		{Name: "project-skill", Source: "ntk148v/skills", Scope: skills.ScopeProject, Enabled: true},
	}

	out := m.renderInstalled()
	for _, want := range []string{"G", "global-skill", "P", "project-skill"} {
		if !strings.Contains(out, want) {
			t.Fatalf("installed view missing %q:\n%s", want, out)
		}
	}
}

func TestInstallCommandProjectHasNoProjectFlag(t *testing.T) {
	item := skills.Skill{Name: "grill-with-doc", Source: "ntk148v/skills"}
	got := installCommand(item, false)
	if got != "npx skills add ntk148v/skills --skill grill-with-doc -y" {
		t.Fatalf("command=%q", got)
	}
}

func TestInstallCommandGlobalHasGlobalFlag(t *testing.T) {
	item := skills.Skill{Name: "grill-with-doc", Source: "ntk148v/skills"}
	got := installCommand(item, true)
	if got != "npx skills add ntk148v/skills --skill grill-with-doc -g -y" {
		t.Fatalf("command=%q", got)
	}
}

func TestLogsEnterShowsLogDetail(t *testing.T) {
	m := newTestModel()
	m.tab = TabLogs
	m.focus = focusList
	m.logsSel = 0
	m.logs = []skills.LogEntry{{
		At:      time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC),
		Action:  "install",
		Command: "npx skills add ntk148v/skills --skill grill-with-doc -g -y",
		Output:  "installed",
	}}

	cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatalf("log detail should not launch async command")
	}
	out := m.View()
	for _, want := range []string{"Log Detail", "install", "npx skills add", "installed"} {
		if !strings.Contains(out, want) {
			t.Fatalf("log detail missing %q:\n%s", want, out)
		}
	}
}

func TestActionOKMessage(t *testing.T) {
	cases := []struct {
		action string
		skill  skills.Skill
		want   string
	}{
		{"install", skills.Skill{Name: "foo"}, "Installed foo"},
		{"update", skills.Skill{Name: "foo"}, "Updated foo"},
		{"uninstall", skills.Skill{Name: "foo"}, "Removed foo"},
		{"unknown", skills.Skill{Name: "foo"}, "unknown ok"},
	}
	for _, c := range cases {
		got := actionOKMessage(c.action, c.skill)
		if got != c.want {
			t.Fatalf("actionOKMessage(%q, %q) = %q, want %q", c.action, c.skill.Name, got, c.want)
		}
	}
}

func TestInstallCommandEmptyScope(t *testing.T) {
	got := installCommand(skills.Skill{Name: "x", Source: "s"}, false)
	if strings.Contains(got, "-g") {
		t.Fatalf("project install should not have -g: %q", got)
	}
	got2 := installCommand(skills.Skill{Name: "x", Source: "s"}, true)
	if !strings.Contains(got2, "-g") {
		t.Fatalf("global install should have -g: %q", got2)
	}
}

func TestRemoveCommand(t *testing.T) {
	got := removeCommand(skills.Skill{Name: "x", Scope: skills.ScopeGlobal})
	if !strings.Contains(got, "-g") {
		t.Fatalf("global remove should have -g: %q", got)
	}
	got2 := removeCommand(skills.Skill{Name: "x", Scope: skills.ScopeProject})
	if !strings.Contains(got2, "-p") {
		t.Fatalf("project remove should have -p: %q", got2)
	}
	if strings.Contains(got2, "-g") {
		t.Fatalf("project remove should not have -g: %q", got2)
	}
}

func TestEmptyDash(t *testing.T) {
	if emptyDash("") != "-" {
		t.Fatalf("emptyDash('') should return '-'")
	}
	if emptyDash("foo") != "foo" {
		t.Fatalf("emptyDash('foo') should return 'foo'")
	}
}

func TestRowStyleSelectedMatches(t *testing.T) {
	s := newStyles()
	if rowStyle(s, true).Render("x") != s.rowSelected.Render("x") {
		t.Fatal("selected row style mismatch")
	}
	if rowStyle(s, false).Render("x") != s.rowMuted.Render("x") {
		t.Fatal("unselected row style mismatch")
	}
}

// Verify scope badge content matches scope field.
func TestScopeBadgeContent(t *testing.T) {
	s := newStyles()
	if !strings.Contains(scopeBadge(s, skills.ScopeGlobal), "G") {
		t.Fatal("global badge missing G")
	}
	if !strings.Contains(scopeBadge(s, skills.ScopeProject), "P") {
		t.Fatal("project badge missing P")
	}
	if !strings.Contains(scopeBadge(s, skills.ScopeUser), "G") {
		t.Fatal("user badge missing G")
	}
	if !strings.Contains(scopeBadge(s, ""), "-") {
		t.Fatal("empty scope badge missing -")
	}
}
