package skills

import (
	"bufio"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

type Scope string

const (
	ScopeProject Scope = "project"
	ScopeGlobal  Scope = "global"
	ScopeUser    Scope = "user"
)

type SkillStatus string

const (
	SkillStatusEnabled  SkillStatus = "enabled"
	SkillStatusDisabled SkillStatus = "disabled"
)

type Skill struct {
	Name        string      `json:"name"`
	Source      string      `json:"source,omitempty"`
	Scope       Scope       `json:"scope,omitempty"`
	Path        string      `json:"path,omitempty"`
	Folder      string      `json:"folder,omitempty"`
	Agents      []string    `json:"agents,omitempty"`
	Enabled     bool        `json:"enabled"`
	Favorite    bool        `json:"favorite"`
	Description string      `json:"description,omitempty"`
	Preview     string      `json:"preview,omitempty"`
	ID          string      `json:"id,omitempty"`
	Warnings    []string    `json:"warnings,omitempty"`
	Installs    int         `json:"installs,omitempty"`
	Status      SkillStatus `json:"status,omitempty"`
}

type Source struct {
	Name       string    `json:"name"`
	Repo       string    `json:"repo"`
	Available  int       `json:"available,omitempty"`
	Installed  int       `json:"installed,omitempty"`
	Updated    time.Time `json:"updated,omitempty"`
	RawUpdated string    `json:"rawUpdated,omitempty"`
}

type LogEntry struct {
	At      time.Time `json:"at"`
	Action  string    `json:"action"`
	Command string    `json:"command"`
	Output  string    `json:"output,omitempty"`
	Err     string    `json:"err,omitempty"`
}

type LockFile struct {
	Version int         `json:"version"`
	Skills  []LockSkill `json:"skills"`
}

type LockSkill struct {
	Name         string `json:"name"`
	Source       string `json:"source"`
	Repo         string `json:"repo,omitempty"`
	SourceType   string `json:"sourceType,omitempty"`
	SkillPath    string `json:"skillPath,omitempty"`
	ComputedHash string `json:"computedHash,omitempty"`
}

type Client interface {
	ListInstalled(context.Context) ([]Skill, error)
	Find(context.Context, string) ([]Skill, error)
	ListSources(context.Context) ([]Source, error)
	AddSource(context.Context, string) error
	UpdateSource(context.Context, string) error
	RemoveSource(context.Context, string) error
	InstallSkill(context.Context, Skill, bool) error
	UpdateSkill(context.Context, Skill) error
	UninstallSkill(context.Context, Skill) error
	SkillDetail(context.Context, Skill) (Skill, error)
	ListSourceSkills(context.Context, string) ([]Skill, error)
	PruneLocks(context.Context) error
	SyncFromLock(context.Context, string, bool) error
}

type Runner interface {
	Run(context.Context, string, ...string) ([]byte, error)
}

const (
	defaultCommandTimeout = 2 * time.Minute
	defaultHTTPTimeout    = 15 * time.Second
)

var httpClient = &http.Client{Timeout: defaultHTTPTimeout}

func withDefaultTimeout(ctx context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, d)
}

type execRunner struct{}

func (execRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	ctx, cancel := withDefaultTimeout(ctx, defaultCommandTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = append(os.Environ(), "NO_COLOR=1")
	out, err := cmd.CombinedOutput()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return out, fmt.Errorf("%s %s: timed out after %v", name, strings.Join(args, " "), defaultCommandTimeout)
	}
	return out, err
}

type NpxClient struct {
	mu          sync.RWMutex
	runner      Runner
	detailCache map[string]Skill
}

func (c *NpxClient) getCachedDetail(key string) (Skill, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.detailCache[key]
	return v, ok
}

func (c *NpxClient) setCachedDetail(key string, v Skill) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.detailCache[key] = v
}

func NewNpxClient() *NpxClient {
	return &NpxClient{runner: execRunner{}, detailCache: map[string]Skill{}}
}
func NewNpxClientWithRunner(r Runner) *NpxClient {
	return &NpxClient{runner: r, detailCache: map[string]Skill{}}
}

type knitConfig struct {
	Sources []string `json:"sources"`
	Removed []string `json:"removed,omitempty"`
}

func knitConfigPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "knit", "knit.json"), nil
}

func readKnitConfig() (knitConfig, error) {
	path, err := knitConfigPath()
	if err != nil {
		return knitConfig{}, err
	}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return knitConfig{}, nil
	}
	if err != nil {
		return knitConfig{}, err
	}
	var cfg knitConfig
	if err := json.Unmarshal(b, &cfg); err != nil {
		// Ignore unparseable config so a format change or manual edit
		// does not brick source listing.
		return knitConfig{}, nil
	}
	return cfg, nil
}

func writeKnitConfig(cfg knitConfig) error {
	path, err := knitConfigPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o644)
}

func addConfigSource(source string) error {
	source = strings.TrimSpace(source)
	cfg, err := readKnitConfig()
	if err != nil {
		return err
	}
	for _, s := range cfg.Sources {
		if s == source {
			return nil
		}
	}
	cfg.Sources = append(cfg.Sources, source)
	sort.Strings(cfg.Sources)
	// Clear the removed flag so re-added sources show up immediately.
	cfg.Removed = removeString(cfg.Removed, source)
	return writeKnitConfig(cfg)
}

func removeConfigSource(source string) error {
	cfg, err := readKnitConfig()
	if err != nil {
		return err
	}
	out := cfg.Sources[:0]
	for _, s := range cfg.Sources {
		if s != source {
			out = append(out, s)
		}
	}
	cfg.Sources = out
	// Track as removed so ListSources can filter lock-only entries.
	if !contains(cfg.Removed, source) {
		cfg.Removed = append(cfg.Removed, source)
	}
	return writeKnitConfig(cfg)
}

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

func removeString(slice []string, s string) []string {
	out := slice[:0]
	for _, v := range slice {
		if v != s {
			out = append(out, v)
		}
	}
	return out
}

// ─── ListInstalled ───────────────────────────────────────────────────

// ListInstalled returns skills from both project and global scopes,
// deduplicated by the composite key scope+path+name.
//
// ponytail: two sequential npx calls. Parallelise only if latency is
// measurably too high for a startup list.
func (c *NpxClient) ListInstalled(ctx context.Context) ([]Skill, error) {
	all := []Skill{}
	seen := map[string]bool{}

	appendFrom := func(args ...string) {
		out, err := c.run(ctx, "npx", args...)
		if err != nil {
			return
		}
		for _, item := range parseListJSON(out) {
			key := string(item.Scope) + "|" + item.Path + "|" + item.Name
			if !seen[key] {
				all = append(all, item)
				seen[key] = true
			}
		}
	}

	appendFrom("skills", "list", "--json")
	appendFrom("skills", "list", "-g", "--json")
	return c.enrichInstalled(all), nil
}

// enrichInstalled enriches Skills from npx skills list with real source repo,
// agents, and health info from lock files.
//
// ponytail: re-reads lock files on every call. Cache lock artifacts only if
// ListInstalled latency is measurably too high.
func (c *NpxClient) enrichInstalled(skills []Skill) []Skill {
	projectSrc := findProjectLock(skills)
	globalSrc := c.readLockSources(findGlobalLock())
	out := make([]Skill, 0, len(skills))
	for _, s := range skills {
		repo := ""
		if s.Scope == ScopeProject {
			repo = projectSrc[s.Name]
		} else {
			repo = globalSrc[s.Name]
		}
		if repo != "" {
			s.Source = repo
		}
		if s.Path != "" {
			if fi, err := os.Stat(filepath.Join(s.Path, "SKILL.md")); err != nil || !fi.Mode().IsRegular() {
				s.Warnings = appendWarning(s.Warnings, "broken (SKILL.md missing)")
			}
		}
		out = append(out, applySkillHealth(s))
	}
	return out
}

// findProjectLock walks up from the first project-scoped skill Path to find
// a skills-lock.json in node_modules/.skills/.
//
// ponytail: reads the lock once from the first project skill's path, assumes
// all project skills share the same lock. Expand to per-skill lock files only
// if skills.sh ever supports multiple project skill roots.
func findProjectLock(skills []Skill) map[string]string {
	for _, s := range skills {
		if s.Scope == ScopeProject && s.Path != "" {
			// Walk up from skill path to find the lock file.
			// Skills are at .../node_modules/.skills/<name>/SKILL.md
			// Lock is at .../node_modules/.skills/skills-lock.json
			// But Path could be anything, so check one level up from node_modules.
			candidate := filepath.Join(s.Path, "..", "skills-lock.json")
			if data, err := os.ReadFile(candidate); err == nil {
				return parseLockSources(data)
			}
			// Check if Path is under node_modules/.skills/<name>
			// The lock is two levels up from the skill dir.
			candidate = filepath.Join(s.Path, "..", "..", "skills-lock.json")
			if data, err := os.ReadFile(candidate); err == nil {
				return parseLockSources(data)
			}
		}
	}
	// Fallback: try CWD-relative path.
	if data, err := os.ReadFile(filepath.Join(".", "node_modules", ".skills", "skills-lock.json")); err == nil {
		return parseLockSources(data)
	}
	return nil
}

func (c *NpxClient) readLockSources(lockPath string) map[string]string {
	data, err := os.ReadFile(lockPath)
	if err != nil {
		return nil
	}
	return parseLockSources(data)
}

func parseLockSources(data []byte) map[string]string {
	var raw struct {
		Skills json.RawMessage `json:"skills"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}
	m := make(map[string]string)

	// Try map format: {"caveman": {"source":"...", "repo":"..."}}
	var byName map[string]struct {
		Source string `json:"source"`
		Repo   string `json:"repo"`
	}
	if err := json.Unmarshal(raw.Skills, &byName); err == nil && byName != nil {
		for name, entry := range byName {
			if entry.Repo != "" {
				m[name] = entry.Repo
			} else if entry.Source != "" {
				m[name] = entry.Source
			}
		}
		return m
	}

	// Try array format: [{"name":"caveman", "source":"...", "repo":"..."}]
	var list []struct {
		Name   string `json:"name"`
		Source string `json:"source"`
		Repo   string `json:"repo"`
	}
	if err := json.Unmarshal(raw.Skills, &list); err != nil {
		return nil
	}
	for _, s := range list {
		if s.Repo != "" {
			m[s.Name] = s.Repo
		} else if s.Source != "" {
			m[s.Name] = s.Source
		}
	}
	return m
}

// findGlobalLock locates the global skills lock file.
//
// ponytail: checks HOME and XDG_CONFIG_HOME. Expand if skills.sh changes
// their default global install path.
func findGlobalLock() string {
	home, _ := os.UserHomeDir()
	candidates := []string{
		filepath.Join(home, ".config", "skills", "skills-lock.json"),
	}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		candidates = append([]string{filepath.Join(xdg, "skills", "skills-lock.json")}, candidates...)
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func parseListJSON(b []byte) []Skill {
	var raw []struct {
		Name   string   `json:"name"`
		Path   string   `json:"path"`
		Scope  string   `json:"scope"`
		Agents []string `json:"agents"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil
	}
	out := make([]Skill, 0, len(raw))
	for _, item := range raw {
		out = append(out, Skill{
			Name:    item.Name,
			Source:  filepath.Base(item.Path),
			Scope:   Scope(item.Scope),
			Path:    item.Path,
			Agents:  item.Agents,
			Enabled: true,
			Status:  SkillStatusEnabled,
		})
	}
	return out
}

// ─── LoadLockFile ────────────────────────────────────────────────────

func LoadLockFile(path string) (LockFile, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return LockFile{}, err
	}
	var raw struct {
		Version int             `json:"version"`
		Skills  json.RawMessage `json:"skills"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return LockFile{}, err
	}
	lock := LockFile{Version: raw.Version}

	var byName map[string]LockSkill
	if err := json.Unmarshal(raw.Skills, &byName); err == nil && byName != nil {
		for name, item := range byName {
			item.Name = name
			if item.Source == "" {
				item.Source = item.Repo
			}
			lock.Skills = append(lock.Skills, item)
		}
		sort.Slice(lock.Skills, func(i, j int) bool { return lock.Skills[i].Name < lock.Skills[j].Name })
		return lock, nil
	}

	var list []LockSkill
	if err := json.Unmarshal(raw.Skills, &list); err != nil {
		return LockFile{}, err
	}
	for i := range list {
		if list[i].Source == "" {
			list[i].Source = list[i].Repo
		}
	}
	lock.Skills = list
	return lock, nil
}

// ─── Find ────────────────────────────────────────────────────────────

func (c *NpxClient) Find(ctx context.Context, q string) ([]Skill, error) {
	q = strings.TrimSpace(q)
	if len([]rune(q)) < 2 {
		return nil, nil
	}
	out, err := c.run(ctx, "npx", "skills", "find", q)
	if err != nil {
		return nil, err
	}
	return parseFindOutput(string(out)), nil
}

// ─── ListSources ─────────────────────────────────────────────────────

// ListSources returns installed-skill counts from project/global lock files,
// merged with user-added sources from Knit's config.
func (c *NpxClient) ListSources(ctx context.Context) ([]Source, error) {
	lockCounts := map[string]int{}
	for _, path := range []string{projectLockPath(), globalLockPath()} {
		b, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var parsed struct {
			Skills map[string]map[string]any `json:"skills"`
		}
		if json.Unmarshal(b, &parsed) == nil {
			for _, entry := range parsed.Skills {
				source, _ := entry["source"].(string)
				if source != "" {
					lockCounts[source]++
				}
			}
		}
	}

	seen := map[string]Source{}
	cfg, err := readKnitConfig()
	if err != nil {
		return nil, err
	}
	for _, name := range cfg.Sources {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		seen[name] = Source{Name: name, Repo: name}
	}
	for name, n := range lockCounts {
		if contains(cfg.Removed, name) {
			continue
		}
		s := seen[name]
		s.Name = name
		s.Repo = name
		s.Installed = n
		s.Available = n
		seen[name] = s
	}

	out := make([]Source, 0, len(seen))
	for _, s := range seen {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// ─── Source mutations ────────────────────────────────────────────────

// AddSource validates a source by inspecting available skills,
// then persists it in Knit's config.
func (c *NpxClient) AddSource(ctx context.Context, source string) error {
	source = strings.TrimSpace(source)
	if source == "" {
		return errors.New("source required")
	}
	_, err := c.run(ctx, "npx", "skills", "add", source, "--list")
	if err != nil {
		return fmt.Errorf("source %q not valid: %w", source, err)
	}
	return addConfigSource(source)
}

// RemoveSource removes the source from Knit's config so it no longer
// appears in the Sources tab. Installed skills from this source are
// preserved — the user can uninstall them individually from Installed.
func (c *NpxClient) RemoveSource(ctx context.Context, source string) error {
	return removeConfigSource(source)
}

// UpdateSource re-runs --list to validate the source still works.
// Config persistence is handled by the app layer.
func (c *NpxClient) UpdateSource(ctx context.Context, name string) error {
	_, err := c.run(ctx, "npx", "skills", "add", name, "--list")
	return err
}

// ─── Skill mutations ─────────────────────────────────────────────────

func (c *NpxClient) InstallSkill(ctx context.Context, skill Skill, global bool) error {
	args := []string{"skills", "add", skill.Source, "--skill", skill.Name, "-y"}
	if global {
		args = append(args, "-g")
	}
	_, err := c.run(ctx, "npx", args...)
	return err
}

func (c *NpxClient) UpdateSkill(ctx context.Context, skill Skill) error {
	args := []string{"skills", "update", skill.Name, "-y"}
	if skill.Scope == ScopeGlobal {
		args = append(args, "-g")
	} else if skill.Scope == ScopeProject {
		args = append(args, "-p")
	}
	_, err := c.run(ctx, "npx", args...)
	return err
}

func (c *NpxClient) UninstallSkill(ctx context.Context, skill Skill) error {
	args := []string{"skills", "remove", skill.Name, "-y"}
	if skill.Scope == ScopeGlobal || skill.Scope == ScopeUser {
		args = append(args, "-g")
	} else if skill.Scope == ScopeProject {
		args = append(args, "-p")
	}
	_, err := c.run(ctx, "npx", args...)
	return err
}

// PruneLocks is a no-op unless the upstream CLI gains a dedicated command.
//
// ponytail: npx skills has no prune command. Return nil to match the
// existing interface contract without a stub.
func (c *NpxClient) PruneLocks(context.Context) error { return nil }

// ─── SkillDetail ─────────────────────────────────────────────────────

// SkillDetail returns full metadata + preview for a skill. For installed
// skills (Path != "") the SKILL.md is read from disk. For discovered skills
// the skills.sh detail API is called. Results are cached in-memory so that
// navigating between skills does not re-fetch.
func (c *NpxClient) SkillDetail(ctx context.Context, skill Skill) (Skill, error) {
	key := cacheKey(skill)
	if cached, ok := c.getCachedDetail(key); ok {
		return cached, nil
	}

	// Local SKILL.md when available.
	if skill.Path != "" {
		content, err := os.ReadFile(filepath.Join(skill.Path, "SKILL.md"))
		if err != nil {
			// Cache the original skill so we don't retry a broken path.
			c.setCachedDetail(key, skill)
			return skill, nil
		}
		d := applySkillHealth(mergeSkillMarkdown(skill, string(content)))
		c.setCachedDetail(key, d)
		return d, nil
	}

	// Source git cache: lazy clone and read SKILL.md for non-installed
	// source skills without hitting the remote API.
	if skill.Source != "" && skill.Name != "" {
		if md, err := c.loadSourceSkillMarkdown(ctx, skill.Source, skill.Name); err == nil {
			d := applySkillHealth(mergeSkillMarkdown(skill, md))
			c.setCachedDetail(key, d)
			return d, nil
		}
	}

	// Remote API.
	id := skill.ID
	if id == "" {
		id = strings.Trim(skill.Source, "/") + "/" + strings.Trim(skill.Name, "/")
	}
	base := os.Getenv("SKILLS_API_URL")
	if base == "" {
		base = "https://skills.sh"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/api/v1/skills/"+id, nil)
	if err != nil {
		return skill, err
	}
	res, err := httpClient.Do(req)
	if err != nil {
		return skill, err
	}
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		c.setCachedDetail(key, skill)
		return skill, nil
	}
	var data struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Slug   string `json:"slug"`
		Source string `json:"source"`
		Files  []struct {
			Path     string `json:"path"`
			Contents string `json:"contents"`
		} `json:"files"`
	}
	if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
		return skill, err
	}
	for _, f := range data.Files {
		if strings.EqualFold(filepath.Base(f.Path), "SKILL.md") {
			baseSkill := skill
			if data.Source != "" {
				baseSkill.Source = data.Source
			}
			d := applySkillHealth(mergeSkillMarkdown(baseSkill, f.Contents))
			c.setCachedDetail(key, d)
			return d, nil
		}
	}
	// No SKILL.md found — cache the original.
	c.setCachedDetail(key, skill)
	return skill, nil
}

func cacheKey(s Skill) string {
	if s.ID != "" {
		return s.ID
	}
	return s.Source + "/" + s.Name
}

// ListSourceSkills runs `npx skills add <source> --list` and parses the
// output into a slice of available Skills.
func (c *NpxClient) ListSourceSkills(ctx context.Context, source string) ([]Skill, error) {
	out, err := c.run(ctx, "npx", "skills", "add", source, "--list")
	if err != nil {
		return nil, err
	}
	return parseListAvailable(string(out), source), nil
}

func isSourceSkillName(s string) bool {
	if s == "" {
		return false
	}
	// Reject reserved output labels from npx skills prompts.
	switch s {
	case "skills", "skill", "available", "available-skills", "general", "source", "sources":
		return false
	}
	// First character must be a lowercase letter or digit.
	r := rune(s[0])
	if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')) {
		return false
	}
	// Rest must be lowercase letters, digits, dots, hyphens, underscores.
	for i := 1; i < len(s); i++ {
		c := s[i]
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' || c == '.' || c == '_') {
			return false
		}
	}
	return true
}

// stripBoxDrawing removes leading box-drawing and prompt characters from a line.
func stripBoxDrawing(s string) string {
	for len(s) > 0 {
		r, n := utf8.DecodeRuneInString(s)
		switch r {
		case '\u2502', '\u2514', '\u250C', '\u251C', '\u2510', '\u2524', '\u2518', '\u2500',
			'\u25C6', '\u25C7', '\u2714', '\u2716':
			s = s[n:]
		default:
			return s
		}
	}
	return s
}

func parseListAvailable(out, source string) []Skill {
	var res []Skill
	seen := map[string]bool{}
	scanner := bufio.NewScanner(strings.NewReader(stripANSI(out)))
	for scanner.Scan() {
		trimmed := strings.TrimSpace(scanner.Text())
		trimmed = stripBoxDrawing(trimmed)
		trimmed = strings.TrimSpace(trimmed)
		if trimmed == "" {
			continue
		}
		if !isSourceSkillName(trimmed) {
			continue
		}
		if seen[trimmed] {
			continue
		}
		seen[trimmed] = true
		res = append(res, Skill{
			Name:   trimmed,
			Source: source,
			ID:     source + "/" + trimmed,
		})
	}
	return res
}

// ─── SyncFromLock ─────────────────────────────────────────────────

func (c *NpxClient) SyncFromLock(ctx context.Context, lockPath string, global bool) error {
	lock, err := LoadLockFile(lockPath)
	if err != nil {
		return err
	}
	for _, s := range lock.Skills {
		if strings.TrimSpace(s.Name) == "" || strings.TrimSpace(s.Source) == "" {
			continue
		}
		if err := c.InstallSkill(ctx, Skill{Name: s.Name, Source: s.Source}, global); err != nil {
			return err
		}
	}
	return nil
}

// ─── Runner ──────────────────────────────────────────────────────────

func (c *NpxClient) run(ctx context.Context, name string, args ...string) ([]byte, error) {
	if c.runner == nil {
		c.runner = execRunner{}
	}
	out, err := c.runner.Run(ctx, name, args...)
	if err != nil {
		return out, fmt.Errorf("%s %s: %w: %s",
			name, strings.Join(args, " "), err, strings.TrimSpace(stripANSI(string(out))))
	}
	return out, nil
}

// ─── Parsers ─────────────────────────────────────────────────────────

var findLineRE = regexp.MustCompile(`^(.+?)@(.+?)(?:\s+([0-9.]+[kKmM]?(?:\s+installs?)))?$`)

func parseFindOutput(out string) []Skill {
	var res []Skill
	scan := bufio.NewScanner(strings.NewReader(stripANSI(out)))
	for scan.Scan() {
		line := strings.TrimSpace(scan.Text())
		if line == "" ||
			strings.HasPrefix(line, "Install with") ||
			strings.HasPrefix(line, "No skills found") ||
			strings.HasPrefix(line, "Usage:") {
			continue
		}
		if strings.HasPrefix(line, "└") {
			if len(res) > 0 {
				if u := strings.TrimSpace(strings.TrimPrefix(line, "└")); strings.HasPrefix(u, "https://skills.sh/") {
					res[len(res)-1].ID = strings.TrimPrefix(u, "https://skills.sh/")
				}
			}
			continue
		}
		m := findLineRE.FindStringSubmatch(line)
		if len(m) == 4 {
			res = append(res, Skill{
				Source:   strings.TrimSpace(m[1]),
				Name:     strings.TrimSpace(m[2]),
				Installs: parseInstalls(strings.TrimSpace(m[3])),
			})
		}
	}
	for i := range res {
		if res[i].ID == "" {
			res[i].ID = res[i].Source + "/" + res[i].Name
		}
	}
	return res
}

func parseInstalls(s string) int {
	if s == "" {
		return 0
	}
	f := strings.ToLower(strings.TrimSpace(s))
	f = strings.TrimSuffix(f, " installs")
	f = strings.TrimSuffix(f, " install")
	f = strings.TrimSpace(f)
	switch {
	case strings.HasSuffix(f, "m"):
		if v, err := strconv.ParseFloat(strings.TrimSuffix(f, "m"), 64); err == nil {
			return int(v * 1_000_000)
		}
	case strings.HasSuffix(f, "k"):
		if v, err := strconv.ParseFloat(strings.TrimSuffix(f, "k"), 64); err == nil {
			return int(v * 1_000)
		}
	default:
		if v, err := strconv.Atoi(f); err == nil {
			return v
		}
	}
	return 0
}

// ─── Markdown helpers ────────────────────────────────────────────────

var frontmatterRE = regexp.MustCompile(`(?s)^---\n(.*?)\n---\n(.*)$`)
var skillBlockRE = regexp.MustCompile(`(?s)<SKILL\.md>\n(.*?)\n</SKILL\.md>`)
var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)

func applySkillHealth(s Skill) Skill {
	s.Warnings = removeManagedWarnings(s.Warnings)
	if strings.TrimSpace(s.Description) == "" {
		s.Warnings = appendWarning(s.Warnings, "missing description")
	}
	if len(s.Agents) == 0 {
		s.Warnings = appendWarning(s.Warnings, "no agents reported")
	}
	return s
}

func removeManagedWarnings(warnings []string) []string {
	out := warnings[:0]
	for _, w := range warnings {
		if w != "missing description" && w != "no agents reported" {
			out = append(out, w)
		}
	}
	return out
}

func appendWarning(warnings []string, msg string) []string {
	for _, w := range warnings {
		if w == msg {
			return warnings
		}
	}
	return append(warnings, msg)
}

func mergeSkillMarkdown(base Skill, md string) Skill {
	md = strings.TrimSpace(md)
	parsed := parseMarkdownSkill(md)
	if parsed.Name != "" {
		base.Name = parsed.Name
	}
	if parsed.Description != "" {
		base.Description = parsed.Description
	}
	if parsed.Source != "" {
		base.Source = parsed.Source
	}
	base.Preview = parsed.Preview
	if base.Preview == "" {
		base.Preview = md
	}
	return base
}

func parseMarkdownSkill(md string) Skill {
	md = strings.TrimSpace(md)
	res := Skill{}
	if m := frontmatterRE.FindStringSubmatch(md); len(m) == 3 {
		for _, line := range strings.Split(m[1], "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			key, val, ok := strings.Cut(line, ":")
			if !ok {
				continue
			}
			switch strings.TrimSpace(strings.ToLower(key)) {
			case "name":
				res.Name = strings.TrimSpace(val)
			case "description":
				res.Description = strings.TrimSpace(val)
			case "source":
				res.Source = strings.TrimSpace(val)
			}
		}
		res.Preview = strings.TrimSpace(m[2])
		return res
	}
	res.Preview = md
	return res
}

func extractSkillMarkdown(out string) string {
	if m := skillBlockRE.FindStringSubmatch(out); len(m) == 2 {
		return strings.TrimSpace(m[1])
	}
	return out
}

func stripANSI(s string) string { return ansiRE.ReplaceAllString(s, "") }

// ─── Path helpers ────────────────────────────────────────────────────

func projectLockPath() string { return filepath.Join(mustCwd(), "skills-lock.json") }

func globalLockPath() string {
	if xdg := os.Getenv("XDG_STATE_HOME"); xdg != "" {
		return filepath.Join(xdg, "skills", ".skill-lock.json")
	}
	return filepath.Join(mustHome(), ".agents", ".skill-lock.json")
}

func mustCwd() string  { cwd, _ := os.Getwd(); return cwd }
func mustHome() string { home, _ := os.UserHomeDir(); return home }

func parseCount(s, suffix string) int {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	switch strings.ToUpper(suffix) {
	case "K":
		f *= 1_000
	case "M":
		f *= 1_000_000
	case "B":
		f *= 1_000_000_000
	}
	return int(f)
}

func IsNotFound(err error) bool { return err != nil && errors.Is(err, exec.ErrNotFound) }

// ─── Source git URL and cache ────────────────────────────────────────

func sourceGitURL(source string) (string, bool) {
	source = strings.TrimSpace(source)
	if source == "" || strings.HasSuffix(source, ".json") {
		return "", false
	}
	if strings.HasPrefix(source, "git@") {
		return source, true
	}
	if strings.HasPrefix(source, "http://") && os.Getenv("KNIT_ALLOW_INSECURE_SOURCES") != "1" {
		return "", false
	}
	if strings.HasPrefix(source, "https://") || strings.HasPrefix(source, "http://") {
		if strings.HasSuffix(source, ".git") {
			return source, true
		}
		return strings.TrimRight(source, "/") + ".git", true
	}
	parts := strings.Split(source, "/")
	if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
		return "https://github.com/" + source + ".git", true
	}
	return "", false
}

func sourceCacheDir(source string) string {
	if root := os.Getenv("KNIT_SOURCE_CACHE_DIR"); root != "" {
		return filepath.Join(root, cacheKeyPart(source))
	}
	root, err := os.UserCacheDir()
	if err != nil {
		root = os.TempDir()
	}
	return filepath.Join(root, "knit", "sources", cacheKeyPart(source))
}

func cacheKeyPart(s string) string {
	sum := sha1.Sum([]byte(s))
	return hex.EncodeToString(sum[:])
}

// loadSourceSkillMarkdown reads SKILL.md for source+name, cloning the repo
// if no cached copy exists yet. Only git sources are supported.
func (c *NpxClient) loadSourceSkillMarkdown(ctx context.Context, source, name string) (string, error) {
	if strings.TrimSpace(source) == "" || strings.TrimSpace(name) == "" {
		return "", errors.New("source and skill name required")
	}
	dir := sourceCacheDir(source)
	if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
		url, ok := sourceGitURL(source)
		if !ok {
			return "", errors.New("source preview supports git sources only")
		}
		if _, err := c.run(ctx, "git", "clone", "--depth", "1", url, dir); err != nil {
			return "", err
		}
	}

	subdirs := []string{".", "skills", ".agents/skills"}
	for _, sub := range subdirs {
		candidate, err := safeJoinUnder(dir, sub, name, "SKILL.md")
		if err != nil {
			continue
		}
		b, err := os.ReadFile(candidate)
		if err == nil {
			return string(b), nil
		}
	}
	return "", fmt.Errorf("SKILL.md not found for %s in %s", name, source)
}

// safeJoinUnder joins path parts and verifies the result stays under root.
// This prevents path traversal attacks via malicious name or source inputs.
func safeJoinUnder(root string, parts ...string) (string, error) {
	rootClean, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	p := filepath.Join(append([]string{rootClean}, parts...)...)
	pClean, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(rootClean, pClean)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("path %q escapes cache root %q", pClean, rootClean)
	}
	return pClean, nil
}
