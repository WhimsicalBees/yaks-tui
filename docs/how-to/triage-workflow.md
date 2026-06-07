# Triage your yaks

Move yaks through their states without leaving the keyboard. Put the cursor on
a yak and press one key:

| Key | Sets state to |
|-----|---------------|
| `t` | todo |
| `w` | wip (work in progress) |
| `b` | blocked |
| `d` | done |

After each change the tree reloads and the cursor stays on the same yak (matched
by its stable id), so you can triage several in a row.

To reload the tree manually — for example after another process ran `yx sync` —
press `r`.
