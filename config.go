package main

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// DocConfig holds per-document settings that persist across sessions.
type DocConfig struct {
	FitMode       string  `json:"fit_mode"`
	ScaleFactor   float64 `json:"scale_factor"`
	DarkMode      string  `json:"dark_mode"`
	DualPageMode  string  `json:"dual_page_mode"`
	ForceMode     string  `json:"force_mode"`
	HTMLPageWidth int     `json:"html_page_width"`
	CropTop       float64 `json:"crop_top"`
	CropBottom    float64 `json:"crop_bottom"`
	CropLeft      float64 `json:"crop_left"`
	CropRight     float64 `json:"crop_right"`
}

// configDir returns the directory used to store per-document config files.
func configDir() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "docviewer")
}

// configPath returns the config file path for a given document (by absolute path).
func configPath(absPath string) string {
	hash := md5.Sum([]byte(absPath))
	return filepath.Join(configDir(), fmt.Sprintf("%x.json", hash))
}

// loadDocConfig loads persisted settings for a document, returning defaults if
// no saved config exists.
func loadDocConfig(absPath string) DocConfig {
	cfg := DocConfig{
		FitMode:       "height",
		ScaleFactor:   1.0,
		HTMLPageWidth: 1000,
	}

	data, err := os.ReadFile(configPath(absPath))
	if err != nil {
		return cfg
	}

	_ = json.Unmarshal(data, &cfg)

	// Sanity-check loaded values
	if cfg.ScaleFactor < 0.1 || cfg.ScaleFactor > 2.0 {
		cfg.ScaleFactor = 1.0
	}
	if cfg.HTMLPageWidth < 200 || cfg.HTMLPageWidth > 3000 {
		cfg.HTMLPageWidth = 1000
	}

	return cfg
}

// saveConfig persists the current document settings to disk.
func (d *DocumentViewer) saveConfig() {
	absPath, err := filepath.Abs(d.path)
	if err != nil {
		return
	}

	cfg := DocConfig{
		FitMode:       d.fitMode,
		ScaleFactor:   d.scaleFactor,
		DarkMode:      d.darkMode,
		DualPageMode:  d.dualPageMode,
		ForceMode:     d.forceMode,
		HTMLPageWidth: d.htmlPageWidth,
		CropTop:       d.cropTop,
		CropBottom:    d.cropBottom,
		CropLeft:      d.cropLeft,
		CropRight:     d.cropRight,
	}

	dir := configDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return
	}

	_ = os.WriteFile(configPath(absPath), data, 0o644)
}
