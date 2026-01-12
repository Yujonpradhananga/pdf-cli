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
	fmt.Print("\033[2J")
	fmt.Print("\033[3J")
	fmt.Print("\033[H")
	fmt.Print("\033[1G")
	fmt.Print("\033[0m")
	os.Stdout.Sync()
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
	fmt.Print("\033[9999;1H")
	os.Stdout.Sync()
}

func (d *DocumentViewer) getPageContentType(pageNum int) string {
	// Honor forced mode if set (toggled with 't')
	if d.forceMode == "text" {
		return "text"
	}
	if d.forceMode == "image" {
		return "image"
	}

	// For PDFs, prefer image rendering - it's more faithful to the original
	// especially for math, diagrams, and formatted content
	if d.fileType == "pdf" {
		if d.pageHasVisualContent(pageNum) {
			return "image"
		}
	}

	// For EPUBs or PDFs without visual content, use text-based logic
	text, err := d.doc.Text(pageNum)
	hasText := err == nil && len(strings.Fields(strings.TrimSpace(text))) >= 3
	textWordCount := 0
	if err == nil {
		words := strings.Fields(strings.TrimSpace(text))
		for _, word := range words {
			if len(word) > 1 {
				textWordCount++
			}
		}
	}
	hasVisual := d.pageHasVisualContent(pageNum)
	if textWordCount >= 50 {
		return "text"
	} else if textWordCount >= 3 && textWordCount < 20 && hasVisual {
		return "mixed"
	} else if textWordCount < 3 && hasVisual {
		return "image"
	} else if hasText {
		return "text"
	} else {
		return "text"
	}
}

func (d *DocumentViewer) displayTextPage(pageNum, termWidth, termHeight int) {
	text, err := d.doc.Text(pageNum)
	if err != nil {
		fmt.Printf("Error extracting text: %v\n", err)
		return
	}
	effectiveWidth := termWidth - 3
	reflowedLines := d.reflowText(text, effectiveWidth)
	reserved := 2
	available := termHeight - reserved
	row := 1
	for i, line := range reflowedLines {
		if row > available {
			break
		}
		fmt.Printf("\033[%d;1H", row)
		fmt.Printf("  %s", line)
		row++
		if i == len(reflowedLines)-1 {
			break
		}
	}
	for row <= available {
		fmt.Printf("\033[%d;1H", row)
		fmt.Print(strings.Repeat(" ", termWidth))
		row++
	}
	fmt.Printf("\033[%d;1H", termHeight-1)
	fmt.Print(strings.Repeat(" ", termWidth))
	fmt.Printf("\033[%d;1H", termHeight)
	// page info
	d.displayPageInfo(pageNum, termWidth, "Text")
}

func (d *DocumentViewer) displayImagePage(pageNum, termWidth, termHeight int) {
	reserved := 2
	verticalPadding := 1 // top padding
	availableHeight := termHeight - reserved - verticalPadding
	fmt.Print("\033[1;1H")
	fmt.Print("\r\n")
	fmt.Print("\033[2;1H")
	imageHeight := d.renderPageImage(pageNum, termWidth, availableHeight)
	if imageHeight <= 0 {
		fmt.Print("\033[2;1H")
		fmt.Printf("  [Image content - page %d]", pageNum+1)
		fmt.Print("\033[3;1H")
		fmt.Print("  (Image rendering failed)")
		imageHeight = 2
	}
	for row := imageHeight + 1 + verticalPadding; row <= termHeight-reserved; row++ {
		fmt.Printf("\033[%d;1H", row)
		fmt.Print(strings.Repeat(" ", termWidth))
	}
	fmt.Printf("\033[%d;1H", termHeight)
	d.displayPageInfo(pageNum, termWidth, "Image")
}

func (d *DocumentViewer) displayMixedPage(pageNum, termWidth, termHeight int) {
	reserved := 3
	verticalPadding := 1
	available := termHeight - reserved - verticalPadding
	maxImageHeight := available / 2
	if maxImageHeight > 12 {
		maxImageHeight = 12
	}
	fmt.Print("\033[1;1H")
	fmt.Print("\r\n")
	fmt.Print("\033[2;1H")
	imageHeight := d.renderPageImage(pageNum, termWidth, maxImageHeight)
	if imageHeight <= 0 {
		imageHeight = 0
	}
	currentRow := imageHeight + 1 + verticalPadding
	separatorUsed := 0
	if imageHeight > 0 && available-imageHeight > 2 {
		fmt.Printf("\033[%d;1H", currentRow)
		fmt.Print(strings.Repeat("─", termWidth))
		currentRow++
		separatorUsed = 1
	}
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
				fmt.Printf("\033[%d;1H", currentRow)
				fmt.Printf("  %s", line)
				currentRow++
				textLinesDisplayed++
				if i == len(reflowedLines)-1 {
					break
				}
			}
			for textLinesDisplayed < textAvailable {
				fmt.Printf("\033[%d;1H", currentRow)
				fmt.Print(strings.Repeat(" ", termWidth))
				currentRow++
				textLinesDisplayed++
			}
		} else {
			for i := 0; i < textAvailable; i++ {
				fmt.Printf("\033[%d;1H", currentRow)
				fmt.Print(strings.Repeat(" ", termWidth))
				currentRow++
			}
		}
	}
	fmt.Printf("\033[%d;1H", termHeight-1)
	fmt.Print(strings.Repeat(" ", termWidth))
	fmt.Printf("\033[%d;1H", termHeight)
	d.displayPageInfo(pageNum, termWidth, "Image+Text")
}

func (d *DocumentViewer) displayPageInfo(pageNum, termWidth int, contentType string) {
	modeIndicator := ""
	if d.forceMode != "" {
		modeIndicator = fmt.Sprintf(" [%s]", d.forceMode)
	}
	fitIndicator := ""
	if d.fitMode != "height" {
		fitIndicator = fmt.Sprintf(" [fit:%s]", d.fitMode)
	}
	var pageInfo string
	if d.fileType == "epub" {
		pageInfo = fmt.Sprintf("Page %d/%d (%s)%s%s - EPUB  [b]ack [t]ext/img [f]it [h]elp [q]uit", d.currentPage+1, len(d.textPages), contentType, modeIndicator, fitIndicator)
	} else {
		pageInfo = fmt.Sprintf("Page %d/%d (%s)%s%s - PDF  [b]ack [t]ext/img [f]it [h]elp [q]uit", d.currentPage+1, len(d.textPages), contentType, modeIndicator, fitIndicator)
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

func (d *DocumentViewer) reflowText(text string, termWidth int) []string {
	if termWidth <= 0 {
		termWidth = 80
	}
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	if d.fileType == "epub" {
		text = d.cleanEpubText(text)
	}
	lines := strings.Split(text, "\n")
	hasShortLines := false
	shortLineCount := 0
	for _, line := range lines {
		if len(strings.TrimSpace(line)) > 0 && len(strings.TrimSpace(line)) < termWidth/2 {
			shortLineCount++
		}
	}
	if float64(shortLineCount)/float64(len(lines)) > 0.3 {
		hasShortLines = true
	}
	var reflowedLines []string
	if hasShortLines {
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				reflowedLines = append(reflowedLines, "")
				continue
			}
			if len(trimmed) > termWidth {
				wrapped := d.wrapText(trimmed, termWidth)
				reflowedLines = append(reflowedLines, wrapped...)
			} else {
				reflowedLines = append(reflowedLines, trimmed)
			}
		}
	} else {
		paragraphs := strings.Split(text, "\n\n")
		for _, paragraph := range paragraphs {
			if strings.TrimSpace(paragraph) == "" {
				reflowedLines = append(reflowedLines, "")
				continue
			}
			cleanParagraph := strings.ReplaceAll(paragraph, "\n", " ")
			cleanParagraph = d.normalizeWhitespace(cleanParagraph)
			if strings.TrimSpace(cleanParagraph) == "" {
				continue
			}
			wrappedLines := d.wrapText(cleanParagraph, termWidth)
			reflowedLines = append(reflowedLines, wrappedLines...)
			reflowedLines = append(reflowedLines, "")
		}
	}
	for len(reflowedLines) > 0 && reflowedLines[len(reflowedLines)-1] == "" {
		reflowedLines = reflowedLines[:len(reflowedLines)-1]
	}
	return reflowedLines
}

func (d *DocumentViewer) cleanEpubText(text string) string {
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

func (d *DocumentViewer) wrapText(text string, width int) []string {
	if width <= 0 {
		width = 80 // Fallback
	}
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
		if len(word) > width {
			if currentLine.Len() > 0 {
				lines = append(lines, currentLine.String())
				currentLine.Reset()
			}
			for len(word) > width {
				lines = append(lines, word[:width])
				word = word[width:]
			}
			if len(word) > 0 {
				currentLine.WriteString(word)
			}
			continue
		}
		proposedLength := currentLine.Len()
		if proposedLength > 0 {
			proposedLength += 1 // for the space
		}
		proposedLength += len(word)
		if proposedLength <= width {
			if currentLine.Len() > 0 {
				currentLine.WriteString(" ")
			}
			currentLine.WriteString(word)
		} else {
			if currentLine.Len() > 0 {
				lines = append(lines, currentLine.String())
				currentLine.Reset()
			}
			currentLine.WriteString(word)
		}
	}
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
	fmt.Println("  j or Space  - Next page")
	fmt.Println("  k           - Previous page")
	fmt.Println("  g           - Go to specific page")
	fmt.Println("  b           - Back to file list")
	fmt.Println("  t           - Toggle view mode (auto → text → image)")
	fmt.Println("  f           - Toggle fit-to-height")
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
