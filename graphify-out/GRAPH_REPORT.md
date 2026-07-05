# Graph Report - .  (2026-07-05)

## Corpus Check
- 26 files · ~107,282 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 415 nodes · 1095 edges · 20 communities (18 shown, 2 thin omitted)
- Extraction: 83% EXTRACTED · 17% INFERRED · 0% AMBIGUOUS · INFERRED: 188 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Community Hubs (Navigation)
- [[_COMMUNITY_Test Suite|Test Suite]]
- [[_COMMUNITY_Runner & Client|Runner & Client]]
- [[_COMMUNITY_Skill Management|Skill Management]]
- [[_COMMUNITY_Application UI|Application UI]]
- [[_COMMUNITY_Project Overview|Project Overview]]
- [[_COMMUNITY_UI Event Handlers|UI Event Handlers]]
- [[_COMMUNITY_Render Functions|Render Functions]]
- [[_COMMUNITY_Client Interface|Client Interface]]
- [[_COMMUNITY_Source Filtering|Source Filtering]]
- [[_COMMUNITY_Skill Filtering|Skill Filtering]]
- [[_COMMUNITY_Skill Concepts|Skill Concepts]]
- [[_COMMUNITY_Table Rendering|Table Rendering]]
- [[_COMMUNITY_Entry Point|Entry Point]]
- [[_COMMUNITY_Preview Rendering|Preview Rendering]]
- [[_COMMUNITY_Error Handling|Error Handling]]
- [[_COMMUNITY_Record Script|Record Script]]
- [[_COMMUNITY_Check Fixtures|Check Fixtures]]
- [[_COMMUNITY_Styles|Styles]]
- [[_COMMUNITY_Install Script (sh)|Install Script (sh)]]

## God Nodes (most connected - your core abstractions)
1. `model` - 70 edges
2. `newTestModel()` - 65 edges
3. `contains()` - 54 edges
4. `T` - 50 edges
5. `T` - 34 edges
6. `T` - 32 edges
7. `Cmd` - 24 edges
8. `Skill` - 22 edges
9. `NpxClient` - 22 edges
10. `NewNpxClientWithRunner()` - 21 edges

## Surprising Connections (you probably didn't know these)
- `KNIT Brand Logo` --conceptually_related_to--> `Knit Terminal UI`  [INFERRED]
  assets/logo-dark.png → README.md
- `KNIT Brand Logo with Weave Motif` --conceptually_related_to--> `Knit Terminal UI`  [INFERRED]
  assets/logo-transparent.png → README.md
- `main()` --calls--> `New()`  [INFERRED]
  cmd/knit/main.go → internal/app/app.go
- `main()` --calls--> `NewNpxClient()`  [INFERRED]
  cmd/knit/main.go → internal/skills/skills.go
- `CI Workflow` --references--> `Knit Terminal UI`  [INFERRED]
  .github/workflows/ci.yml → README.md

## Import Cycles
- None detected.

## Hyperedges (group relationships)
- **Knit CI/CD Pipeline** — workflows_ci, workflows_release, _goreleaser, readme_knit [INFERRED 0.85]
- **Agent Skill System Collection** — caveman_skill_caveman, code_reviewer_skill_code_reviewer, frontend_design_skill_frontend_design [INFERRED 0.75]
- **Knit Tab Navigation System** — assets_knit_demo_installed_tab, assets_knit_demo_discover_tab, assets_knit_demo_sources_tab, assets_knit_demo_skill_detail, assets_knit_demo_search_bar [EXTRACTED 1.00]
- **Knit Brand Identity** — assets_logo_transparent_knit_logo, assets_logo_dark_knit_logo, readme_knit, assets_knit_demo_knit [INFERRED 0.85]

## Communities (20 total, 2 thin omitted)

### Community 0 - "Test Suite"
Cohesion: 0.07
Nodes (90): TestActionOKMessage(), TestActionResultErrorShowsBriefFooterAndDetailedErrorLog(), TestActionResultShowsBriefFooterAndDetailedLog(), TestDigitSwitchesTabWhenSearchNotFocused(), TestDigitTypesIntoFocusedInstalledSearch(), TestDiscoverEmptyStateSuggestsSearch(), TestEmptyDash(), TestInstallCommandEmptyScope() (+82 more)

### Community 1 - "Runner & Client"
Cohesion: 0.08
Nodes (40): Context, Client, execRunner, knitConfig, LockSkill, LogEntry, NpxClient, Runner (+32 more)

### Community 2 - "Skill Management"
Cohesion: 0.09
Nodes (48): Context, Skill, T, LockSkill, fakeRunner, LockFile, recordingRunner, sequenceRunner (+40 more)

### Community 3 - "Application UI"
Cohesion: 0.10
Nodes (20): actionResultMsg, addSourceDoneMsg, agentMatrix(), detailActions(), detailScopeText(), detailStatusText(), emptyDash(), helpBody() (+12 more)

### Community 4 - "Project Overview"
Cohesion: 0.10
Nodes (26): GoReleaser Configuration, Knit TUI Demo GIF, Caveman Skill - Ultra-compressed Communication, Code Reviewer Skill - Expert Code Review, Discover Tab, Frontend Design Skill - UI Design, Installed Tab, knit - Terminal UI for npx skills (+18 more)

### Community 5 - "UI Event Handlers"
Cohesion: 0.26
Nodes (5): actionOKMessage(), removeCommand(), updateCommand(), Cmd, KeyMsg

### Community 6 - "Render Functions"
Cohesion: 0.29
Nodes (4): clampOffset(), renderListLine(), rowStyle(), styles

### Community 7 - "Client Interface"
Cohesion: 0.23
Nodes (5): bulkFakeClient, fakeClient, Context, Skill, Source

### Community 8 - "Source Filtering"
Cohesion: 0.19
Nodes (6): filterSources(), model, Client, Context, Source, LogEntry

### Community 9 - "Skill Filtering"
Cohesion: 0.22
Nodes (6): filterSkills(), installedKey(), mergeDetailSkill(), sameSourceSkill(), Skill, loadedMsg

### Community 10 - "Skill Concepts"
Cohesion: 0.14
Nodes (15): Caveman Mode, Ultra-compressed Communication, Token Usage Reduction (~75%), Code Reviewer, Automated Fix Suggestions, Performance Bottleneck Detection, PR Review Integration, Static Analysis (+7 more)

### Community 11 - "Table Rendering"
Cohesion: 0.39
Nodes (8): rowCell, clampIndex(), numberCell(), renderBlockRow(), renderCells(), truncateText(), visibleWidth(), Style

### Community 12 - "Entry Point"
Cohesion: 0.39
Nodes (6): T, commandConfig, main(), parseArgs(), TestParseArgsSyncLockFileAndGlobal(), TestParseArgsSyncRequiresLockFile()

### Community 13 - "Preview Rendering"
Cohesion: 0.62
Nodes (6): highlightText(), lineWidth(), renderCodeBlock(), renderPreview(), stripFrontmatter(), styles

### Community 14 - "Error Handling"
Cohesion: 0.33
Nodes (4): errString(), stripANSI(), Msg, sourceSkillsLoadedMsg

### Community 15 - "Record Script"
Cohesion: 0.33
Nodes (4): record.sh script, HOME, PATH, TERM

### Community 16 - "Check Fixtures"
Cohesion: 0.50
Nodes (4): check-fixtures.sh script, HOME, PATH, require_contains()

## Knowledge Gaps
- **41 isolated node(s):** `LogEntry`, `Msg`, `loadedMsg`, `sourceSkillsLoadedMsg`, `addSourceDoneMsg` (+36 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **2 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `contains()` connect `Test Suite` to `Source Filtering`, `Skill Filtering`, `Skill Management`, `Runner & Client`?**
  _High betweenness centrality (0.278) - this node is a cross-community bridge._
- **Why does `model` connect `Source Filtering` to `Test Suite`, `Application UI`, `UI Event Handlers`, `Render Functions`, `Skill Filtering`, `Error Handling`?**
  _High betweenness centrality (0.183) - this node is a cross-community bridge._
- **Why does `New()` connect `Test Suite` to `Source Filtering`, `Runner & Client`, `Application UI`, `Entry Point`?**
  _High betweenness centrality (0.178) - this node is a cross-community bridge._
- **Are the 24 inferred relationships involving `newTestModel()` (e.g. with `TestActionResultErrorShowsBriefFooterAndDetailedErrorLog()` and `TestActionResultShowsBriefFooterAndDetailedLog()`) actually correct?**
  _`newTestModel()` has 24 INFERRED edges - model-reasoned connections that need verification._
- **Are the 51 inferred relationships involving `contains()` (e.g. with `TestActionResultErrorShowsBriefFooterAndDetailedErrorLog()` and `TestDiscoverEmptyStateSuggestsSearch()`) actually correct?**
  _`contains()` has 51 INFERRED edges - model-reasoned connections that need verification._
- **What connects `LogEntry`, `Msg`, `loadedMsg` to the rest of the system?**
  _42 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Test Suite` be split into smaller, more focused modules?**
  _Cohesion score 0.07118967988533206 - nodes in this community are weakly interconnected._