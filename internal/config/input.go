package config

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/AlecAivazis/survey/v2"
	"github.com/aws/aws-sdk-go/service/synthetics"
	"github.com/daaru00/aws-canary-cli/internal/canary"
	"github.com/urfave/cli/v2"
)

// FilterCanariesByName filter canaries by name flag
func FilterCanariesByName(canaries *[]*canary.Canary, names *[]string) []*canary.Canary {
	selectedCanaries := []*canary.Canary{}

	// Check for provided canaries name
	for _, canary := range *canaries {
		for _, name := range *names {
			match, _ := filepath.Match(name, canary.Name)
			if match {
				selectedCanaries = append(selectedCanaries, canary)
				break
			}
		}
	}

	return selectedCanaries
}

// AskMultipleCanariesSelection ask user to select multiple canaries
func AskMultipleCanariesSelection(c *cli.Context, canaries []*canary.Canary) (*[]*canary.Canary, error) {
	selectedCanaries := []*canary.Canary{}

	// Check if single canary
	if len(canaries) == 1 {
		return &canaries, nil
	}

	// Check if all flag is present
	if c.Bool("all") {
		return &canaries, nil
	}

	// Check for provided canaries name
	names := c.StringSlice("name")
	if len(c.StringSlice("name")) > 0 {
		selectedCanaries = FilterCanariesByName(&canaries, &names)

		// Check if at least one canary was found
		if len(selectedCanaries) == 0 {
			return &selectedCanaries, errors.New("Cannot find any canaries that match provided name filters")
		}

		return &selectedCanaries, nil
	}

	// Build table
	header := fmt.Sprintf("%-25s\t%-20s", "Name", "Tags")
	var options []string
	for _, canary := range canaries {
		options = append(options, fmt.Sprintf("%-20s\t%-20s", canary.Name, *canary.GetFlatTags(",")))
	}

	// Ask selection
	canariesSelectedIndexes := []int{}
	prompt := &survey.MultiSelect{
		Message:  "Select canaries: \n\n  " + header + "\n",
		Options:  options,
		PageSize: 15,
	}
	survey.AskOne(prompt, &canariesSelectedIndexes)
	fmt.Println("")

	// Check response
	if len(canariesSelectedIndexes) == 0 {
		return &selectedCanaries, errors.New("No canaries selected")
	}

	// Load selected canaries
	for _, index := range canariesSelectedIndexes {
		selectedCanaries = append(selectedCanaries, canaries[index])
	}

	return &selectedCanaries, nil
}

// AskSingleCanarySelection ask user to select canaries
func AskSingleCanarySelection(c *cli.Context, canaries []*canary.Canary) (*canary.Canary, error) {
	selectedCanaries := []*canary.Canary{}

	// Check if single canary
	if len(canaries) == 1 {
		return canaries[0], nil
	}

	// Check for provided canaries name
	name := c.String("name")
	if len(c.String("name")) > 0 {
		selectedCanaries = FilterCanariesByName(&canaries, &[]string{
			name,
		})

		// Check if a canary was found
		if len(selectedCanaries) == 0 {
			return nil, errors.New("Cannot find canary that match provided name filter")
		}

		return selectedCanaries[0], nil
	}

	// Build table
	header := fmt.Sprintf("%-25s\t%-20s", "Name", "Tags")
	var options []string
	for _, canary := range canaries {
		options = append(options, fmt.Sprintf("%-20s\t%-20s", canary.Name, *canary.GetFlatTags(",")))
	}

	// Ask selection
	canarySelectedIndex := -1
	prompt := &survey.Select{
		Message:  "Select a canary: \n\n  " + header + "\n",
		Options:  options,
		Help:     "",
		PageSize: 15,
	}
	survey.AskOne(prompt, &canarySelectedIndex)
	fmt.Println("")

	// Check response
	if canarySelectedIndex == -1 {
		return nil, errors.New("No canaries selected")
	}

	return canaries[canarySelectedIndex], nil
}

// AskSingleCanaryRun ask user to select canary run
func AskSingleCanaryRun(runs []*synthetics.CanaryRun) (*synthetics.CanaryRun, error) {
	// Build table
	header := fmt.Sprintf("%-36s\t%-7s\t%-25s\t%-25s", "Id", "Status", "Started At", "Compleated At")
	var options []string
	for _, run := range runs {
		options = append(options, fmt.Sprintf("%-36s\t%-7s\t%-25s\t%-25s", *run.Id, *run.Status.State, *run.Timeline.Started, *run.Timeline.Completed))
	}

	// Ask selection
	canaryRunIndex := -1
	prompt := &survey.Select{
		Message:  "Select canary run: \n\n  " + header + "\n",
		Options:  options,
		PageSize: 15,
	}
	survey.AskOne(prompt, &canaryRunIndex)
	fmt.Println("")

	// Check response
	if canaryRunIndex == -1 {
		return nil, errors.New("No canary run selected")
	}

	return runs[canaryRunIndex], nil
}
