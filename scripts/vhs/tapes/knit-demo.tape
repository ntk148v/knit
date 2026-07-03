Output assets/knit-demo.gif
Set Width 1200
Set Height 800
Set FontSize 16
Set FontFamily "JetBrains Mono, Fira Code, monospace"
Set Theme "Catppuccin Mocha"
Set Padding 24
Set Framerate 30

# ── Launch knit ──────────────────────────────────────────────────
Type "./.tmp/vhs/knit"
Sleep 300ms
Enter
Sleep 3s

# ── Installed tab: prove installed skills load ───────────────────
Sleep 1.5s
Down
Sleep 900ms
Enter
Sleep 3s
Escape
Sleep 900ms

# ── Installed tab: search filter ─────────────────────────────────
Type "/"
Sleep 300ms
Type "design"
Sleep 2.5s
Escape
Sleep 900ms

# ── Discover tab: search marketplace results ─────────────────────
Tab
Sleep 1.2s
Type "/"
Sleep 300ms
Type "review"
Sleep 2.5s
Enter
Sleep 2.5s
Escape
Sleep 900ms

# ── Sources tab: open source detail with available skills ─────────
Tab
Sleep 1.2s
Down
Sleep 800ms
Down
Sleep 800ms
Enter
Sleep 3s

# ── Quit ─────────────────────────────────────────────────────────
Escape
Sleep 700ms
Type "q"
Sleep 700ms
