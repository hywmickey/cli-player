package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"cli-player/internal/metadata"
	"cli-player/internal/player"
	"cli-player/internal/playlist"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7C3AED")).
			Padding(0, 1)

	artistStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9CA3AF")).
			Padding(0, 1)

	albumStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280")).
			Padding(0, 1)

	progressBarStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#7C3AED"))

	timeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9CA3AF"))

	controlsStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#D1D5DB")).
			Align(lipgloss.Center)

	activeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#10B981"))

	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#4B5563")).
			Padding(1, 2)
)

// PlayerView renders the now playing view
type PlayerView struct {
	width   int
	height  int
	focused bool
}

// NewPlayerView creates a new player view
func NewPlayerView() *PlayerView {
	return &PlayerView{}
}

// SetSize sets the view dimensions
func (v *PlayerView) SetSize(width, height int) {
	v.width = width
	v.height = height
}

// SetFocused sets focus state
func (v *PlayerView) SetFocused(focused bool) {
	v.focused = focused
}

// Render renders the player view
func (v *PlayerView) Render(
	track *metadata.Track,
	state player.State,
	position, duration time.Duration,
	volume float64,
	repeat playlist.RepeatMode,
	shuffle bool,
) string {
	var sb strings.Builder

	// Title section
	if track != nil {
		sb.WriteString(titleStyle.Render(track.Title))
		sb.WriteString("\n")

		if track.Artist != "" {
			sb.WriteString(artistStyle.Render(track.Artist))
			sb.WriteString("\n")
		}

		if track.Album != "" {
			sb.WriteString(albumStyle.Render("Album: " + track.Album))
			sb.WriteString("\n")
		}
	} else {
		sb.WriteString(titleStyle.Render("No Track Playing"))
		sb.WriteString("\n")
	}

	sb.WriteString("\n")

	// Progress bar
	progressBar := v.renderProgressBar(position, duration)
	sb.WriteString(progressBar)
	sb.WriteString("\n")

	// State indicator
	stateStr := v.renderState(state)
	sb.WriteString(stateStr)
	sb.WriteString("\n\n")

	// Controls
	controls := v.renderControls(volume, repeat, shuffle)
	sb.WriteString(controls)

	content := sb.String()
	return borderStyle.Render(content)
}

func (v *PlayerView) renderProgressBar(position, duration time.Duration) string {
	posStr := metadata.FormatDuration(position)
	durStr := metadata.FormatDuration(duration)

	// Calculate available width for the bar
	// Account for: time strings (5 chars each), spaces (2), border padding (6)
	timeLen := len(posStr) + len(durStr) + 2 // +2 for spaces
	availableWidth := v.width - timeLen - 8   // -8 for border and padding

	if availableWidth < 10 {
		availableWidth = 10
	}
	if availableWidth > 30 {
		availableWidth = 30 // Limit max width
	}

	var progress float64
	if duration > 0 {
		progress = float64(position) / float64(duration)
	}
	if progress > 1 {
		progress = 1
	}

	filled := int(float64(availableWidth) * progress)
	if filled > availableWidth {
		filled = availableWidth
	}
	if filled < 0 {
		filled = 0
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", availableWidth-filled)

	return fmt.Sprintf("%s %s %s",
		timeStyle.Render(posStr),
		progressBarStyle.Render(bar),
		timeStyle.Render(durStr),
	)
}

func (v *PlayerView) renderState(state player.State) string {
	switch state {
	case player.StatePlaying:
		return activeStyle.Render("▶ Playing")
	case player.StatePaused:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#F59E0B")).Render("⏸ Paused")
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).Render("⏹ Stopped")
	}
}

func (v *PlayerView) renderControls(volume float64, repeat playlist.RepeatMode, shuffle bool) string {
	var controls []string

	// Shuffle
	if shuffle {
		controls = append(controls, activeStyle.Render("⤮ Shuffle"))
	} else {
		controls = append(controls, "⤮ Shuffle")
	}

	// Repeat
	switch repeat {
	case playlist.RepeatAll:
		controls = append(controls, activeStyle.Render("⟳ Repeat All"))
	case playlist.RepeatOne:
		controls = append(controls, activeStyle.Render("🔂 Repeat One"))
	default:
		controls = append(controls, "⟳ Repeat")
	}

	// Volume
	volBar := renderVolumeBar(volume)
	controls = append(controls, fmt.Sprintf("🔊 %s", volBar))

	return controlsStyle.Render(strings.Join(controls, "  "))
}

func renderVolumeBar(volume float64) string {
	width := 5
	filled := int(float64(width) * volume)
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	return bar
}
