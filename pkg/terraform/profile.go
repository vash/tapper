package terraform

import (
	"fmt"

	"tapper/pkg/utils"
)

// Profile represents a Terraform configuration profile
type Profile struct {
	Name          string `json:"name"`
	BackendConfig string `json:"backendconfig"`
	VarFile       string `json:"varfile"`
	BackendDir    string `json:"backenddir"`
	VarsDir       string `json:"varsdir"`
	LastUsed      string `json:"lastused"`
}

// Config represents the application configuration
type Config struct {
	Profiles []Profile `json:"profiles"`
}

// DetectProfiles scans the filesystem and returns detected profiles
func DetectProfiles() (*Config, error) {
	backendDir := "backend"
	varsDir := "vars"

	// Check if required directories exist
	for _, dir := range []string{backendDir, varsDir} {
		exists, err := utils.CheckDirExists(dir)
		if err != nil {
			return nil, fmt.Errorf("error checking %s directory: %w", dir, err)
		}
		if !exists {
			return &Config{Profiles: []Profile{}}, nil
		}
	}

	// Scan for backend and var files
	backendFiles, err := utils.ScanFilesWithExtension(backendDir, ".tfbackend")
	if err != nil {
		return nil, fmt.Errorf("error scanning backend directory: %w", err)
	}

	varFiles, err := utils.ScanFilesWithExtension(varsDir, ".tfvars")
	if err != nil {
		return nil, fmt.Errorf("error scanning vars directory: %w", err)
	}

	// Create profiles for matching backend and var files
	var profiles []Profile
	for profileName, backendFile := range backendFiles {
		if varFile, exists := varFiles[profileName]; exists {
			profiles = append(profiles, Profile{
				Name:          profileName,
				BackendConfig: backendFile,
				VarFile:       varFile,
				BackendDir:    backendDir,
				VarsDir:       varsDir,
				LastUsed:      "",
			})
		}
	}

	return &Config{Profiles: profiles}, nil
}

// LoadConfig loads the configuration by detecting profiles from filesystem
func LoadConfig() (*Config, error) {
	return DetectProfiles()
}

// GetProfile gets a profile by name
func GetProfile(config *Config, name string) (Profile, bool) {
	for _, profile := range config.Profiles {
		if profile.Name == name {
			return profile, true
		}
	}
	return Profile{}, false
}

// ListProfiles returns a list of all profile names
func ListProfiles(config *Config) []string {
	names := make([]string, len(config.Profiles))
	for i, profile := range config.Profiles {
		names[i] = profile.Name
	}
	return names
}
