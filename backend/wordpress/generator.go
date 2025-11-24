package wordpress

import (
	"archive/zip"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

// GeneratePlugin creates a WP plugin zip with a unique key
func GeneratePlugin(templatePath, destDir string) (string, string, error) {
	// Generate Key
	key, err := generateKey()
	if err != nil {
		return "", "", err
	}

	// Read Template
	content, err := ioutil.ReadFile(templatePath)
	if err != nil {
		return "", "", fmt.Errorf("failed to read template: %v", err)
	}

	// Replace Key
	pluginCode := strings.ReplaceAll(string(content), "{{API_KEY}}", key)

	// Create Zip
	zipPath := fmt.Sprintf("%s/dback-sync-plugin.zip", destDir)
	file, err := os.Create(zipPath)
	if err != nil {
		return "", "", err
	}
	defer file.Close()

	w := zip.NewWriter(file)
	defer w.Close()

	// Add file to zip
	f, err := w.Create("dback-sync/dback-sync.php")
	if err != nil {
		return "", "", err
	}
	_, err = f.Write([]byte(pluginCode))
	if err != nil {
		return "", "", err
	}

	return key, zipPath, nil
}

func generateKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
