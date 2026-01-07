package prd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const DefaultFilename = "prd.json"

// Load loads a PRD from the specified path
func Load(path string) (*PRD, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("prd.json not found at %s", path)
		}
		return nil, fmt.Errorf("failed to read prd.json: %w", err)
	}

	var prd PRD
	if err := json.Unmarshal(data, &prd); err != nil {
		return nil, fmt.Errorf("failed to parse prd.json: %w", err)
	}

	return &prd, nil
}

// LoadFromDir loads a PRD from the default filename in the specified directory
func LoadFromDir(dir string) (*PRD, error) {
	return Load(filepath.Join(dir, DefaultFilename))
}

// LoadFromCurrentDir loads a PRD from the current working directory
func LoadFromCurrentDir() (*PRD, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}
	return LoadFromDir(cwd)
}

// Save saves the PRD to the specified path
func Save(p *PRD, path string) error {
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal prd: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write prd.json: %w", err)
	}

	return nil
}

// SaveToDir saves the PRD to the default filename in the specified directory
func SaveToDir(p *PRD, dir string) error {
	return Save(p, filepath.Join(dir, DefaultFilename))
}

// SaveToCurrentDir saves the PRD to the current working directory
func SaveToCurrentDir(p *PRD) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	return SaveToDir(p, cwd)
}

// Exists checks if a PRD file exists at the specified path
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// ExistsInDir checks if a PRD file exists in the specified directory
func ExistsInDir(dir string) bool {
	return Exists(filepath.Join(dir, DefaultFilename))
}

// ExistsInCurrentDir checks if a PRD file exists in the current directory
func ExistsInCurrentDir() bool {
	cwd, err := os.Getwd()
	if err != nil {
		return false
	}
	return ExistsInDir(cwd)
}

// GetPath returns the path to the PRD file in the current directory
func GetPath() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}
	return filepath.Join(cwd, DefaultFilename), nil
}
