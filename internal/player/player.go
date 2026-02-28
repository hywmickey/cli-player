// Package player provides audio playback functionality
package player

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/effects"
	"github.com/faiface/beep/flac"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	"github.com/faiface/beep/vorbis"
	"github.com/faiface/beep/wav"
)

// State represents the playback state
type State int

const (
	StateStopped State = iota
	StatePlaying
	StatePaused
)

// Player represents an audio player
type Player struct {
	state      State
	volume     float64
	volCtrl    *effects.Volume
	streamer   beep.StreamSeekCloser
	resampled  *beep.Resampler
	ctrl       *beep.Ctrl
	file       *os.File
	mu         sync.Mutex
	done       chan struct{}
	format     beep.Format
	sampleRate beep.SampleRate
	onTrackEnd func()
}

// New creates a new audio player
func New() *Player {
	return &Player{
		state:  StateStopped,
		volume: 1.0,
		done:   make(chan struct{}),
	}
}

// Init initializes the audio system
func (p *Player) Init() error {
	// Initialize speaker with 44100Hz sample rate (standard CD quality)
	p.sampleRate = beep.SampleRate(44100)
	if err := speaker.Init(p.sampleRate, p.sampleRate.N(time.Second/10)); err != nil {
		return fmt.Errorf("failed to initialize speaker: %w", err)
	}
	return nil
}

// Play starts playing a file
func (p *Player) Play(path string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Stop any current playback
	p.stopInternal()

	// Open the file
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}

	// Decode based on extension
	ext := strings.ToLower(filepath.Ext(path))
	var streamer beep.StreamSeekCloser
	var format beep.Format

	switch ext {
	case ".mp3":
		streamer, format, err = mp3.Decode(file)
	case ".wav":
		streamer, format, err = wav.Decode(file)
	case ".flac":
		streamer, format, err = flac.Decode(file)
	case ".ogg", ".oga":
		streamer, format, err = vorbis.Decode(file)
	default:
		file.Close()
		return fmt.Errorf("unsupported format: %s", ext)
	}

	if err != nil {
		file.Close()
		return fmt.Errorf("failed to decode file: %w", err)
	}

	p.file = file
	p.streamer = streamer
	p.format = format

	// Resample to match speaker's sample rate if needed
	p.resampled = beep.Resample(4, format.SampleRate, p.sampleRate, streamer)

	p.ctrl = &beep.Ctrl{Streamer: p.resampled, Paused: false}

	// Create volume control - volume is in decibels, 0 is normal, negative is quieter
	// Convert our 0.0-1.0 scale to decibels
	volumeDb := volumeToDb(p.volume)
	p.volCtrl = &effects.Volume{
		Streamer: p.ctrl,
		Base:     2,
		Volume:   volumeDb,
		Silent:   p.volume == 0,
	}

	// Play through speaker
	speaker.Play(beep.Seq(p.volCtrl, beep.Callback(func() {
		if p.onTrackEnd != nil {
			p.onTrackEnd()
		}
	})))

	p.state = StatePlaying
	return nil
}

// Pause pauses playback
func (p *Player) Pause() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.ctrl != nil && p.state == StatePlaying {
		speaker.Lock()
		p.ctrl.Paused = true
		speaker.Unlock()
		p.state = StatePaused
	}
}

// Resume resumes playback
func (p *Player) Resume() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.ctrl != nil && p.state == StatePaused {
		speaker.Lock()
		p.ctrl.Paused = false
		speaker.Unlock()
		p.state = StatePlaying
	}
}

// TogglePause toggles between play and pause
func (p *Player) TogglePause() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.ctrl == nil {
		return
	}

	speaker.Lock()
	if p.state == StatePlaying {
		p.ctrl.Paused = true
		p.state = StatePaused
	} else if p.state == StatePaused {
		p.ctrl.Paused = false
		p.state = StatePlaying
	}
	speaker.Unlock()
}

// Stop stops playback
func (p *Player) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stopInternal()
}

func (p *Player) stopInternal() {
	if p.streamer != nil {
		speaker.Lock()
		p.ctrl.Paused = true
		speaker.Unlock()
		p.streamer.Close()
		p.streamer = nil
	}
	if p.file != nil {
		p.file.Close()
		p.file = nil
	}
	p.ctrl = nil
	p.state = StateStopped
}

// SetVolume sets the volume (0.0 to 1.0)
func (p *Player) SetVolume(v float64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if v < 0 {
		v = 0
	} else if v > 1 {
		v = 1
	}
	p.volume = v

	// Apply volume to the volume control
	if p.volCtrl != nil {
		speaker.Lock()
		p.volCtrl.Volume = volumeToDb(v)
		p.volCtrl.Silent = v == 0
		speaker.Unlock()
	}
}

// Volume returns the current volume
func (p *Player) Volume() float64 {
	return p.volume
}

// State returns the current playback state
func (p *Player) State() State {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.state
}

// Position returns the current position in seconds
func (p *Player) Position() time.Duration {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.streamer == nil {
		return 0
	}

	speaker.Lock()
	pos := p.streamer.Position()
	speaker.Unlock()

	samples := p.streamer.Len()
	if samples == 0 {
		return 0
	}

	return time.Duration(float64(pos) / float64(p.format.SampleRate) * float64(time.Second))
}

// Length returns the total duration
func (p *Player) Length() time.Duration {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.streamer == nil {
		return 0
	}

	samples := p.streamer.Len()
	return time.Duration(float64(samples) / float64(p.format.SampleRate) * float64(time.Second))
}

// Seek seeks to a position
func (p *Player) Seek(pos time.Duration) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.streamer == nil {
		return fmt.Errorf("no track loaded")
	}

	sample := int(float64(p.format.SampleRate) * float64(pos) / float64(time.Second))
	if sample < 0 {
		sample = 0
	}
	if sample > p.streamer.Len() {
		sample = p.streamer.Len()
	}

	speaker.Lock()
	err := p.streamer.Seek(sample)
	speaker.Unlock()

	return err
}

// SeekForward skips forward by the specified duration
func (p *Player) SeekForward(d time.Duration) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.streamer == nil {
		return fmt.Errorf("no track loaded")
	}

	speaker.Lock()
	currentPos := p.streamer.Position()
	speaker.Unlock()

	newPos := currentPos + int(float64(p.format.SampleRate)*float64(d)/float64(time.Second))
	if newPos > p.streamer.Len() {
		newPos = p.streamer.Len()
	}

	speaker.Lock()
	err := p.streamer.Seek(newPos)
	speaker.Unlock()

	return err
}

// SeekBackward skips backward by the specified duration
func (p *Player) SeekBackward(d time.Duration) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.streamer == nil {
		return fmt.Errorf("no track loaded")
	}

	speaker.Lock()
	currentPos := p.streamer.Position()
	speaker.Unlock()

	newPos := currentPos - int(float64(p.format.SampleRate)*float64(d)/float64(time.Second))
	if newPos < 0 {
		newPos = 0
	}

	speaker.Lock()
	err := p.streamer.Seek(newPos)
	speaker.Unlock()

	return err
}

// OnTrackEnd sets a callback for when a track ends
func (p *Player) OnTrackEnd(fn func()) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onTrackEnd = fn
}

// Close closes the player and releases resources
func (p *Player) Close() {
	p.Stop()
	speaker.Close()
}

// volumeToDb converts a linear volume (0.0-1.0) to decibels
// Using logarithmic scale for natural perceived volume
func volumeToDb(v float64) float64 {
	if v <= 0 {
		return -10 // Very quiet but not completely silent
	}
	// Convert linear to logarithmic scale
	// v=1.0 -> 0dB, v=0.5 -> -6dB, v=0.1 -> -20dB
	return -6 * (1 - v) * (1 - v) / v
}
