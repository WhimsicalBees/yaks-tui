# CLI preconditions

yaks-tui checks these at startup and exits with a message if they aren't met.

| Requirement | Why | If missing |
|-------------|-----|------------|
| `yx` on PATH | all data comes from the yaks CLI | exits: "`yx` not found on PATH", with an install link |
| A yaks repo in the current directory, with `.yaks` gitignored | yaks-tui reads the repo via `yx list` | exits: "no yaks repo here (or `.yaks` not gitignored)", with a `yx add` hint |
| `fzf` on PATH | the `/` fuzzy jump hands off to fzf | optional; `/` shows a friendly message if absent |

Markdown is rendered in-process with
[glamour](https://github.com/charmbracelet/glamour); no external renderer such
as `glow` is required.
