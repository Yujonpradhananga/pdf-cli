package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sahilm/fuzzy"
)

type FileResult struct {
	Path         string
	RelativePath string
	Score        int
	Matches      []int
}

// FileSearcher handles searching for PDF and EPUB files
type FileSearcher struct {
	files []string
	cache map[string][]string // Cache for directory scans
}

func NewFileSearcher() *FileSearcher {
	return &FileSearcher{
		files: []string{},
		cache: make(map[string][]string),
	}
}

// ScanDirectories scans common directories for PDF and EPUB files
func (fs *FileSearcher) ScanDirectories() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	// Common directories to search
	searchDirs := []string{
		homeDir,
		filepath.Join(homeDir, "Documents"),
		filepath.Join(homeDir, "Downloads"),
		filepath.Join(homeDir, "Desktop"),
		".", // Current directory
	}

	// Add custom directories if they exist
	customDirs := []string{
		"/usr/share/doc",
		filepath.Join(homeDir, ".local/share/books"),
	}
	searchDirs = append(searchDirs, customDirs...)

	fmt.Println("Scanning for PDF and EPUB files...")
	fmt.Println("This may take a moment on first run...")

	allFiles := make(map[string]bool) // Use map to avoid duplicates

	for _, dir := range searchDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}

		files := fs.scanDirectory(dir, 3) // Max depth of 3
		for _, file := range files {
			allFiles[file] = true
		}
	}

	// Convert map to slice
	fs.files = make([]string, 0, len(allFiles))
	for file := range allFiles {
		fs.files = append(fs.files, file)
	}

	fmt.Printf("Found %d files\n\n", len(fs.files))
	return nil
}

// scanDirectory recursively scans a directory up to maxDepth
func (fs *FileSearcher) scanDirectory(dir string, maxDepth int) []string {
	if maxDepth <= 0 {
		return nil
	}

	// Check cache
	if cached, ok := fs.cache[dir]; ok {
		return cached
	}

	var results []string

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	for _, entry := range entries {
		// Skip hidden directories and common large directories
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		if entry.Name() == "node_modules" || entry.Name() == "vendor" {
			continue
		}

		fullPath := filepath.Join(dir, entry.Name())

		if entry.IsDir() {
			// Recursively scan subdirectories
			subResults := fs.scanDirectory(fullPath, maxDepth-1)
			results = append(results, subResults...)
		} else {
			// Check if file is PDF or EPUB
			ext := strings.ToLower(filepath.Ext(entry.Name()))
			if ext == ".pdf" || ext == ".epub" {
				results = append(results, fullPath)
			}
		}
	}

	// Cache results
	fs.cache[dir] = results
	return results
}

// Search performs fuzzy search on the file list
func (fs *FileSearcher) Search(query string) []FileResult {
	if query == "" {
		// Return all files if no query
		results := make([]FileResult, 0, len(fs.files))
		for _, file := range fs.files {
			results = append(results, FileResult{
				Path:         file,
				RelativePath: fs.getDisplayPath(file),
				Score:        0,
			})
		}
		return results
	}

	// Prepare data for fuzzy search
	// We'll search on the display path (more user-friendly)
	displayPaths := make([]string, len(fs.files))
	for i, file := range fs.files {
		displayPaths[i] = fs.getDisplayPath(file)
	}

	// Perform fuzzy search
	matches := fuzzy.Find(query, displayPaths)

	// Convert to FileResult
	results := make([]FileResult, 0, len(matches))
	for _, match := range matches {
		if match.Index < len(fs.files) {
			results = append(results, FileResult{
				Path:         fs.files[match.Index],
				RelativePath: displayPaths[match.Index],
				Score:        match.Score,
				Matches:      match.MatchedIndexes,
			})
		}
	}

	// Sort by score (lower is better in fuzzy library)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score < results[j].Score
	})

	return results
}

// getDisplayPath returns a user-friendly display path
func (fs *FileSearcher) getDisplayPath(path string) string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return path
	}

	// Replace home directory with ~
	if strings.HasPrefix(path, homeDir) {
		return "~" + strings.TrimPrefix(path, homeDir)
	}

	return path
}

// GetAllFiles returns all scanned files
func (fs *FileSearcher) GetAllFiles() []FileResult {
	results := make([]FileResult, 0, len(fs.files))
	for _, file := range fs.files {
		results = append(results, FileResult{
			Path:         file,
			RelativePath: fs.getDisplayPath(file),
		})
	}
	return results
}

// HighlightMatches returns a string with matched characters highlighted
func (fr *FileResult) HighlightMatches() string {
	if len(fr.Matches) == 0 {
		return fr.RelativePath
	}

	// Build string with ANSI color codes for matched characters
	var result strings.Builder
	matchSet := make(map[int]bool)
	for _, idx := range fr.Matches {
		matchSet[idx] = true
	}

	for i, char := range fr.RelativePath {
		if matchSet[i] {
			// Yellow/bold for matched characters
			result.WriteString("\033[1;33m")
			result.WriteRune(char)
			result.WriteString("\033[0m")
		} else {
			result.WriteRune(char)
		}
	}

	return result.String()
}
