package terraform

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// InteractionHandler handles user interactions like approval prompts
type InteractionHandler struct{}

// NewInteractionHandler creates a new user interaction handler
func NewInteractionHandler() *InteractionHandler {
	return &InteractionHandler{}
}

// ReviewAndApproveResults displays complete results and handles approval
func (h *InteractionHandler) ReviewAndApproveResults(results []ExecutionResult) ([]string, error) {
	var approvedProfiles []string

	for _, result := range results {
		fmt.Printf("=== Profile: %s ===\n", result.ProfileName)
		fmt.Printf("Duration: %v\n", result.Duration)
		fmt.Printf("Working Directory: %s\n", result.WorkingDir)

		if result.Error != nil {
			fmt.Printf("Status: Failed\n")
			fmt.Printf("Error: %v\n", result.Error)
		} else if result.Success {
			fmt.Printf("Status: Success\n")
		}

		if result.Output != "" {
			fmt.Printf("\nComplete Output:\n%s\n", result.Output)
		}

		approved := h.PromptForApproval(result.ProfileName)
		if approved {
			approvedProfiles = append(approvedProfiles, result.ProfileName)
			fmt.Printf("Approved: %s\n", result.ProfileName)
		} else {
			fmt.Printf("Rejected: %s\n", result.ProfileName)
		}

		fmt.Println(strings.Repeat("-", 80))
	}

	if len(approvedProfiles) == 0 {
		fmt.Println("No profiles approved for execution.")
		return nil, nil
	}
	// If there's exactly one profile - don't verify
	if len(results) == 1 {
		return approvedProfiles, nil
	}
	return h.ConfirmBatchExecution(approvedProfiles)
}

// PromptForApproval prompts the user for approval of a specific profile
func (h *InteractionHandler) PromptForApproval(profileName string) bool {
	fmt.Printf("Approve execution for profile '%s'? (y/n): ", profileName)
	return h.getYesNoResponse()
}

// ConfirmBatchExecution confirms execution of multiple approved profiles
func (h *InteractionHandler) ConfirmBatchExecution(approvedProfiles []string) ([]string, error) {
	fmt.Printf("\nApproved profiles: %s\n", strings.Join(approvedProfiles, ", "))
	fmt.Print("Proceed with execution? (y/n): ")

	if h.getYesNoResponse() {
		return approvedProfiles, nil
	}

	fmt.Println("Execution cancelled.")
	return nil, nil
}

// getYesNoResponse gets a yes/no response from the user
func (h *InteractionHandler) getYesNoResponse() bool {
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("Error reading input: %v, defaulting to 'no'\n", err)
		return false
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}
