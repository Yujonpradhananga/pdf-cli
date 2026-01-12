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

// detecting Terminal
func (d *DocumentViewer) detectTerminalType() string {
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
	if os.Getenv("KITTY_WINDOW_ID") != "" || os.Getenv("KITTY_PID") != "" {
		return "kitty"
	}
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

func (d *DocumentViewer) getTerminalCellSize() (float64, float64) {
	termType := d.detectTerminalType()
	// Cell sizes in pixels - tuned for HiDPI/Retina displays
	switch termType {
	case "kitty":
		return 18.0, 36.0 // HiDPI Kitty
	case "foot":
		return 15.0, 25.0
	case "alacritty":
		return 14.0, 28.0
	case "wezterm":
		return 18.0, 36.0
	case "iterm2":
		return 16.0, 32.0
	case "xterm":
		return 7.0, 14.0
	default:
		return 15.0, 30.0
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
	buf := make([]byte, 1)
	n, _ := os.Stdin.Read(buf)
	if n > 0 {
		return buf[0]
	}
	return 0
}
