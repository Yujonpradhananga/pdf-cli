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
- `f` - Cycle fit modes
- `h` - Help
- `q` - Quit

## Dependencies
- `github.com/gen2brain/go-fitz` - PDF/EPUB parsing (MuPDF)
- `github.com/blacktop/go-termimg` - Terminal image rendering
- `github.com/disintegration/imaging` - Image processing
- `github.com/sahilm/fuzzy` - Fuzzy file search

## Fork Info
Original: https://github.com/Yujonpradhananga/CLI-PDF-EPUB-reader
Fork: https://github.com/lenis2000/CLI-PDF-EPUB-reader
