// Package playlist provides playlist management functionality
package playlist

import (
	"math/rand"
	"time"

	"cli-player/internal/metadata"
)

// RepeatMode represents the repeat mode
type RepeatMode int

const (
	RepeatNone RepeatMode = iota
	RepeatOne
	RepeatAll
)

// Playlist represents a music playlist
type Playlist struct {
	Tracks     []*metadata.Track
	CurrentIdx int
	Repeat     RepeatMode
	Shuffle    bool
	shuffled   []int
	rand       *rand.Rand
}

// New creates a new empty playlist
func New() *Playlist {
	return &Playlist{
		Tracks:     make([]*metadata.Track, 0),
		CurrentIdx: -1,
		rand:       rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Add appends tracks to the playlist (skips duplicates)
func (p *Playlist) Add(tracks ...*metadata.Track) {
	for _, track := range tracks {
		// Check if track already exists (by path)
		if p.contains(track.Path) {
			continue
		}
		p.Tracks = append(p.Tracks, track)
	}
	if p.CurrentIdx < 0 && len(p.Tracks) > 0 {
		p.CurrentIdx = 0
	}
}

// contains checks if a track with the given path already exists
func (p *Playlist) contains(path string) bool {
	for _, t := range p.Tracks {
		if t.Path == path {
			return true
		}
	}
	return false
}

// Remove removes a track at the given index
func (p *Playlist) Remove(idx int) {
	if idx < 0 || idx >= len(p.Tracks) {
		return
	}

	p.Tracks = append(p.Tracks[:idx], p.Tracks[idx+1:]...)

	if p.CurrentIdx >= len(p.Tracks) {
		p.CurrentIdx = len(p.Tracks) - 1
	} else if idx < p.CurrentIdx {
		p.CurrentIdx--
	}

	if len(p.Tracks) == 0 {
		p.CurrentIdx = -1
	}
}

// Current returns the current track
func (p *Playlist) Current() *metadata.Track {
	if p.CurrentIdx < 0 || p.CurrentIdx >= len(p.Tracks) {
		return nil
	}
	return p.Tracks[p.CurrentIdx]
}

// Next moves to the next track and returns it
func (p *Playlist) Next() *metadata.Track {
	if len(p.Tracks) == 0 {
		return nil
	}

	if p.Shuffle {
		return p.nextShuffled()
	}

	if p.Repeat == RepeatOne {
		return p.Current()
	}

	p.CurrentIdx++
	if p.CurrentIdx >= len(p.Tracks) {
		if p.Repeat == RepeatAll {
			p.CurrentIdx = 0
		} else {
			p.CurrentIdx = len(p.Tracks) - 1
			return nil
		}
	}

	return p.Current()
}

// Prev moves to the previous track and returns it
func (p *Playlist) Prev() *metadata.Track {
	if len(p.Tracks) == 0 {
		return nil
	}

	if p.Shuffle {
		return p.prevShuffled()
	}

	if p.Repeat == RepeatOne {
		return p.Current()
	}

	p.CurrentIdx--
	if p.CurrentIdx < 0 {
		if p.Repeat == RepeatAll {
			p.CurrentIdx = len(p.Tracks) - 1
		} else {
			p.CurrentIdx = 0
		}
	}

	return p.Current()
}

// JumpTo jumps to a specific track by index
func (p *Playlist) JumpTo(idx int) *metadata.Track {
	if idx < 0 || idx >= len(p.Tracks) {
		return nil
	}
	p.CurrentIdx = idx
	return p.Current()
}

// ToggleShuffle toggles shuffle mode
func (p *Playlist) ToggleShuffle() {
	p.Shuffle = !p.Shuffle
	if p.Shuffle {
		p.generateShuffled()
	}
}

// ToggleRepeat cycles through repeat modes
func (p *Playlist) ToggleRepeat() {
	switch p.Repeat {
	case RepeatNone:
		p.Repeat = RepeatAll
	case RepeatAll:
		p.Repeat = RepeatOne
	case RepeatOne:
		p.Repeat = RepeatNone
	}
}

// Clear removes all tracks from the playlist
func (p *Playlist) Clear() {
	p.Tracks = make([]*metadata.Track, 0)
	p.CurrentIdx = -1
	p.shuffled = nil
}

// Len returns the number of tracks
func (p *Playlist) Len() int {
	return len(p.Tracks)
}

func (p *Playlist) generateShuffled() {
	p.shuffled = make([]int, len(p.Tracks))
	for i := range p.shuffled {
		p.shuffled[i] = i
	}
	p.rand.Shuffle(len(p.shuffled), func(i, j int) {
		p.shuffled[i], p.shuffled[j] = p.shuffled[j], p.shuffled[i]
	})
}

func (p *Playlist) nextShuffled() *metadata.Track {
	if len(p.shuffled) == 0 {
		p.generateShuffled()
	}

	// Find current position in shuffled list
	currentPos := -1
	for i, idx := range p.shuffled {
		if idx == p.CurrentIdx {
			currentPos = i
			break
		}
	}

	nextPos := currentPos + 1
	if nextPos >= len(p.shuffled) {
		if p.Repeat == RepeatAll {
			nextPos = 0
			p.generateShuffled()
		} else {
			return nil
		}
	}

	p.CurrentIdx = p.shuffled[nextPos]
	return p.Current()
}

func (p *Playlist) prevShuffled() *metadata.Track {
	if len(p.shuffled) == 0 {
		p.generateShuffled()
	}

	// Find current position in shuffled list
	currentPos := -1
	for i, idx := range p.shuffled {
		if idx == p.CurrentIdx {
			currentPos = i
			break
		}
	}

	prevPos := currentPos - 1
	if prevPos < 0 {
		if p.Repeat == RepeatAll {
			prevPos = len(p.shuffled) - 1
		} else {
			prevPos = 0
		}
	}

	p.CurrentIdx = p.shuffled[prevPos]
	return p.Current()
}
