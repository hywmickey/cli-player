package views

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"cli-player/internal/metadata"
	"cli-player/internal/playlist"
)

var (
	playlistItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#D1D5DB"))

	playlistSelectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#7C3AED")).
				Bold(true)

	playlistPlayingStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#10B981")).
				Bold(true)

	playlistBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#4B5563")).
				Padding(0, 1)
)

// PlaylistItem represents an item in the playlist
type PlaylistItem struct {
	Track    *metadata.Track
	IsPlaying bool
	Idx      int
}

// FilterValue implements list.Item interface
func (p PlaylistItem) FilterValue() string {
	return p.Track.Title
}

// Title implements list.DefaultItem interface
func (p PlaylistItem) Title() string {
	prefix := ""
	if p.IsPlaying {
		prefix = "▶ "
	}
	return fmt.Sprintf("%s%d. %s", prefix, p.Idx+1, p.Track.Title)
}

// Description implements list.DefaultItem interface
func (p PlaylistItem) Description() string {
	artist := p.Track.Artist
	if artist == "" {
		artist = "Unknown Artist"
	}
	duration := metadata.FormatDuration(p.Track.Duration)
	return fmt.Sprintf("%s • %s", artist, duration)
}

// PlaylistView renders the playlist view
type PlaylistView struct {
	list     list.Model
	playlist *playlist.Playlist
	width    int
	height   int
	focused  bool
}

// NewPlaylistView creates a new playlist view
func NewPlaylistView(pl *playlist.Playlist) *PlaylistView {
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), 20, 10)
	l.Title = "Playlist"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)

	return &PlaylistView{
		list:     l,
		playlist: pl,
	}
}

// SetSize sets the view dimensions
func (v *PlaylistView) SetSize(width, height int) {
	v.width = width
	v.height = height
	v.list.SetSize(width-2, height-4)
}

// SetFocused sets focus state
func (v *PlaylistView) SetFocused(focused bool) {
	v.focused = focused
}

// Update handles messages
func (v *PlaylistView) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	v.list, cmd = v.list.Update(msg)
	return cmd
}

// View renders the playlist view
func (v *PlaylistView) View() string {
	// Update items
	items := make([]list.Item, v.playlist.Len())
	current := v.playlist.Current()
	for i := 0; i < v.playlist.Len(); i++ {
		track := v.playlist.Tracks[i]
		items[i] = PlaylistItem{
			Track:     track,
			IsPlaying: current != nil && track.Path == current.Path,
			Idx:       i,
		}
	}
	v.list.SetItems(items)

	// Style the title
	if v.focused {
		v.list.Styles.Title = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7C3AED")).
			Bold(true).
			Padding(0, 1)
	} else {
		v.list.Styles.Title = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280")).
			Padding(0, 1)
	}

	return playlistBorderStyle.Render(v.list.View())
}

// SelectedIndex returns the selected index
func (v *PlaylistView) SelectedIndex() int {
	return v.list.Index()
}

// SetSelectedIndex sets the selected index
func (v *PlaylistView) SetSelectedIndex(idx int) {
	v.list.Select(idx)
}
