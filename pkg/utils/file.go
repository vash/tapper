package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CheckFileOrDirExists checks if a file or directory exists, following symlinks
func CheckFileOrDirExists(path string) (bool, error) {
	// First check if the path exists
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	// If it's a symlink, resolve it
	if info.Mode()&os.ModeSymlink != 0 {
		_, err := filepath.EvalSymlinks(path)
		if err != nil {
			return false, err
		}
		// Check if the resolved path exists
		_, err = os.Stat(path)
		if os.IsNotExist(err) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
	}

	return true, nil
}

// ResolveIfSymlink resolves a path if it's a symlink, otherwise returns the original path
func ResolveIfSymlink(path string) (string, os.FileInfo, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return path, nil, err
	}

	// If it's a symlink, resolve it
	if info.Mode()&os.ModeSymlink != 0 {
		resolvedPath, err := filepath.EvalSymlinks(path)
		if err != nil {
			return path, info, err
		}
		resolvedInfo, err := os.Stat(resolvedPath)
		if err != nil {
			return path, info, err
		}
		return resolvedPath, resolvedInfo, nil
	}

	return path, info, nil
}

// CheckDirExists checks if a directory exists, following symlinks
func CheckDirExists(dirPath string) (bool, error) {
	_, info, err := ResolveIfSymlink(dirPath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return info.IsDir(), nil
}

// ScanFilesWithExtension scans a directory for files with the given extension and returns a map of profile names to filenames
func ScanFilesWithExtension(dirPath, extension string) (map[string]string, error) {
	resolvedDir, _, err := ResolveIfSymlink(dirPath)
	if err != nil {
		return nil, fmt.Errorf("error resolving directory %s: %w", dirPath, err)
	}

	files := make(map[string]string)

	err = filepath.Walk(resolvedDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Handle symlinks by resolving them
		_, actualInfo, resolveErr := ResolveIfSymlink(path)
		if resolveErr != nil {
			// If we can't resolve the symlink, skip it
			return nil
		}

		if !actualInfo.IsDir() && strings.HasSuffix(actualInfo.Name(), extension) {
			// Extract profile name (remove extension)
			profileName := strings.TrimSuffix(actualInfo.Name(), extension)
			files[profileName] = actualInfo.Name()
		}
		return nil
	})

	return files, err
}
