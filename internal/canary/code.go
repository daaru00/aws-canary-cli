package canary

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

// Code structure
type Code struct {
	archivename     string
	archivepath     string
	archives3bucket string
	archives3key    string
	clients         *clients

	Src     string `yaml:"src" json:"src"`
	Handler string `yaml:"handler" json:"handler"`
}

// CreateArchive create a ZIP archive from code path
func (c *Code) CreateArchive(name *string, pathprefix *string) error {
	c.archivename = fmt.Sprintf("%s.zip", *name)
	c.archivepath = path.Join(os.TempDir(), c.archivename)

	// Create ZIP archive
	destinationFile, err := os.Create(c.archivepath)
	if err != nil {
		return err
	}

	// Initialize write
	codeZip := zip.NewWriter(destinationFile)

	// Walk for each files in source path
	err = filepath.Walk(c.Src, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Elaborate destination path
		destPath := filePath
		if len(c.Src) != 0 && c.Src != "." && c.Src != "./" {
			destPath = strings.TrimPrefix(destPath, c.Src)
		}
		destPath = path.Join(*pathprefix, destPath)

		// Add file to ZIP archive
		zipFile, err := codeZip.Create(destPath)
		if err != nil {
			return err
		}
		fsFile, err := os.Open(filePath)
		if err != nil {
			return err
		}
		_, err = io.Copy(zipFile, fsFile)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Close ZIP archive
	err = codeZip.Close()
	if err != nil {
		return err
	}
	return nil
}

// ReadArchive will return the archive data
func (c *Code) ReadArchive() ([]byte, error) {
	return ioutil.ReadFile(c.archivepath)
}

// DeleteArchive will delete the temporary archive
func (c *Code) DeleteArchive() error {
	return os.Remove(c.archivepath)
}

// Upload will upload archive to S3
func (c *Code) Upload(bucket *string) error {
	// Open archive file
	file, err := os.Open(c.archivepath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Set archive s3 location
	c.archives3bucket = *bucket
	c.archives3key = c.archivename

	// Upload archive
	_, err = c.clients.s3uploader.Upload(&s3manager.UploadInput{
		Bucket: &c.archives3bucket,
		Key:    &c.archives3key,
		Body:   file,
	})

	return err
}

// InstallNpmDependencies will install npm dependencies
func (c *Code) InstallNpmDependencies() (string, error) {
	var outBuffer, errBuffer bytes.Buffer

	// Check if package.json exist
	if _, err := os.Stat(path.Join(c.Src, "package.json")); os.IsNotExist(err) {
		return outBuffer.String(), nil
	}

	// Prepare npm dependencies install command
	cmd := exec.Command("npm", "install", "--production")
	cmd.Dir = c.Src

	// Set outputs
	cmd.Stdout = &outBuffer
	cmd.Stderr = &errBuffer

	// Run command
	err := cmd.Run()
	if err != nil {
		return outBuffer.String(), fmt.Errorf("Error installing npm dependencies in %s: %s", c.Src, errBuffer.String())
	}

	return outBuffer.String(), nil
}

// InstallPipDependencies will install pip dependencies
func (c *Code) InstallPipDependencies() (string, error) {
	var outBuffer, errBuffer bytes.Buffer

	// Check if requirements.txt exist
	if _, err := os.Stat(path.Join(c.Src, "requirements.txt")); os.IsNotExist(err) {
		return outBuffer.String(), nil
	}

	// Prepare npm dependencies install command
	cmd := exec.Command("pip", "install", "-r", "requirements.txt")
	cmd.Dir = c.Src

	// Set outputs
	cmd.Stdout = &outBuffer
	cmd.Stderr = &errBuffer

	// Run command
	err := cmd.Run()
	if err != nil {
		return outBuffer.String(), fmt.Errorf("Error installing npm dependencies in %s: %s", c.Src, errBuffer.String())
	}

	return outBuffer.String(), nil
}
