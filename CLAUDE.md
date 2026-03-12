# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Run

```bash
# Build
go build -o cli-player .

# Run with directory or file path
./cli-player /path/to/audio/files
./cli-player /path/to/audio.mp3
./cli-player .  # current directory
```

## Architecture

The project is a Go-based TUI music player with a modular architecture:

### Core Packages

- **`internal/player`**: Audio playback engine using the `beep` library
  - Manages audio decoding (MP3, FLAC, OGG, WAV)
  - Handles playback state (playing/paused/stopped), seeking, volume
  - Speaker runs at 44100Hz sample rate, uses resampling for format matching
  - Important: `speaker.Lock()` must be used before accessing streamer position/state

- **`internal/playlist`**: Playlist management with shuffle and repeat modes
  - Tracks are `*metadata.Track` references
  - Supports `RepeatNone`, `RepeatOne`, `RepeatAll` modes
  - Shuffle uses Fisher-Yates algorithm with a shuffled index array

- **`internal/metadata`**: Audio file metadata extraction
  - Uses `dhowden/tag` library for ID3/vorbis/flac metadata
  - Falls back to filename for title if no metadata
  - Duration is calculated by fully decoding the audio file

- **`internal/lyrics`**: LRC lyrics parsing and synchronization
  - Parses `[mm:ss.xx]` or `[mm:ss:xx]` timestamp format
  - Handles multiple timestamps per line
  - Auto-syncs to current playback position

- **`internal/tui`**: Bubbletea-based terminal UI
  - `app.go`: Main model handling keyboard input and orchestration
  - `views/` package contains individual view components
  - Two-panel layout: left (player + playlist), right (browser or lyrics)

### TUI Structure

The TUI uses bubbletea's Elm architecture:
- `Init()` returns initial commands
- `Update(msg)` handles `tea.KeyMsg`, `tea.WindowSizeMsg`, `TickMsg`, and `TrackEndMsg`
- `View()` renders the UI using lipgloss for styling

Views are in `internal/tui/views/`:
- `PlayerView`: Shows current track info, progress bar, volume
- `PlaylistView`: Uses bubbles/list component for track list
- `BrowserView`: File browser for selecting audio files

### Key Concepts

- **Focus management**: `FocusView` enum (Player, Playlist, Browser, Lyrics) determines which view receives keyboard input
- **Right panel modes**: `RightPanelLyrics` or `RightPanelBrowser` toggles right panel content
- **Lyrics sync mode**: Auto-sync by default, manual mode allows navigation with `↑/↓` and `Enter` to jump
- **Track end detection**: Checked in `TickMsg` (100ms interval) via `player.State()` == `StateStopped`

### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `q`, `Ctrl+C` | Quit |
| `Space` | Play/Pause |
| `n`, `p` | Next/Prev track |
| `</>` | Seek 10 seconds backward/forward |
| `+/-` | Volume up/down |
| `s`, `r` | Toggle shuffle/repeat |
| `l` | Toggle lyrics/browser panel |
| `Tab` | Cycle focus between views |
| `Enter` | Play selected (playlist), Open/Add (browser) |

## Notes

- Lyrics are looked up by replacing audio file extension with `.lrc` or `.LRC`
- Volume uses logarithmic scale: 0dB at 1.0, -6dB at 0.5, -20dB at 0.1
- The binary is ignored by git (see `.gitignore`)
