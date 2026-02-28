// Package lyrics provides LRC file parsing and display functionality
package lyrics

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Line represents a single lyrics line with timestamp
type Line struct {
	Time    time.Duration
	Text    string
}

// Lyrics represents parsed lyrics with lines
type Lyrics struct {
	Lines    []Line
	FilePath string
}

// Parse parses an LRC file and returns Lyrics
func Parse(path string) (*Lyrics, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open lrc file: %w", err)
	}
	defer file.Close()

	lyrics := &Lyrics{
		FilePath: path,
		Lines:    make([]Line, 0),
	}

	// Regex to match [mm:ss.xx] or [mm:ss:xx] format
	timeRegex := regexp.MustCompile(`\[(\d{1,2}):(\d{2})[.:](\d{2,3})\]`)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		// Find all timestamps in the line (some lines have multiple)
		matches := timeRegex.FindAllStringSubmatchIndex(line, -1)
		if len(matches) == 0 {
			continue
		}

		// Extract the text after all timestamps
		lastMatchEnd := matches[len(matches)-1][1]
		text := strings.TrimSpace(line[lastMatchEnd:])

		// Parse each timestamp and create a line entry
		for _, match := range matches {
			fullMatch := line[match[0]:match[1]]
			submatches := timeRegex.FindStringSubmatch(fullMatch)
			if len(submatches) != 4 {
				continue
			}

			minutes := parseUint(submatches[1])
			seconds := parseUint(submatches[2])
			millis := parseUint(submatches[3])

			// Handle 2-digit milliseconds (centiseconds)
			if len(submatches[3]) == 2 {
				millis *= 10
			}

			t := time.Duration(minutes)*time.Minute + time.Duration(seconds)*time.Second + time.Duration(millis)*time.Millisecond

			lyrics.Lines = append(lyrics.Lines, Line{
				Time: t,
				Text: text,
			})
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading lrc file: %w", err)
	}

	// Sort lines by time
	sort.Slice(lyrics.Lines, func(i, j int) bool {
		return lyrics.Lines[i].Time < lyrics.Lines[j].Time
	})

	return lyrics, nil
}

// FindLRCFile finds the LRC file for a given audio file
func FindLRCFile(audioPath string) string {
	ext := filepath.Ext(audioPath)
	lrcPath := audioPath[:len(audioPath)-len(ext)] + ".lrc"

	if _, err := os.Stat(lrcPath); err == nil {
		return lrcPath
	}

	// Also try uppercase .LRC
	lrcPathUpper := audioPath[:len(audioPath)-len(ext)] + ".LRC"
	if _, err := os.Stat(lrcPathUpper); err == nil {
		return lrcPathUpper
	}

	return ""
}

// GetCurrentLine returns the current lyrics line based on position
func (l *Lyrics) GetCurrentLine(position time.Duration) (int, string) {
	if l == nil || len(l.Lines) == 0 {
		return -1, ""
	}

	// Find the line that should be displayed at current position
	idx := sort.Search(len(l.Lines), func(i int) bool {
		return l.Lines[i].Time > position
	})

	// The current line is the one before the found index
	if idx == 0 {
		return -1, ""
	}

	return idx - 1, l.Lines[idx-1].Text
}

// GetLinesAround returns lines around the current position for display
func (l *Lyrics) GetLinesAround(position time.Duration, before, after int) ([]Line, int) {
	if l == nil || len(l.Lines) == 0 {
		return nil, -1
	}

	currentIdx, _ := l.GetCurrentLine(position)
	if currentIdx < 0 {
		currentIdx = 0
	}

	start := currentIdx - before
	if start < 0 {
		start = 0
	}

	end := currentIdx + after + 1
	if end > len(l.Lines) {
		end = len(l.Lines)
	}

	result := make([]Line, end-start)
	copy(result, l.Lines[start:end])

	// Return the index of current line in the result slice
	currentInResult := currentIdx - start
	return result, currentInResult
}

func parseUint(s string) int {
	var result int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			result = result*10 + int(c-'0')
		}
	}
	return result
}
