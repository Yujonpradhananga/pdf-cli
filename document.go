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
}

func NewDocumentViewer(path string) *DocumentViewer {
	ext := strings.ToLower(filepath.Ext(path))
	fileType := strings.TrimPrefix(ext, ".")

	// Create temp directory for image files
	tempDir := filepath.Join(os.TempDir(), fmt.Sprintf("docviewer_%d", time.Now().UnixNano()))

	return &DocumentViewer{
		path:     path,
		fileType: fileType,
		reader:   bufio.NewReader(os.Stdin),
		tempDir:  tempDir,
	}
}

func (d *DocumentViewer) Open() error {
	doc, err := fitz.New(d.path)
	if err != nil {
		return fmt.Errorf("error opening %s: %v", d.fileType, err)
	}
	d.doc = doc

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

		// Check for text content first
		text, err := d.doc.Text(i)
		if err == nil && len(strings.Fields(strings.TrimSpace(text))) >= 3 {
			hasContent = true
		}

		// If no meaningful text, check if page has meaningful visual content
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
	// Try to render the page as image to check for visual content
	img, err := d.doc.Image(pageNum)
	if err != nil {
		return false
	}

	bounds := img.Bounds()
	// Check if image has reasonable dimensions
	if bounds.Dx() < 50 || bounds.Dy() < 50 {
		return false
	}

	// Check if the image is mostly blank/white
	return d.hasNonBlankContent(img)
}

// hasNonBlankContent checks if an image has meaningful visual content
// by sampling pixels and checking for non-white/non-blank content
func (d *DocumentViewer) hasNonBlankContent(img image.Image) bool {
	bounds := img.Bounds()

	// Sample configuration
	sampleRate := 10             // Sample every 10th pixel
	nonWhiteThreshold := 20      // How many non-white pixels we need to consider it content
	whiteThreshold := uint8(240) // RGB values above this are considered "white-ish"

	nonWhitePixels := 0
	sampledPixels := 0

	// Sample pixels across the image
	for y := bounds.Min.Y; y < bounds.Max.Y; y += sampleRate {
		for x := bounds.Min.X; x < bounds.Max.X; x += sampleRate {
			sampledPixels++

			// Get pixel color
			c := img.At(x, y)
			r, g, b, a := c.RGBA()

			// Convert to 8-bit values
			r8 := uint8(r >> 8)
			g8 := uint8(g >> 8)
			b8 := uint8(b >> 8)
			a8 := uint8(a >> 8)

			// Skip fully transparent pixels
			if a8 < 10 {
				continue
			}

			// Check if pixel is non-white
			if r8 < whiteThreshold || g8 < whiteThreshold || b8 < whiteThreshold {
				nonWhitePixels++

				// Early exit if we found enough content
				if nonWhitePixels >= nonWhiteThreshold {
					return true
				}
			}
		}
	}

	// Additional check: look for any significantly colored pixels
	// This catches pages that might have light gray backgrounds or subtle content
	colorVariance := d.checkColorVariance(img)
	if colorVariance > 100 { // Threshold for color variance
		return true
	}

	// Consider it meaningful content if we have enough non-white pixels
	return nonWhitePixels >= nonWhiteThreshold
}

// checkColorVariance calculates the variance in pixel colors to detect subtle content
func (d *DocumentViewer) checkColorVariance(img image.Image) float64 {
	bounds := img.Bounds()

	// Sample fewer pixels for variance check
	sampleRate := 20
	var rSum, gSum, bSum uint64
	var rSumSq, gSumSq, bSumSq uint64
	sampleCount := 0

	// Calculate mean
	for y := bounds.Min.Y; y < bounds.Max.Y; y += sampleRate {
		for x := bounds.Min.X; x < bounds.Max.X; x += sampleRate {
			c := img.At(x, y)
			r, g, b, a := c.RGBA()

			// Skip transparent pixels
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

	// Calculate variance
	rMean := float64(rSum) / float64(sampleCount)
	gMean := float64(gSum) / float64(sampleCount)
	bMean := float64(bSum) / float64(sampleCount)

	rVar := float64(rSumSq)/float64(sampleCount) - rMean*rMean
	gVar := float64(gSumSq)/float64(sampleCount) - gMean*gMean
	bVar := float64(bSumSq)/float64(sampleCount) - bMean*bMean

	// Return combined variance
	return rVar + gVar + bVar
}

func (d *DocumentViewer) Run() {
	defer d.doc.Close()
	defer d.cleanup()

	fmt.Printf("Press any key to start reading %s, or 'q' to quit\n", strings.ToUpper(d.fileType))
	input, _ := d.reader.ReadString('\n')
	if strings.TrimSpace(input) == "q" {
		return
	}

	oldState, err := d.setRawMode()
	if err != nil {
		fmt.Printf("Error setting raw mode: %v\n", err)
		return
	}
	defer d.restoreTerminal(oldState)

	// Hide cursor
	fmt.Print("\033[?25l")
	defer fmt.Print("\033[?25h") // Show cursor on exit

	d.currentPage = 0

	// Main loop - display page and wait for input
	for {
		// Display the current page
		d.displayCurrentPage()

		// Wait for user input - this keeps the page visible
		char := d.readSingleChar()

		// Handle input and check if we should quit
		if d.handleInput(char) {
			break // Exit the loop to quit
		}

		// Continue loop - next iteration will display the new page
	}

	// Clear screen before showing exit message
	fmt.Print("\033[2J\033[H")
	fmt.Println("Thanks for reading!")
}

func (d *DocumentViewer) cleanup() {
	if d.tempDir != "" {
		os.RemoveAll(d.tempDir)
	}
}

func (d *DocumentViewer) handleInput(c byte) bool {
	switch c {
	case 'q':
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
	case 27: // ESC key - could be arrow keys
		d.handleArrowKeys()
	}
	return false
}

func (d *DocumentViewer) handleArrowKeys() {
	// Read next 2 bytes to see if it's an arrow key
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
