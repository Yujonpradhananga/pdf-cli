package main

import (
	"bufio"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gen2brain/go-fitz"
	"golang.org/x/term"
)

type DocumentViewer struct {
	doc         *fitz.Document
	currentPage int
	textPages   []int
	reader      *bufio.Reader
	path        string
	oldState    *term.State
	fileType    string // "pdf" or "epub"
	tempDir     string // for storing temporary image files
	forceMode   string // "", "text", or "image" - override auto-detection
	fitMode      string  // "auto", "height", "width"
	wantBack     bool    // signal to go back to file picker
	searchQuery  string  // current search query
	searchHits   []int     // pages with matches
	searchHitIdx int       // current index in searchHits
	scaleFactor  float64   // image scale adjustment (1.0 = default)
	lastModTime  time.Time // for auto-reload detection
}

func NewDocumentViewer(path string) *DocumentViewer {
	ext := strings.ToLower(filepath.Ext(path))
	fileType := strings.TrimPrefix(ext, ".")

	tempDir := filepath.Join(os.TempDir(), fmt.Sprintf("docviewer_%d", time.Now().UnixNano()))

	return &DocumentViewer{
		path:        path,
		fileType:    fileType,
		reader:      bufio.NewReader(os.Stdin),
		tempDir:     tempDir,
		fitMode:     "height", // default: fit to height
		scaleFactor: 1.0,
	}
}

func (d *DocumentViewer) Open() error {
	doc, err := fitz.New(d.path)
	if err != nil {
		return fmt.Errorf("error opening %s: %v", d.fileType, err)
	}
	d.doc = doc

	// Store modification time for auto-reload
	if info, err := os.Stat(d.path); err == nil {
		d.lastModTime = info.ModTime()
	}

	d.findContentPages()
	if len(d.textPages) == 0 {
		return fmt.Errorf("no pages with extractable content found")
	}
	return nil
}

func (d *DocumentViewer) findContentPages() {
	d.textPages = []int{}
	for i := 0; i < d.doc.NumPage(); i++ {
		hasContent := false

		text, err := d.doc.Text(i)
		if err == nil && len(strings.Fields(strings.TrimSpace(text))) >= 3 {
			hasContent = true
		}

		if !hasContent {
			if d.pageHasVisualContent(i) {
				hasContent = true
			}
		}

		if hasContent {
			d.textPages = append(d.textPages, i)
		}
	}
}

func (d *DocumentViewer) pageHasVisualContent(pageNum int) bool {
	img, err := d.doc.Image(pageNum)
	if err != nil {
		return false
	}

	bounds := img.Bounds()
	if bounds.Dx() < 50 || bounds.Dy() < 50 {
		return false
	}

	return d.hasNonBlankContent(img)
}

func (d *DocumentViewer) hasNonBlankContent(img image.Image) bool {
	bounds := img.Bounds()

	sampleRate := 10
	nonWhiteThreshold := 20
	whiteThreshold := uint8(240)

	nonWhitePixels := 0
	sampledPixels := 0

	for y := bounds.Min.Y; y < bounds.Max.Y; y += sampleRate {
		for x := bounds.Min.X; x < bounds.Max.X; x += sampleRate {
			sampledPixels++

			c := img.At(x, y)
			r, g, b, a := c.RGBA()

			r8 := uint8(r >> 8)
			g8 := uint8(g >> 8)
			b8 := uint8(b >> 8)
			a8 := uint8(a >> 8)

			if a8 < 10 {
				continue
			}

			if r8 < whiteThreshold || g8 < whiteThreshold || b8 < whiteThreshold {
				nonWhitePixels++

				if nonWhitePixels >= nonWhiteThreshold {
					return true
				}
			}
		}
	}

	colorVariance := d.checkColorVariance(img)
	if colorVariance > 100 {
		return true
	}

	return nonWhitePixels >= nonWhiteThreshold
}

func (d *DocumentViewer) checkColorVariance(img image.Image) float64 {
	bounds := img.Bounds()

	sampleRate := 20
	var rSum, gSum, bSum uint64
	var rSumSq, gSumSq, bSumSq uint64
	sampleCount := 0

	for y := bounds.Min.Y; y < bounds.Max.Y; y += sampleRate {
		for x := bounds.Min.X; x < bounds.Max.X; x += sampleRate {
			c := img.At(x, y)
			r, g, b, a := c.RGBA()

			if uint8(a>>8) < 10 {
				continue
			}

			r8 := uint8(r >> 8)
			g8 := uint8(g >> 8)
			b8 := uint8(b >> 8)

			rSum += uint64(r8)
			gSum += uint64(g8)
			bSum += uint64(b8)

			rSumSq += uint64(r8) * uint64(r8)
			gSumSq += uint64(g8) * uint64(g8)
			bSumSq += uint64(b8) * uint64(b8)

			sampleCount++
		}
	}

	if sampleCount < 10 {
		return 0
	}

	rMean := float64(rSum) / float64(sampleCount)
	gMean := float64(gSum) / float64(sampleCount)
	bMean := float64(bSum) / float64(sampleCount)

	rVar := float64(rSumSq)/float64(sampleCount) - rMean*rMean
	gVar := float64(gSumSq)/float64(sampleCount) - gMean*gMean
	bVar := float64(bSumSq)/float64(sampleCount) - bMean*bMean

	return rVar + gVar + bVar
}

func (d *DocumentViewer) Run() bool {
	defer d.doc.Close()
	defer d.cleanup()

	oldState, err := d.setRawMode()
	if err != nil {
		fmt.Printf("Error setting raw mode: %v\n", err)
		return false
	}
	defer d.restoreTerminal(oldState)
	fmt.Print("\033[?25l")
	defer fmt.Print("\033[?25h") // Show cursor on exit

	d.currentPage = 0

	// Start file watcher goroutine
	reloadChan := make(chan bool, 1)
	stopChan := make(chan bool)
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if d.checkAndReload() {
					select {
					case reloadChan <- true:
					default:
					}
				}
			case <-stopChan:
				return
			}
		}
	}()
	defer close(stopChan)

	for {
		d.displayCurrentPage()

		// Check for reload signal (non-blocking)
		select {
		case <-reloadChan:
			continue // Redisplay after reload
		default:
		}

		char := d.readSingleChar()

		if d.handleInput(char) {
			break // Exit the loop to quit
		}
	}

	// Clear screen
	fmt.Print("\033[2J\033[H")
	return d.wantBack
}

func (d *DocumentViewer) cleanup() {
	if d.tempDir != "" {
		os.RemoveAll(d.tempDir)
	}
}

func (d *DocumentViewer) checkAndReload() bool {
	info, err := os.Stat(d.path)
	if err != nil {
		return false
	}

	if info.ModTime().After(d.lastModTime) {
		// File changed - reload
		savedPage := d.currentPage
		d.doc.Close()

		// Small delay to ensure file is fully written
		time.Sleep(100 * time.Millisecond)

		doc, err := fitz.New(d.path)
		if err != nil {
			return false
		}
		d.doc = doc
		d.lastModTime = info.ModTime()
		d.findContentPages()

		// Restore page position (clamp to valid range)
		if savedPage >= len(d.textPages) {
			savedPage = len(d.textPages) - 1
		}
		if savedPage < 0 {
			savedPage = 0
		}
		d.currentPage = savedPage
		return true
	}
	return false
}

func (d *DocumentViewer) handleInput(c byte) bool {
	switch c {
	case 'q':
		return true
	case 'b':
		d.wantBack = true
		return true
	case 'j', ' ':
		if d.currentPage < len(d.textPages)-1 {
			d.currentPage++
		}
	case 'k':
		if d.currentPage > 0 {
			d.currentPage--
		}
	case 'g':
		d.goToPage()
	case 'h':
		d.showHelp()
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
		d.startSearch()
	case 'n':
		d.nextSearchHit()
	case 'N':
		d.prevSearchHit()
	case '+', '=':
		d.scaleFactor += 0.1
		if d.scaleFactor > 2.0 {
			d.scaleFactor = 2.0
		}
	case '-', '_':
		d.scaleFactor -= 0.1
		if d.scaleFactor < 0.1 {
			d.scaleFactor = 0.1
		}
	case 27: // ESC key - could be arrow keys
		d.handleArrowKeys()
	}
	return false
}

func (d *DocumentViewer) startSearch() {
	d.restoreTerminal(d.oldState)
	fmt.Print("\033[?25h") // show cursor
	fmt.Printf("\nSearch: ")
	line, _ := d.reader.ReadString('\n')
	query := strings.TrimSpace(line)
	fmt.Print("\033[?25l") // hide cursor
	d.setRawMode()

	if query == "" {
		d.searchQuery = ""
		d.searchHits = nil
		return
	}

	d.searchQuery = strings.ToLower(query)
	d.searchHits = nil
	d.searchHitIdx = 0

	// Search all pages
	for _, pageNum := range d.textPages {
		text, err := d.doc.Text(pageNum)
		if err == nil && strings.Contains(strings.ToLower(text), d.searchQuery) {
			d.searchHits = append(d.searchHits, pageNum)
		}
	}

	if len(d.searchHits) > 0 {
		// Jump to first hit
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

func (d *DocumentViewer) handleArrowKeys() {
	buf := make([]byte, 2)
	n, _ := os.Stdin.Read(buf)

	if n >= 2 && buf[0] == '[' {
		switch buf[1] {
		case 'B': // Down arrow
			if d.currentPage > 0 {
				d.currentPage--
			}
		case 'A': // Up arrow
			if d.currentPage < len(d.textPages)-1 {
				d.currentPage++
			}
		case 'C': // Right arrow
			if d.currentPage < len(d.textPages)-1 {
				d.currentPage++
			}
		case 'D': // Left arrow
			if d.currentPage > 0 {
				d.currentPage--
			}
		}
	}
}

func (d *DocumentViewer) goToPage() {
	d.restoreTerminal(d.oldState)
	fmt.Printf("\nGo to page (1-%d): ", len(d.textPages))
	line, _ := d.reader.ReadString('\n')
	var num int
	if _, err := fmt.Sscanf(strings.TrimSpace(line), "%d", &num); err == nil {
		if num >= 1 && num <= len(d.textPages) {
			d.currentPage = num - 1
		}
	}
	d.setRawMode()
}
