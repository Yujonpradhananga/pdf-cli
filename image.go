package main

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"

	"github.com/blacktop/go-termimg"
)

func (d *DocumentViewer) renderPageImage(pageNum, maxWidth, maxHeight int) int {
	if maxHeight <= 0 {
		return 0
	}

	imagePath, actualHeight, imageWidthInChars, err := d.savePageAsImage(pageNum, maxWidth, maxHeight)
	if err != nil {
		return 0
	}
	defer os.Remove(imagePath)

	horizontalOffset := (maxWidth - imageWidthInChars) / 2
	if horizontalOffset < 0 {
		horizontalOffset = 0
	}

	return d.renderWithTermImg(imagePath, actualHeight, horizontalOffset, imageWidthInChars)
}

func (d *DocumentViewer) savePageAsImage(pageNum, termWidth, termHeight int) (string, int, int, error) {
	if err := os.MkdirAll(d.tempDir, 0o755); err != nil {
		return "", 0, 0, err
	}

	pixelsPerChar, pixelsPerLine := d.getTerminalCellSize()

	// Calculate target pixel dimensions based on terminal size
	horizontalPadding := 4
	verticalPadding := 3
	effectiveWidth := termWidth - horizontalPadding
	effectiveHeight := termHeight - verticalPadding

	// Apply user scale factor
	scale := d.scaleFactor
	if scale == 0 {
		scale = 1.0
	}

	targetPixelWidth := int(float64(effectiveWidth) * pixelsPerChar * scale)
	targetPixelHeight := int(float64(effectiveHeight) * pixelsPerLine * scale)

	// Get page dimensions at 72 DPI to calculate proper render DPI
	testImg, err := d.doc.ImageDPI(pageNum, 72.0)
	if err != nil {
		return "", 0, 0, err
	}
	testBounds := testImg.Bounds()
	pageWidthAt72 := testBounds.Dx()
	pageHeightAt72 := testBounds.Dy()
	aspectRatio := float64(pageHeightAt72) / float64(pageWidthAt72)

	// Calculate final dimensions based on fit mode
	var finalWidth, finalHeight int
	switch d.fitMode {
	case "height":
		finalHeight = targetPixelHeight
		finalWidth = int(float64(finalHeight) / aspectRatio)
		if finalWidth > targetPixelWidth {
			finalWidth = targetPixelWidth
			finalHeight = int(float64(finalWidth) * aspectRatio)
		}
	case "width":
		finalWidth = targetPixelWidth
		finalHeight = int(float64(finalWidth) * aspectRatio)
	default: // "auto"
		finalWidth = targetPixelWidth
		finalHeight = int(float64(finalWidth) * aspectRatio)
		if finalHeight > targetPixelHeight {
			finalHeight = targetPixelHeight
			finalWidth = int(float64(finalHeight) / aspectRatio)
		}
	}

	// Calculate DPI needed to render at exactly the right size
	dpiForWidth := float64(finalWidth) / float64(pageWidthAt72) * 72.0
	dpiForHeight := float64(finalHeight) / float64(pageHeightAt72) * 72.0
	dpi := dpiForWidth
	if dpiForHeight < dpi {
		dpi = dpiForHeight
	}

	// Clamp DPI to reasonable range
	if dpi < 36 {
		dpi = 36
	}
	if dpi > 300 {
		dpi = 300
	}

	// Render at calculated DPI - no resizing needed
	img, err := d.doc.ImageDPI(pageNum, dpi)
	if err != nil {
		return "", 0, 0, err
	}

	bounds := img.Bounds()
	actualWidth := bounds.Dx()
	actualHeight := bounds.Dy()

	actualLines := int(float64(actualHeight)/pixelsPerLine) + 1
	if actualLines > termHeight {
		actualLines = termHeight
	}

	imageWidthInChars := int(float64(actualWidth)/pixelsPerChar) + 1

	filename := fmt.Sprintf("page_%d.png", pageNum)
	imagePath := filepath.Join(d.tempDir, filename)

	file, err := os.Create(imagePath)
	if err != nil {
		return "", 0, 0, err
	}
	defer file.Close()

	err = png.Encode(file, img)
	if err != nil {
		os.Remove(imagePath)
		return "", 0, 0, err
	}

	return imagePath, actualLines, imageWidthInChars, nil
}

func (d *DocumentViewer) renderWithTermImg(imagePath string, estimatedLines int, horizontalOffset int, widthChars int) int {
	if horizontalOffset > 0 {
		fmt.Printf("\033[%dC", horizontalOffset) // Move cursor right
	}

	// Use termimg fluent API to control size in terminal cells
	img, err := termimg.Open(imagePath)
	if err != nil {
		return 0
	}

	// Use ScaleNone - we already rendered at the correct size
	err = img.Width(widthChars).Height(estimatedLines).Scale(termimg.ScaleNone).Print()
	if err != nil {
		return 0
	}

	return estimatedLines
}
