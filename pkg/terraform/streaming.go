package terraform

import (
	"fmt"
	"strings"
	"sync"
	"tapper/pkg/utils"
	"time"
)

// StreamingOutput represents a line of output from a streaming execution
type StreamingOutput struct {
	ProfileName string
	Line        string
	IsError     bool
	Timestamp   time.Time
}

// StreamingOutputHandler handles the real-time display of streaming output
type StreamingOutputHandler struct {
	outputMutex  sync.Mutex
	colorManager *utils.ProfileColorManager
}

// NewStreamingOutputHandler creates a new streaming output handler
func NewStreamingOutputHandler() *StreamingOutputHandler {
	return &StreamingOutputHandler{
		colorManager: utils.NewProfileColorManager(),
	}
}

// DisplayStreamingOutput handles the real-time display of streaming output
func (h *StreamingOutputHandler) DisplayStreamingOutput(streamChan <-chan StreamingOutput, done chan<- bool) {
	for output := range streamChan {
		h.outputMutex.Lock()
		h.printStreamingLine(output)
		h.outputMutex.Unlock()
	}
	done <- true
}

// printStreamingLine formats and prints a single streaming output line
func (h *StreamingOutputHandler) printStreamingLine(output StreamingOutput) {
	timestamp := output.Timestamp.Format("15:04:05.000")
	profileColor := h.colorManager.GetProfileColor(output.ProfileName)

	var prefix string
	if output.IsError {
		prefix = fmt.Sprintf("[%s] %s%s%s %sERROR%s:",
			timestamp,
			profileColor, output.ProfileName, utils.ColorReset,
			utils.ColorRed, utils.ColorReset)
	} else {
		// Check if this is a step message
		line := output.Line
		if h.isStepMessage(line) {
			// This is a step message, color it
			prefix = fmt.Sprintf("[%s] %s%s%s:",
				timestamp,
				profileColor, output.ProfileName, utils.ColorReset)
			line = fmt.Sprintf("%s%s%s", profileColor, line, utils.ColorReset)
		} else {
			// This is regular terraform output, don't color the content
			prefix = fmt.Sprintf("[%s] %s%s%s:",
				timestamp,
				profileColor, output.ProfileName, utils.ColorReset)
		}

		// Print each line with the profile prefix
		lines := strings.Split(strings.TrimRight(line, "\n"), "\n")
		for _, outputLine := range lines {
			if strings.TrimSpace(outputLine) != "" {
				fmt.Printf("%s %s\n", prefix, outputLine)
			}
		}
		return
	}

	// Print each line with the profile prefix (for error case)
	lines := strings.Split(strings.TrimRight(output.Line, "\n"), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			fmt.Printf("%s %s\n", prefix, line)
		}
	}
}

// isStepMessage checks if a line is a step message that should be colored
func (h *StreamingOutputHandler) isStepMessage(line string) bool {
	stepPrefixes := []string{
		"Starting execution",
		"Running terraform",
		"Executing:",
		"INIT:",
		"✅ Execution completed",
	}

	for _, prefix := range stepPrefixes {
		if strings.HasPrefix(line, prefix) || strings.Contains(line, "Execution completed") {
			return true
		}
	}
	return false
}
