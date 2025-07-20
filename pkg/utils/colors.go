package utils

import "sync"

// ANSI color codes for profile differentiation
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorWhite  = "\033[37m"
	ColorBold   = "\033[1m"
)

// ProfileColorManager manages color assignment for profiles
type ProfileColorManager struct {
	profileColorMap map[string]string
	colorMutex      sync.Mutex
	colors          []string
}

// NewProfileColorManager creates a new color manager
func NewProfileColorManager() *ProfileColorManager {
	return &ProfileColorManager{
		profileColorMap: make(map[string]string),
		colors: []string{
			ColorCyan,
			ColorYellow,
			ColorGreen,
			ColorPurple,
			ColorBlue,
			ColorRed,
			ColorWhite,
		},
	}
}

// GetProfileColor assigns and returns a consistent color for a profile
func (pcm *ProfileColorManager) GetProfileColor(profileName string) string {
	pcm.colorMutex.Lock()
	defer pcm.colorMutex.Unlock()

	// If we already have a color for this profile, return it
	if color, exists := pcm.profileColorMap[profileName]; exists {
		return color
	}

	// Assign a new color based on the number of profiles we've seen
	colorIndex := len(pcm.profileColorMap) % len(pcm.colors)
	color := pcm.colors[colorIndex]
	pcm.profileColorMap[profileName] = color

	return color
}
