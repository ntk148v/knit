package skills

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// fakeRunner returns the same output for every run call.
type fakeRunner struct{ out string }

func (f fakeRunner) Run(context.Context, string, ...string) ([]byte, error) {
	return []byte(f.out), nil
}

// recordingRunner captures every command invocation for assertion.
type recordingRunner struct {
	out   []byte
	calls [][]string
}

func (r *recordingRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	r.calls = append(r.calls, append([]string{name}, args...))
	return r.out, nil
}

// sequenceRunner returns outputs sequentially for multiple Run calls.
type sequenceRunner struct {
	outs [][]byte
	idx  int
}

func (r *sequenceRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	if r.idx >= len(r.outs) {
		return nil, nil
	}
	out := r.outs[r.idx]
	r.idx++
	return out, nil
}

// ─── Existing tests ──────────────────────────────────────────────────

func TestListInstalledJSON(t *testing.T) {
	runner := fakeRunner{
		out: `[{"name":"caveman","path":"/tmp/skills/caveman","scope":"global","agents":["codex"]}]`,
	}
	client := NewNpxClientWithRunner(runner)
	items, err := client.ListInstalled(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Name != "caveman" || items[0].Path != "/tmp/skills/caveman" {
		t.Fatalf("unexpected: %#v", items)
	}
}

func TestApplySkillHealthAddsMinimalWarnings(t *testing.T) {
	item := applySkillHealth(Skill{Name: "demo", Enabled: true})
	for _, want := range []string{"missing description", "no agents reported"} {
		if !hasWarning(item, want) {
			t.Fatalf("missing warning %q in %#v", want, item.Warnings)
		}
	}
}

func TestApplySkillHealthRemovesStaleDescriptionWarning(t *testing.T) {
	item := applySkillHealth(Skill{Name: "demo", Description: "exists", Warnings: []string{"missing description", "no agents reported"}, Agents: []string{"codex"}})
	for _, stale := range []string{"missing description", "no agents reported"} {
		if hasWarning(item, stale) {
			t.Fatalf("stale warning %q still present in %#v", stale, item.Warnings)
		}
	}
}

func hasWarning(s Skill, want string) bool {
	for _, w := range s.Warnings {
		if w == want {
			return true
		}
	}
	return false
}

func TestFindCLIOutput(t *testing.T) {
	runner := fakeRunner{
		out: "Install with npx skills add\n\nvercel-labs/agent-skills@frontend-design 1.2K installs\n└ https://skills.sh/vercel-labs/agent-skills/frontend-design\n\nowner/repo@React Native\n└ https://skills.sh/owner/repo/react-native\n",
	}
	items, err := NewNpxClientWithRunner(runner).Find(context.Background(), "frontend")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("got %d items", len(items))
	}
	if items[0].Source != "vercel-labs/agent-skills" || items[0].Name != "frontend-design" || items[0].Installs != 1200 || items[0].ID != "vercel-labs/agent-skills/frontend-design" {
		t.Fatalf("unexpected first: %#v", items[0])
	}
	if items[1].Name != "React Native" || items[1].ID != "owner/repo/react-native" {
		t.Fatalf("unexpected second: %#v", items[1])
	}
}

func TestFindShortQuerySkipsRequest(t *testing.T) {
	items, err := NewNpxClientWithRunner(fakeRunner{out: ""}).Find(context.Background(), "a")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("expected no results, got %d", len(items))
	}
}

func TestSkillDetailAPI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/skills/vercel-labs/agent-skills/frontend-design" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":     "vercel-labs/agent-skills/frontend-design",
			"source": "vercel-labs/agent-skills",
			"slug":   "frontend-design",
			"files": []map[string]any{
				{"path": "SKILL.md", "contents": "---\nname: frontend-design\nsource: vercel-labs/agent-skills\ndescription: Pretty UI\n---\n# Frontend\nHello"},
			},
		})
	}))
	defer server.Close()
	old := os.Getenv("SKILLS_API_URL")
	os.Setenv("SKILLS_API_URL", server.URL)
	defer os.Setenv("SKILLS_API_URL", old)
	item, err := NewNpxClient().SkillDetail(context.Background(), Skill{
		Name: "frontend-design", Source: "vercel-labs/agent-skills",
		ID: "vercel-labs/agent-skills/frontend-design",
	})
	if err != nil {
		t.Fatal(err)
	}
	if item.Name != "frontend-design" || item.Description != "Pretty UI" || item.Preview == "" {
		t.Fatalf("unexpected: %#v", item)
	}
	if item.Source != "vercel-labs/agent-skills" {
		t.Fatalf("source=%q", item.Source)
	}
}

func TestMergeSkillMarkdownPreservesSourceFrontmatter(t *testing.T) {
	got := mergeSkillMarkdown(Skill{Name: "grill-with-doc", Source: "grill-with-doc"}, "---\nname: grill-with-doc\nsource: ntk148v/skills\ndescription: Grill docs\n---\n# Grill\n")
	if got.Source != "ntk148v/skills" {
		t.Fatalf("Source=%q, want ntk148v/skills", got.Source)
	}
	if got.Name != "grill-with-doc" || got.Description != "Grill docs" {
		t.Fatalf("unexpected metadata: %#v", got)
	}
}

func TestSkillDetailFromDisk(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "skill")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	md := "---\nname: demo\ndescription: Demo skill\n---\n\n# Demo\nHello\n"
	if err := os.WriteFile(filepath.Join(path, "SKILL.md"), []byte(md), 0o644); err != nil {
		t.Fatal(err)
	}
	item, err := NewNpxClientWithRunner(fakeRunner{out: ""}).SkillDetail(context.Background(), Skill{Path: path})
	if err != nil {
		t.Fatal(err)
	}
	if item.Name != "demo" || item.Description != "Demo skill" || item.Preview == "" {
		t.Fatalf("unexpected: %#v", item)
	}
}

// ─── Task 4: CLI command arg tests ───────────────────────────────────

func TestMutatingCommandsUseYes(t *testing.T) {
	r := &recordingRunner{out: []byte("[]")}
	c := NewNpxClientWithRunner(r)
	ctx := context.Background()

	_ = c.InstallSkill(ctx, Skill{Name: "frontend-design", Source: "vercel-labs/agent-skills"}, true)
	_ = c.UpdateSkill(ctx, Skill{Name: "frontend-design"})
	_ = c.UninstallSkill(ctx, Skill{Name: "frontend-design"})

	want := [][]string{
		{"npx", "skills", "add", "vercel-labs/agent-skills", "--skill", "frontend-design", "-y", "-g"},
		{"npx", "skills", "update", "frontend-design", "-y"},
		{"npx", "skills", "remove", "frontend-design", "-y"},
	}
	if !reflect.DeepEqual(r.calls, want) {
		t.Fatalf("calls mismatch\nwant %#v\n got %#v", want, r.calls)
	}
}

func TestMutatingCommandsScopeFlags(t *testing.T) {
	r := &recordingRunner{out: []byte("[]")}
	c := NewNpxClientWithRunner(r)
	ctx := context.Background()

	// Global scope for update/remove.
	_ = c.UpdateSkill(ctx, Skill{Name: "g-skill", Scope: ScopeGlobal})
	_ = c.UninstallSkill(ctx, Skill{Name: "g-skill", Scope: ScopeGlobal})

	// Project scope for update.
	_ = c.UpdateSkill(ctx, Skill{Name: "p-skill", Scope: ScopeProject})

	want := [][]string{
		{"npx", "skills", "update", "g-skill", "-y", "-g"},
		{"npx", "skills", "remove", "g-skill", "-y", "-g"},
		{"npx", "skills", "update", "p-skill", "-y", "-p"},
	}
	if !reflect.DeepEqual(r.calls, want) {
		t.Fatalf("calls mismatch\nwant %#v\n got %#v", want, r.calls)
	}
}

func TestAddSourceValidatesWithList(t *testing.T) {
	r := &recordingRunner{out: []byte("some output")}
	c := NewNpxClientWithRunner(r)

	if err := c.AddSource(context.Background(), "vercel-labs/agent-skills"); err != nil {
		t.Fatal(err)
	}
	want := [][]string{{"npx", "skills", "add", "vercel-labs/agent-skills", "--list"}}
	if !reflect.DeepEqual(r.calls, want) {
		t.Fatalf("calls mismatch\nwant %#v\n got %#v", want, r.calls)
	}
}

// ─── Task 3: Enrichment ────────────────────────────────────────────

func TestEnrichInstalledSourceFromLock(t *testing.T) {
	dir := t.TempDir()
	lockDir := filepath.Join(dir, "node_modules", ".skills")
	if err := os.MkdirAll(lockDir, 0o755); err != nil {
		t.Fatal(err)
	}
	lockFile := filepath.Join(lockDir, "skills-lock.json")
	lockContent := `{"skills":[{"name":"caveman","source":"ntk148v/skills","repo":"github.com/ntk148v/skills"}]}`
	if err := os.WriteFile(lockFile, []byte(lockContent), 0o644); err != nil {
		t.Fatal(err)
	}

	skillDir := filepath.Join(dir, "node_modules", ".skills", "caveman")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# test"), 0o644); err != nil {
		t.Fatal(err)
	}

	c := &NpxClient{}
	got := c.enrichInstalled([]Skill{
		{Name: "caveman", Path: skillDir, Scope: ScopeProject},
	})
	if len(got) != 1 {
		t.Fatalf("got %d skills", len(got))
	}
	if got[0].Source != "github.com/ntk148v/skills" {
		t.Fatalf("source not enriched: %q", got[0].Source)
	}
	if !hasWarning(got[0], "missing description") || !hasWarning(got[0], "no agents reported") {
		t.Fatalf("missing health warnings: %v", got[0].Warnings)
	}
}

func TestEnrichInstalledBrokenSkill(t *testing.T) {
	dir := t.TempDir()
	badDir := filepath.Join(dir, "broken-skill")
	if err := os.MkdirAll(badDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// No SKILL.md - broken
	c := &NpxClient{}
	got := c.enrichInstalled([]Skill{
		{Name: "broken", Path: badDir, Scope: ScopeGlobal},
	})
	if len(got) != 1 || len(got[0].Warnings) == 0 || got[0].Warnings[0] != "broken (SKILL.md missing)" {
		t.Fatalf("expected broken warning, got %#v", got[0].Warnings)
	}
}

// ─── Task 7: Installed skills merge ──────────────────────────────────

func TestListInstalledMergesProjectAndGlobal(t *testing.T) {
	r := &sequenceRunner{outs: [][]byte{
		[]byte(`[{"name":"project-skill","path":"/repo/.agents/skills/project-skill","scope":"project","agents":["codex"]}]`),
		[]byte(`[{"name":"global-skill","path":"/home/me/.agents/skills/global-skill","scope":"global","agents":["codex"]}]`),
	}}
	c := NewNpxClientWithRunner(r)
	got, err := c.ListInstalled(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 skills, got %d: %#v", len(got), got)
	}
	names := map[string]bool{}
	for _, s := range got {
		names[s.Name] = true
	}
	if !names["project-skill"] || !names["global-skill"] {
		t.Fatalf("missing skills: %#v", names)
	}
}

func TestListInstalledDeduplicates(t *testing.T) {
	r := &sequenceRunner{outs: [][]byte{
		[]byte(`[{"name":"same","path":"/p","scope":"project","agents":["codex"]}]`),
		[]byte(`[{"name":"same","path":"/p","scope":"project","agents":["codex"]}]`),
	}}
	c := NewNpxClientWithRunner(r)
	got, err := c.ListInstalled(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 after dedup, got %d", len(got))
	}
}

// ─── Task 6: Detail cache ────────────────────────────────────────────

func TestSkillDetailCachesResult(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":     "src/name",
			"source": "src",
			"slug":   "name",
			"files":  []map[string]any{{"path": "SKILL.md", "contents": "---\nname: cached\n---\nContent"}},
		})
	}))
	defer server.Close()
	old := os.Getenv("SKILLS_API_URL")
	os.Setenv("SKILLS_API_URL", server.URL)
	defer os.Setenv("SKILLS_API_URL", old)

	c := NewNpxClient()
	ctx := context.Background()
	sk := Skill{Name: "name", Source: "src", ID: "src/name"}

	// First call hits API.
	_, _ = c.SkillDetail(ctx, sk)
	// Second call should use cache.
	item, err := c.SkillDetail(ctx, sk)
	if err != nil {
		t.Fatal(err)
	}
	if callCount != 1 {
		t.Fatalf("expected 1 API call, got %d", callCount)
	}
	if item.Name != "cached" {
		t.Fatalf("expected cached name 'cached', got %q", item.Name)
	}
}

func TestParseListAvailableCurrentGroupedOutput(t *testing.T) {
	// Real npx skills add --list output format (with │ prefixes).
	out := strings.Join([]string{
		"◇  Available Skills",
		"Agent Skills Collection",
		"│",
		"│    grill-with-doc",
		"│",
		"│      Interview the user using documentation context.",
		"│",
		"│    code-reviewer",
		"│",
		"│      Review code for correctness and maintainability.",
		"│",
		"General",
		"│",
		"│    repo-analyzer",
		"│",
		"└  Use --skill <name> to install specific skills",
	}, "\n")

	got := parseListAvailable(out, "ntk148v/skills")
	want := []Skill{
		{Name: "grill-with-doc", Source: "ntk148v/skills", ID: "ntk148v/skills/grill-with-doc"},
		{Name: "code-reviewer", Source: "ntk148v/skills", ID: "ntk148v/skills/code-reviewer"},
		{Name: "repo-analyzer", Source: "ntk148v/skills", ID: "ntk148v/skills/repo-analyzer"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parsed skills mismatch\nwant %#v\n got %#v", want, got)
	}
}

func TestLoadLockFileMapShape(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "skills-lock.json")
	data := `{
		"version": 1,
		"skills": {
			"caveman": {
				"source": "ntk148v/skills",
				"sourceType": "github",
				"skillPath": "skills/caveman/SKILL.md",
				"computedHash": "abc"
			}
		}
	}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}

	lock, err := LoadLockFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if lock.Version != 1 || len(lock.Skills) != 1 {
		t.Fatalf("bad lock: %#v", lock)
	}
	got := lock.Skills[0]
	if got.Name != "caveman" || got.Source != "ntk148v/skills" || got.SkillPath != "skills/caveman/SKILL.md" {
		t.Fatalf("bad skill: %#v", got)
	}
}

func TestLoadLockFileArrayShape(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "skills-lock.json")
	data := `{"version":1,"skills":[{"name":"handoff","source":"ntk148v/skills","repo":"github.com/ntk148v/skills"}]}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}

	lock, err := LoadLockFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(lock.Skills) != 1 || lock.Skills[0].Name != "handoff" || lock.Skills[0].Source != "ntk148v/skills" {
		t.Fatalf("bad lock: %#v", lock)
	}
}

func TestSyncFromLockInstallsEachSkillWithNpxAdd(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "skills-lock.json")
	data := `{"version":1,"skills":{"caveman":{"source":"ntk148v/skills"},"handoff":{"source":"ntk148v/skills"}}}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	r := &recordingRunner{}
	c := NewNpxClientWithRunner(r)

	if err := c.SyncFromLock(context.Background(), path, false); err != nil {
		t.Fatal(err)
	}
	want := [][]string{
		{"npx", "skills", "add", "ntk148v/skills", "--skill", "caveman", "-y"},
		{"npx", "skills", "add", "ntk148v/skills", "--skill", "handoff", "-y"},
	}
	if !reflect.DeepEqual(r.calls, want) {
		t.Fatalf("calls mismatch\nwant %#v\n got %#v", want, r.calls)
	}
}

func TestSyncFromLockGlobalAddsGlobalFlag(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "skills-lock.json")
	if err := os.WriteFile(path, []byte(`{"skills":{"uv":{"source":"owner/repo"}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	r := &recordingRunner{}
	c := NewNpxClientWithRunner(r)

	if err := c.SyncFromLock(context.Background(), path, true); err != nil {
		t.Fatal(err)
	}
	got := strings.Join(r.calls[0], " ")
	if !strings.Contains(got, " -g") {
		t.Fatalf("global sync missing -g: %#v", r.calls[0])
	}
}

func TestInstallSkillScopeFlags(t *testing.T) {
	r := &recordingRunner{out: []byte("[]")}
	c := NewNpxClientWithRunner(r)
	ctx := context.Background()

	_ = c.InstallSkill(ctx, Skill{Name: "grill-with-doc", Source: "ntk148v/skills"}, false)
	_ = c.InstallSkill(ctx, Skill{Name: "grill-with-doc", Source: "ntk148v/skills"}, true)

	want := [][]string{
		{"npx", "skills", "add", "ntk148v/skills", "--skill", "grill-with-doc", "-y"},
		{"npx", "skills", "add", "ntk148v/skills", "--skill", "grill-with-doc", "-y", "-g"},
	}
	if !reflect.DeepEqual(r.calls, want) {
		t.Fatalf("calls mismatch\nwant %#v\n got %#v", want, r.calls)
	}
}

func TestMutationsUseNpxSkillsSoCliUpdatesLocks(t *testing.T) {
	r := &recordingRunner{}
	c := NewNpxClientWithRunner(r)
	ctx := context.Background()

	if err := c.InstallSkill(ctx, Skill{Name: "caveman", Source: "ntk148v/skills"}, false); err != nil {
		t.Fatal(err)
	}
	if err := c.InstallSkill(ctx, Skill{Name: "uv", Source: "owner/repo"}, true); err != nil {
		t.Fatal(err)
	}
	if err := c.UpdateSkill(ctx, Skill{Name: "caveman", Scope: ScopeProject}); err != nil {
		t.Fatal(err)
	}
	if err := c.UpdateSkill(ctx, Skill{Name: "uv", Scope: ScopeGlobal}); err != nil {
		t.Fatal(err)
	}
	if err := c.UninstallSkill(ctx, Skill{Name: "caveman", Scope: ScopeProject}); err != nil {
		t.Fatal(err)
	}
	if err := c.UninstallSkill(ctx, Skill{Name: "uv", Scope: ScopeGlobal}); err != nil {
		t.Fatal(err)
	}

	want := [][]string{
		{"npx", "skills", "add", "ntk148v/skills", "--skill", "caveman", "-y"},
		{"npx", "skills", "add", "owner/repo", "--skill", "uv", "-y", "-g"},
		{"npx", "skills", "update", "caveman", "-y", "-p"},
		{"npx", "skills", "update", "uv", "-y", "-g"},
		{"npx", "skills", "remove", "caveman", "-y", "-p"},
		{"npx", "skills", "remove", "uv", "-y", "-g"},
	}
	if !reflect.DeepEqual(r.calls, want) {
		t.Fatalf("calls mismatch\nwant %#v\n got %#v", want, r.calls)
	}
}

func TestParseListAvailableFiltersBoxDrawingAndHeaders(t *testing.T) {
	// Full realistic output: boxes, prompts, blank lines, section headers, real names.
	out := strings.Join([]string{
		"◇  Available Skills",
		"Agent Skills Collection",
		"│",
		"│    caveman",
		"│",
		"│      Ultra-compressed communication mode.",
		"│",
		"│    handoff",
		"│",
		"│      Compact the current conversation.",
		"│",
		"General",
		"│",
		"│    code-reviewer",
		"│",
		"│      Review code.",
		"│",
		"│    conventional-commits",
		"│",
		"│      Use when creating git commits.",
		"│",
		"│    repo-analyzer",
		"│",
		"└  Use --skill <name> to install specific skills",
	}, "\n")

	got := parseListAvailable(out, "ntk148v/skills")
	want := []Skill{
		{Name: "caveman", Source: "ntk148v/skills", ID: "ntk148v/skills/caveman"},
		{Name: "handoff", Source: "ntk148v/skills", ID: "ntk148v/skills/handoff"},
		{Name: "code-reviewer", Source: "ntk148v/skills", ID: "ntk148v/skills/code-reviewer"},
		{Name: "conventional-commits", Source: "ntk148v/skills", ID: "ntk148v/skills/conventional-commits"},
		{Name: "repo-analyzer", Source: "ntk148v/skills", ID: "ntk148v/skills/repo-analyzer"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parsed skills mismatch\nwant %#v\n got %#v", want, got)
	}
}

func TestParseListAvailableDeduplicates(t *testing.T) {
	out := "│    caveman\n│    caveman\n│    handoff\n"
	got := parseListAvailable(out, "ntk148v/skills")
	if len(got) != 2 || got[0].Name != "caveman" || got[1].Name != "handoff" {
		t.Fatalf("expected 2 deduplicated skills, got %#v", got)
	}
}

func TestParseListAvailableEmptyWhenNoSkillNames(t *testing.T) {
	// Lines with box-drawing chars only, no real skill names.
	out := strings.Join([]string{
		"┌─────────────────┐",
		"│                                 │",
		"└───────────────────────┘",
		"◇  Some message",
		"└  Use --skill <name> to install",
	}, "\n")
	got := parseListAvailable(out, "ntk148v/skills")
	if len(got) != 0 {
		t.Fatalf("expected 0 skills from decorative-only output, got %d: %#v", len(got), got)
	}
}

func TestRemoveSourceRemovesInstalledSkillsFromThatSource(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	c := NewNpxClientWithRunner(&recordingRunner{})

	// Add a source first so the config has it.
	if err := c.AddSource(context.Background(), "ntk148v/skills"); err != nil {
		t.Fatal(err)
	}
	// Remove it.
	if err := c.RemoveSource(context.Background(), "ntk148v/skills"); err != nil {
		t.Fatal(err)
	}
	// Source must not appear in ListSources.
	got, err := c.ListSources(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range got {
		if s.Name == "ntk148v/skills" {
			t.Fatalf("source still listed after remove: %#v", got)
		}
	}
	// Config should have the source in Removed list.
	cfg, err := readKnitConfig()
	if err != nil {
		t.Fatal(err)
	}
	if !contains(cfg.Removed, "ntk148v/skills") {
		t.Fatalf("source not in Removed list: %#v", cfg.Removed)
	}
}

// ─── Hash helpers ────────────────────────────────────────────────────

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestSourceGitURLFromOwnerRepo(t *testing.T) {
	got, ok := sourceGitURL("ntk148v/skills")
	if !ok || got != "https://github.com/ntk148v/skills.git" {
		t.Fatalf("got %q/%v", got, ok)
	}
}

func TestSourceGitURLFromHTTPS(t *testing.T) {
	got, ok := sourceGitURL("https://github.com/ntk148v/skills")
	if !ok || got != "https://github.com/ntk148v/skills.git" {
		t.Fatalf("got %q/%v", got, ok)
	}
}

func TestSourceGitURLRejectsMarketplaceJSON(t *testing.T) {
	got, ok := sourceGitURL("https://example.com/marketplace.json")
	if ok || got != "" {
		t.Fatalf("got %q/%v", got, ok)
	}
}

func TestSourceGitURLRejectsInsecureHTTPByDefault(t *testing.T) {
	// Ensure the env is not set for this test.
	os.Unsetenv("KNIT_ALLOW_INSECURE_SOURCES")
	got, ok := sourceGitURL("http://github.com/ntk148v/skills.git")
	if ok || got != "" {
		t.Fatalf("insecure http:// should be rejected, got %q/%v", got, ok)
	}
}

func TestSourceGitURLRejectsInsecureHTTPPlain(t *testing.T) {
	os.Unsetenv("KNIT_ALLOW_INSECURE_SOURCES")
	got, ok := sourceGitURL("http://example.com/repo")
	if ok || got != "" {
		t.Fatalf("insecure http:// should be rejected, got %q/%v", got, ok)
	}
}

func TestSkillDetailLoadsPreviewFromCachedSource(t *testing.T) {
	t.Setenv("KNIT_SOURCE_CACHE_DIR", t.TempDir())
	cache := sourceCacheDir("ntk148v/skills")
	if err := os.MkdirAll(filepath.Join(cache, "conventional-commits"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(cache, "conventional-commits", "SKILL.md"), "---\nname: conventional-commits\ndescription: Commit messages\n---\n\n# Conventional Commits\n")

	c := NewNpxClientWithRunner(&recordingRunner{})
	got, err := c.SkillDetail(context.Background(), Skill{Name: "conventional-commits", Source: "ntk148v/skills"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Description != "Commit messages" || !strings.Contains(got.Preview, "Conventional Commits") {
		t.Fatalf("bad detail: %#v", got)
	}
}

func TestAddSourcePersistsToKnitConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	r := &recordingRunner{out: []byte("caveman\nreviewer\n")}
	c := NewNpxClientWithRunner(r)

	if err := c.AddSource(context.Background(), "ntk148v/skills"); err != nil {
		t.Fatal(err)
	}
	got, err := c.ListSources(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, s := range got {
		if s.Name == "ntk148v/skills" && s.Repo == "ntk148v/skills" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("source not found in list: %#v", got)
	}
	want := [][]string{{"npx", "skills", "add", "ntk148v/skills", "--list"}}
	if !reflect.DeepEqual(r.calls, want) {
		t.Fatalf("calls mismatch\nwant %#v\n got %#v", want, r.calls)
	}
}

func TestAddSourceDeduplicatesConfigSources(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	c := NewNpxClientWithRunner(&recordingRunner{out: []byte("caveman\n")})

	if err := c.AddSource(context.Background(), "ntk148v/skills"); err != nil {
		t.Fatal(err)
	}
	if err := c.AddSource(context.Background(), "ntk148v/skills"); err != nil {
		t.Fatal(err)
	}
	got, err := c.ListSources(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	// Accept additional lock-based sources; just check for no duplicate name.
	count := 0
	for _, s := range got {
		if s.Name == "ntk148v/skills" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("duplicate source persisted: %#v", got)
	}
}

func TestRemoveSourceDeletesConfigSource(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	c := NewNpxClientWithRunner(&recordingRunner{out: []byte("caveman\n")})

	if err := c.AddSource(context.Background(), "ntk148v/skills"); err != nil {
		t.Fatal(err)
	}
	if err := c.RemoveSource(context.Background(), "ntk148v/skills"); err != nil {
		t.Fatal(err)
	}
	got, err := c.ListSources(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range got {
		if s.Name == "ntk148v/skills" {
			t.Fatalf("source still listed after remove: %#v", got)
		}
	}
}

func TestSafeJoinUnderRejectsTraversal(t *testing.T) {
	tests := []struct {
		name    string
		root    string
		parts   []string
		wantErr bool
	}{
		{"normal name", "/tmp/cache", []string{"myskill", "SKILL.md"}, false},
		{"traversal with ..", "/tmp/cache", []string{"..", "etc", "passwd"}, true},
		{"double traversal", "/tmp/cache", []string{"..", "..", "etc"}, true},
		{"abs path inside", "/tmp/cache", []string{"/tmp/cache/../cache/foo"}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := safeJoinUnder(tc.root, tc.parts...)
			if tc.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
