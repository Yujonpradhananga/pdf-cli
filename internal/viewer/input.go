package viewer

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"pdf-cli/internal/layout"
)

// handleInput returns: 0 = continue, 1 = quit, -1 = search, -2 = goto page, -3 = help, -4 = debug, -5 = chapter list
func (d *DocumentViewer) handleInput(c byte) int {
	switch c {
	case 'q':
		return 1
	case 'b':
		d.wantBack = true
		return 1
	case 'j', ' ':
		if d.dualPageMode == "half" {
			if d.halfPageOffset == 0 {
				d.halfPageOffset = 1
			} else {
				d.halfPageOffset = 0
				if d.currentPage < len(d.textPages)-1 {
					d.currentPage++
				}
			}
		} else if d.currentPage < len(d.textPages)-1 {
			d.currentPage++
		}
	case 'k':
		if d.dualPageMode == "half" {
			if d.halfPageOffset == 1 {
				d.halfPageOffset = 0
			} else {
				d.halfPageOffset = 1
				if d.currentPage > 0 {
					d.currentPage--
				}
			}
		} else if d.currentPage > 0 {
			d.currentPage--
		}
	case 'g':
		return -2
	case 'c':
		return -5
	case '>':
		d.nextChapter()
	case '<':
		d.prevChapter()
	case 'h', '?':
		return -3
	case 't':
		d.toggleViewMode()
	case 'f':
		switch d.fitMode {
		case "height":
			d.fitMode = "width"
		case "width":
			d.fitMode = "auto"
		default:
			d.fitMode = "height"
		}
	case '/':
		return -1
	case 'n':
		d.nextSearchHit()
	case 'N':
		d.prevSearchHit()
	case '+', '=':
		if d.isReflowable {
			d.adjustHTMLZoom(-100)
		} else {
			d.scaleFactor += 0.1
			if d.scaleFactor > 2.0 {
				d.scaleFactor = 2.0
			}
		}
	case '-', '_':
		if d.isReflowable {
			d.adjustHTMLZoom(100)
		} else {
			d.scaleFactor -= 0.1
			if d.scaleFactor < 0.1 {
				d.scaleFactor = 0.1
			}
		}
	case 'r':
		d.refreshCellSize()
	case 'S':
		d.openInExternalApp("Skim")
	case 'P':
		d.openInExternalApp("Preview")
	case 'O':
		absPath, _ := filepath.Abs(d.path)
		exec.Command("open", "-R", absPath).Start()
	case 'i':
		if d.darkMode == "smart" {
			d.darkMode = ""
		} else {
			d.darkMode = "smart"
		}
	case 'd':
		if d.darkMode == "invert" {
			d.darkMode = ""
		} else {
			d.darkMode = "invert"
		}
	case 'D':
		return -4
	case '2':
		switch d.dualPageMode {
		case "":
			d.dualPageMode = "vertical"
		case "vertical":
			d.dualPageMode = "horizontal"
		case "horizontal":
			d.dualPageMode = "half"
			d.halfPageOffset = 0
		default:
			d.dualPageMode = ""
		}
	case 'J':
		if d.dualPageMode != "" {
			if d.currentPage < len(d.textPages)-2 {
				d.currentPage += 2
			} else if d.currentPage < len(d.textPages)-1 {
				d.currentPage = len(d.textPages) - 1
			}
		}
	case 'K':
		if d.dualPageMode != "" {
			if d.currentPage >= 2 {
				d.currentPage -= 2
			} else {
				d.currentPage = 0
			}
		}
	case '{':
		d.cropTop = min(d.cropTop+0.02, 0.45)
	case '}':
		d.cropBottom = min(d.cropBottom+0.02, 0.45)
	case '[':
		d.cropLeft = min(d.cropLeft+0.02, 0.45)
	case ']':
		d.cropRight = min(d.cropRight+0.02, 0.45)
	case '\\':
		d.cropTop, d.cropBottom, d.cropLeft, d.cropRight = 0, 0, 0, 0
	case 27:
		// Do nothing for plain ESC
	}
	return 0
}

func (d *DocumentViewer) openInExternalApp(appName string) {
	absPath, _ := filepath.Abs(d.path)
	page := d.currentPage + 1
	switch appName {
	case "Skim":
		script := fmt.Sprintf(`
set theFile to POSIX file "%s"
tell application "Skim"
  set theDocs to get documents whose path is (get POSIX path of theFile)
  if (count of theDocs) > 0 then
    try
      revert theDocs
    end try
  end if
  open theFile
  set index of current page of document 1 to %d
end tell
`, absPath, page)
		go exec.Command("osascript", "-e", script).Run()
	case "Preview":
		exec.Command("open", "-a", appName, absPath).Start()
	}
}

func (d *DocumentViewer) toggleViewMode() {
	switch d.forceMode {
	case "":
		d.forceMode = "text"
	case "text":
		d.forceMode = "image"
	case "image":
		d.forceMode = ""
	}
}

func (d *DocumentViewer) startSearch(inputChan <-chan byte) {
	_, rows := d.getTerminalSize()
	fmt.Printf("\033[%d;1H\033[K", rows)
	fmt.Print("\033[?25h")
	fmt.Print("Search: ")

	var query []byte
	for {
		ch := <-inputChan
		switch ch {
		case 13, 10:
			goto done
		case 27:
			fmt.Print("\033[?25l")
			return
		case 127, 8:
			if len(query) > 0 {
				query = query[:len(query)-1]
				fmt.Printf("\033[%d;1H\033[K", rows)
				fmt.Printf("Search: %s", string(query))
			}
		default:
			if ch >= 32 && ch < 127 {
				query = append(query, ch)
				fmt.Printf("%c", ch)
			}
		}
	}
done:
	fmt.Print("\033[?25l")
	queryStr := strings.TrimSpace(string(query))

	if queryStr == "" {
		d.searchQuery = ""
		d.searchHits = nil
		return
	}

	d.searchQuery = strings.ToLower(queryStr)
	d.searchHits = nil
	d.searchHitIdx = 0

	for _, pageNum := range d.textPages {
		text, err := d.doc.Text(pageNum)
		if err == nil && strings.Contains(strings.ToLower(text), d.searchQuery) {
			d.searchHits = append(d.searchHits, pageNum)
		}
	}

	if len(d.searchHits) > 0 {
		for i, p := range d.textPages {
			if p == d.searchHits[0] {
				d.currentPage = i
				break
			}
		}
	}
}

func (d *DocumentViewer) nextSearchHit() {
	if len(d.searchHits) == 0 {
		return
	}
	d.searchHitIdx = (d.searchHitIdx + 1) % len(d.searchHits)
	targetPage := d.searchHits[d.searchHitIdx]
	for i, p := range d.textPages {
		if p == targetPage {
			d.currentPage = i
			break
		}
	}
}

func (d *DocumentViewer) prevSearchHit() {
	if len(d.searchHits) == 0 {
		return
	}
	d.searchHitIdx--
	if d.searchHitIdx < 0 {
		d.searchHitIdx = len(d.searchHits) - 1
	}
	targetPage := d.searchHits[d.searchHitIdx]
	for i, p := range d.textPages {
		if p == targetPage {
			d.currentPage = i
			break
		}
	}
}

func (d *DocumentViewer) showHelp(inputChan <-chan byte) {
	fmt.Print("\033[2J\033[H")
	termWidth, _ := d.getTerminalSize()

	p := func(s string) { fmt.Print(s + "\r\n") }

	p(strings.Repeat("=", termWidth))
	p(fmt.Sprintf("%s Viewer Help", strings.ToUpper(d.fileType)))
	p(strings.Repeat("=", termWidth))
	p("")
	p("Navigation:")
	p("  j/Space/Down/Right  - Next page")
	p("  k/Up/Left           - Previous page")
	p("  g                   - Go to specific page")
	p("  c                   - Show chapter list (Table of Contents)")
	p("  >                   - Next chapter")
	p("  <                   - Previous chapter")
	p("  b                   - Back to file list")
	p("")
	p("Search:")
	p("  /                   - Search text in document")
	p("  n                   - Next search result")
	p("  N                   - Previous search result")
	p("")
	p("Display:")
	p("  t                   - Toggle view mode (auto/text/image)")
	p("  f                   - Cycle fit mode (height/width/auto)")
	p("  i                   - Toggle dark mode (smart invert, preserves hue)")
	p("  D                   - Show debug info")
	p("  +/-                 - Zoom in/out (10%-200%)")
	p("  2                   - Cycle view (off/vertical/horizontal/half-page)")
	p("  Shift+Left/Right    - Jump 2 pages (in dual page mode)")
	p("  Arrow/j/k           - Navigate by half-page (in half-page mode)")
	p("  r                   - Refresh cell size (after resolution change)")
	p("")
	p("Crop (trim page edges, session-only):")
	p("  {                   - Crop top edge (press multiple times)")
	p("  }                   - Crop bottom edge")
	p("  [                   - Crop left edge")
	p("  ]                   - Crop right edge")
	p("  \\                   - Reset all crops")
	p("  d                   - Toggle dark mode (simple color invert)")
	p("  S                   - Open in Skim")
	p("  P                   - Open in Preview")
	p("  O                   - Reveal in Finder")
	p("  h or ?              - Show this help")
	p("  q                   - Quit")
	p("")
	p("Features:")
	p("  - Auto-reload when file changes (for LaTeX workflows)")
	p("  - Text is reflowed to fit terminal width")
	p("  - Images rendered via Kitty/Sixel/iTerm2 graphics")
	if d.fileType == "epub" {
		p("  - HTML entities are converted to readable text")
	}
	p("")
	p("Supported formats: PDF, EPUB, DOCX, HTML")
	p("")
	p(strings.Repeat("=", termWidth))
	p("Press any key to return...")
	<-inputChan
}

func (d *DocumentViewer) showChapterList(inputChan <-chan byte) {
	fmt.Print("\033[2J\033[H")
	termWidth, termHeight := d.getTerminalSize()

	p := func(s string) { fmt.Print(s + "\r\n") }

	if len(d.chapters) == 0 {
		p("No chapters found in this document.")
		p("")
		p("Press any key to return...")
		<-inputChan
		return
	}

	d.updateCurrentChapter()

	p(strings.Repeat("=", termWidth))
	p("Table of Contents")
	p(strings.Repeat("=", termWidth))
	p("")

	available := termHeight - 7
	for i, ch := range d.chapters {
		if i >= available {
			p(fmt.Sprintf("  ... and %d more chapters", len(d.chapters)-i))
			break
		}
		indent := strings.Repeat("  ", ch.Level-1)
		marker := "  "
		if i == d.currentChapter {
			marker = "> "
		}
		title := ch.Title
		maxTitleLen := termWidth - len(indent) - len(marker) - 15
		if maxTitleLen < 20 {
			maxTitleLen = 20
		}
		if len(title) > maxTitleLen {
			title = title[:maxTitleLen-3] + "..."
		}
		p(fmt.Sprintf("%s%s%2d. %s (p.%d)", marker, indent, i+1, title, ch.Page+1))
	}

	p("")
	p(strings.Repeat("=", termWidth))
	fmt.Print("Enter chapter number (or ESC to cancel): ")
	fmt.Print("\033[?25h")

	var input []byte
	for {
		ch := <-inputChan
		switch ch {
		case 13, 10:
			goto selectChapter
		case 27:
			fmt.Print("\033[?25l")
			return
		case 127, 8:
			if len(input) > 0 {
				input = input[:len(input)-1]
				_, rows := d.getTerminalSize()
				fmt.Printf("\033[%d;1H\033[K", rows)
				fmt.Printf("Enter chapter number (or ESC to cancel): %s", string(input))
			}
		default:
			if ch >= '0' && ch <= '9' {
				input = append(input, ch)
				fmt.Printf("%c", ch)
			}
		}
	}
selectChapter:
	fmt.Print("\033[?25l")
	var num int
	if _, err := fmt.Sscanf(string(input), "%d", &num); err == nil {
		if num >= 1 && num <= len(d.chapters) {
			d.currentChapter = num - 1
			d.goToChapterPage(d.chapters[d.currentChapter].Page)
		}
	}
}

func (d *DocumentViewer) showDebugInfo(inputChan <-chan byte) {
	fmt.Print("\033[2J\033[H")
	cols, rows := d.getTerminalSize()
	cellW, cellH := d.getTerminalCellSize()
	pixelW, pixelH := d.getTerminalPixelSize()

	p := func(s string) { fmt.Print(s + "\r\n") }
	p("=== Debug Info ===")
	p(fmt.Sprintf("Terminal size: %d cols x %d rows", cols, rows))
	p(fmt.Sprintf("Cell size: %.1f x %.1f pixels", cellW, cellH))
	p(fmt.Sprintf("Pixel size (TIOCGWINSZ): %d x %d", pixelW, pixelH))
	p(fmt.Sprintf("Calculated terminal pixels: %.0f x %.0f", float64(cols)*cellW, float64(rows)*cellH))
	p(fmt.Sprintf("Fit mode: %s", d.fitMode))
	p(fmt.Sprintf("Scale factor: %.1f", d.scaleFactor))
	p("")
	p("Press any key to return...")
	<-inputChan
}

func (d *DocumentViewer) goToPage(inputChan <-chan byte) {
	_, rows := d.getTerminalSize()
	fmt.Printf("\033[%d;1H\033[K", rows)
	fmt.Print("\033[?25h")
	fmt.Printf("Go to page (1-%d): ", len(d.textPages))

	var input []byte
	for {
		ch := <-inputChan
		switch ch {
		case 13, 10:
			goto done
		case 27:
			fmt.Print("\033[?25l")
			return
		case 127, 8:
			if len(input) > 0 {
				input = input[:len(input)-1]
				fmt.Printf("\033[%d;1H\033[K", rows)
				fmt.Printf("Go to page (1-%d): %s", len(d.textPages), string(input))
			}
		default:
			if ch >= '0' && ch <= '9' {
				input = append(input, ch)
				fmt.Printf("%c", ch)
			}
		}
	}
done:
	fmt.Print("\033[?25l")
	var num int
	if _, err := fmt.Sscanf(string(input), "%d", &num); err == nil {
		if num >= 1 && num <= len(d.textPages) {
			d.currentPage = num - 1
		}
	}
}

// loadChapters extracts the table of contents from the document.
func (d *DocumentViewer) loadChapters() {
	outline, err := d.doc.ToC()
	if err != nil || len(outline) == 0 {
		d.chapters = nil
		return
	}
	d.chapters = make([]Chapter, 0, len(outline))
	for _, entry := range outline {
		page := entry.Page
		if page < 0 && entry.URI != "" {
			page = layout.ResolveLink(d.doc, entry.URI)
		}
		if page < 0 {
			page = 0
		}
		d.chapters = append(d.chapters, Chapter{
			Title: entry.Title,
			Page:  page,
			Level: entry.Level,
		})
	}
}

func (d *DocumentViewer) updateCurrentChapter() {
	if len(d.chapters) == 0 {
		return
	}
	actualPage := d.textPages[d.currentPage]
	d.currentChapter = 0
	for i, ch := range d.chapters {
		if ch.Page <= actualPage {
			d.currentChapter = i
		} else {
			break
		}
	}
}

func (d *DocumentViewer) nextChapter() {
	if len(d.chapters) == 0 {
		return
	}
	d.updateCurrentChapter()
	if d.currentChapter < len(d.chapters)-1 {
		d.currentChapter++
		d.goToChapterPage(d.chapters[d.currentChapter].Page)
	}
}

func (d *DocumentViewer) prevChapter() {
	if len(d.chapters) == 0 {
		return
	}
	d.updateCurrentChapter()
	if d.currentChapter > 0 {
		d.currentChapter--
		d.goToChapterPage(d.chapters[d.currentChapter].Page)
	}
}

func (d *DocumentViewer) goToChapterPage(targetPage int) {
	for i, p := range d.textPages {
		if p == targetPage {
			d.currentPage = i
			return
		}
	}
	for i, p := range d.textPages {
		if p >= targetPage {
			d.currentPage = i
			return
		}
	}
	if len(d.textPages) > 0 {
		d.currentPage = len(d.textPages) - 1
	}
}
