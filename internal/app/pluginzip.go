package app

import (
	"fmt"
	"os"

	"dback/backend/wordpress"
)

func (a *App) BuildWordPressPluginZip(siteURL, apiKey string) ([]byte, string, error) {
	return wordpress.BuildPluginZip(siteURL, apiKey)
}

func (a *App) SaveWordPressPluginZip(path, siteURL, apiKey string) (string, error) {
	data, filename, err := wordpress.BuildPluginZip(siteURL, apiKey)
	if err != nil {
		return "", err
	}
	target := path
	if target == "" {
		return "", fmt.Errorf("save path is required")
	}
	if err := os.WriteFile(target, data, 0644); err != nil {
		return "", err
	}
	_ = filename
	return target, nil
}
