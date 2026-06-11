# Changelog

All notable changes to this project are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Added
- Add, rename, and remove yaks from inside the TUI: `a` adds a child under the
  cursor, `A` adds a root, `R` renames in place, `x` removes (with a
  confirmation; recursive when the yak has children).
- View filters and search: `H` hides done yaks, `W` focuses wip / blocked, `f`
  searches by name, and `esc` clears active filters.

## [1.1.0]

### Added
- Inline context editor: press `e` to edit a yak's markdown body in the TUI,
  `ctrl+s` to save, `esc` to cancel.

## [1.0.0]

### Added
- Two-pane TUI: yak tree on the left, rendered markdown detail on the right.
- Keyboard navigation, fold/unfold, and pane focus switching.
- Triage: set a yak's state (todo / wip / blocked / done) with a single key.
- Fuzzy jump to a yak with `/` (via fzf).
- In-process markdown rendering with glamour.
