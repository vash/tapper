package workspace

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Profile represents a simplified profile for workspace operations
type Profile struct {
	Name string
}

// WorkspaceManager handles creating and managing temporary workspaces for multi-profile execution
type WorkspaceManager struct {
	BaseDirPath   string
	OperationID   string            // Unique ID for this operation
	ProfileSpaces map[string]string // profile name -> workspace path
}

func NewWorkspaceManager() (*WorkspaceManager, error) {
	bytes := make([]byte, 4) // 4 bytes = 8 hex characters
	_, err := rand.Read(bytes)
	if err != nil {
		return nil, err
	}
	operationID := fmt.Sprintf("%x", bytes)

	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	return &WorkspaceManager{
		BaseDirPath:   cwd,
		OperationID:   operationID,
		ProfileSpaces: make(map[string]string),
	}, nil
}

func (wm *WorkspaceManager) CreateWorkspaces(profiles []Profile) error {
	workspaceParent := filepath.Dir(wm.BaseDirPath)

	for _, profile := range profiles {
		// Create profile-specific workspace directory alongside BaseDir
		// Pattern: .dir-<PROFILE>-<OPERATION_ID>

		baseDir := filepath.Base(wm.BaseDirPath)
		profileWorkspaceName := fmt.Sprintf(".%s-%s-%s", baseDir, profile.Name, wm.OperationID)
		profileWorkspace := filepath.Join(workspaceParent, profileWorkspaceName)

		if err := os.MkdirAll(profileWorkspace, 0755); err != nil {
			return fmt.Errorf("error creating profile workspace %s: %w", profileWorkspace, err)
		}

		// Store the mapping
		wm.ProfileSpaces[profile.Name] = profileWorkspace

		// Create symlinks for all files and directories (including special .terraform handling)
		if err := wm.symlink(profileWorkspace); err != nil {
			return fmt.Errorf("error creating symlinks for profile %s: %w", profile.Name, err)
		}
	}

	return nil
}

// symlink creates symlinks for all files and directories in the base directory
func (wm *WorkspaceManager) symlink(targetDir string) error {
	entries, err := os.ReadDir(wm.BaseDirPath)
	if err != nil {
		return fmt.Errorf("error reading base directory: %w", err)
	}

	for _, entry := range entries {
		name := entry.Name()

		sourcePath := filepath.Join(wm.BaseDirPath, name)
		targetPath := filepath.Join(targetDir, name)

		// Terraform.tfstate needs to be unique for every workspace
		if name == ".terraform" {
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return fmt.Errorf("error creating .terraform directory: %w", err)
			}
			skipFunc := func(name string) bool {
				return strings.Contains(name, "terraform.tfstate")
			}
			if err := wm.conditionalSymlink(sourcePath, targetPath, skipFunc); err != nil {
				return fmt.Errorf("error creating symlinks in .terraform directory: %w", err)
			}
		} else {
			relPath, err := filepath.Rel(targetDir, sourcePath)
			if err != nil {
				return fmt.Errorf("error calculating relative path from %s to %s: %w", targetDir, sourcePath, err)
			}
			if err := os.Symlink(relPath, targetPath); err != nil {
				return fmt.Errorf("error creating symlink from %s to %s: %w", relPath, targetPath, err)
			}
		}
	}

	return nil
}

func (wm *WorkspaceManager) conditionalSymlink(sourceDir, targetDir string, skipFunc func(string) bool) error {
	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		return fmt.Errorf("error reading directory %s: %w", sourceDir, err)
	}

	for _, entry := range entries {
		name := entry.Name()

		// Skip if the skip function returns true
		if skipFunc != nil && skipFunc(name) {
			continue
		}

		sourcePath := filepath.Join(sourceDir, name)
		targetPath := filepath.Join(targetDir, name)

		// Calculate relative path from target to source
		relPath, err := filepath.Rel(targetDir, sourcePath)
		if err != nil {
			return fmt.Errorf("error calculating relative path from %s to %s: %w", targetDir, sourcePath, err)
		}

		// Create symlink for both files and directories
		if err := os.Symlink(relPath, targetPath); err != nil {
			return fmt.Errorf("error creating symlink from %s to %s: %w", relPath, targetPath, err)
		}
	}

	return nil
}

// Cleanup removes only the workspaces created by this operation
func (wm *WorkspaceManager) Cleanup() error {
	// Get the directory where workspaces were created
	workspaceParent := filepath.Dir(wm.BaseDirPath)
	workspaceDir := filepath.Base(wm.BaseDirPath)

	// Read the workspace parent directory to find directories with our pattern
	entries, err := os.ReadDir(workspaceParent)
	if err != nil {
		return fmt.Errorf("error reading workspace parent directory %s: %w", workspaceParent, err)
	}

	// Remove directories that match our pattern: .baseDirName-*-operationID
	suffix := fmt.Sprintf("-%s", wm.OperationID)
	prefix := fmt.Sprintf(".%s-", workspaceDir)

	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), prefix) && strings.HasSuffix(entry.Name(), suffix) {
			workspacePath := filepath.Join(workspaceParent, entry.Name())

			if err := os.RemoveAll(workspacePath); err != nil {
				return fmt.Errorf("error removing workspace %s: %w", workspacePath, err)
			}
		}
	}
	// Clear the ProfileSpaces map
	wm.ProfileSpaces = make(map[string]string)
	return nil
}

// GetWorkspacePath returns the workspace path for a given profile
func (wm *WorkspaceManager) GetWorkspacePath(profileName string) (string, bool) {
	path, exists := wm.ProfileSpaces[profileName]
	return path, exists
}
