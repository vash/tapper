package utils

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const (
	// SSOTokenExpiredError is the error message for expired SSO tokens
	SSOTokenExpiredError = "SSOProviderInvalidToken: the SSO session has expired or is invalid"
)

// ExtractProfileFromBackendConfig parses the backend config content and extracts the profile value
func ExtractProfileFromBackendConfig(content string) (string, error) {
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip comments and empty lines
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}

		// Look for profile parameter (handle both quoted and unquoted values)
		if strings.HasPrefix(line, "profile") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				profileValue := strings.TrimSpace(parts[1])
				// Remove quotes if present
				profileValue = strings.Trim(profileValue, `"'`)
				return profileValue, nil
			}
		}
	}

	return "", fmt.Errorf("profile parameter not found in backend config")
}

// RefreshAWSSSO runs aws sso login with the specified profile
func RefreshAWSSSO(profileName string) error {
	fmt.Printf("Running AWS SSO login for profile '%s'...\n", profileName)

	// Run aws sso login with the profile
	cmd := exec.Command("aws", "sso", "login", "--profile", profileName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error running aws sso login: %w", err)
	}

	return nil
}

// RefreshAWSSSOFromBackendConfig reads the backend config file and refreshes SSO for the profile found
func RefreshAWSSSOFromBackendConfig(backendConfigPath string) error {
	data, err := os.ReadFile(backendConfigPath)
	if err != nil {
		return fmt.Errorf("error reading backend config file: %w", err)
	}

	profileName, err := ExtractProfileFromBackendConfig(string(data))
	if err != nil {
		return fmt.Errorf("error extracting profile from backend config: %w", err)
	}

	return RefreshAWSSSO(profileName)
}

// IsAWSSSOTokenExpired checks if the given error output indicates an expired SSO token
func IsAWSSSOTokenExpired(output string) bool {
	return strings.Contains(output, SSOTokenExpiredError)
}
