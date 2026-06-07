# Edit a yak's context

Edit the markdown body of a yak without leaving the TUI.

1. Move the cursor to the yak you want to edit.
2. Press `e`. The right pane becomes an editable text area pre-filled with the
   yak's current context (empty if it has none).
3. Edit the text. All normal typing and cursor keys work.
4. Press `ctrl+s` to save. The pane returns to rendered markdown showing your
   new body.

To discard your changes instead, press `esc` — nothing is written.

If the save fails (for example, `yx` returns an error), the message appears in
the status bar and you stay in the editor, so your edits aren't lost.
