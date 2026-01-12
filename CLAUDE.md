# CLI PDF/EPUB Reader - Project Notes

## Project Overview
Terminal-based PDF/EPUB viewer written in Go, using MuPDF (go-fitz) for rendering and terminal graphics protocols (Kitty/Sixel/iTerm2) for image display.

## Build & Install
```bash
go build -o docviewer . && mv docviewer ~/bin/
```

## Key Design Decisions

### Image Rendering for PDFs
- PDFs render as images by default (not text extraction) - essential for math, diagrams, and formatted content
- Use 300 DPI for crisp rendering on HiDPI displays
- Terminal cell sizes tuned for HiDPI: Kitty uses 18x36 pixels

### Fit Modes
- `height` (default): Fit to terminal height, no scrolling needed
- `width`: Fit to terminal width, may exceed height
- `auto`: Fit within both bounds

### User Preferences
- No unnecessary prompts - opens immediately
- Quiet mode for directory scanning (no "Scanning..." messages)
- Status bar shows available shortcuts inline
- Back button (`b`) returns to file picker for quick browsing

## Keyboard Shortcuts
- `j`/`Space` - Next page
- `k` - Previous page
- `g` - Go to page
- `b` - Back to file list
- `/` - Search, `n`/`N` - next/prev result
- `t` - Toggle text/image mode
- `f` - Cycle fit modes (height → width → auto)
- `+`/`-` - Manual zoom (10% to 200%)
- `h` - Help
- `q` - Quit

## Auto-Reload (LaTeX Workflow)
- Checks file modification every 500ms
- Waits for file size to stabilize before reloading
- Retries opening PDF up to 3 times if corrupted
- **Silent error handling**: MuPDF warnings suppressed via stderr redirect
- Preserves current page position after reload
- Perfect for `latexmk -pvc` continuous compilation

## Dependencies
- `github.com/gen2brain/go-fitz` - PDF/EPUB parsing (MuPDF)
- `github.com/blacktop/go-termimg` - Terminal image rendering
- `github.com/disintegration/imaging` - Image processing
- `github.com/sahilm/fuzzy` - Fuzzy file search

## Development Notes

### Error Handling Philosophy
- Silent errors preferred - don't spam terminal with warnings
- Use stderr redirect to /dev/null for noisy libraries (MuPDF)
- Graceful degradation: keep showing current content if reload fails

### Terminal Cell Sizes (HiDPI)
Tuned for Retina/HiDPI displays:
- Kitty: 18x36 pixels
- WezTerm: 18x36 pixels
- iTerm2: 16x32 pixels
- Alacritty: 14x28 pixels

### User's Workflow
- Uses `dv` alias for `docviewer`
- LaTeX compilation with `latexmk -pvc`
- Prefers immediate feedback, no prompts
- Wants zoom range down to 10% for overview

## Fork Info
Original: https://github.com/Yujonpradhananga/CLI-PDF-EPUB-reader
Fork: https://github.com/lenis2000/CLI-PDF-EPUB-reader
