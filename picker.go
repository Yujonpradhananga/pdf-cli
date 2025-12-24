package main

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

type FilePicker struct {
	searcher      *FileSearcher
	query         string
	results       []FileResult
	selectedIndex int
	displayOffset int
	termHeight    int
	termWidth     int
	oldState      *term.State
}

func NewFilePicker(searcher *FileSearcher) *FilePicker {
	width, height, _ := term.GetSize(int(os.Stdout.Fd()))
	return &FilePicker{
		searcher:      searcher,
		query:         "",
		selectedIndex: 0,
		displayOffset: 0,
		termHeight:    height,
		termWidth:     width,
	}
}

// Run starts the interactive file picker
func (fp *FilePicker) Run() (string, error) {
	// Set terminal to raw mode
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return "", err
	}
	fp.oldState = oldState
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	// Hide cursor
	fmt.Print("\033[?25l")
	defer fmt.Print("\033[?25h") // Show cursor on exit

	// Initial search (empty query shows all)
	fp.updateResults()

	// Main loop
	for {
		fp.render()

		// Read single character
		char := fp.readChar()

		// Handle input
		switch char {
		case 3: // Ctrl+C
			return "", fmt.Errorf("cancelled")
		case 27: // ESC or arrow keys
			if fp.handleEscapeSequence() {
				return "", fmt.Errorf("cancelled")
			}
		case 127, 8: // Backspace/Delete
			if len(fp.query) > 0 {
				fp.query = fp.query[:len(fp.query)-1]
				fp.updateResults()
			}
		case 13: // Enter
			if len(fp.results) > 0 && fp.selectedIndex < len(fp.results) {
				return fp.results[fp.selectedIndex].Path, nil
			}
		case 9: // Tab - cycle through results
			if len(fp.results) > 0 {
				fp.selectedIndex = (fp.selectedIndex + 1) % len(fp.results)
				fp.ensureSelectedVisible()
			}
		default:
			// Regular character input
			if char >= 32 && char < 127 {
				fp.query += string(char)
				fp.updateResults()
			}
		}
	}
}

func (fp *FilePicker) handleEscapeSequence() bool {
	// Read next characters to determine if it's an arrow key
	seq := make([]byte, 2)
	n, _ := os.Stdin.Read(seq)

	if n == 0 {
		// Just ESC pressed
		return true
	}

	if n >= 2 && seq[0] == '[' {
		switch seq[1] {
		case 'A': // Up arrow
			if fp.selectedIndex > 0 {
				fp.selectedIndex--
				fp.ensureSelectedVisible()
			}
		case 'B': // Down arrow
			if fp.selectedIndex < len(fp.results)-1 {
				fp.selectedIndex++
				fp.ensureSelectedVisible()
			}
		}
	}

	return false
}

func (fp *FilePicker) readChar() byte {
	buf := make([]byte, 1)
	n, _ := os.Stdin.Read(buf)
	if n > 0 {
		return buf[0]
	}
	return 0
}

func (fp *FilePicker) updateResults() {
	fp.results = fp.searcher.Search(fp.query)
	fp.selectedIndex = 0
	fp.displayOffset = 0
}

func (fp *FilePicker) ensureSelectedVisible() {
	// Calculate visible area (reserve lines for header and input)
	headerLines := 3
	visibleLines := fp.termHeight - headerLines - 1

	if fp.selectedIndex < fp.displayOffset {
		fp.displayOffset = fp.selectedIndex
	} else if fp.selectedIndex >= fp.displayOffset+visibleLines {
		fp.displayOffset = fp.selectedIndex - visibleLines + 1
	}
}

func (fp *FilePicker) render() {
	// Clear screen and move to top
	fmt.Print("\033[2J\033[H")

	// Header
	fmt.Print("\033[1;36m╔═══════════════════════════════════════════════════════════════╗\033[0m\r\n")
	fmt.Print("\033[1;36m║\033[0m              \033[1;37mPDF/EPUB File Selector\033[0m                     \033[1;36m║\033[0m\r\n")
	fmt.Print("\033[1;36m╚═══════════════════════════════════════════════════════════════╝\033[0m\r\n")

	// Search input
	fmt.Printf("\033[1;32m>\033[0m %s\033[0m\r\n", fp.query)
	fmt.Print(strings.Repeat("─", fp.termWidth))
	fmt.Print("\r\n")

	// Calculate visible area
	headerLines := 5
	visibleLines := fp.termHeight - headerLines - 2

	// Display results
	if len(fp.results) == 0 {
		fmt.Print("\033[2m  No files found\033[0m\r\n")
		fmt.Print("\r\n")
		fmt.Print("\033[2m  Try a different search query or press Ctrl+C to exit\033[0m\r\n")
	} else {
		// Show result count
		fmt.Printf("\033[2m  Found %d file(s)\033[0m\r\n\r\n", len(fp.results))

		// Display visible results
		endIndex := fp.displayOffset + visibleLines
		if endIndex > len(fp.results) {
			endIndex = len(fp.results)
		}

		for i := fp.displayOffset; i < endIndex; i++ {
			result := fp.results[i]

			if i == fp.selectedIndex {
				// Highlight selected item
				fmt.Print("\033[7m► ") // Reverse video
				fmt.Print(result.HighlightMatches())
				fmt.Print("\033[0m\r\n") // Use CRLF
			} else {
				fmt.Print("  ")
				fmt.Print(result.HighlightMatches())
				fmt.Print("\r\n") // Use CRLF
			}
		}

		// Show scroll indicator if needed
		if len(fp.results) > visibleLines {
			fmt.Printf("\r\n\033[2m  [%d-%d of %d]\033[0m", fp.displayOffset+1, endIndex, len(fp.results))
		}
	}

	// Instructions at bottom
	fmt.Print("\r\n\r\n")
	fmt.Print("\033[2m  ↑/↓: Navigate  Enter: Select  Tab: Next  Backspace: Clear  Esc/Ctrl+C: Exit\033[0m")
}
