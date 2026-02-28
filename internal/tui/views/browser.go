package views

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"cli-player/internal/metadata"
)

var (
	dirStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#60A5FA"))

	fileStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#D1D5DB"))

	supportedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#10B981"))

	browserBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#4B5563")).
				Padding(0, 1)
)

// FileItem represents a file or directory
type FileItem struct {
	Name     string
	Path     string
	IsDir    bool
	IsAudio  bool
}

// FilterValue implements list.Item interface
func (f FileItem) FilterValue() string {
	return f.Name
}

// Title implements list.DefaultItem interface
func (f FileItem) Title() string {
	if f.IsDir {
		return dirStyle.Render("📁 " + f.Name)
	}
	if f.IsAudio {
		return supportedStyle.Render("🎵 " + f.Name)
	}
	return fileStyle.Render(f.Name)
}

// Description implements list.DefaultItem interface
func (f FileItem) Description() string {
	if f.IsDir {
		return "Directory"
	}
	if f.IsAudio {
		return "Audio file"
	}
	return "File"
}

// BrowserView renders the file browser
type BrowserView struct {
	list     list.Model
	path     string
	width    int
	height   int
	focused  bool
}

// NewBrowserView creates a new browser view
func NewBrowserView(startPath string) *BrowserView {
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), 20, 10)
	l.Title = "Browser"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)

	v := &BrowserView{
		list: l,
		path: startPath,
	}
	v.loadDirectory()
	return v
}

// SetSize sets the view dimensions
func (v *BrowserView) SetSize(width, height int) {
	v.width = width
	v.height = height
	v.list.SetSize(width-2, height-4)
}

// SetFocused sets focus state
func (v *BrowserView) SetFocused(focused bool) {
	v.focused = focused
}

// Update handles messages
func (v *BrowserView) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	v.list, cmd = v.list.Update(msg)
	return cmd
}

// View renders the browser view
func (v *BrowserView) View() string {
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

	return browserBorderStyle.Render(v.list.View())
}

// SelectedItem returns the selected item
func (v *BrowserView) SelectedItem() FileItem {
	item, ok := v.list.SelectedItem().(FileItem)
	if !ok {
		return FileItem{}
	}
	return item
}

// NavigateUp navigates to the parent directory
func (v *BrowserView) NavigateUp() {
	parent := filepath.Dir(v.path)
	if parent != v.path {
		v.path = parent
		v.loadDirectory()
	}
}

// NavigateDown navigates into a directory
func (v *BrowserView) NavigateDown(dir string) {
	v.path = dir
	v.loadDirectory()
}

// Refresh refreshes the current directory
func (v *BrowserView) Refresh() {
	v.loadDirectory()
}

// Path returns the current path
func (v *BrowserView) Path() string {
	return v.path
}

func (v *BrowserView) loadDirectory() {
	entries, err := os.ReadDir(v.path)
	if err != nil {
		v.list.SetItems([]list.Item{})
		v.list.Title = "Browser - Error reading directory"
		return
	}

	items := make([]list.Item, 0, len(entries)+1)

	// Add parent directory entry if not at root
	if v.path != "/" {
		items = append(items, FileItem{
			Name:  "..",
			Path:  filepath.Dir(v.path),
			IsDir: true,
		})
	}

	// Separate directories and files
	var dirs []FileItem
	var files []FileItem

	for _, entry := range entries {
		name := entry.Name()
		// Skip hidden files
		if strings.HasPrefix(name, ".") {
			continue
		}

		path := filepath.Join(v.path, name)
		isDir := entry.IsDir()
		isAudio := !isDir && metadata.IsSupported(name)

		item := FileItem{
			Name:    name,
			Path:    path,
			IsDir:   isDir,
			IsAudio: isAudio,
		}

		if isDir {
			dirs = append(dirs, item)
		} else {
			files = append(files, item)
		}
	}

	// Add directories first, then files
	for _, dir := range dirs {
		items = append(items, dir)
	}
	for _, file := range files {
		items = append(items, file)
	}

	v.list.SetItems(items)
	v.list.Title = "Browser - " + v.path
}
