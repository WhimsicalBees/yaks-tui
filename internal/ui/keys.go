package ui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Up       key.Binding
	Down     key.Binding
	Expand   key.Binding
	Collapse key.Binding
	Toggle   key.Binding
	Focus    key.Binding
	Wip      key.Binding
	Blocked  key.Binding
	Done     key.Binding
	Todo     key.Binding
	Edit     key.Binding
	HideDone key.Binding
	WipFocus key.Binding
	Search   key.Binding
	Find     key.Binding
	Reload   key.Binding
	Help     key.Binding
	Quit     key.Binding
}

func defaultKeys() keyMap {
	return keyMap{
		Up:       key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:     key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		Expand:   key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("→/l", "expand")),
		Collapse: key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("←/h", "collapse")),
		Toggle:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "toggle")),
		Focus:    key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "focus")),
		Wip:      key.NewBinding(key.WithKeys("w"), key.WithHelp("w", "wip")),
		Blocked:  key.NewBinding(key.WithKeys("b"), key.WithHelp("b", "blocked")),
		Done:     key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "done")),
		Todo:     key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "todo")),
		Edit:     key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit")),
		HideDone: key.NewBinding(key.WithKeys("H"), key.WithHelp("H", "hide done")),
		WipFocus: key.NewBinding(key.WithKeys("W"), key.WithHelp("W", "wip focus")),
		Search:   key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "search")),
		Find:     key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "find")),
		Reload:   key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reload")),
		Help:     key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		Quit:     key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	}
}

// ShortHelp / FullHelp satisfy bubbles/help.KeyMap.
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Collapse, k.Expand, k.Wip, k.Blocked, k.Done, k.Edit, k.Find, k.Help, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Expand, k.Collapse, k.Toggle, k.Focus},
		{k.Wip, k.Blocked, k.Done, k.Todo, k.Edit},
		{k.HideDone, k.WipFocus, k.Search, k.Find, k.Reload, k.Help, k.Quit},
	}
}
