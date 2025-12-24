package main

import (
	"os"
	"strings"

	"golang.org/x/term"
)

func (d *DocumentViewer) getTerminalSize() (int, int) {
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return 80, 24 // Fallback default
	}
	return width, height
}

// detectTerminalType detects which terminal emulator is being used
func (d *DocumentViewer) detectTerminalType() string {
	// Check TERM_PROGRAM first (most reliable for modern terminals)
	if termProgram := os.Getenv("TERM_PROGRAM"); termProgram != "" {
		switch termProgram {
		case "WezTerm":
			return "wezterm"
		case "iTerm.app":
			return "iterm2"
		case "Apple_Terminal":
			return "apple_terminal"
		}
	}

	// Check for Kitty
	if os.Getenv("KITTY_WINDOW_ID") != "" || os.Getenv("KITTY_PID") != "" {
		return "kitty"
	}

	// Check TERM variable for other terminals
	term := os.Getenv("TERM")
	switch {
	case strings.Contains(term, "kitty"):
		return "kitty"
	case strings.Contains(term, "foot"):
		return "foot"
	case strings.Contains(term, "alacritty"):
		return "alacritty"
	case strings.Contains(term, "wezterm"):
		return "wezterm"
	case strings.Contains(term, "xterm"):
		return "xterm"
	case strings.Contains(term, "tmux"):
		return "tmux"
	case strings.Contains(term, "screen"):
		return "screen"
	}

	return "unknown"
}

// getTerminalCellSize returns the approximate pixel dimensions per character cell
// for known terminals. Returns (pixelsPerChar, pixelsPerLine)
func (d *DocumentViewer) getTerminalCellSize() (float64, float64) {
	termType := d.detectTerminalType()

	switch termType {
	case "kitty":
		// Kitty - adjusted for better image sizing
		return 6.0, 13.9
	case "foot":
		// Foot uses more standard cell sizes
		return 15.0, 25.0
	case "alacritty":
		// Alacritty varies but typically around these values
		return 7.0, 14.0
	case "wezterm":
		// WezTerm similar to Kitty
		return 9.0, 18.0
	case "iterm2":
		// iTerm2 on macOS
		return 8.0, 16.0
	case "xterm":
		// Classic xterm
		return 7.0, 14.0
	default:
		// Conservative fallback
		return 15.0, 25.0
	}
}

func (d *DocumentViewer) setRawMode() (*term.State, error) {
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return nil, err
	}
	d.oldState = oldState
	return oldState, nil
}

func (d *DocumentViewer) restoreTerminal(old *term.State) {
	if old != nil {
		term.Restore(int(os.Stdin.Fd()), old)
	}
}

func (d *DocumentViewer) readSingleChar() byte {
	// In raw mode, we can just read 1 byte from stdin
	buf := make([]byte, 1)
	n, _ := os.Stdin.Read(buf)
	if n > 0 {
		return buf[0]
	}
	return 0
}
