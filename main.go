package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	var filePath string
	var err error

	// Determine target: argument or current directory
	arg := "."
	if len(os.Args) > 1 {
		arg = os.Args[1]
	}

	// Expand ~ to home directory
	if strings.HasPrefix(arg, "~/") {
		homeDir, _ := os.UserHomeDir()
		arg = filepath.Join(homeDir, arg[2:])
	}

	// Check if argument is a directory or file
	info, statErr := os.Stat(arg)
	if statErr != nil {
		fmt.Printf("Path not found: %s\n", arg)
		return
	}

	if info.IsDir() {
		// It's a directory - search within it
		filePath, err = selectFileWithPickerInDir(arg)
		if err != nil {
			fmt.Printf("File selection cancelled: %v\n", err)
			return
		}
	} else {
		// It's a file - use directly
		filePath = arg
	}

	if filePath == "" {
		fmt.Println("No file selected. Exiting.")
		return
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		fmt.Printf("File not found at: %s\n", filePath)
		return
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	if ext != ".pdf" && ext != ".epub" && ext != ".docx" {
		fmt.Printf("Unsupported file format: %s\nSupported formats: .pdf, .epub, .docx\n", ext)
		return
	}

	viewer := NewDocumentViewer(filePath)
	if err := viewer.Open(); err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		return
	}

	viewer.Run()
}

func selectFileWithPickerInDir(dir string) (string, error) {
	searcher := NewFileSearcher()
	if err := searcher.ScanDirectory(dir); err != nil {
		return "", fmt.Errorf("error scanning directory: %v", err)
	}
	allFiles := searcher.GetAllFiles()
	if len(allFiles) == 0 {
		return "", fmt.Errorf("no PDF or EPUB files found in %s", dir)
	}
	picker := NewFilePicker(searcher)
	return picker.Run()
}
