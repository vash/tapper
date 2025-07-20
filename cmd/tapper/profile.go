package main

import (
	"fmt"
	"os"

	"tapper/pkg/terraform"
	"tapper/pkg/utils"

	"github.com/spf13/cobra"
)

var (
	profileName   string
	backendConfig string
	varFile       string
	backendDir    string
	varsDir       string
)

// profileCmd represents the profile command
var profileCmd = &cobra.Command{
	Use:     "profile",
	Aliases: []string{"p"},
	Short:   "Manage Terraform profiles",
	Long:    `Create, list, and manage Terraform profiles.`,
}

// createProfileCmd creates a new profile
var createProfileCmd = &cobra.Command{
	Use:     "create",
	Aliases: []string{"c"},
	Short:   "Create a new profile",
	Long:    `Create a new Terraform profile with the specified backend config and var file.`,
	Run: func(cmd *cobra.Command, args []string) {
		utils.IsActiveDir()

		fmt.Println("Note: Profiles are now auto-detected from filesystem.")
		fmt.Println("To create a profile, simply add matching .tfbackend and .tfvars files")
		fmt.Println("to the backend/ and vars/ directories respectively.")
		fmt.Printf("Example: backend/%s.tfbackend and vars/%s.tfvars\n", profileName, profileName)

		if profileName == "" {
			fmt.Println("Profile name is required")
			os.Exit(1)
		}

		if backendConfig == "" {
			fmt.Println("Backend config is required")
			os.Exit(1)
		}

		if varFile == "" {
			fmt.Println("Var file is required")
			os.Exit(1)
		}

		// Set default directories if not provided
		if backendDir == "" {
			backendDir = "backend"
		}

		if varsDir == "" {
			varsDir = "vars"
		}

		fmt.Printf("To create profile '%s', ensure these files exist:\n", profileName)
		fmt.Printf("  - %s/%s\n", backendDir, backendConfig)
		fmt.Printf("  - %s/%s\n", varsDir, varFile)
		fmt.Println("The profile will be automatically detected when you run tapper commands.")
	},
}

// listProfilesCmd lists all profiles
var listProfilesCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"l", "ls"},
	Short:   "List all profiles",
	Long:    `List all Terraform profiles.`,
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := terraform.LoadConfig()
		if err != nil {
			fmt.Printf("Error loading config: %v\n", err)
			os.Exit(1)
		}

		if len(cfg.Profiles) == 0 {
			fmt.Println("No profiles found")
			fmt.Println("Make sure you have matching .tfbackend and .tfvars files in backend/ and vars/ directories")
			return
		}

		fmt.Println("Available profiles:")
		for _, profile := range cfg.Profiles {
			fmt.Printf("- %s (Backend: %s, Vars: %s, Last used: %s)\n",
				profile.Name,
				profile.BackendConfig,
				profile.VarFile,
				profile.LastUsed)
		}
	},
}

// deleteProfileCmd deletes a profile
var deleteProfileCmd = &cobra.Command{
	Use:     "delete",
	Aliases: []string{"d", "rm"},
	Short:   "Delete a profile",
	Long:    `Delete a Terraform profile.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Note: Profiles are now auto-detected from filesystem.")
		fmt.Println("To delete a profile, remove the corresponding .tfbackend and .tfvars files")
		fmt.Println("from the backend/ and vars/ directories respectively.")

		if profileName == "" {
			fmt.Println("Profile name is required")
			os.Exit(1)
		}

		fmt.Printf("To delete profile '%s', remove these files:\n", profileName)
		fmt.Printf("  - backend/%s.tfbackend\n", profileName)
		fmt.Printf("  - vars/%s.tfvars\n", profileName)
		fmt.Println("The profile will no longer be detected after the files are removed.")
	},
}

func init() {
	rootCmd.AddCommand(profileCmd)
	profileCmd.AddCommand(createProfileCmd, listProfilesCmd, deleteProfileCmd)

	// Add flags for the create command
	createProfileCmd.Flags().StringVarP(&profileName, "name", "n", "", "Profile name (required)")
	createProfileCmd.Flags().StringVarP(&backendConfig, "backend-config", "b", "", "Backend config file (required)")
	createProfileCmd.Flags().StringVarP(&varFile, "var-file", "v", "", "Var file (required)")
	createProfileCmd.Flags().StringVarP(&backendDir, "backend-dir", "", "backend", "Backend directory")
	createProfileCmd.Flags().StringVarP(&varsDir, "vars-dir", "", "vars", "Variables directory")

	createProfileCmd.MarkFlagRequired("name")
	createProfileCmd.MarkFlagRequired("backend-config")
	createProfileCmd.MarkFlagRequired("var-file")

	// Add flags for the delete command
	deleteProfileCmd.Flags().StringVarP(&profileName, "name", "n", "", "Profile name (required)")
	deleteProfileCmd.MarkFlagRequired("name")
}
