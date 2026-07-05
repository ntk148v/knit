# Graph Report - .  (2026-07-05)

## Corpus Check
- 12 files · ~107,186 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 345 nodes · 627 edges · 13 communities detected
- Extraction: 100% EXTRACTED · 0% INFERRED · 0% AMBIGUOUS
- Token cost: 0 input · 0 output

## God Nodes (most connected - your core abstractions)
1. `model` - 59 edges
2. `newTestModel()` - 40 edges
3. `NpxClient` - 20 edges
4. `max()` - 15 edges
5. `fakeClient` - 14 edges
6. `min()` - 12 edges
7. `writeFile()` - 8 edges
8. `readKnitConfig()` - 5 edges
9. `addConfigSource()` - 5 edges
10. `removeConfigSource()` - 5 edges

## Surprising Connections (you probably didn't know these)
- None detected - all connections are within the same source files.

## Communities

### Community 0 - "Community 0"
Cohesion: 0.05
Nodes (45): addConfigSource(), appendWarning(), applySkillHealth(), cacheKey(), cacheKeyPart(), Client, contains(), execRunner (+37 more)

### Community 1 - "Community 1"
Cohesion: 0.1
Nodes (9): actionOKMessage(), emptyDash(), filterSkills(), installedKey(), max(), min(), model, removeCommand() (+1 more)

### Community 2 - "Community 2"
Cohesion: 0.06
Nodes (42): bulkFakeClient, newTestModel(), TestActionViewRowsAreSeparate(), TestDiscoverRefreshErrorKeepsExistingList(), TestDiscoverRowsUseFullWidthLikeSources(), TestDiscoverScrollClipsOutput(), TestDownFromSearchSelectsAddSource(), TestDownFromSearchSelectsFirstDiscover() (+34 more)

### Community 3 - "Community 3"
Cohesion: 0.06
Nodes (15): fakeRunner, recordingRunner, sequenceRunner, hasWarning(), TestApplySkillHealthAddsMinimalWarnings(), TestApplySkillHealthRemovesStaleDescriptionWarning(), TestEnrichInstalledSourceFromLock(), TestLoadLockFileArrayShape() (+7 more)

### Community 4 - "Community 4"
Cohesion: 0.06
Nodes (0): 

### Community 5 - "Community 5"
Cohesion: 0.08
Nodes (21): agentMatrix(), detailActions(), detailScopeText(), detailStatusText(), dropLastRune(), errString(), filterSources(), focusArea (+13 more)

### Community 6 - "Community 6"
Cohesion: 0.14
Nodes (1): fakeClient

### Community 7 - "Community 7"
Cohesion: 0.27
Nodes (6): rowCell, renderBlockRow(), renderCells(), renderListLine(), truncateText(), visibleWidth()

### Community 8 - "Community 8"
Cohesion: 0.22
Nodes (8): actionResultMsg, addSourceDoneMsg, clearLogsMsg, confirmResultMsg, debouncedSearchMsg, detailLoadedMsg, loadedMsg, sourceSkillsLoadedMsg

### Community 9 - "Community 9"
Cohesion: 0.67
Nodes (5): highlightText(), lineWidth(), renderCodeBlock(), renderPreview(), stripFrontmatter()

### Community 10 - "Community 10"
Cohesion: 0.67
Nodes (1): styles

### Community 11 - "Community 11"
Cohesion: 0.67
Nodes (0): 

### Community 12 - "Community 12"
Cohesion: 1.0
Nodes (0): 

## Knowledge Gaps
- **23 isolated node(s):** `SkillStatus`, `Skill`, `Source`, `LogEntry`, `LockFile` (+18 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **Thin community `Community 12`** (1 nodes): `install.ps1`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `model` connect `Community 1` to `Community 5`?**
  _High betweenness centrality (0.169) - this node is a cross-community bridge._
- **Why does `fakeClient` connect `Community 6` to `Community 2`?**
  _High betweenness centrality (0.059) - this node is a cross-community bridge._
- **What connects `SkillStatus`, `Skill`, `Source` to the rest of the system?**
  _23 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Community 0` be split into smaller, more focused modules?**
  _Cohesion score 0.05 - nodes in this community are weakly interconnected._
- **Should `Community 1` be split into smaller, more focused modules?**
  _Cohesion score 0.1 - nodes in this community are weakly interconnected._
- **Should `Community 2` be split into smaller, more focused modules?**
  _Cohesion score 0.06 - nodes in this community are weakly interconnected._
- **Should `Community 3` be split into smaller, more focused modules?**
  _Cohesion score 0.06 - nodes in this community are weakly interconnected._