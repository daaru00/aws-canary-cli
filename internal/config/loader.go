package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/daaru00/aws-canary-cli/internal/canary"
	"github.com/joho/godotenv"
	"github.com/urfave/cli/v2"
)

// LoadDotEnv will load environment variable from .env file
func LoadDotEnv() error {
	env := os.Getenv("CANARY_ENV")
	envFile := ".env"

	// Build env file name
	if len(env) != 0 {
		envFile += "." + env
	}

	// Check if file exist
	_, err := os.Stat(envFile)
	if os.IsNotExist(err) {
		return nil
	}

	// Load environment variables
	return godotenv.Load(envFile)
}

// LoadCanaries load canary using user input
func LoadCanaries(c *cli.Context, ses *session.Session) (*[]*canary.Canary, error) {
	canaries := []*canary.Canary{}

	// Search config in sources
	fileName := c.String("config-file")
	parser := c.String("config-parser")

	// Check tests source path argument
	searchPaths := []string{"."}
	if c.Args().Len() > 0 {
		searchPaths = c.Args().Slice()
	}

	// Iterate over search paths provided
	for _, searchPath := range searchPaths {

		// Check provided path type
		info, err := os.Stat(searchPath)
		if err != nil {
			return &canaries, err
		}

		// Check if path is a directory or file
		fileMode := info.Mode()
		if fileMode.IsDir() {
			// Found canary in directory
			canariesFound, err := LoadCanariesFromDir(ses, &searchPath, &fileName, &parser)
			if err != nil {
				return nil, err
			}

			// Append canaries
			canaries = append(canaries, canariesFound...)
		} else if fileMode.IsRegular() {
			// Load canary from file
			canaryFound, err := LoadCanaryFromFile(ses, &searchPath, &parser)
			if err != nil {
				return nil, err
			}

			// Append canaries
			canaries = append(canaries, canaryFound)
		} else {
			return &canaries, fmt.Errorf("Path %s has a unsupported type", searchPath)
		}
	}

	return &canaries, nil
}

// LoadCanaryFromFile load canary from file
func LoadCanaryFromFile(ses *session.Session, filePath *string, parser *string) (*canary.Canary, error) {
	// If file match read content
	fileContent, err := ioutil.ReadFile(*filePath)
	if err != nil {
		return nil, err
	}

	// Interpolate file content
	fileContentInterpolated := InterpolateContent(&fileContent)

	// Parse file content into config object
	fileName := filepath.Base(*filePath)
	extension := filepath.Ext(fileName)
	canaryName := fileName[0 : len(fileName)-len(extension)]
	canary := canary.New(ses, canaryName)
	err = ParseContent(fileContentInterpolated, parser, canary)
	if err != nil {
		return nil, err
	}

	// Add path to config
	if len(canary.Code.Src) == 0 {
		canary.Code.Src = filepath.Dir(*filePath)
	} else {
		canary.Code.Src = path.Join(canary.Code.Src, filepath.Dir(*filePath))
	}

	return canary, nil
}

// LoadCanariesFromDir search config files and load canaries
func LoadCanariesFromDir(ses *session.Session, searchPath *string, fileNameToMatch *string, parser *string) ([]*canary.Canary, error) {
	start := time.Now()
	filesCount := 0
	canaries := []*canary.Canary{}

	// Walk for each files in source path
	err := filepath.Walk(*searchPath, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}
		filesCount++

		// Check if file match name
		fileName := filepath.Base(filePath)
		if fileName != *fileNameToMatch {
			return nil
		}

		// Parse canary from file
		canary, err := LoadCanaryFromFile(ses, &filePath, parser)
		if err != nil {
			return err
		}
		// Add canary to slice
		canaries = append(canaries, canary)
		return nil
	})

	// Check for errors
	if err != nil {
		return canaries, err
	}

	// Check canaries length
	if len(canaries) == 0 {
		round, _ := time.ParseDuration("5ms")
		elapsed := time.Since(start).Round(round)
		return canaries, fmt.Errorf("No canaries found in path %s (%d files scanned in %s)", *searchPath, filesCount, elapsed)
	}

	// Return canaries
	return canaries, err
}
