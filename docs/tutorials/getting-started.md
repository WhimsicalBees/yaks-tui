# Getting started

This tutorial takes you from nothing to your first triage session. By the end
you'll have browsed a yak tree, opened a yak's detail, and changed its state —
all from the keyboard.

## Before you begin

You need the `yx` (yaks) binary on your PATH. If you don't have it, install it
from [yaks](https://github.com/mattwynne/yaks) first.

## 1. Get a yaks repo

yaks-tui reads an existing yaks repo. If you don't have one, make a throwaway:

    mkdir yak-demo && cd yak-demo
    git init
    echo '.yaks' >> .gitignore
    yx add "Try yaks-tui"
    yx add "Read the docs" --under "Try yaks-tui"

## 2. Build and launch

From the yaks-tui source directory:

    make build

Then, from inside your yaks repo, run the binary:

    /path/to/yaks-tui/bin/yaks-tui

You'll see two panes: the yak tree on the left, rendered detail on the right.

## 3. Move around

Press `j` and `k` (or the arrow keys) to move the cursor up and down the tree.
Watch the detail pane on the right update as you move. Press `l` to expand a yak
that has children, `h` to collapse it.

## 4. Do your first triage

Put the cursor on a yak and press `w`. Its state changes to *wip* (work in
progress) and the tree reloads with the cursor still on your yak. Try `d` to
mark one *done*.

## 5. Quit

Press `q`. You're back at your shell.

That's a full session: browse, read, triage. Next, learn how to
[edit a yak's context](../how-to/edit-context.md) or see the complete
[keybindings](../reference/keybindings.md).
