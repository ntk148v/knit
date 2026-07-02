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
Sleep 200ms
Enter
Sleep 2s

# ── Installed tab: navigate and open detail ──────────────────────
Down
Sleep 400ms
Down
Sleep 300ms
Down
Sleep 300ms
Enter
Sleep 1.2s
Escape
Sleep 400ms

# ── Installed tab: search filter ─────────────────────────────────
Type "/"
Sleep 200ms
Type "design"
Sleep 1.5s
Escape
Sleep 500ms

# ── Switch to Discover tab, search ───────────────────────────────
Tab
Sleep 600ms
Type "/"
Sleep 200ms
Type "review"
Sleep 2s

# ── Switch to Sources tab, open detail ───────────────────────────
Tab
Sleep 600ms
Down
Sleep 300ms
Down
Sleep 300ms
Enter
Sleep 2s
Escape
Sleep 400ms

# ── Quit ─────────────────────────────────────────────────────────
Type "q"
Sleep 500ms
