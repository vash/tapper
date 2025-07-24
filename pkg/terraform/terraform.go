package terraform

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"tapper/pkg/utils"
	"tapper/pkg/workspace"
)

// Executor handles parallel execution of terraform commands across multiple profiles
type Executor struct {
	MaxConcurrency   int
	streamingHandler *StreamingOutputHandler
	userInteraction  *InteractionHandler
	workspaceManager *workspace.WorkspaceManager
	AdditionalArgs   []string // Additional arguments to pass to terraform commands
}

type ExecutionOptions struct {
	Command string
	Args    []string
	DryRun  bool
}

const PREVIEW_COMMAND = "plan"

// NewExecutor creates a new parallel executor
func NewExecutor() (*Executor, error) {
	wm, err := workspace.NewWorkspaceManager()
	if err != nil {
		return nil, fmt.Errorf("error creating workspace manager: %w", err)
	}
	return &Executor{
		MaxConcurrency:   5, // Default to 5 concurrent executions
		streamingHandler: NewStreamingOutputHandler(),
		userInteraction:  NewInteractionHandler(),
		workspaceManager: wm,
	}, nil
}

// SetAdditionalArgs sets additional arguments to be passed to terraform commands
func (e *Executor) SetAdditionalArgs(args []string) error {
	e.AdditionalArgs = args
	return nil
}

// PlanExecution creates an execution plan by running the corresponding command in dry-run mode
func (e *Executor) PlanExecution(command string, profiles []Profile) (*ExecutionPlan, error) {
	if len(profiles) == 0 {
		return nil, fmt.Errorf("no profiles provided")
	}

	if err := e.Init(profiles[0]); err != nil {
		return nil, fmt.Errorf("error running terraform init: %w", err)
	}

	// Create workspaces
	workspaceProfiles := make([]workspace.Profile, len(profiles))
	for i, profile := range profiles {
		workspaceProfiles[i] = workspace.Profile{Name: profile.Name}
	}
	if err := e.workspaceManager.CreateWorkspaces(workspaceProfiles); err != nil {
		return nil, fmt.Errorf("error creating workspaces: %w", err)
	}

	plan := &ExecutionPlan{
		Command:  command,
		Profiles: profiles,
		Results:  make([]ExecutionResult, 0, len(profiles)),
	}

	fmt.Printf("\n=== Streaming Execution for %s ===\n", command)
	fmt.Printf("Executing %d profiles with real-time output...\n\n", len(profiles))

	previewArgs := []string{"--detailed-exitcode"}

	// Emulate destruction with command (otherwise plain plan will show)
	if command == "destroy" {
		previewArgs = append(previewArgs, "--destroy")
	}

	// Add additional arguments to preview args
	previewArgs = append(previewArgs, e.AdditionalArgs...)

	executionOptions := &ExecutionOptions{
		Command: PREVIEW_COMMAND,
		Args:    previewArgs,
		DryRun:  true,
	}

	results, err := e.parallelExecution(profiles, executionOptions)
	if err != nil {
		return nil, err
	}

	// Display review and get approval
	fmt.Printf("\n" + strings.Repeat("=", 80) + "\n")
	fmt.Printf("=== EXECUTION COMPLETED - PLAN REVIEW ===\n")
	fmt.Printf(strings.Repeat("=", 80) + "\n\n")

	approvedProfiles, err := e.userInteraction.ReviewAndApproveResults(results)
	if err != nil {
		return nil, fmt.Errorf("error during streaming execution: %w", err)
	}

	plan.ApprovedProfiles = approvedProfiles
	return plan, nil
}

// ExecutePlan executes the approved execution plan
func (e *Executor) ExecutePlan(plan *ExecutionPlan) ([]ExecutionResult, error) {
	approvedProfileStructs := e.filterApprovedProfiles(plan.Profiles, plan.ApprovedProfiles)
	fmt.Printf("Executing %d profiles with real-time output...\n\n", len(approvedProfileStructs))
	execOpts := &ExecutionOptions{
		Command: plan.Command,
		Args:    e.AdditionalArgs, // Include additional arguments
		DryRun:  false,
	}

	results, err := e.parallelExecution(approvedProfileStructs, execOpts)
	if err != nil {
		return nil, err
	}

	fmt.Println() // Add a blank line for clean separation
	return results, nil
}

// parallelExecution prepares the environment for parallel streaming
func (e *Executor) parallelExecution(profiles []Profile, execOpts *ExecutionOptions) ([]ExecutionResult, error) {
	fmt.Printf("EXECUTING COMMAND %s\n", execOpts.Command)

	// Create channels for streaming communication
	streamChan := make(chan StreamingOutput, 100)
	resultsChan := make(chan ExecutionResult, len(profiles))
	var wg sync.WaitGroup

	// Start goroutine to handle streaming output display
	displayDone := make(chan bool)
	go e.streamingHandler.DisplayStreamingOutput(streamChan, displayDone)

	// Starts the execution
	e.executeParallelCommand(profiles, execOpts, streamChan, resultsChan, &wg)

	// Wait for all executions to complete
	wg.Wait()
	close(streamChan)
	close(resultsChan)

	// Wait for display to finish
	<-displayDone

	// Collect all results
	var results []ExecutionResult
	for result := range resultsChan {
		results = append(results, result)
	}

	return results, nil
}

// executeParallelCommand executes terraform commands in parallel
func (e *Executor) executeParallelCommand(profiles []Profile, execOpts *ExecutionOptions, streamChan chan<- StreamingOutput, resultsChan chan<- ExecutionResult, wg *sync.WaitGroup) {
	// Create a semaphore to limit concurrency
	semaphore := make(chan struct{}, e.MaxConcurrency)

	for _, profile := range profiles {
		wg.Add(1)
		go func(prof Profile) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Execute the command for this profile with streaming
			result := e.executeForProfileWithStreaming(prof, execOpts, streamChan)
			resultsChan <- result
		}(profile)
	}
}

// executeForProfileWithStreaming executes a terraform command for a specific profile with streaming output
func (e *Executor) executeForProfileWithStreaming(profile Profile, execOpts *ExecutionOptions, streamChan chan<- StreamingOutput) ExecutionResult {
	startTime := time.Now()
	workspacePath, exists := e.workspaceManager.GetWorkspacePath(profile.Name)
	if !exists {
		return e.errorResultWithStreaming(ExecutionResult{
			ProfileName: profile.Name,
		}, fmt.Errorf("workspace path not found for profile %s", profile.Name), startTime, streamChan)
	}

	result := ExecutionResult{
		ProfileName: profile.Name,
		WorkingDir:  workspacePath,
	}

	// Send start message
	streamChan <- StreamingOutput{
		ProfileName: profile.Name,
		Line:        "Starting execution...",
		IsError:     false,
		Timestamp:   time.Now(),
	}

	// Initialize terraform if needed
	workspacePathForInit, _ := e.workspaceManager.GetWorkspacePath(profile.Name)
	if err := e.initInWorkspaceWithStreaming(profile, workspacePathForInit, streamChan); err != nil {
		return e.errorResultWithStreaming(result, fmt.Errorf("terraform init failed: %w", err), startTime, streamChan)
	}

	// Build command
	cmdBuilder := NewCommandBuilder()
	cmd, err := cmdBuilder.BuildCommandFromProfile(profile, workspacePath, execOpts)
	if err != nil {
		return e.errorResultWithStreaming(result, fmt.Errorf("command build failed: %w", err), startTime, streamChan)
	}

	// Execute command with streaming
	return e.executeCommandWithStreaming(cmd, result, startTime, streamChan)
}

// executeCommandWithStreaming executes a command and streams the output
func (e *Executor) executeCommandWithStreaming(cmd *exec.Cmd, result ExecutionResult, startTime time.Time, streamChan chan<- StreamingOutput) ExecutionResult {
	var outputBuffer bytes.Buffer
	var stderrBuffer bytes.Buffer

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return e.errorResultWithStreaming(result, err, startTime, streamChan)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return e.errorResultWithStreaming(result, err, startTime, streamChan)
	}

	if err := cmd.Start(); err != nil {
		return e.errorResultWithStreaming(result, err, startTime, streamChan)
	}

	var wg sync.WaitGroup
	wg.Add(2)

	// stdout
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			outputBuffer.WriteString(line + "\n")
			streamChan <- StreamingOutput{
				ProfileName: result.ProfileName,
				Line:        line,
				IsError:     false,
				Timestamp:   time.Now(),
			}
		}
	}()

	// stderr
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			stderrBuffer.WriteString(line + "\n")
			streamChan <- StreamingOutput{
				ProfileName: result.ProfileName,
				Line:        line,
				IsError:     true,
				Timestamp:   time.Now(),
			}
		}
	}()

	// Wait for both goroutines to finish
	wg.Wait()

	// Wait for command to complete
	err = cmd.Wait()
	duration := time.Since(startTime)

	// Combine outputs
	combinedOutput := outputBuffer.String() + stderrBuffer.String()

	if err != nil {
		// Check if this is an SSO token error
		stderrOutput := stderrBuffer.String()
		if ssoErr := e.handleSSOTokenError(err, stderrOutput, result.ProfileName, streamChan); ssoErr != nil {
			result.Error = ssoErr
			result.Success = false
			result.Output = combinedOutput
			result.Duration = duration
			return result
		}

		result.Error = err
		result.Success = false
		result.Output = combinedOutput
		result.Duration = duration

		// Send completion message
		streamChan <- StreamingOutput{
			ProfileName: result.ProfileName,
			Line:        fmt.Sprintf("❌ Execution failed after %v", duration),
			IsError:     true,
			Timestamp:   time.Now(),
		}

		return result
	}

	result.Success = true
	result.Output = combinedOutput
	result.Duration = duration

	// Send completion message
	streamChan <- StreamingOutput{
		ProfileName: result.ProfileName,
		Line:        fmt.Sprintf("✅ Execution completed successfully in %v", duration),
		IsError:     false,
		Timestamp:   time.Now(),
	}

	return result
}

func (e *Executor) Init(profile Profile) error {
	cmdBuilder := NewCommandBuilder().
		WithBackendConfig(profile.BackendConfig).
		WithBackendDir(profile.BackendDir)

	backendConfigPath := cmdBuilder.GetBackendConfigPath()
	exists, err := utils.CheckFileOrDirExists(backendConfigPath)
	if err != nil {
		return fmt.Errorf("error checking backend config file: %w", err)
	}
	if !exists {
		return fmt.Errorf("backend config file not found: %s", backendConfigPath)
	}

	cmd := cmdBuilder.BuildInitCommand()
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("error creating stderr pipe: %w", err)
	}
	cmd.Stdout = os.Stdout

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error starting terraform init: %w", err)
	}

	stderrBytes, _ := io.ReadAll(stderr)
	stderrOutput := string(stderrBytes)

	// Wait for command to finish
	err = cmd.Wait()

	// If there was an error, check for SSO token error
	// Currently checks specifically for AWS-related errors.
	if err != nil && utils.IsAWSSSOTokenExpired(stderrOutput) {
		fmt.Println("AWS SSO session has expired. Attempting to login...")

		if refreshErr := utils.RefreshAWSSSOFromBackendConfig(backendConfigPath); refreshErr != nil {
			return fmt.Errorf("error refreshing AWS SSO token: %w", refreshErr)
		}

		// Run init again
		retryCmd := cmdBuilder.BuildInitCommand()
		retryCmd.Stdout = os.Stdout
		retryCmd.Stderr = os.Stderr

		return retryCmd.Run()
	}

	// Write stderr output to os.Stderr for user to see
	if err != nil {
		os.Stderr.Write(stderrBytes)
	}

	return err
}

// filterApprovedProfiles filters the profiles to only include approved ones
func (e *Executor) filterApprovedProfiles(profiles []Profile, approvedNames []string) []Profile {
	var approvedProfiles []Profile
	for _, profile := range profiles {
		for _, approvedName := range approvedNames {
			if profile.Name == approvedName {
				approvedProfiles = append(approvedProfiles, profile)
				break
			}
		}
	}
	return approvedProfiles
}

// errorResultWithStreaming creates an error result and sends error message to stream
func (e *Executor) errorResultWithStreaming(result ExecutionResult, err error, startTime time.Time, streamChan chan<- StreamingOutput) ExecutionResult {
	result.Error = err
	result.Success = false
	result.Duration = time.Since(startTime)

	streamChan <- StreamingOutput{
		ProfileName: result.ProfileName,
		Line:        fmt.Sprintf("❌ Error: %v", err),
		IsError:     true,
		Timestamp:   time.Now(),
	}

	return result
}

// WorkspaceCleanup cleans up the created workspaces by the last execution
func (e *Executor) WorkspaceCleanup(plan *ExecutionPlan) error {
	if e.workspaceManager != nil {
		return e.workspaceManager.Cleanup()
	}
	return nil
}

// initInWorkspaceWithStreaming runs terraform init in a workspace with streaming output
func (e *Executor) initInWorkspaceWithStreaming(profile Profile, workspacePath string, streamChan chan<- StreamingOutput) error {
	cmd := NewCommandBuilder().WithWorkingDir(workspacePath).
		WithBackendConfig(profile.BackendConfig).
		WithBackendDir(profile.BackendDir).
		BuildInitCommand()

	streamChan <- StreamingOutput{
		ProfileName: profile.Name,
		Line:        "INIT: Initializing Terraform...",
		IsError:     false,
		Timestamp:   time.Now(),
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	var wg sync.WaitGroup
	wg.Add(2)

	// stdout
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			streamChan <- StreamingOutput{
				ProfileName: profile.Name,
				Line:        fmt.Sprintf("INIT: %s", line),
				IsError:     false,
				Timestamp:   time.Now(),
			}
		}
	}()

	// stderr
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			streamChan <- StreamingOutput{
				ProfileName: profile.Name,
				Line:        fmt.Sprintf("INIT ERROR: %s", line),
				IsError:     true,
				Timestamp:   time.Now(),
			}
		}
	}()

	wg.Wait()

	if err := cmd.Wait(); err != nil {
		streamChan <- StreamingOutput{
			ProfileName: profile.Name,
			Line:        fmt.Sprintf("INIT: ❌ Failed: %v", err),
			IsError:     true,
			Timestamp:   time.Now(),
		}
		return err
	}

	streamChan <- StreamingOutput{
		ProfileName: profile.Name,
		Line:        "INIT: ✅ Terraform initialized successfully",
		IsError:     false,
		Timestamp:   time.Now(),
	}

	return nil
}

// handleSSOTokenError handles SSO token errors
func (e *Executor) handleSSOTokenError(err error, stderrOutput string, profileName string, streamChan chan<- StreamingOutput) error {
	// Check if the error is related to SSO token issues
	if strings.Contains(stderrOutput, "SSO") || strings.Contains(stderrOutput, "token") {
		streamChan <- StreamingOutput{
			ProfileName: profileName,
			Line:        "⚠️  SSO token error detected. Please refresh your SSO token and try again.",
			IsError:     true,
			Timestamp:   time.Now(),
		}
		return fmt.Errorf("SSO token error: %w", err)
	}
	return nil
}
