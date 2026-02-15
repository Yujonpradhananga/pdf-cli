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

type FileSearcher struct {
	files []string
	cache map[string][]string
}

func NewFileSearcher() *FileSearcher {
	return &FileSearcher{
		files: []string{},
		cache: make(map[string][]string),
	}
}

func (fs *FileSearcher) ScanDirectories() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	// Common directories
	searchDirs := []string{
		homeDir,
		filepath.Join(homeDir, "Documents"),
		filepath.Join(homeDir, "Downloads"),
		filepath.Join(homeDir, "Desktop"),
		".", // Current directory
	}

	customDirs := []string{
		"/usr/share/doc",
		filepath.Join(homeDir, ".local/share/books"),
	}
	searchDirs = append(searchDirs, customDirs...)

	fmt.Println("Scanning for PDF and EPUB files...")
	fmt.Println("This may take a moment on first run...")

	allFiles := make(map[string]bool)

	for _, dir := range searchDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}

		files := fs.scanDirectory(dir, 3) // Max depth of 3
		for _, file := range files {
			allFiles[file] = true
		}
	}

	fs.files = make([]string, 0, len(allFiles))
	for file := range allFiles {
		fs.files = append(fs.files, file)
	}

	fmt.Printf("Found %d files\n\n", len(fs.files))
	return nil
}

// ScanDirectory scans a single directory for PDF/EPUB files (quiet mode)
func (fs *FileSearcher) ScanDirectory(dir string) error {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}

	files := fs.scanDirectory(absDir, 10)
	fs.files = files
	return nil
}

func (fs *FileSearcher) scanDirectory(dir string, maxDepth int) []string {
	if maxDepth <= 0 {
		return nil
	}

	if cached, ok := fs.cache[dir]; ok {
		return cached
	}

	var results []string

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		if entry.Name() == "node_modules" || entry.Name() == "vendor" {
			continue
		}

		fullPath := filepath.Join(dir, entry.Name())

		if entry.IsDir() {
			subResults := fs.scanDirectory(fullPath, maxDepth-1)
			results = append(results, subResults...)
		} else {
			ext := strings.ToLower(filepath.Ext(entry.Name()))
			if ext == ".pdf" || ext == ".epub" || ext == ".docx" || ext == ".html" || ext == ".htm" {
				results = append(results, fullPath)
			}
		}
	}

	fs.cache[dir] = results
	return results
}

func (fs *FileSearcher) Search(query string) []FileResult {
	if query == "" {
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

	displayPaths := make([]string, len(fs.files))
	for i, file := range fs.files {
		displayPaths[i] = fs.getDisplayPath(file)
	}

	matches := fuzzy.Find(query, displayPaths)

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

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score < results[j].Score
	})

	return results
}

func (fs *FileSearcher) getDisplayPath(path string) string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return path
	}

	if strings.HasPrefix(path, homeDir) {
		return "~" + strings.TrimPrefix(path, homeDir)
	}

	return path
}

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

func (fr *FileResult) HighlightMatches() string {
	if len(fr.Matches) == 0 {
		return fr.RelativePath
	}

	var result strings.Builder
	matchSet := make(map[int]bool)
	for _, idx := range fr.Matches {
		matchSet[idx] = true
	}

	for i, char := range fr.RelativePath {
		if matchSet[i] {
			result.WriteString("\033[1;33m")
			result.WriteRune(char)
			result.WriteString("\033[0m")
		} else {
			result.WriteRune(char)
		}
	}

	return result.String()
}
