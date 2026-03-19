<h1 align=center>pdf-cli</h1>
<div align=center>

![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=for-the-badge&logo=go&labelColor=181825)
![Stars](https://img.shields.io/github/stars/Yujonpradhananga/pdf-cli?style=for-the-badge&labelColor=181825&color=f9e2af)
![Latest Release](https://img.shields.io/github/v/release/Yujonpradhananga/pdf-cli?style=for-the-badge&labelColor=181825&color=b4befe)
![Last Commit](https://img.shields.io/github/last-commit/Yujonpradhananga/pdf-cli?style=for-the-badge&labelColor=181825&color=a6e3a1)

</div>
A terminal-based PDF, EPUB and DOCX viewer with fuzzy file search, high-resolution image rendering, auto-reload for LaTeX workflows intelligent text reflow and double page mode.

<https://github.com/user-attachments/assets/7ba77b3f-7f7a-48aa-bf70-1f432650bdf1>

## Features

- **Fuzzy File Search**: Interactive file picker with fuzzy search to quickly find your PDFs and EPUBs
- **Smart Content Detection**: Automatically detects and displays text, images, or mixed content pages
- **High-Resolution Image Rendering**: Uses terminal graphics protocols (Sixel/Kitty/iTerm2) for crisp image display
- **Half Page View**:Supports screen splitting to display pages in halfpage view with high quality rendering.
- **Image Invert**: Inverts the Image while preserving the core colors of the image.
- **HiDPI/Retina Support**: Dynamic cell size detection for sharp rendering on high-DPI displays
- **Auto-Reload**: Automatically reloads when the PDF changes (perfect for LaTeX compilation with `latexmk -pvc`)
- **Fit Modes**: Toggle between height-fit, width-fit, and auto-fit modes
- **Manual Zoom**: Adjust zoom from 10% to 200%
- **In-Document Search**: Search for text within documents
- **Intelligent Text Reflow**: Automatically reformats text to fit your terminal width while preserving paragraphs
- **Terminal-Aware**: Detects your terminal type and optimizes rendering accordingly
- **Multiple Formats**: Supports PDF, EPUB, and DOCX documents

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `j` / `Space` / `Down` / `Right` | Next page |
| `k` / `Up` / `Left` | Previous page |
| `g` | Go to specific page |
| `b` | Back to file picker |
| `/` | Search in document |
| `n` | Next search result |
| `N` | Previous search result |
| `t` | Toggle text/image/auto mode |
| `f` | Cycle fit modes (height/width/auto) |
| `+` / `=` | Zoom in |
| `-` | Zoom out |
| `r` | Refresh display (re-detect cell size) |
| `d` | Show debug info |
| `h` | Show help |
| `q` | Quit |
| `2` | Cycle page modes |

## Installation

### NixOSNixOS Installation

Add the `pdf-cli.nix` file to your nixos configuration directory, then add the overlay to your `flake.nix`:

```nix
let
  myOverlay = final: prev: {
    pdf-cli = prev.callPackage ./pdf-cli.nix { };
  };
in
```

Then apply the overlay in your `nixosConfigurations`:

```nix
({ config, pkgs, ... }: {
  nixpkgs.overlays = [ myOverlay ];
})
```

Then in your `home.nix`:

```nix
home.packages = with pkgs; [
  pdf-cli
];
```

### Arch Linux Installation

Package is coming to the AUR soon.
=======
### Arch Linux Installation (AUR)

Using yay

```bash
yay -S pdf-cli
```

Using paru

```
paru -S pdf-cli
```

### Building from source

```bash
# Clone this repository
git clone https://github.com/Yujonpradhananga/pdf-cli

# Install dependencies
go mod tidy

# Build
go build -o pdf-cli .

# Optionally move to your PATH
mv pdf-cli ~/local/bin/
```

### Usage

```bash
# Search current directory (default)
pdf-cli

# Search specific directory
pdf-cli ~/Documents/papers/

# Open a specific file directly
pdf-cli paper.pdf
```

## LaTeX Workflow

The auto-reload feature makes this viewer ideal for LaTeX editing:

1. Open your PDF: `pdf-cli paper.pdf`
2. Run LaTeX compiler in another terminal: `latexmk -pvc paper.tex`
3. The viewer automatically reloads when the PDF updates, preserving your page position

The viewer handles partially-written PDFs gracefully, waiting for the file to stabilize before reloading.

## Dependencies

- Go 1.21+
- [go-fitz](https://github.com/gen2brain/go-fitz) - PDF/EPUB parsing (MuPDF)
- [go-termimg](https://github.com/blacktop/go-termimg) - Terminal image rendering
- [fuzzy](https://github.com/sahilm/fuzzy) - Fuzzy search
- [golang.org/x/term](https://golang.org/x/term) - Terminal control

## Supported Terminals

Optimized for terminals with graphics support:

- **Kitty** (recommended) - Native cell size detection via escape sequences
- Foot
- WezTerm
- iTerm2
- Alacritty
- xterm (with Sixel support)

Works in any terminal, but image rendering quality depends on terminal capabilities.

## How It Works

The reader scans the current directory (or specified directory) for PDF, EPUB, and DOCX files. Use the fuzzy search to quickly filter and select a file. The viewer intelligently detects whether pages contain text, images, or both, and renders them appropriately for terminal display.

PDFs are rendered as images by default (essential for math, diagrams, and formatted content) at a DPI calculated to match your terminal's pixel dimensions for optimal sharpness.

## License

MIT

---

By [Yujon Pradhananga](https://github.com/Yujonpradhananga)
