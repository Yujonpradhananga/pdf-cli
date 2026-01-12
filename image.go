package main

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"

	"github.com/blacktop/go-termimg"
	"github.com/disintegration/imaging"
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

	return d.renderWithTermImg(imagePath, actualHeight, horizontalOffset)
}

func (d *DocumentViewer) savePageAsImage(pageNum, termWidth, termHeight int) (string, int, int, error) {
	if err := os.MkdirAll(d.tempDir, 0o755); err != nil {
		return "", 0, 0, err
	}

	img, err := d.doc.ImageDPI(pageNum, 300.0)
	if err != nil {
		return "", 0, 0, err
	}

	horizontalPadding := 4 // 2 chars on each side
	verticalPadding := 2   // 1 line top and bottom

	effectiveWidth := termWidth - horizontalPadding
	effectiveHeight := termHeight - verticalPadding

	pixelsPerChar, pixelsPerLine := d.getTerminalCellSize()

	targetPixelWidth := int(float64(effectiveWidth) * pixelsPerChar)
	targetPixelHeight := int(float64(effectiveHeight) * pixelsPerLine)

	bounds := img.Bounds()
	imgWidth := bounds.Dx()
	imgHeight := bounds.Dy()

	aspectRatio := float64(imgHeight) / float64(imgWidth)

	newWidth := targetPixelWidth
	newHeight := int(float64(newWidth) * aspectRatio)

	if newHeight > targetPixelHeight {
		newHeight = targetPixelHeight
		newWidth = int(float64(newHeight) / aspectRatio)
	}

	if newWidth < 100 {
		newWidth = 100
	}
	if newHeight < 100 {
		newHeight = 100
	}

	resizedImg := imaging.Resize(img, newWidth, newHeight, imaging.Lanczos)

	actualLines := int(float64(newHeight)/pixelsPerLine) + 1

	if actualLines > termHeight {
		actualLines = termHeight
	}

	imageWidthInChars := int(float64(newWidth)/pixelsPerChar) + 1

	filename := fmt.Sprintf("page_%d.png", pageNum)
	imagePath := filepath.Join(d.tempDir, filename)

	file, err := os.Create(imagePath)
	if err != nil {
		return "", 0, 0, err
	}
	defer file.Close()

	err = png.Encode(file, resizedImg)
	if err != nil {
		os.Remove(imagePath)
		return "", 0, 0, err
	}

	return imagePath, actualLines, imageWidthInChars, nil
}

func (d *DocumentViewer) renderWithTermImg(imagePath string, estimatedLines int, horizontalOffset int) int {
	if horizontalOffset > 0 {
		fmt.Printf("\033[%dC", horizontalOffset) // Move cursor right
	}

	err := termimg.PrintFile(imagePath)
	if err != nil {
		return 0
	}

	return estimatedLines
}
