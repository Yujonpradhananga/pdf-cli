# Dual Page View Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a 2-page view mode with vertical/horizontal layouts, 1-page arrow navigation, and Shift+Arrow 2-page jumps.

**Architecture:** New `dualPageMode` state on `DocumentViewer` controls rendering. Key `2` cycles off→vertical→horizontal. Dual mode always renders both pages as images at half-dimensions. Shift+Arrow escape sequences detected in `readSingleChar` for 2-page jumps.

**Tech Stack:** Go, existing go-fitz + go-termimg pipeline, ANSI/Kitty terminal escape sequences.

---

### Task 1: Add state and key cycling

**Files:**
- Modify: `document.go:19-44` (DocumentViewer struct)
- Modify: `document.go:509-593` (handleInput)

**Step 1: Add `dualPageMode` field to DocumentViewer struct**

In `document.go`, add after the `darkMode` field (line 43):

```go
dualPageMode  string // "": off, "vertical": stacked, "horizontal": side-by-side
```

**Step 2: Add key `2` handler in handleInput**

In `document.go` `handleInput()`, add a new case before the `case 27:` line:

```go
case '2':
    switch d.dualPageMode {
    case "":
        d.dualPageMode = "vertical"
    case "vertical":
        d.dualPageMode = "horizontal"
    default:
        d.dualPageMode = ""
    }
```

**Step 3: Build and verify it compiles**

Run: `cd /Users/leo/__code/CLI-PDF-EPUB-reader && go build .`
Expected: Compiles successfully

**Step 4: Commit**

```bash
git add document.go
git commit -m "feat: add dualPageMode state and key 2 cycling"
```

---

### Task 2: Detect Shift+Arrow escape sequences

**Files:**
- Modify: `terminal.go:248-275` (readSingleChar)

**Step 1: Extend readSingleChar to handle modifier arrow sequences**

Shift+Arrow sends `ESC [ 1 ; 2 A/B/C/D`. The current code reads ESC then 2 bytes (`[` + letter). We need to handle the longer `[ 1 ; 2 letter` sequence.

Replace the escape sequence handling block in `readSingleChar()` (lines 256-272):

```go
// Handle escape sequences (arrow keys, shift+arrow)
if buf[0] == 27 {
    seq := make([]byte, 5)
    n, _ := os.Stdin.Read(seq[:2])
    if n >= 2 && seq[0] == '[' {
        switch seq[1] {
        case 'A':
            return 'k'
        case 'B':
            return 'j'
        case 'C':
            return 'j'
        case 'D':
            return 'k'
        case '1':
            // Could be shift+arrow: ESC [ 1 ; 2 A/B/C/D
            n2, _ := os.Stdin.Read(seq[2:5])
            if n2 >= 3 && seq[2] == ';' && seq[3] == '2' {
                switch seq[4] {
                case 'A': // Shift+Up
                    return 'K' // uppercase = shift
                case 'B': // Shift+Down
                    return 'J' // uppercase = shift
                case 'C': // Shift+Right
                    return 'J' // uppercase = shift (forward)
                case 'D': // Shift+Left
                    return 'K' // uppercase = shift (backward)
                }
            }
        }
    }
    return 27 // Plain ESC
}
```

We map Shift+Arrow to `J`/`K` (uppercase) as internal signals.

**Step 2: Add Shift+Arrow (J/K) handler in handleInput**

In `document.go` `handleInput()`, add cases for `J` and `K`:

```go
case 'J': // Shift+Down/Right: jump 2 pages (in dual mode)
    if d.dualPageMode != "" {
        if d.currentPage < len(d.textPages)-2 {
            d.currentPage += 2
        } else if d.currentPage < len(d.textPages)-1 {
            d.currentPage = len(d.textPages) - 1
        }
    }
case 'K': // Shift+Up/Left: jump back 2 pages (in dual mode)
    if d.dualPageMode != "" {
        if d.currentPage >= 2 {
            d.currentPage -= 2
        } else {
            d.currentPage = 0
        }
    }
```

**Step 3: Build and verify**

Run: `cd /Users/leo/__code/CLI-PDF-EPUB-reader && go build .`
Expected: Compiles successfully

**Step 4: Commit**

```bash
git add terminal.go document.go
git commit -m "feat: detect Shift+Arrow keys for 2-page jumps in dual mode"
```

---

### Task 3: Implement dual page rendering

**Files:**
- Modify: `display.go:10-47` (displayCurrentPage)
- Modify: `display.go` (new function displayDualPage)

**Step 1: Route to dual page display in displayCurrentPage**

In `display.go` `displayCurrentPage()`, after the `fmt.Print("\033[0m")` line (line 29), add a check that routes to the new dual display function when in dual mode:

```go
if d.dualPageMode != "" {
    d.displayDualPage(termWidth, termHeight)
    fmt.Print("\033[9999;1H")
    fmt.Print("\033[?2026l")
    os.Stdout.Sync()
    return
}
```

**Step 2: Add displayDualPage function**

Add new function in `display.go`:

```go
func (d *DocumentViewer) displayDualPage(termWidth, termHeight int) {
	page1 := d.textPages[d.currentPage]
	hasPage2 := d.currentPage+1 < len(d.textPages)

	reserved := 2 // status bar

	if d.dualPageMode == "vertical" {
		d.displayDualVertical(page1, hasPage2, termWidth, termHeight, reserved)
	} else {
		d.displayDualHorizontal(page1, hasPage2, termWidth, termHeight, reserved)
	}
}

func (d *DocumentViewer) displayDualVertical(page1 int, hasPage2 bool, termWidth, termHeight, reserved int) {
	availableHeight := termHeight - reserved
	halfHeight := availableHeight / 2

	// Render page 1 in top half
	fmt.Print("\033[1;1H")
	imgHeight1 := d.renderPageImage(page1, termWidth, halfHeight)
	if imgHeight1 <= 0 {
		fmt.Print("\033[1;1H")
		fmt.Printf("  [Page %d - render failed]", page1+1)
		imgHeight1 = 1
	}

	// Render page 2 in bottom half
	startRow2 := halfHeight + 1
	fmt.Printf("\033[%d;1H", startRow2)

	if hasPage2 {
		page2 := d.textPages[d.currentPage+1]
		imgHeight2 := d.renderPageImage(page2, termWidth, halfHeight)
		if imgHeight2 <= 0 {
			fmt.Printf("  [Page %d - render failed]", page2+1)
		}
	} else {
		// No second page - show placeholder
		fmt.Print("  [End of document]")
	}

	// Clear remaining lines
	clearStart := termHeight - reserved + 1
	for row := clearStart; row < termHeight; row++ {
		fmt.Printf("\033[%d;1H\033[K", row)
	}

	// Status bar
	fmt.Printf("\033[%d;1H", termHeight)
	d.displayDualPageInfo(page1, hasPage2, termWidth, "2pg-v")
}

func (d *DocumentViewer) displayDualHorizontal(page1 int, hasPage2 bool, termWidth, termHeight, reserved int) {
	availableHeight := termHeight - reserved
	halfWidth := termWidth / 2

	// Render page 1 on left half
	fmt.Print("\033[1;1H")
	imgHeight1 := d.renderPageImage(page1, halfWidth, availableHeight)
	if imgHeight1 <= 0 {
		fmt.Print("\033[1;1H")
		fmt.Printf("  [Page %d - render failed]", page1+1)
	}

	// Render page 2 on right half
	fmt.Printf("\033[1;%dH", halfWidth+1)

	if hasPage2 {
		page2 := d.textPages[d.currentPage+1]
		imgHeight2 := d.renderPageImage(page2, halfWidth, availableHeight)
		if imgHeight2 <= 0 {
			fmt.Printf("  [Page %d - render failed]", page2+1)
		}
	} else {
		fmt.Print("  [End of document]")
	}

	// Status bar
	fmt.Printf("\033[%d;1H", termHeight)
	d.displayDualPageInfo(page1, hasPage2, termWidth, "2pg-h")
}
```

**Step 3: Add displayDualPageInfo function**

```go
func (d *DocumentViewer) displayDualPageInfo(page1 int, hasPage2 bool, termWidth int, modeLabel string) {
	page1Num := d.currentPage + 1
	page2Num := page1Num + 1
	totalPages := len(d.textPages)

	var pageRange string
	if hasPage2 {
		pageRange = fmt.Sprintf("Pages %d-%d/%d", page1Num, page2Num, totalPages)
	} else {
		pageRange = fmt.Sprintf("Page %d/%d", page1Num, totalPages)
	}

	fitIndicator := fmt.Sprintf(" [fit:%s]", d.fitMode)
	scaleIndicator := ""
	if d.isReflowable {
		zoomPct := 595 * 100 / d.htmlPageWidth
		scaleIndicator = fmt.Sprintf(" [zoom:%d%%]", zoomPct)
	} else if d.scaleFactor != 1.0 {
		scaleIndicator = fmt.Sprintf(" [%.0f%%]", d.scaleFactor*100)
	}
	darkIndicator := ""
	switch d.darkMode {
	case "smart":
		darkIndicator = " [dark]"
	case "invert":
		darkIndicator = " [dark:inv]"
	}
	searchIndicator := ""
	if d.searchQuery != "" {
		if len(d.searchHits) > 0 {
			searchIndicator = fmt.Sprintf(" [/%s: %d/%d]", d.searchQuery, d.searchHitIdx+1, len(d.searchHits))
		} else {
			searchIndicator = fmt.Sprintf(" [/%s: no matches]", d.searchQuery)
		}
	}

	typeLabel := strings.ToUpper(d.fileType)
	pageInfo := fmt.Sprintf("%s (Image) [%s]%s%s%s%s - %s",
		pageRange, modeLabel, fitIndicator, scaleIndicator, darkIndicator, searchIndicator, typeLabel)

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
```

**Step 4: Build and verify**

Run: `cd /Users/leo/__code/CLI-PDF-EPUB-reader && go build .`
Expected: Compiles successfully

**Step 5: Manual test**

Run: `./CLI-PDF-EPUB-reader <some-pdf-file>`
- Press `2` → should show vertical dual page view
- Press `2` again → should show horizontal dual page view
- Press `2` again → should return to single page view
- In dual mode, arrows should flip by 1 page
- In dual mode, Shift+Arrow should flip by 2 pages

**Step 6: Commit**

```bash
git add display.go
git commit -m "feat: implement dual page rendering (vertical and horizontal layouts)"
```

---

### Task 4: Update help screen and status bar

**Files:**
- Modify: `display.go:484-529` (showHelp)

**Step 1: Add dual page entries to help screen**

In `showHelp()`, after the `"  r                   - Refresh cell size..."` line, add:

```go
fmt.Println()
fmt.Println("Dual Page View:")
fmt.Println("  2                   - Cycle dual page (off/vertical/horizontal)")
fmt.Println("  Shift+Left/Right    - Jump 2 pages (in dual page mode)")
```

**Step 2: Build and verify**

Run: `cd /Users/leo/__code/CLI-PDF-EPUB-reader && go build .`

**Step 3: Commit**

```bash
git add display.go
git commit -m "feat: add dual page view to help screen"
```
