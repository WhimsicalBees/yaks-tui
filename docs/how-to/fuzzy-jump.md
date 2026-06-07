# Jump to a yak with fuzzy find

When the tree is large, jump straight to a yak by name instead of scrolling.

This feature needs [`fzf`](https://github.com/junegunn/fzf) on your PATH. If
it's not installed, pressing `/` shows a friendly message instead.

1. Press `/`. The terminal hands off to fzf, listing the currently visible yaks.
2. Type to filter, use the arrow keys to highlight a match.
3. Press `enter` to select. The cursor jumps to that yak in the tree.
4. Press `esc` in fzf to cancel — you return to the tree unchanged.
