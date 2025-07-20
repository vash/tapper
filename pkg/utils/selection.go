package utils

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
)

// SelectionConfig holds configuration for interactive selection
type SelectionConfig struct {
	Prompt        string
	Header        string
	Height        string
	Border        bool
	Reverse       bool
	Multi         bool
	Preview       string
	PreviewWindow string
}

// DefaultSingleSelectConfig returns default config for single selection
func DefaultSingleSelectConfig(prompt, header string) SelectionConfig {
	return SelectionConfig{
		Prompt:  prompt,
		Header:  header,
		Height:  "40%",
		Border:  true,
		Reverse: true,
		Multi:   false,
	}
}

// DefaultMultiSelectConfig returns default config for multi-selection
func DefaultMultiSelectConfig(prompt, header string) SelectionConfig {
	return SelectionConfig{
		Prompt:  prompt,
		Header:  header,
		Height:  "40%",
		Border:  true,
		Reverse: true,
		Multi:   true,
	}
}

// InteractiveSelect provides unified fzf-based selection with fallback
func InteractiveSelect(items []string, config SelectionConfig) ([]string, error) {
	if len(items) == 0 {
		return nil, fmt.Errorf("no items provided for selection")
	}

	if len(items) == 1 && !config.Multi {
		fmt.Printf("Using the only available option: %s\n", items[0])
		return []string{items[0]}, nil
	}

	// Check if fzf is available
	if _, err := exec.LookPath("fzf"); err != nil {
		// Fallback to simple selection if fzf is not available
		return fallbackSelect(items, config)
	}

	// Use fzf for interactive selection
	return fzfSelect(items, config)
}

// fzfSelect uses fzf for interactive selection
func fzfSelect(items []string, config SelectionConfig) ([]string, error) {
	// Build fzf command arguments
	args := []string{
		"--prompt=" + config.Prompt,
		"--height=" + config.Height,
		"--header=" + config.Header,
	}

	if config.Border {
		args = append(args, "--border")
	}
	if config.Reverse {
		args = append(args, "--reverse")
	}
	if config.Multi {
		args = append(args, "--multi")
	}
	if config.Preview != "" {
		args = append(args, "--preview="+config.Preview)
	}
	if config.PreviewWindow != "" {
		args = append(args, "--preview-window="+config.PreviewWindow)
	}

	cmd := exec.Command("fzf", args...)

	// Set up stdin pipe to send items to fzf
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("error creating stdin pipe for fzf: %w", err)
	}

	// Set up stdout pipe to read selected items from fzf
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("error creating stdout pipe for fzf: %w", err)
	}

	// Connect stderr to terminal so user can see fzf interface
	cmd.Stderr = os.Stderr

	// Start fzf
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("error starting fzf: %w", err)
	}

	// Send items to fzf
	go func() {
		defer stdin.Close()
		for _, item := range items {
			fmt.Fprintln(stdin, item)
		}
	}()

	// Read the selected items
	output, err := io.ReadAll(stdout)
	if err != nil {
		cmd.Wait()
		return nil, fmt.Errorf("error reading fzf output: %w", err)
	}

	// Wait for fzf to finish
	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("fzf selection cancelled or failed")
	}

	// Parse the output
	selectedOutput := strings.TrimSpace(string(output))
	if selectedOutput == "" {
		return nil, fmt.Errorf("no items selected")
	}

	// Split by newlines for multi-select, single item for single-select
	var selected []string
	if config.Multi {
		selected = strings.Split(selectedOutput, "\n")
	} else {
		selected = []string{selectedOutput}
	}

	// Filter out empty strings
	var result []string
	for _, item := range selected {
		item = strings.TrimSpace(item)
		if item != "" {
			result = append(result, item)
		}
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("no valid items selected")
	}

	return result, nil
}

// fallbackSelect provides simple numbered selection when fzf is not available
func fallbackSelect(items []string, config SelectionConfig) ([]string, error) {
	fmt.Println("fzf not found, using fallback selection method")
	fmt.Printf("%s\n", config.Header)
	fmt.Println("Available options:")

	for i, item := range items {
		fmt.Printf("  %d. %s\n", i+1, item)
	}

	if config.Multi {
		fmt.Print("Select options (enter numbers separated by commas, e.g., 1,3,4): ")
		return handleMultiSelectInput(items)
	} else {
		fmt.Print("Select an option (enter number): ")
		return handleSingleSelectInput(items)
	}
}

// handleSingleSelectInput handles single selection input parsing
func handleSingleSelectInput(items []string) ([]string, error) {
	var selection int
	_, err := fmt.Scanln(&selection)
	if err != nil {
		return nil, fmt.Errorf("error reading input: %w", err)
	}

	if selection < 1 || selection > len(items) {
		return nil, fmt.Errorf("invalid selection. Please enter a number between 1 and %d", len(items))
	}

	return []string{items[selection-1]}, nil
}

// handleMultiSelectInput handles multi-selection input parsing
func handleMultiSelectInput(items []string) ([]string, error) {
	var input string
	_, err := fmt.Scanln(&input)
	if err != nil {
		return nil, fmt.Errorf("error reading input: %w", err)
	}

	// Parse comma-separated numbers
	selections := strings.Split(input, ",")
	var selectedItems []string

	for _, selStr := range selections {
		selStr = strings.TrimSpace(selStr)
		selection, err := strconv.Atoi(selStr)
		if err != nil {
			return nil, fmt.Errorf("invalid selection '%s': must be a number", selStr)
		}

		if selection < 1 || selection > len(items) {
			return nil, fmt.Errorf("invalid selection %d. Valid range is 1-%d", selection, len(items))
		}

		selectedItems = append(selectedItems, items[selection-1])
	}

	if len(selectedItems) == 0 {
		return nil, fmt.Errorf("no items selected")
	}

	return selectedItems, nil
}

// HierarchicalSelect handles hierarchical selection with parent-child relationships
func HierarchicalSelect(items []string, hierarchyMap map[string][]string, config SelectionConfig) ([]string, error) {
	selected, err := InteractiveSelect(items, config)
	if err != nil {
		return nil, err
	}

	// Apply hierarchical selection logic
	finalSelection := make(map[string]bool)

	for _, selection := range selected {
		if children, exists := hierarchyMap[selection]; exists {
			// This is a hierarchy item, select all children
			for _, child := range children {
				finalSelection[child] = true
			}
		} else {
			// Regular item selection
			finalSelection[selection] = true
		}
	}

	// Convert back to slice
	var result []string
	for item := range finalSelection {
		result = append(result, item)
	}

	// Sort for consistent output
	sort.Strings(result)
	return result, nil
}
