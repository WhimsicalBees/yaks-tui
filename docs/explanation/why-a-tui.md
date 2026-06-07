# Why a TUI

yaks is a CLI for tracking work as a tree of nested "yaks." The CLI is great for
quick adds and scripted use, but browsing and triaging a large tree means
running `yx list`, reading IDs, and typing follow-up commands. That loop is slow
when you're working through many yaks at once.

yaks-tui exists to make browsing and triage *interactive*: move with the
keyboard, see rendered detail as you go, change state with a single key. It
deliberately **composes existing tools** rather than reimplementing them:

- It shells out to `yx` for every operation — yaks-tui is a front end, not a
  second source of truth. Anything `yx` can do to the data, it remains the
  authority on.
- It renders markdown with [glamour](https://github.com/charmbracelet/glamour),
  the same engine behind `glow`, in-process.
- It delegates fuzzy finding to [`fzf`](https://github.com/junegunn/fzf) rather
  than building a matcher.

The result is a thin, focused layer: the parts unique to "browsing a yak tree"
are ours; everything else is borrowed from tools that already do it well.
