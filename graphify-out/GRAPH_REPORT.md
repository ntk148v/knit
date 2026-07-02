# Graph Report - .  (2026-07-02)

## Corpus Check
- 10 files · ~84,767 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 298 nodes · 546 edges · 11 communities detected
- Extraction: 100% EXTRACTED · 0% INFERRED · 0% AMBIGUOUS
- Token cost: 0 input · 0 output

## God Nodes (most connected - your core abstractions)
1. `model` - 56 edges
2. `newTestModel()` - 38 edges
3. `NpxClient` - 18 edges
4. `max()` - 15 edges
5. `fakeClient` - 14 edges
6. `min()` - 12 edges
7. `writeFile()` - 8 edges
8. `parseListAvailable()` - 5 edges
9. `parseFindOutput()` - 4 edges
10. `stripANSI()` - 4 edges

## Surprising Connections (you probably didn't know these)
- None detected - all connections are within the same source files.

## Communities

### Community 0 - "Community 0"
Cohesion: 0.06
Nodes (32): cacheKey(), cacheKeyPart(), Client, execRunner, findGlobalLock(), findProjectLock(), globalLockPath(), isSourceSkillName() (+24 more)

### Community 1 - "Community 1"
Cohesion: 0.11
Nodes (9): actionOKMessage(), filterSkills(), filterSources(), max(), mergeDetailSkill(), min(), model, removeCommand() (+1 more)

### Community 2 - "Community 2"
Cohesion: 0.08
Nodes (39): newTestModel(), TestActionViewRowsAreSeparate(), TestDiscoverRefreshErrorKeepsExistingList(), TestDiscoverRowsUseFullWidthLikeSources(), TestDiscoverScrollClipsOutput(), TestDownFromSearchSelectsAddSource(), TestDownFromSearchSelectsFirstDiscover(), TestDownFromSearchSelectsFirstInstalled() (+31 more)

### Community 3 - "Community 3"
Cohesion: 0.07
Nodes (23): actionResultMsg, addSourceDoneMsg, confirmResultMsg, detailActions(), detailScopeText(), detailStatusText(), emptyDash(), errString() (+15 more)

### Community 4 - "Community 4"
Cohesion: 0.07
Nodes (11): fakeRunner, recordingRunner, sequenceRunner, TestEnrichInstalledSourceFromLock(), TestLoadLockFileArrayShape(), TestLoadLockFileMapShape(), TestSkillDetailFromDisk(), TestSkillDetailLoadsPreviewFromCachedSource() (+3 more)

### Community 5 - "Community 5"
Cohesion: 0.07
Nodes (0): 

### Community 6 - "Community 6"
Cohesion: 0.14
Nodes (1): fakeClient

### Community 7 - "Community 7"
Cohesion: 0.27
Nodes (6): rowCell, renderBlockRow(), renderCells(), renderListLine(), truncateText(), visibleWidth()

### Community 8 - "Community 8"
Cohesion: 0.8
Nodes (4): highlightText(), lineWidth(), renderCodeBlock(), renderPreview()

### Community 9 - "Community 9"
Cohesion: 0.67
Nodes (1): styles

### Community 10 - "Community 10"
Cohesion: 0.67
Nodes (0): 

## Knowledge Gaps
- **19 isolated node(s):** `SkillStatus`, `Skill`, `Source`, `LogEntry`, `LockFile` (+14 more)
  These have ≤1 connection - possible missing edges or undocumented components.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `model` connect `Community 1` to `Community 3`?**
  _High betweenness centrality (0.186) - this node is a cross-community bridge._
- **Why does `fakeClient` connect `Community 6` to `Community 2`?**
  _High betweenness centrality (0.069) - this node is a cross-community bridge._
- **What connects `SkillStatus`, `Skill`, `Source` to the rest of the system?**
  _19 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Community 0` be split into smaller, more focused modules?**
  _Cohesion score 0.06 - nodes in this community are weakly interconnected._
- **Should `Community 1` be split into smaller, more focused modules?**
  _Cohesion score 0.11 - nodes in this community are weakly interconnected._
- **Should `Community 2` be split into smaller, more focused modules?**
  _Cohesion score 0.08 - nodes in this community are weakly interconnected._
- **Should `Community 3` be split into smaller, more focused modules?**
  _Cohesion score 0.07 - nodes in this community are weakly interconnected._