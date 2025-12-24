# CLI PDF/EPUB Reader

A terminal-based PDF and EPUB reader with fuzzy file search, high-resolution image rendering, and intelligent text reflow.

## Screenshots

![Screenshot 1](screenshots/ss1.png)
![Screenshot 2](screenshots/ss2.png)
![Screenshot 3](screenshots/ss3.png)
![Screenshot 4](screenshots/ss4.png)
![Screenshot 5](screenshots/ss5.png)
![Screenshot 6](screenshots/ss6.png)
![Screenshot 7](screenshots/ss7.png)
![Screenshot 8](screenshots/ss8.png)
![Screenshot 9](screenshots/ss9.png)

## Features

- **Fuzzy File Search**: Interactive file picker with fuzzy search to quickly find your PDFs and EPUBs
- **Smart Content Detection**: Automatically detects and displays text, images, or mixed content pages
- **High-Resolution Image Rendering**: Uses terminal graphics protocols (Sixel/Kitty/iTerm2) for crisp image display
- **Intelligent Text Reflow**: Automatically reformats text to fit your terminal width while preserving paragraphs
- **Terminal-Aware**: Detects your terminal type and optimizes rendering accordingly
- **Both Formats**: Supports PDF and EPUB documents

## Navigation

- `j` or `Space` - Next page
- `k` - Previous page  
- `g` - Go to specific page
- `h` - Show help
- `q` - Quit

## Installation

1. Clone this repository:
```bash
git clone https://github.com/Yujonpradhananga/CLI-PDF-EPUB-reader.git
cd CLI-PDF-EPUB-reader
```

2. Run the binary:
```bash
./lnreader
```

**Building from source** (optional):
```bash
go mod download
go build -o lnreader
```

## Dependencies

- Go 1.21+
- [go-fitz](https://github.com/gen2brain/go-fitz) - PDF/EPUB parsing
- [go-termimg](https://github.com/blacktop/go-termimg) - Terminal image rendering
- [imaging](https://github.com/disintegration/imaging) - Image processing
- [fuzzy](https://github.com/sahilm/fuzzy) - Fuzzy search
- [golang.org/x/term](https://golang.org/x/term) - Terminal control

## Supported Terminals

Optimized for terminals with graphics support:
- Kitty
- WezTerm
- iTerm2
- Alacritty
- Foot
- xterm (with Sixel support)

Works in any terminal, but image rendering quality depends on terminal capabilities.

## How It Works

The reader scans your home directory, Documents, Downloads, and Desktop for PDF/EPUB files. Use the fuzzy search to quickly filter and select a file. The viewer intelligently detects whether pages contain text, images, or both, and renders them appropriately for terminal display.

## License

MIT

---

Made with ❤️ by [Yujon Pradhananga](https://github.com/Yujonpradhananga)
