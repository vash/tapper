package main

import (
	"fmt"

	"tapper/pkg/terraform"
	"tapper/pkg/utils"
)

// selectMultipleProfiles allows the user to interactively select multiple profiles
func selectMultipleProfiles(cfg *terraform.Config) ([]string, error) {
	profiles := terraform.ListProfiles(cfg)

	if len(profiles) == 0 {
		return nil, fmt.Errorf("no profiles found. Make sure you have matching .tfbackend and .tfvars files in backend/ and vars/ directories")
	}

	config := utils.DefaultMultiSelectConfig(
		"Select profiles (use Tab to select multiple): ",
		"Available Terraform profiles - Tab to select, Enter to confirm",
	)
	return utils.InteractiveSelect(profiles, config)
}
