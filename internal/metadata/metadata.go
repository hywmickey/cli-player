// Package metadata provides audio file metadata extraction
package metadata

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dhowden/tag"
	"github.com/faiface/beep"
	"github.com/faiface/beep/flac"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/vorbis"
	"github.com/faiface/beep/wav"
)

// Track represents an audio track with its metadata
type Track struct {
	Path     string
	Title    string
	Artist   string
	Album    string
	Duration time.Duration
}

// Extract reads metadata from an audio file
func Extract(path string) (*Track, error) {
	// Open the file
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	track := &Track{
		Path: path,
	}

	// Try to read metadata using tag library
	metadata, err := tag.ReadFrom(file)
	if err == nil {
		track.Title = metadata.Title()
		track.Artist = metadata.Artist()
		track.Album = metadata.Album()
	}

	// If no title, use filename
	if track.Title == "" {
		track.Title = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}

	// Get duration by decoding the audio file
	// Need to reopen the file for decoding
	file.Seek(0, 0)
	duration, err := getDuration(file, path)
	if err == nil {
		track.Duration = duration
	}

	return track, nil
}

// getDuration returns the duration of an audio file
func getDuration(file *os.File, path string) (time.Duration, error) {
	ext := strings.ToLower(filepath.Ext(path))

	var streamer beep.StreamSeekCloser
	var format beep.Format
	var err error

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
		return 0, fmt.Errorf("unsupported format: %s", ext)
	}

	if err != nil {
		return 0, fmt.Errorf("failed to decode: %w", err)
	}
	defer streamer.Close()

	samples := streamer.Len()
	duration := time.Duration(float64(samples) / float64(format.SampleRate) * float64(time.Second))

	return duration, nil
}

// FormatDuration formats a duration as mm:ss
func FormatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	mins := int(d.Minutes())
	secs := int(d.Seconds()) % 60
	return fmt.Sprintf("%d:%02d", mins, secs)
}

// IsSupported checks if a file extension is supported
func IsSupported(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".mp3", ".wav", ".flac", ".ogg", ".oga":
		return true
	default:
		return false
	}
}
