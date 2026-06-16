package iam

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func readTrustedConfigFile(rawPath string) ([]byte, error) {
	path, err := cleanTrustedConfigFilePath(rawPath)
	if err != nil {
		return nil, err
	}
	// #nosec G304 G703 -- path comes from deployment-controlled env/config and is constrained to a cleaned absolute path.
	return os.ReadFile(path)
}

func cleanTrustedConfigFilePath(rawPath string) (string, error) {
	path := strings.TrimSpace(rawPath)
	if path == "" {
		return "", fmt.Errorf("trusted config file path is empty")
	}
	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("trusted config file path must be absolute")
	}
	return filepath.Clean(path), nil
}
