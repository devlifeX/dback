package db

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"dback/models"
)

var wordpressDBNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_]{1,64}$`)

func ValidateProfileForWordPress(p models.Profile) error {
	siteURL := strings.TrimSpace(p.WPUrl)
	if siteURL == "" {
		siteURL = strings.TrimSpace(p.Host)
	}
	if siteURL == "" {
		return errors.New("wordpress site URL is required")
	}
	u, err := url.Parse(siteURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("invalid wordpress site URL: %q", siteURL)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("wordpress site URL must use http or https")
	}
	if strings.TrimSpace(p.WPKey) == "" {
		return errors.New("wordpress API key is required")
	}
	return nil
}

func ValidateWordPressDatabaseName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	if !wordpressDBNamePattern.MatchString(name) {
		return fmt.Errorf("invalid wordpress database name: %q", name)
	}
	return nil
}

func WordPressImportDatabase(p models.Profile) string {
	return strings.TrimSpace(p.TargetDBName)
}

func ValidateProfileForOps(p models.Profile) error {
	if p.UsesWordPress() {
		return ValidateProfileForWordPress(p)
	}
	return ValidateProfileForRemoteOps(p)
}
