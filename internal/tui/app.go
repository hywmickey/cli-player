// Package tui provides the terminal user interface
package tui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"cli-player/internal/lyrics"
	"cli-player/internal/metadata"
	"cli-player/internal/player"
	"cli-player/internal/playlist"
	"cli-player/internal/tui/views"
)

// FocusView represents which view is focused
type FocusView int

const (
	FocusPlayer FocusView = iota
	FocusPlaylist
	FocusBrowser
	FocusLyrics
)

// RightPanelMode represents what to display in right panel
type RightPanelMode int

const (
	RightPanelLyrics RightPanelMode = iota
	RightPanelBrowser
)

// TickMsg is sent periodically for UI updates
type TickMsg time.Time

// TrackEndMsg is sent when a track ends
type TrackEndMsg struct{}

// Model represents the main TUI model
type Model struct {
	player         *player.Player
	playlist       *playlist.Playlist
	playerView     *views.PlayerView
	playlistView   *views.PlaylistView
	browserView    *views.BrowserView
	currentLyrics  *lyrics.Lyrics

	focused        FocusView
	rightPanelMode RightPanelMode
	width          int
	height         int
	err            error
	quitting       bool

	// Lyrics navigation
	lyricsSelectedIdx int    // Selected lyric line index
	lyricsManualMode  bool   // Whether user is manually navigating lyrics
	lyricsScrollOffset int   // Scroll offset for displaying lyrics
}

// New creates a new TUI model
func New(p *player.Player, pl *playlist.Playlist, startPath string) *Model {
	m := &Model{
		player:         p,
		playlist:       pl,
		playerView:     views.NewPlayerView(),
		playlistView:   views.NewPlaylistView(pl),
		browserView:    views.NewBrowserView(startPath),
		focused:        FocusBrowser,
		rightPanelMode: RightPanelBrowser,
	}

	// Set up track end callback
	p.OnTrackEnd(func() {
		// This will be handled through the main loop
	})

	return m
}

// Init initializes the model
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(),
	)
}

// Update handles messages
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateViewSizes()
		return m, nil

	case TickMsg:
		// Check for track end
		if m.player.State() == player.StateStopped && m.playlist.Current() != nil {
			// Auto-advance to next track
			m.nextTrack()
		}
		return m, tickCmd()

	case TrackEndMsg:
		m.nextTrack()
		return m, nil
	}

	// Update focused view
	switch m.focused {
	case FocusPlaylist:
		cmd := m.playlistView.Update(msg)
		cmds = append(cmds, cmd)
	case FocusBrowser:
		cmd := m.browserView.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Global keys
	switch msg.String() {
	case "q", "ctrl+c":
		m.quitting = true
		return m, tea.Quit

	case "tab":
		m.cycleFocus()
		return m, nil

	case " ":
		m.togglePause()
		return m, nil

	case "n":
		m.nextTrack()
		return m, nil

	case "p":
		m.prevTrack()
		return m, nil

	case "+", "=":
		m.adjustVolume(0.1)
		return m, nil

	case "-", "_":
		m.adjustVolume(-0.1)
		return m, nil

	case "s":
		m.playlist.ToggleShuffle()
		return m, nil

	case "r":
		m.playlist.ToggleRepeat()
		return m, nil

	case "l":
		// Toggle right panel between lyrics and browser
		hasLyrics := m.currentLyrics != nil && len(m.currentLyrics.Lines) > 0
		hasTrack := m.playlist.Current() != nil

		if m.rightPanelMode == RightPanelBrowser {
			// Switch to lyrics only if there's a track with lyrics
			if hasTrack && hasLyrics {
				m.rightPanelMode = RightPanelLyrics
				m.focused = FocusLyrics
				m.lyricsManualMode = false
			}
		} else {
			// Switch to browser
			m.rightPanelMode = RightPanelBrowser
			m.focused = FocusBrowser
		}
		return m, nil

	case "up", "k":
		// Navigate lyrics up when in lyrics panel
		if m.rightPanelMode == RightPanelLyrics && m.focused == FocusLyrics {
			m.navigateLyricsUp()
			return m, nil
		}

	case "down", "j":
		// Navigate lyrics down when in lyrics panel
		if m.rightPanelMode == RightPanelLyrics && m.focused == FocusLyrics {
			m.navigateLyricsDown()
			return m, nil
		}

	case "enter":
		// Jump to selected lyric line
		if m.rightPanelMode == RightPanelLyrics && m.focused == FocusLyrics {
			m.jumpToSelectedLyric()
			return m, nil
		}

	case ">", ".":
		// Fast forward 10 seconds
		m.player.SeekForward(10 * time.Second)
		return m, nil

	case "<", ",":
		// Rewind 10 seconds
		m.player.SeekBackward(10 * time.Second)
		return m, nil
	}

	// View-specific keys
	switch m.focused {
	case FocusPlaylist:
		return m.handlePlaylistKey(msg)
	case FocusBrowser:
		if m.rightPanelMode == RightPanelBrowser {
			return m.handleBrowserKey(msg)
		}
	case FocusLyrics:
		return m.handleLyricsKey(msg)
	}

	return m, nil
}

func (m *Model) handlePlaylistKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		idx := m.playlistView.SelectedIndex()
		if idx >= 0 && idx < m.playlist.Len() {
			m.playlist.JumpTo(idx)
			m.playCurrent()
		}
		return m, nil

	case "delete", "backspace":
		idx := m.playlistView.SelectedIndex()
		m.playlist.Remove(idx)
		return m, nil
	}

	cmd := m.playlistView.Update(msg)
	return m, cmd
}

func (m *Model) handleBrowserKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter", "right":
		item := m.browserView.SelectedItem()
		if item.IsDir {
			m.browserView.NavigateDown(item.Path)
		} else if item.IsAudio {
			m.addToPlaylist(item.Path)
		}
		return m, nil

	case "left", "backspace":
		m.browserView.NavigateUp()
		return m, nil

	case "a":
		// Add selected file to playlist
		item := m.browserView.SelectedItem()
		if item.IsAudio {
			m.addToPlaylist(item.Path)
		}
		return m, nil
	}

	cmd := m.browserView.Update(msg)
	return m, cmd
}

func (m *Model) handleLyricsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		m.navigateLyricsUp()
		return m, nil

	case "down", "j":
		m.navigateLyricsDown()
		return m, nil

	case "enter":
		m.jumpToSelectedLyric()
		return m, nil

	case "esc":
		// Exit manual mode, return to auto-sync
		m.lyricsManualMode = false
		return m, nil
	}

	return m, nil
}

func (m *Model) navigateLyricsUp() {
	if m.currentLyrics == nil || len(m.currentLyrics.Lines) == 0 {
		return
	}

	m.lyricsManualMode = true
	if m.lyricsSelectedIdx > 0 {
		m.lyricsSelectedIdx--
	}
}

func (m *Model) navigateLyricsDown() {
	if m.currentLyrics == nil || len(m.currentLyrics.Lines) == 0 {
		return
	}

	m.lyricsManualMode = true
	if m.lyricsSelectedIdx < len(m.currentLyrics.Lines)-1 {
		m.lyricsSelectedIdx++
	}
}

func (m *Model) jumpToSelectedLyric() {
	if m.currentLyrics == nil || len(m.currentLyrics.Lines) == 0 {
		return
	}

	if m.lyricsSelectedIdx < 0 || m.lyricsSelectedIdx >= len(m.currentLyrics.Lines) {
		return
	}

	line := m.currentLyrics.Lines[m.lyricsSelectedIdx]
	m.player.Seek(line.Time)
	m.lyricsManualMode = false // Return to auto-sync after jumping
}

func (m *Model) View() string {
	if m.quitting {
		return "\n  Thanks for using the Music Player!  \n\n"
	}

	// Render views
	playerContent := m.playerView.Render(
		m.playlist.Current(),
		m.player.State(),
		m.player.Position(),
		m.player.Length(),
		m.player.Volume(),
		m.playlist.Repeat,
		m.playlist.Shuffle,
	)

	playlistContent := m.playlistView.View()

	// Layout
	leftPanel := lipgloss.JoinVertical(lipgloss.Left, playerContent, playlistContent)

	// Right panel - lyrics or browser
	// Show browser when: no current track, no lyrics, or explicitly in browser mode
	var rightPanel string
	hasLyrics := m.currentLyrics != nil && len(m.currentLyrics.Lines) > 0
	hasTrack := m.playlist.Current() != nil

	if m.rightPanelMode == RightPanelBrowser || !hasTrack || !hasLyrics {
		rightPanel = m.browserView.View()
	} else {
		rightPanel = m.renderLyricsPanel()
	}

	// Calculate widths
	leftWidth := m.width / 3
	if leftWidth < 40 {
		leftWidth = 40
	}
	rightWidth := m.width - leftWidth - 4

	// Apply width constraints
	leftPanel = lipgloss.NewStyle().Width(leftWidth).Render(leftPanel)
	rightPanel = lipgloss.NewStyle().Width(rightWidth).Render(rightPanel)

	// Create main layout
	mainLayout := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, "  ", rightPanel)

	// Help bar
	helpBar := m.renderHelpBar()

	return lipgloss.JoinVertical(lipgloss.Left, mainLayout, helpBar)
}

func (m *Model) renderLyricsPanel() string {
	width := m.width / 2
	if width < 40 {
		width = 40
	}
	height := m.height - 6

	// Styles
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7C3AED")).
		Bold(true).
		Padding(1, 2)

	lyricsCurrentStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#A78BFA")).
		Bold(true)

	lyricsSelectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#10B981")).
		Bold(true).
		Background(lipgloss.Color("#1F2937"))

	lyricsOtherStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280"))

	lyricsPastStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#4B5563"))

	emptyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#4B5563")).
		Italic(true).
		Align(lipgloss.Center)

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#4B5563")).
		Padding(1, 2).
		Width(width).
		Height(height)

	var content strings.Builder

	// Title with mode indicator
	title := "♪ Lyrics ♪"
	if m.lyricsManualMode {
		title = "♪ Lyrics (Manual - Enter to jump, Esc to auto) ♪"
	}
	content.WriteString(titleStyle.Render(title))
	content.WriteString("\n\n")

	if m.currentLyrics == nil || len(m.currentLyrics.Lines) == 0 {
		content.WriteString(emptyStyle.Render("No lyrics available\n\nPlace .lrc file with same name as audio file"))
		return borderStyle.Render(content.String())
	}

	// Get current position
	position := m.player.Position()

	// Determine current line index based on position
	currentLineIdx := -1
	for i, line := range m.currentLyrics.Lines {
		if line.Time <= position {
			currentLineIdx = i
		} else {
			break
		}
	}

	// If not in manual mode, sync selected index with current playback
	if !m.lyricsManualMode {
		m.lyricsSelectedIdx = currentLineIdx
	}

	// Calculate scroll offset to show selected line in the middle
	visibleLines := height - 6 // Account for title and padding
	halfVisible := visibleLines / 2

	scrollOffset := m.lyricsSelectedIdx - halfVisible
	if scrollOffset < 0 {
		scrollOffset = 0
	}
	if scrollOffset > len(m.currentLyrics.Lines)-visibleLines {
		scrollOffset = len(m.currentLyrics.Lines) - visibleLines
	}
	if scrollOffset < 0 {
		scrollOffset = 0
	}

	endIdx := scrollOffset + visibleLines
	if endIdx > len(m.currentLyrics.Lines) {
		endIdx = len(m.currentLyrics.Lines)
	}

	// Render visible lines
	for i := scrollOffset; i < endIdx; i++ {
		line := m.currentLyrics.Lines[i]
		var styledLine string

		if i == m.lyricsSelectedIdx && m.lyricsManualMode {
			// Selected line (in manual mode)
			styledLine = lyricsSelectedStyle.Render("▶ " + line.Text)
		} else if i == currentLineIdx {
			// Current playing line (highlighted)
			styledLine = lyricsCurrentStyle.Render("♪ " + line.Text + " ♪")
		} else if i < currentLineIdx {
			// Past lyrics (dimmed)
			styledLine = lyricsPastStyle.Render("  " + line.Text)
		} else {
			// Future lyrics
			styledLine = lyricsOtherStyle.Render("  " + line.Text)
		}
		content.WriteString(styledLine)
		content.WriteString("\n")
	}

	return borderStyle.Render(content.String())
}

func (m *Model) renderHelpBar() string {
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280")).
		Padding(1, 2)

	var helps []string

	hasLyrics := m.currentLyrics != nil && len(m.currentLyrics.Lines) > 0
	hasTrack := m.playlist.Current() != nil

	if m.focused == FocusBrowser || (m.rightPanelMode == RightPanelBrowser || !hasTrack || !hasLyrics) {
		helps = []string{
			"Enter: Open/Add",
			"Backspace: Up",
			"a: Add to playlist",
		}
	} else if m.focused == FocusLyrics {
		helps = []string{
			"↑/↓: Navigate lyrics",
			"Enter: Jump to line",
			"Esc: Auto-sync mode",
		}
	}

	if m.focused == FocusPlaylist {
		helps = append(helps, "Enter: Play", "Del: Remove")
	}

	globalHelp := []string{
		"l: Lyrics/Browser",
		"Tab: Switch view",
		"Space: Play/Pause",
		"n/p: Next/Prev",
		"</>: Seek 10s",
		"q: Quit",
	}

	allHelp := append(helps, globalHelp...)
	return helpStyle.Render(strings.Join(allHelp, "  •  "))
}

func (m *Model) updateViewSizes() {
	playerHeight := 12
	playlistHeight := m.height - playerHeight - 6
	browserHeight := m.height - 6

	leftWidth := m.width / 3
	if leftWidth < 40 {
		leftWidth = 40
	}
	rightWidth := m.width - leftWidth - 4

	m.playerView.SetSize(leftWidth-4, playerHeight)
	m.playlistView.SetSize(leftWidth-4, playlistHeight)
	m.browserView.SetSize(rightWidth-4, browserHeight)
}

func (m *Model) cycleFocus() {
	hasLyrics := m.currentLyrics != nil && len(m.currentLyrics.Lines) > 0
	hasTrack := m.playlist.Current() != nil

	switch m.focused {
	case FocusPlayer:
		m.focused = FocusPlaylist
	case FocusPlaylist:
		// Can only focus lyrics if there's a track with lyrics and in lyrics mode
		if m.rightPanelMode == RightPanelLyrics && hasTrack && hasLyrics {
			m.focused = FocusLyrics
		} else if m.rightPanelMode == RightPanelBrowser {
			m.focused = FocusBrowser
		} else {
			m.focused = FocusPlayer
		}
	case FocusLyrics:
		m.focused = FocusPlayer
	case FocusBrowser:
		m.focused = FocusPlayer
	}

	m.playerView.SetFocused(m.focused == FocusPlayer)
	m.playlistView.SetFocused(m.focused == FocusPlaylist)
	m.browserView.SetFocused(m.focused == FocusBrowser)
}

func (m *Model) togglePause() {
	if m.player.State() == player.StatePlaying {
		m.player.Pause()
	} else if m.player.State() == player.StatePaused {
		m.player.Resume()
	} else if m.playlist.Current() != nil {
		m.playCurrent()
	}
}

func (m *Model) playCurrent() {
	track := m.playlist.Current()
	if track == nil {
		return
	}

	if err := m.player.Play(track.Path); err != nil {
		m.err = err
	}

	// Load lyrics for the track
	m.loadLyrics(track.Path)
}

func (m *Model) loadLyrics(audioPath string) {
	m.currentLyrics = nil
	m.lyricsSelectedIdx = 0
	m.lyricsManualMode = false
	lrcPath := lyrics.FindLRCFile(audioPath)
	if lrcPath == "" {
		return
	}

	l, err := lyrics.Parse(lrcPath)
	if err != nil {
		return
	}
	m.currentLyrics = l

	// Auto-switch to lyrics panel when lyrics are found
	if len(l.Lines) > 0 {
		m.rightPanelMode = RightPanelLyrics
		m.focused = FocusLyrics
	}
}

func (m *Model) nextTrack() {
	track := m.playlist.Next()
	if track != nil {
		if err := m.player.Play(track.Path); err != nil {
			m.err = err
		}
		m.loadLyrics(track.Path)
	} else {
		m.player.Stop()
		m.currentLyrics = nil
	}
}

func (m *Model) prevTrack() {
	track := m.playlist.Prev()
	if track != nil {
		if err := m.player.Play(track.Path); err != nil {
			m.err = err
		}
		m.loadLyrics(track.Path)
	}
}

func (m *Model) adjustVolume(delta float64) {
	newVol := m.player.Volume() + delta
	m.player.SetVolume(newVol)
}

func (m *Model) addToPlaylist(path string) {
	track, err := metadata.Extract(path)
	if err != nil {
		m.err = err
		return
	}

	wasEmpty := m.playlist.Len() == 0
	m.playlist.Add(track)

	// Auto-play if this is the first track
	if wasEmpty {
		m.playCurrent()
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}
