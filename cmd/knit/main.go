package main

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ntk148v/knit/internal/app"
	"github.com/ntk148v/knit/internal/skills"
)

func main() {
	ctx := context.Background()
	client := skills.NewNpxClient()
	m := app.New(client)
	p := tea.NewProgram(m, tea.WithContext(ctx), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
