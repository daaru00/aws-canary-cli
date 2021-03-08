package config

import (
	"errors"
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/aws/aws-sdk-go/service/synthetics"
	"github.com/daaru00/aws-canary-cli/internal/canary"
	"github.com/urfave/cli/v2"
)

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
	// Check if single canary
	if len(canaries) == 1 {
		return canaries[0], nil
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
