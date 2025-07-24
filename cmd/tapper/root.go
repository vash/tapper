package main

import (
	"fmt"
	"os"

	"tapper/pkg/terraform"
	"tapper/pkg/utils"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "tapper",
	Short: "Tapper - A Terraform profile manager",
	Long: `Tapper is a CLI tool that simplifies running Terraform init and apply commands
with different backend configurations and variable files.

It automatically detects profiles from matching .tfbackend and .tfvars files
in backend/ and vars/ directories.`,
}

// applyCmd represents the apply command
var applyCmd = &cobra.Command{
	Use:     "apply [profile...]",
	Aliases: []string{"run", "r"},
	Short:   "Run terraform apply with selected profile(s)",
	Long: `Run terraform apply with one or more profiles.
If no profile is specified, displays an interactive selection menu.
If one profile is specified, runs on that profile only.
If multiple profiles are specified, runs in parallel across all profiles.`,
	Run: func(cmd *cobra.Command, args []string) {
		executeCommand("apply", args, cmd)
	},
}

// planCmd represents the plan command
var planCmd = &cobra.Command{
	Use:     "plan [profile...]",
	Aliases: []string{"pl", "p"},
	Short:   "Run terraform plan with selected profile(s)",
	Long: `Run terraform plan with one or more profiles.
If no profile is specified, displays an interactive selection menu.
If one profile is specified, runs on that profile only.
If multiple profiles are specified, runs in parallel across all profiles.`,
	Run: func(cmd *cobra.Command, args []string) {
		executeCommand("plan", args, cmd)
	},
}

// destroyCmd represents the destroy command
var destroyCmd = &cobra.Command{
	Use:     "destroy [profile...]",
	Aliases: []string{"d"},
	Short:   "Run terraform destroy with selected profile(s)",
	Long: `Run terraform destroy with one or more profiles.
If no profile is specified, displays an interactive selection menu.
If one profile is specified, runs on that profile only.
If multiple profiles are specified, runs in parallel across all profiles.`,
	Run: func(cmd *cobra.Command, args []string) {
		executeCommand("destroy", args, cmd)
	},
}

// executeCommand handles the execution logic for all terraform commands
func executeCommand(command string, profileArgs []string, cmd *cobra.Command) {
	utils.IsActiveDir()

	cfg, err := terraform.LoadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	var profileNames []string
	if len(profileArgs) == 0 {
		// No profiles specified, let user select
		var err error
		profileNames, err = selectMultipleProfiles(cfg)
		if err != nil {
			fmt.Printf("Error selecting profiles: %v\n", err)
			os.Exit(1)
		}
		if len(profileNames) == 0 {
			fmt.Println("No profiles selected.")
			return
		}
	} else {
		profileNames = profileArgs
	}

	var profiles []terraform.Profile
	for _, profileName := range profileNames {
		profile, exists := terraform.GetProfile(cfg, profileName)
		if !exists {
			fmt.Printf("Profile '%s' not found\n", profileName)
			os.Exit(1)
		}
		profiles = append(profiles, profile)
	}
	fmt.Printf("Selected profiles: %v\n", profiles)

	executor, err := terraform.NewExecutor()
	if err != nil {
		fmt.Printf("Error creating executor: %v\n", err)
		os.Exit(1)
	}

	var additionalArgs []string
	lockValue, err := cmd.Flags().GetBool("lock")
	if err == nil {
		if lockValue {
			additionalArgs = append(additionalArgs, "-lock=true")
		} else {
			additionalArgs = append(additionalArgs, "-lock=false")
		}
	}

	// Set additional args on the executor
	if err := executor.SetAdditionalArgs(additionalArgs); err != nil {
		fmt.Printf("Error setting additional arguments: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Creating execution plan for %s across %d profile(s)...\n", command, len(profiles))
	//TODO: Add target selection
	plan, err := executor.PlanExecution(command, profiles)
	if err != nil {
		fmt.Printf("Error creating execution plan: %v\n", err)
		os.Exit(1)
	}

	defer func() {
		if err := executor.WorkspaceCleanup(plan); err != nil {
			fmt.Printf("Warning: Error cleaning up workspaces: %v\n", err)
		}
	}()

	if len(plan.ApprovedProfiles) == 0 {
		fmt.Println("No profiles approved or execution cancelled.")
		return
	}

	// Execute the approved plan
	fmt.Printf("Executing %s for approved profile(s)...\n", command)
	//TODO: Show errors on failed execution
	_, err = executor.ExecutePlan(plan)
	if err != nil {
		fmt.Printf("Error executing plan: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(applyCmd, planCmd, destroyCmd)

	// Add -lock flag to commands that support it (apply, plan, destroy)
	applyCmd.Flags().BoolP("lock", "l", true, "Lock the state file when locking is supported")
	planCmd.Flags().BoolP("lock", "l", true, "Lock the state file when locking is supported")
	destroyCmd.Flags().BoolP("lock", "l", true, "Lock the state file when locking is supported")
}
