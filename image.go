package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"

	"github.com/blacktop/go-termimg"
)

func (d *DocumentViewer) renderPageImage(pageNum, maxWidth, maxHeight int) int {
	if maxHeight <= 0 {
		return 0
	}

	termType := d.detectTerminalType()
	imagePath, actualHeight, imageWidthInChars, actualPixelWidth, actualPixelHeight, err := d.savePageAsImage(pageNum, maxWidth, maxHeight, termType)
	if err != nil {
		return 0
	}
	defer os.Remove(imagePath)

	horizontalOffset := (maxWidth - imageWidthInChars) / 2
	if horizontalOffset < 0 {
		horizontalOffset = 0
	}

	return d.renderWithTermImg(imagePath, actualHeight, horizontalOffset, imageWidthInChars, actualPixelWidth, actualPixelHeight, termType)
}

func (d *DocumentViewer) savePageAsImage(pageNum, termWidth, termHeight int, termType string) (string, int, int, int, int, error) {
	if err := os.MkdirAll(d.tempDir, 0o755); err != nil {
		return "", 0, 0, 0, 0, err
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
		return "", 0, 0, 0, 0, err
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
	// Sixel terminals (Foot) are slower, so use lower max DPI for better performance
	if dpi < 36 {
		dpi = 36
	}
	maxDPI := 300.0
	if termType != "kitty" {
		// Sixel terminals: reduce max DPI significantly for faster rendering
		// 100 DPI is still very readable while being much faster to encode
		maxDPI = 100.0
	}
	if dpi > maxDPI {
		dpi = maxDPI
	}

	// Render at calculated DPI - no resizing needed
	img, err := d.doc.ImageDPI(pageNum, dpi)
	if err != nil {
		return "", 0, 0, 0, 0, err
	}

	// Apply dark mode
	var finalImg image.Image = img
	switch d.darkMode {
	case "smart":
		finalImg = smartInvert(img)
	case "invert":
		finalImg = simpleInvert(img)
	}

	bounds := finalImg.Bounds()
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
		return "", 0, 0, 0, 0, err
	}
	defer file.Close()

	err = png.Encode(file, finalImg)
	if err != nil {
		os.Remove(imagePath)
		return "", 0, 0, 0, 0, err
	}

	return imagePath, actualLines, imageWidthInChars, actualWidth, actualHeight, nil
}

func (d *DocumentViewer) renderWithTermImg(imagePath string, estimatedLines int, horizontalOffset int, widthChars int, pixelWidth int, pixelHeight int, termType string) int {
	if horizontalOffset > 0 {
		fmt.Printf("\033[%dC", horizontalOffset) // Move cursor right
	}

	// Use termimg fluent API to control size in terminal cells
	img, err := termimg.Open(imagePath)
	if err != nil {
		return 0
	}

	// Choose rendering strategy based on terminal type:
	// - Kitty uses native graphics protocol with character-based dimensions
	// - Sixel terminals need pixel-based dimensions for proper scaling
	if termType == "kitty" {
		// Kitty: use character-based dimensions with ScaleNone
		err = img.Width(widthChars).Height(estimatedLines).Scale(termimg.ScaleNone).Print()
	} else {
		// Sixel terminals (Foot, xterm, etc.): use pixel-based dimensions with ScaleFit
		err = img.WidthPixels(pixelWidth).HeightPixels(pixelHeight).Scale(termimg.ScaleFit).Print()
	}

	if err != nil {
		return 0
	}

	return estimatedLines
}

// smartInvert inverts lightness while preserving hue and saturation.
// White backgrounds become black, black text becomes white, colors keep their hue.
func smartInvert(src image.Image) image.Image {
	bounds := src.Bounds()
	dst := image.NewRGBA(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := src.At(x, y).RGBA()
			r8 := float64(r>>8) / 255.0
			g8 := float64(g>>8) / 255.0
			b8 := float64(b>>8) / 255.0

			h, s, l := rgbToHSL(r8, g8, b8)
			l = 0.12 + (1.0-l)*0.88 // invert lightness; dark gray bg instead of pure black
			nr, ng, nb := hslToRGB(h, s, l)

			dst.Set(x, y, color.RGBA{
				R: uint8(nr * 255),
				G: uint8(ng * 255),
				B: uint8(nb * 255),
				A: uint8(a >> 8),
			})
		}
	}
	return dst
}

// simpleInvert does a full RGB color inversion with the same gray background shift.
func simpleInvert(src image.Image) image.Image {
	bounds := src.Bounds()
	dst := image.NewRGBA(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := src.At(x, y).RGBA()
			// Invert and remap to gray bg range: 255→30, 0→255
			nr := 30 + (255-r>>8)*225/255
			ng := 30 + (255-g>>8)*225/255
			nb := 30 + (255-b>>8)*225/255
			dst.Set(x, y, color.RGBA{
				R: uint8(nr),
				G: uint8(ng),
				B: uint8(nb),
				A: uint8(a >> 8),
			})
		}
	}
	return dst
}

func rgbToHSL(r, g, b float64) (h, s, l float64) {
	max := math.Max(r, math.Max(g, b))
	min := math.Min(r, math.Min(g, b))
	l = (max + min) / 2

	if max == min {
		return 0, 0, l
	}

	d := max - min
	if l > 0.5 {
		s = d / (2.0 - max - min)
	} else {
		s = d / (max + min)
	}

	switch max {
	case r:
		h = (g - b) / d
		if g < b {
			h += 6
		}
	case g:
		h = (b-r)/d + 2
	case b:
		h = (r-g)/d + 4
	}
	h /= 6
	return
}

func hslToRGB(h, s, l float64) (r, g, b float64) {
	if s == 0 {
		return l, l, l
	}

	var q float64
	if l < 0.5 {
		q = l * (1 + s)
	} else {
		q = l + s - l*s
	}
	p := 2*l - q

	r = hueToRGB(p, q, h+1.0/3.0)
	g = hueToRGB(p, q, h)
	b = hueToRGB(p, q, h-1.0/3.0)
	return
}

func hueToRGB(p, q, t float64) float64 {
	if t < 0 {
		t++
	}
	if t > 1 {
		t--
	}
	if t < 1.0/6.0 {
		return p + (q-p)*6*t
	}
	if t < 1.0/2.0 {
		return q
	}
	if t < 2.0/3.0 {
		return p + (q-p)*(2.0/3.0-t)*6
	}
	return p
}
