package main

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"

	"github.com/blacktop/go-termimg"
	"github.com/disintegration/imaging"
)

// renderPageImage renders a page as an image using go-termimg and returns the number of lines used
func (d *DocumentViewer) renderPageImage(pageNum, maxWidth, maxHeight int) int {
	if maxHeight <= 0 {
		return 0
	}

	// Create the resized image file
	imagePath, actualHeight, imageWidthInChars, err := d.savePageAsImage(pageNum, maxWidth, maxHeight)
	if err != nil {
		return 0
	}
	defer os.Remove(imagePath) // Clean up temp file

	// Calculate horizontal centering offset
	horizontalOffset := (maxWidth - imageWidthInChars) / 2
	if horizontalOffset < 0 {
		horizontalOffset = 0
	}

	// Render with go-termimg at centered position
	return d.renderWithTermImg(imagePath, actualHeight, horizontalOffset)
}

// savePageAsImage saves a page as a PNG image file, properly sized for terminal display
func (d *DocumentViewer) savePageAsImage(pageNum, termWidth, termHeight int) (string, int, int, error) {
	// Create temp directory if it doesn't exist
	if err := os.MkdirAll(d.tempDir, 0o755); err != nil {
		return "", 0, 0, err
	}

	// Get page as image with moderate DPI
	img, err := d.doc.ImageDPI(pageNum, 200.0)
	if err != nil {
		return "", 0, 0, err
	}

	// Add padding to ensure images don't overflow
	// Reserve space on all sides
	horizontalPadding := 4 // 2 chars on each side
	verticalPadding := 2   // 1 line top and bottom

	effectiveWidth := termWidth - horizontalPadding
	effectiveHeight := termHeight - verticalPadding

	// Get cell size based on detected terminal type
	pixelsPerChar, pixelsPerLine := d.getTerminalCellSize()

	// Calculate target pixel dimensions based on effective terminal space
	targetPixelWidth := int(float64(effectiveWidth) * pixelsPerChar)
	targetPixelHeight := int(float64(effectiveHeight) * pixelsPerLine)

	// Get image dimensions
	bounds := img.Bounds()
	imgWidth := bounds.Dx()
	imgHeight := bounds.Dy()

	// Calculate aspect ratio
	aspectRatio := float64(imgHeight) / float64(imgWidth)

	// Start by fitting to width
	newWidth := targetPixelWidth
	newHeight := int(float64(newWidth) * aspectRatio)

	// If height exceeds available space, fit to height instead
	if newHeight > targetPixelHeight {
		newHeight = targetPixelHeight
		newWidth = int(float64(newHeight) / aspectRatio)
	}

	// Ensure minimum size
	if newWidth < 100 {
		newWidth = 100
	}
	if newHeight < 100 {
		newHeight = 100
	}

	// Resize the image using high-quality algorithm
	resizedImg := imaging.Resize(img, newWidth, newHeight, imaging.Lanczos)

	// Calculate how many terminal lines this will occupy
	actualLines := int(float64(newHeight)/pixelsPerLine) + 1

	// Make sure we don't exceed available height
	if actualLines > termHeight {
		actualLines = termHeight
	}

	// Calculate how many character columns the image will occupy
	imageWidthInChars := int(float64(newWidth)/pixelsPerChar) + 1

	// Create image file path
	filename := fmt.Sprintf("page_%d.png", pageNum)
	imagePath := filepath.Join(d.tempDir, filename)

	// Create and save the image file
	file, err := os.Create(imagePath)
	if err != nil {
		return "", 0, 0, err
	}
	defer file.Close()

	// Encode as PNG
	err = png.Encode(file, resizedImg)
	if err != nil {
		os.Remove(imagePath)
		return "", 0, 0, err
	}

	return imagePath, actualLines, imageWidthInChars, nil
}

// renderWithTermImg renders an image using go-termimg and returns the number of lines used
func (d *DocumentViewer) renderWithTermImg(imagePath string, estimatedLines int, horizontalOffset int) int {
	// Move cursor to the horizontal offset position for centering
	if horizontalOffset > 0 {
		fmt.Printf("\033[%dC", horizontalOffset) // Move cursor right
	}

	// Display the image
	err := termimg.PrintFile(imagePath)
	if err != nil {
		return 0
	}

	// Return estimated lines so the caller can position correctly
	return estimatedLines
}
