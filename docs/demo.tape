# yaks-tui demo. Regenerate with: make demo
Output docs/demo.gif

Set Shell "bash"
Set FontSize 18
Set Width 1200
Set Height 700
Set Padding 20

# Set up a throwaway yaks repo and launch the TUI — all hidden, so the
# recording starts on the running TUI, not the shell scrollback.
Hide
Type "cd $(mktemp -d) && git init -q && echo '.yaks' > .gitignore" Enter
Type "yx add 'Ship yaks-tui v1.1'" Enter
Type "yx add 'Write docs' --under 'Ship yaks-tui v1.1'" Enter
Type "yx add 'Record demo' --under 'Ship yaks-tui v1.1'" Enter
Type "yx add 'Cut release'" Enter
Type "clear" Enter
Type "yaks-tui" Enter
Sleep 1.5s
Show

Sleep 1.5s

# Navigate the tree.
Type "j" Sleep 600ms
Type "j" Sleep 600ms
Type "l" Sleep 800ms

# Triage: mark wip.
Type "w" Sleep 1s

# Edit context inline.
Type "e" Sleep 800ms
Type "Recorded with VHS." Sleep 900ms
Ctrl+S Sleep 1.5s

# Fuzzy jump.
Type "/" Sleep 800ms
Type "Cut" Sleep 800ms
Enter Sleep 1.5s

# Quit — hidden, so the GIF ends on the TUI, not the shell.
Hide
Type "q"
