package main

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"yaks-tui/internal/ui"
	"yaks-tui/internal/yaks"
)

func main() {
	if _, err := yaks.BinaryAvailable(); err != nil {
		fmt.Fprintln(os.Stderr, "yaks-tui: `yx` not found on PATH.")
		fmt.Fprintln(os.Stderr, "Install yaks first: https://github.com/mattwynne/yaks")
		os.Exit(1)
	}

	client := yaks.NewClient(yaks.ExecRunner{})

	ok, err := client.RepoInitialized(context.Background())
	if !ok {
		fmt.Fprintln(os.Stderr, "yaks-tui: no yaks repo here (or `.yaks` not gitignored).")
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %s\n", err)
		}
		fmt.Fprintln(os.Stderr, "Start one with:  yx add \"my first yak\"")
		os.Exit(1)
	}

	p := tea.NewProgram(ui.New(client), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "yaks-tui: %v\n", err)
		os.Exit(1)
	}
}
