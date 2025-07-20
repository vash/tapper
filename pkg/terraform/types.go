package terraform

import (
	"time"
)

// ExecutionPlan represents a plan for execution across multiple profiles
type ExecutionPlan struct {
	Command          string
	Profiles         []Profile
	Results          []ExecutionResult
	ApprovedProfiles []string
}

// ExecutionResult represents the result of executing a terraform command for a profile
type ExecutionResult struct {
	ProfileName string
	Success     bool
	Output      string
	Error       error
	Duration    time.Duration
	WorkingDir  string
}

// ProgressiveResult wraps ExecutionResult with metadata for progressive display
type ProgressiveResult struct {
	Result    ExecutionResult
	Index     int
	Total     int
	Completed bool
}
