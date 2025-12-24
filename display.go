package main

import (
	"fmt"
	"os"
	"strings"
	"unicode"
)

func (d *DocumentViewer) displayCurrentPage() {
	termWidth, termHeight := d.getTerminalSize()
	actualPage := d.textPages[d.currentPage]

	// CRITICAL: Proper clearing and positioning sequence
	// 1. Clear entire screen first
	fmt.Print("\033[2J")
	// 2. Clear scrollback buffer
	fmt.Print("\033[3J")
	// 3. Move cursor to absolute top
	fmt.Print("\033[H")
	// 4. Move to column 1
	fmt.Print("\033[1G")
	// 5. Reset all attributes
	fmt.Print("\033[0m")
	// 6. Flush output
	os.Stdout.Sync()

	// Check what type of content this page has
	contentType := d.getPageContentType(actualPage)

	switch contentType {
	case "text":
		d.displayTextPage(actualPage, termWidth, termHeight)
	case "image":
		d.displayImagePage(actualPage, termWidth, termHeight)
	case "mixed":
		d.displayMixedPage(actualPage, termWidth, termHeight)
	default:
		d.displayTextPage(actualPage, termWidth, termHeight)
	}

	// Force cursor to stay at bottom
	fmt.Print("\033[9999;1H") // Move to bottom
	os.Stdout.Sync()
}

func (d *DocumentViewer) getPageContentType(pageNum int) string {
	// Get text content
	text, err := d.doc.Text(pageNum)
	hasText := err == nil && len(strings.Fields(strings.TrimSpace(text))) >= 3

	// Count meaningful text - more sophisticated check
	textWordCount := 0
	if err == nil {
		words := strings.Fields(strings.TrimSpace(text))
		for _, word := range words {
			// Filter out single characters and common artifacts
			if len(word) > 1 {
				textWordCount++
			}
		}
	}

	// Check for visual content
	hasVisual := d.pageHasVisualContent(pageNum)

	if textWordCount >= 50 {
		// Substantial text - show text only, even if there are images
		return "text"
	} else if textWordCount >= 3 && textWordCount < 20 && hasVisual {
		// Minimal text with images - show mixed
		return "mixed"
	} else if textWordCount < 3 && hasVisual {
		// Almost no text but has images - show image only
		return "image"
	} else if hasText {
		// Has some text - show text only
		return "text"
	} else {
		// Fallback
		return "text"
	}
}

func (d *DocumentViewer) displayTextPage(pageNum, termWidth, termHeight int) {
	text, err := d.doc.Text(pageNum)
	if err != nil {
		fmt.Printf("Error extracting text: %v\n", err)
		return
	}

	// Use more conservative margin
	effectiveWidth := termWidth - 3 // 2 chars margin on each side

	// Reflow text to fit terminal width
	reflowedLines := d.reflowText(text, effectiveWidth)

	reserved := 2 // page info + buffer
	available := termHeight - reserved

	// Display as many lines as possible within available height
	// Use absolute positioning to prevent scrolling
	row := 1
	for i, line := range reflowedLines {
		if row > available {
			break
		}
		// Move cursor to specific row and column
		fmt.Printf("\033[%d;1H", row)
		// Add left margin and print line
		fmt.Printf("  %s", line)
		row++

		if i == len(reflowedLines)-1 {
			break
		}
	}

	// Clear any remaining lines up to the page info
	for row <= available {
		fmt.Printf("\033[%d;1H", row)
		fmt.Print(strings.Repeat(" ", termWidth))
		row++
	}

	// Position cursor for page info (one blank line, then page info)
	fmt.Printf("\033[%d;1H", termHeight-1)
	fmt.Print(strings.Repeat(" ", termWidth))
	fmt.Printf("\033[%d;1H", termHeight)

	// Display page information
	d.displayPageInfo(pageNum, termWidth, "Text")
}

func (d *DocumentViewer) displayImagePage(pageNum, termWidth, termHeight int) {
	// For image-only pages, use almost the entire screen
	// Reserve space for page info at the bottom and padding
	reserved := 2        // page info + buffer
	verticalPadding := 1 // top padding
	availableHeight := termHeight - reserved - verticalPadding

	// Add top padding (blank line)
	fmt.Print("\033[1;1H")
	fmt.Print("\r\n")

	// Position cursor for image rendering (after padding)
	fmt.Print("\033[2;1H")

	// Try to render the image - this returns actual lines used
	imageHeight := d.renderPageImage(pageNum, termWidth, availableHeight)

	// If image rendering failed, show a placeholder
	if imageHeight <= 0 {
		fmt.Print("\033[2;1H")
		fmt.Printf("  [Image content - page %d]", pageNum+1)
		fmt.Print("\033[3;1H")
		fmt.Print("  (Image rendering failed)")
		imageHeight = 2
	}

	// Clear any remaining lines if image doesn't fill the space
	for row := imageHeight + 1 + verticalPadding; row <= termHeight-reserved; row++ {
		fmt.Printf("\033[%d;1H", row)
		fmt.Print(strings.Repeat(" ", termWidth))
	}

	// Position cursor for page info at the very bottom
	fmt.Printf("\033[%d;1H", termHeight)

	// Display page information
	d.displayPageInfo(pageNum, termWidth, "Image")
}

func (d *DocumentViewer) displayMixedPage(pageNum, termWidth, termHeight int) {
	reserved := 3        // page info + buffers
	verticalPadding := 1 // top padding
	available := termHeight - reserved - verticalPadding

	// For mixed pages, give images reasonable space
	maxImageHeight := available / 2 // Half screen for images
	if maxImageHeight > 12 {
		maxImageHeight = 12 // Cap at 12 lines
	}

	// Add top padding
	fmt.Print("\033[1;1H")
	fmt.Print("\r\n")

	// Position cursor for image rendering (after padding)
	fmt.Print("\033[2;1H")

	imageHeight := d.renderPageImage(pageNum, termWidth, maxImageHeight)

	// If image rendering failed, use no space
	if imageHeight <= 0 {
		imageHeight = 0
	}

	currentRow := imageHeight + 1 + verticalPadding

	// Display separator if we have both image and text
	separatorUsed := 0
	if imageHeight > 0 && available-imageHeight > 2 {
		fmt.Printf("\033[%d;1H", currentRow)
		fmt.Print(strings.Repeat("─", termWidth))
		currentRow++
		separatorUsed = 1
	}

	// Calculate remaining space for text
	textAvailable := available - imageHeight - separatorUsed
	if textAvailable > 0 {
		text, err := d.doc.Text(pageNum)
		if err == nil && strings.TrimSpace(text) != "" {
			effectiveWidth := termWidth - 4 // margin
			reflowedLines := d.reflowText(text, effectiveWidth)

			textLinesDisplayed := 0
			for i, line := range reflowedLines {
				if textLinesDisplayed >= textAvailable {
					break
				}
				// Position cursor and add left margin
				fmt.Printf("\033[%d;1H", currentRow)
				fmt.Printf("  %s", line)
				currentRow++
				textLinesDisplayed++

				if i == len(reflowedLines)-1 {
					break
				}
			}

			// Clear remaining text space
			for textLinesDisplayed < textAvailable {
				fmt.Printf("\033[%d;1H", currentRow)
				fmt.Print(strings.Repeat(" ", termWidth))
				currentRow++
				textLinesDisplayed++
			}
		} else {
			// Clear space if no text
			for i := 0; i < textAvailable; i++ {
				fmt.Printf("\033[%d;1H", currentRow)
				fmt.Print(strings.Repeat(" ", termWidth))
				currentRow++
			}
		}
	}

	// Position cursor for page info
	fmt.Printf("\033[%d;1H", termHeight-1)
	fmt.Print(strings.Repeat(" ", termWidth))
	fmt.Printf("\033[%d;1H", termHeight)

	// Display page information
	d.displayPageInfo(pageNum, termWidth, "Image+Text")
}

func (d *DocumentViewer) displayPageInfo(pageNum, termWidth int, contentType string) {
	var pageInfo string
	if d.fileType == "epub" {
		pageInfo = fmt.Sprintf("Page %d/%d (%s) - EPUB", d.currentPage+1, len(d.textPages), contentType)
	} else {
		pageInfo = fmt.Sprintf("Page %d/%d (%s) - PDF", d.currentPage+1, len(d.textPages), contentType)
	}

	if len(pageInfo) > termWidth {
		pageInfo = pageInfo[:termWidth-3] + "..."
	}

	if len(pageInfo) < termWidth {
		padding := (termWidth - len(pageInfo)) / 2
		fmt.Printf("%s%s", strings.Repeat(" ", padding), pageInfo)
	} else {
		fmt.Print(pageInfo)
	}
}

// reflowText takes raw document text and reflows it to fit the terminal width
func (d *DocumentViewer) reflowText(text string, termWidth int) []string {
	if termWidth <= 0 {
		termWidth = 80
	}

	// Clean and normalize the text
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	// For EPUB files, we might want to handle HTML entities or tags
	if d.fileType == "epub" {
		text = d.cleanEpubText(text)
	}

	// First, try to detect if text already has proper line breaks (like in manga/comics)
	lines := strings.Split(text, "\n")
	hasShortLines := false
	shortLineCount := 0
	for _, line := range lines {
		if len(strings.TrimSpace(line)) > 0 && len(strings.TrimSpace(line)) < termWidth/2 {
			shortLineCount++
		}
	}
	// If more than 30% of lines are short, preserve original formatting
	if float64(shortLineCount)/float64(len(lines)) > 0.3 {
		hasShortLines = true
	}

	var reflowedLines []string

	if hasShortLines {
		// Preserve original line breaks for pre-formatted text
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				reflowedLines = append(reflowedLines, "")
				continue
			}
			// Still wrap if line is too long
			if len(trimmed) > termWidth {
				wrapped := d.wrapText(trimmed, termWidth)
				reflowedLines = append(reflowedLines, wrapped...)
			} else {
				reflowedLines = append(reflowedLines, trimmed)
			}
		}
	} else {
		// Reflow text as paragraphs
		// Split by double newlines or more for paragraph breaks
		paragraphs := strings.Split(text, "\n\n")

		for _, paragraph := range paragraphs {
			if strings.TrimSpace(paragraph) == "" {
				reflowedLines = append(reflowedLines, "")
				continue
			}

			// Clean the paragraph - remove extra whitespace and newlines within paragraph
			cleanParagraph := strings.ReplaceAll(paragraph, "\n", " ")
			cleanParagraph = d.normalizeWhitespace(cleanParagraph)

			if strings.TrimSpace(cleanParagraph) == "" {
				continue
			}

			// Wrap the paragraph to terminal width
			wrappedLines := d.wrapText(cleanParagraph, termWidth)
			reflowedLines = append(reflowedLines, wrappedLines...)

			// Add empty line after paragraph
			reflowedLines = append(reflowedLines, "")
		}
	}

	// Remove trailing empty lines
	for len(reflowedLines) > 0 && reflowedLines[len(reflowedLines)-1] == "" {
		reflowedLines = reflowedLines[:len(reflowedLines)-1]
	}

	return reflowedLines
}

// cleanEpubText removes common HTML entities and tags that might appear in EPUB text
func (d *DocumentViewer) cleanEpubText(text string) string {
	// Replace common HTML entities
	replacements := map[string]string{
		"&nbsp;":  " ",
		"&amp;":   "&",
		"&lt;":    "<",
		"&gt;":    ">",
		"&quot;":  "\"",
		"&apos;":  "'",
		"&#8217;": "'",
		"&#8220;": "\"",
		"&#8221;": "\"",
		"&#8230;": "...",
		"&#8212;": "—",
		"&#8211;": "–",
	}

	for entity, replacement := range replacements {
		text = strings.ReplaceAll(text, entity, replacement)
	}

	return text
}

// normalizeWhitespace replaces multiple spaces/tabs with single spaces
func (d *DocumentViewer) normalizeWhitespace(text string) string {
	var result strings.Builder
	var lastWasSpace bool

	for _, r := range text {
		if unicode.IsSpace(r) {
			if !lastWasSpace {
				result.WriteRune(' ')
				lastWasSpace = true
			}
		} else {
			result.WriteRune(r)
			lastWasSpace = false
		}
	}

	return strings.TrimSpace(result.String())
}

// wrapText wraps text to specified width using word wrapping
func (d *DocumentViewer) wrapText(text string, width int) []string {
	if width <= 0 {
		width = 80 // Fallback
	}

	// Ensure we have at least some reasonable width
	if width < 20 {
		width = 20
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}

	var lines []string
	var currentLine strings.Builder

	for _, word := range words {
		// Handle very long words that exceed width
		if len(word) > width {
			// If current line has content, finish it first
			if currentLine.Len() > 0 {
				lines = append(lines, currentLine.String())
				currentLine.Reset()
			}
			// Break the long word across multiple lines
			for len(word) > width {
				lines = append(lines, word[:width])
				word = word[width:]
			}
			// Add remaining part
			if len(word) > 0 {
				currentLine.WriteString(word)
			}
			continue
		}

		// Check if adding this word would exceed the width
		proposedLength := currentLine.Len()
		if proposedLength > 0 {
			proposedLength += 1 // for the space
		}
		proposedLength += len(word)

		if proposedLength <= width {
			// Add word to current line
			if currentLine.Len() > 0 {
				currentLine.WriteString(" ")
			}
			currentLine.WriteString(word)
		} else {
			// Start new line with this word
			if currentLine.Len() > 0 {
				lines = append(lines, currentLine.String())
				currentLine.Reset()
			}
			currentLine.WriteString(word)
		}
	}

	// Add the last line if it has content
	if currentLine.Len() > 0 {
		lines = append(lines, currentLine.String())
	}

	return lines
}

func (d *DocumentViewer) showHelp() {
	fmt.Print("\033[2J\033[H") // clear screen
	termWidth, _ := d.getTerminalSize()

	fmt.Println(strings.Repeat("=", termWidth))
	fmt.Printf("%s Viewer Help\n", strings.ToUpper(d.fileType))
	fmt.Println(strings.Repeat("=", termWidth))
	fmt.Println()
	fmt.Println("Navigation:")
	fmt.Println("  j or Space  - Next page/chapter")
	fmt.Println("  k           - Previous page/chapter")
	fmt.Println("  g           - Go to specific page/chapter")
	fmt.Println("  h           - Show this help")
	fmt.Println("  q           - Quit")
	fmt.Println()
	fmt.Println("Features:")
	fmt.Println("  - Text is reflowed to fit terminal width")
	fmt.Println("  - Images are rendered using high-resolution terminal graphics (Sixel/Kitty/etc)")
	fmt.Println("  - Pages with text, images, or both are shown")
	fmt.Println("  - Paragraphs are preserved with proper spacing")
	if d.fileType == "epub" {
		fmt.Println("  - HTML entities are converted to readable text")
	}
	fmt.Println()
	fmt.Println("Requirements:")

	fmt.Println("  - Terminal with truecolor support recommended")
	fmt.Println()
	fmt.Println("Supported formats: PDF, EPUB")
	fmt.Println()
	fmt.Println(strings.Repeat("=", termWidth))
	fmt.Println("Press any key to return...")
	d.readSingleChar()
}
