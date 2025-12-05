package utils

import (
	"path/filepath"
	"runtime"
)

// TestImagePath is the relative path to the test image file
const TestImagePath = "../data/test_image.png"

// GetBaseDir returns the directory of the current file (consts.go)
func GetBaseDir() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Dir(filename)
}

// GetTestImagePath returns the absolute path to the test image file
func GetTestImagePath() string {
	return filepath.Join(GetBaseDir(), TestImagePath)
}
