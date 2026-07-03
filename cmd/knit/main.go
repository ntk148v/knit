package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ntk148v/knit/internal/app"
	"github.com/ntk148v/knit/internal/skills"
)

var version = "dev"

type commandConfig struct {
	Mode     string
	LockFile string
	Global   bool
}

func parseArgs(args []string) (commandConfig, error) {
	if len(args) == 0 {
		return commandConfig{Mode: "tui"}, nil
	}
	if args[0] != "sync" {
		return commandConfig{}, fmt.Errorf("unknown command %q", args[0])
	}
	fs := flag.NewFlagSet("sync", flag.ContinueOnError)
	lockFile := fs.String("f", "", "skills lock file")
	global := fs.Bool("g", false, "install globally")
	if err := fs.Parse(args[1:]); err != nil {
		return commandConfig{}, err
	}
	if strings.TrimSpace(*lockFile) == "" {
		return commandConfig{}, fmt.Errorf("sync requires -f <lock-file>")
	}
	return commandConfig{Mode: "sync", LockFile: *lockFile, Global: *global}, nil
}

func main() {
	cfg, err := parseArgs(os.Args[1:])
	if err != nil {
		if !errors.Is(err, flag.ErrHelp) {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		return
	}

	ctx := context.Background()
	client := skills.NewNpxClient()
	if cfg.Mode == "sync" {
		if err := client.SyncFromLock(ctx, cfg.LockFile, cfg.Global); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	m := app.New(client)
	p := tea.NewProgram(m, tea.WithContext(ctx), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
