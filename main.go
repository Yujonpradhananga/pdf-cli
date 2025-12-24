package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("╔══════════════════════════════════════════════════╗")
	fmt.Println("║     Terminal PDF/EPUB Viewer                    ║")
	fmt.Println("╚══════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("Choose an option:")
	fmt.Println("  1. Search for a file (recommended)")
	fmt.Println("  2. Enter file path manually")
	fmt.Println()
	fmt.Print("Selection (1/2): ")

	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

	var filePath string
	var err error

	switch choice {
	case "1", "":
		// Use file picker (default)
		filePath, err = selectFileWithPicker()
		if err != nil {
			fmt.Printf("File selection cancelled: %v\n", err)
			return
		}
	case "2":
		// Manual path entry
		filePath, err = getFilePathManually(reader)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
	default:
		fmt.Println("Invalid selection. Exiting.")
		return
	}

	if filePath == "" {
		fmt.Println("No file selected. Exiting.")
		return
	}

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		fmt.Printf("File not found at: %s\n", filePath)
		return
	}

	// Check file extension
	ext := strings.ToLower(filepath.Ext(filePath))
	if ext != ".pdf" && ext != ".epub" && ext != ".docx" {
		fmt.Printf("Unsupported file format: %s\nSupported formats: .pdf, .epub, .docx\n", ext)
		return
	}

	fmt.Printf("\nOpening: %s\n", filePath)

	// Start the document viewer
	viewer := NewDocumentViewer(filePath)
	if err := viewer.Open(); err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		return
	}

	viewer.Run()
}

func selectFileWithPicker() (string, error) {
	// Create file searcher
	searcher := NewFileSearcher()

	// Scan for files
	if err := searcher.ScanDirectories(); err != nil {
		return "", fmt.Errorf("error scanning directories: %v", err)
	}

	// Check if any files were found
	allFiles := searcher.GetAllFiles()
	if len(allFiles) == 0 {
		return "", fmt.Errorf("no PDF or EPUB files found in common directories")
	}

	// Create and run picker
	picker := NewFilePicker(searcher)
	return picker.Run()
}

func getFilePathManually(reader *bufio.Reader) (string, error) {
	fmt.Print("Enter the path to your PDF or EPUB file: ")
	filePath, _ := reader.ReadString('\n')
	filePath = strings.TrimSpace(filePath)

	if filePath == "" {
		return "", fmt.Errorf("no file path provided")
	}

	// Handle quoted paths
	if (strings.HasPrefix(filePath, "\"") && strings.HasSuffix(filePath, "\"")) ||
		(strings.HasPrefix(filePath, "'") && strings.HasSuffix(filePath, "'")) {
		filePath = filePath[1 : len(filePath)-1]
	}

	// Expand home directory if needed
	if strings.HasPrefix(filePath, "~/") {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			filePath = filepath.Join(homeDir, filePath[2:])
		}
	}

	return filePath, nil
}
