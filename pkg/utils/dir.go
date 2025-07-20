package utils

import (
	"fmt"
	"os"
	"strings"
)

// Verifies the current directory has active terraform configuration files
func IsActiveDir() {
	dir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: error occurred while getting working dir: %v\n", err)
		os.Exit(1)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: error occurred while reading module directory: %v\n", err)
		os.Exit(1)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if isActiveFile(name) {
			return
		}
	}
	fmt.Fprintf(os.Stderr, "Error: Current directory does not contain any active terraform files\n")
	os.Exit(1)
}

func isActiveFile(name string) bool {
	// One of these must exist in the directory for it to be considered
	// an active module
	if !(strings.HasSuffix(name, ".tf") || strings.HasSuffix(name, ".tf.json")) {
		return false
	}

	// Handle exceptions for automatically generated files
	if strings.HasPrefix(name, ".") || strings.HasSuffix(name, "~") ||
		(strings.HasPrefix(name, "#") && strings.HasSuffix(name, "#")) {
		return false
	}
	return true
}
