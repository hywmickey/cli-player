package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"cli-player/internal/player"
	"cli-player/internal/playlist"
	"cli-player/internal/tui"
)

func main() {
	// Get starting directory
	startPath := "."
	if len(os.Args) > 1 {
		startPath = os.Args[1]
	}

	// Resolve to absolute path
	absPath, err := filepath.Abs(startPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving path: %v\n", err)
		os.Exit(1)
	}

	// Check if path exists
	info, err := os.Stat(absPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// If a file was provided, use its directory
	if !info.IsDir() {
		absPath = filepath.Dir(absPath)
	}

	// Initialize player
	p := player.New()
	if err := p.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing player: %v\n", err)
		os.Exit(1)
	}
	defer p.Close()

	// Create playlist
	pl := playlist.New()

	// Create and run TUI
	model := tui.New(p, pl, absPath)

	program := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		os.Exit(1)
	}
}
