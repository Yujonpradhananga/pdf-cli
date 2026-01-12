package main

import (
	"os"
	"strings"
	"syscall"
	"unsafe"

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
	// Try to get actual pixel size from terminal
	pixelWidth, pixelHeight := d.getTerminalPixelSize()
	charWidth, charHeight := d.getTerminalSize()

	if pixelWidth > 0 && pixelHeight > 0 && charWidth > 0 && charHeight > 0 {
		cellWidth := float64(pixelWidth) / float64(charWidth)
		cellHeight := float64(pixelHeight) / float64(charHeight)
		if cellWidth > 4 && cellHeight > 8 {
			return cellWidth, cellHeight
		}
	}

	// Fallback to hardcoded values
	termType := d.detectTerminalType()
	switch termType {
	case "kitty":
		return 18.0, 36.0
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

func (d *DocumentViewer) getTerminalPixelSize() (int, int) {
	ws := struct {
		Row    uint16
		Col    uint16
		Xpixel uint16
		Ypixel uint16
	}{}

	_, _, err := syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(syscall.Stdout),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(&ws)))

	if err == 0 && ws.Xpixel > 0 && ws.Ypixel > 0 {
		return int(ws.Xpixel), int(ws.Ypixel)
	}
	return 0, 0
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
	if n == 0 {
		return 0
	}

	// Handle escape sequences (arrow keys)
	if buf[0] == 27 {
		seq := make([]byte, 2)
		n, _ := os.Stdin.Read(seq)
		if n >= 2 && seq[0] == '[' {
			switch seq[1] {
			case 'A': // Up arrow -> previous page (like k)
				return 'k'
			case 'B': // Down arrow -> next page (like j)
				return 'j'
			case 'C': // Right arrow -> next page
				return 'j'
			case 'D': // Left arrow -> previous page
				return 'k'
			}
		}
		return 27 // Plain ESC
	}

	return buf[0]
}
