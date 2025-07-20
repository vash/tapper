package terraform

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectProfiles(t *testing.T) {
	// Create temporary directory structure for testing
	tempDir := t.TempDir()

	// Change to temp directory
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tempDir)

	// Test case 1: No backend/vars directories
	config, err := DetectProfiles()
	if err != nil {
		t.Fatalf("Expected no error when directories don't exist, got: %v", err)
	}
	if len(config.Profiles) != 0 {
		t.Errorf("Expected 0 profiles when directories don't exist, got: %d", len(config.Profiles))
	}

	// Test case 2: Create directories and matching files
	os.MkdirAll("backend", 0755)
	os.MkdirAll("vars", 0755)

	// Create matching backend and vars files
	os.WriteFile(filepath.Join("backend", "dev.tfbackend"), []byte("bucket = \"dev-bucket\""), 0644)
	os.WriteFile(filepath.Join("vars", "dev.tfvars"), []byte("environment = \"dev\""), 0644)
	os.WriteFile(filepath.Join("backend", "prod.tfbackend"), []byte("bucket = \"prod-bucket\""), 0644)
	os.WriteFile(filepath.Join("vars", "prod.tfvars"), []byte("environment = \"prod\""), 0644)

	config, err = DetectProfiles()
	if err != nil {
		t.Fatalf("Expected no error detecting profiles, got: %v", err)
	}
	if len(config.Profiles) != 2 {
		t.Errorf("Expected 2 profiles, got: %d", len(config.Profiles))
	}

	// Verify profile names
	profileNames := ListProfiles(config)
	expectedNames := map[string]bool{"dev": false, "prod": false}
	for _, name := range profileNames {
		if _, exists := expectedNames[name]; exists {
			expectedNames[name] = true
		}
	}
	for name, found := range expectedNames {
		if !found {
			t.Errorf("Expected to find profile '%s'", name)
		}
	}

	// Test case 3: Orphaned files (backend without matching vars)
	os.WriteFile(filepath.Join("backend", "staging.tfbackend"), []byte("bucket = \"staging-bucket\""), 0644)

	config, err = DetectProfiles()
	if err != nil {
		t.Fatalf("Expected no error with orphaned backend file, got: %v", err)
	}
	if len(config.Profiles) != 2 {
		t.Errorf("Expected 2 profiles (orphaned backend should be ignored), got: %d", len(config.Profiles))
	}
}

func TestGetProfile(t *testing.T) {
	config := &Config{
		Profiles: []Profile{
			{Name: "dev", BackendConfig: "dev.tfbackend", VarFile: "dev.tfvars"},
			{Name: "prod", BackendConfig: "prod.tfbackend", VarFile: "prod.tfvars"},
		},
	}

	// Test existing profile
	profile, exists := GetProfile(config, "dev")
	if !exists {
		t.Error("Expected to find 'dev' profile")
	}
	if profile.Name != "dev" {
		t.Errorf("Expected profile name 'dev', got: %s", profile.Name)
	}

	// Test non-existing profile
	_, exists = GetProfile(config, "nonexistent")
	if exists {
		t.Error("Expected not to find 'nonexistent' profile")
	}
}

func TestListProfiles(t *testing.T) {
	config := &Config{
		Profiles: []Profile{
			{Name: "dev"},
			{Name: "prod"},
			{Name: "staging"},
		},
	}

	names := ListProfiles(config)
	if len(names) != 3 {
		t.Errorf("Expected 3 profile names, got: %d", len(names))
	}

	expected := map[string]bool{"dev": false, "prod": false, "staging": false}
	for _, name := range names {
		if _, exists := expected[name]; exists {
			expected[name] = true
		}
	}

	for name, found := range expected {
		if !found {
			t.Errorf("Expected to find profile name '%s'", name)
		}
	}
}
