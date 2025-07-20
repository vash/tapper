package terraform

import (
	"fmt"
	"os/exec"
	"path/filepath"

	"tapper/pkg/utils"
)

// CommandBuilder helps build terraform commands consistently
type CommandBuilder struct {
	WorkingDir    string
	BackendConfig string
	VarFile       string
	BackendDir    string
	VarsDir       string
	Targets       []string
}

// NewCommandBuilder creates a new terraform command builder
func NewCommandBuilder() *CommandBuilder {
	return &CommandBuilder{
		BackendDir: "backend",
		VarsDir:    "vars",
	}
}

// BuildCommandFromProfile builds a terraform command from a profile and command type
// This consolidates the functionality that was in executor.TerraformCommandBuilder
func (cb *CommandBuilder) BuildCommandFromProfile(profile Profile, workspacePath string, execOpts *ExecutionOptions) (*exec.Cmd, error) {
	// Configure the builder with profile settings
	cb.WithWorkingDir(workspacePath).
		WithVarFile(profile.VarFile).
		WithVarsDir(profile.VarsDir)

	// Validate command type
	switch execOpts.Command {
	case "plan", "apply", "destroy":
		// Valid commands
	default:
		return nil, fmt.Errorf("unsupported command: %s", execOpts.Command)
	}

	// Build the command using the generic method
	cmd := cb.buildTerraformCommand(execOpts)

	// Validate that var file exists if specified
	if err := cb.validateVarFile(); err != nil {
		return nil, err
	}

	return cmd, nil
}

// buildTerraformCommand builds a generic terraform command with common arguments
func (cb *CommandBuilder) buildTerraformCommand(execOpts *ExecutionOptions) *exec.Cmd {
	args := []string{execOpts.Command}

	// Add var file if specified
	if cb.VarFile != "" {
		varFilePath := filepath.Join(cb.VarsDir, cb.VarFile)
		args = append(args, fmt.Sprintf("--var-file=%s", varFilePath))
	}

	// Add targets if specified
	for _, target := range cb.Targets {
		args = append(args, fmt.Sprintf("--target=%s", target))
	}

	// Add command-specific dry run flags
	switch execOpts.Command {
	case "plan":
		args = append(args, "--detailed-exitcode")
	case "apply", "destroy":
		if !execOpts.DryRun {
			args = append(args, "--auto-approve")
		}
	}

	// Apply external args
	args = append(args, execOpts.Args...)

	cmd := exec.Command("terraform", args...)
	if cb.WorkingDir != "" {
		cmd.Dir = cb.WorkingDir
	}

	return cmd
}

// GetBackendConfigPath returns the full path to the backend config file
func (cb *CommandBuilder) GetBackendConfigPath() string {
	if cb.BackendConfig == "" {
		return ""
	}

	if cb.WorkingDir != "" {
		return filepath.Join(cb.WorkingDir, cb.BackendDir, cb.BackendConfig)
	}

	return filepath.Join(cb.BackendDir, cb.BackendConfig)
}

// WithWorkingDir sets the working directory
func (cb *CommandBuilder) WithWorkingDir(dir string) *CommandBuilder {
	cb.WorkingDir = dir
	return cb
}

// WithBackendConfig sets the backend config file
func (cb *CommandBuilder) WithBackendConfig(config string) *CommandBuilder {
	cb.BackendConfig = config
	return cb
}

// WithVarFile sets the var file
func (cb *CommandBuilder) WithVarFile(varFile string) *CommandBuilder {
	cb.VarFile = varFile
	return cb
}

// WithBackendDir sets the backend directory
func (cb *CommandBuilder) WithBackendDir(dir string) *CommandBuilder {
	cb.BackendDir = dir
	return cb
}

// WithVarsDir sets the vars directory
func (cb *CommandBuilder) WithVarsDir(dir string) *CommandBuilder {
	cb.VarsDir = dir
	return cb
}

// WithTargets sets the target resources
func (cb *CommandBuilder) WithTargets(targets []string) *CommandBuilder {
	cb.Targets = targets
	return cb
}

// BuildInitCommand builds a terraform init command
func (cb *CommandBuilder) BuildInitCommand() *exec.Cmd {
	args := []string{"init"}

	if cb.BackendConfig != "" {
		backendConfigPath := filepath.Join(cb.BackendDir, cb.BackendConfig)
		args = append(args, fmt.Sprintf("--backend-config=%s", backendConfigPath))
	}

	args = append(args, "--reconfigure")

	cmd := exec.Command("terraform", args...)
	if cb.WorkingDir != "" {
		cmd.Dir = cb.WorkingDir
	}

	return cmd
}

// GetVarFilePath returns the full path to the var file
func (cb *CommandBuilder) GetVarFilePath() string {
	if cb.VarFile == "" {
		return ""
	}

	if cb.WorkingDir != "" {
		return filepath.Join(cb.WorkingDir, cb.VarsDir, cb.VarFile)
	}

	return filepath.Join(cb.VarsDir, cb.VarFile)
}

// validateVarFile checks if the var file exists when specified
func (cb *CommandBuilder) validateVarFile() error {
	varFilePath := cb.GetVarFilePath()
	if varFilePath != "" {
		exists, err := utils.CheckFileOrDirExists(varFilePath)
		if err != nil {
			return fmt.Errorf("error checking var file: %w", err)
		}
		if !exists {
			return fmt.Errorf("var file not found: %s", varFilePath)
		}
	}
	return nil
}
