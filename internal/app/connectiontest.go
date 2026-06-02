package app

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"

	"dback/backend/db"
	"dback/backend/ssh"
	"dback/backend/wordpress"
	"dback/models"
)

const serverProbeMarker = "dback-server-ok"

func (a *App) TestServerConnection(ctx context.Context, profile models.Profile) error {
	if profile.UsesWordPress() {
		client, err := wordpress.NewClient(profile)
		if err != nil {
			return err
		}
		data, err := client.Ping(ctx)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err != nil {
			return err
		}
		if success, ok := data["success"].(bool); ok && !success {
			return fmt.Errorf("wordpress ping failed")
		}
		return nil
	}

	return a.withExecutor(ctx, profile, func(client ssh.Executor) error {
		out, err := client.RunCommand("echo " + serverProbeMarker)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err != nil {
			return fmt.Errorf("%w: %s", err, truncateTestOutput(out))
		}
		if !strings.Contains(out, serverProbeMarker) {
			return fmt.Errorf("server probe failed: %s", truncateTestOutput(out))
		}
		return nil
	})
}

func (a *App) TestDatabaseConnection(ctx context.Context, profile models.Profile) error {
	if profile.UsesWordPress() {
		if err := db.ValidateProfileForWordPress(profile); err != nil {
			return err
		}
		client, err := wordpress.NewClient(profile)
		if err != nil {
			return err
		}
		result, err := client.Query(ctx, "SELECT 1", db.WordPressImportDatabase(profile))
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err != nil {
			return err
		}
		if len(result.Rows) == 0 && result.Message == "" {
			return fmt.Errorf("database returned no output")
		}
		return nil
	}

	if !profile.SupportsSQLQuery() {
		return fmt.Errorf("database test requires MySQL or MariaDB")
	}
	if strings.TrimSpace(profile.TargetDBName) == "" {
		return fmt.Errorf("target database name is required")
	}
	if err := db.ValidateProfileForRemoteOps(profile); err != nil {
		return err
	}
	cmd, err := db.BuildQueryCommand(profile, "SELECT 1", true)
	if err != nil {
		return err
	}
	return a.withExecutor(ctx, profile, func(client ssh.Executor) error {
		out, err := client.RunCommand(cmd)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		out = strings.TrimSpace(out)
		if err != nil {
			if out != "" {
				return fmt.Errorf("%w: %s", err, truncateTestOutput(out))
			}
			return err
		}
		if out == "" {
			return fmt.Errorf("database returned no output")
		}
		return nil
	})
}

func (a *App) withExecutor(ctx context.Context, profile models.Profile, fn func(ssh.Executor) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	client, err := ssh.NewExecutor(profile)
	if err != nil {
		return err
	}
	defer client.Close()

	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = client.Close()
		case <-done:
		}
	}()
	defer close(done)

	if err := fn(client); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return err
	}
	return ctx.Err()
}

func truncateTestOutput(s string) string {
	s = strings.TrimSpace(s)
	if len(s) <= 300 {
		return s
	}
	return s[:300] + "…"
}

func (a *App) GenerateWPKey() (string, error) {
	return GenerateWPAPIKey()
}

func ensureWordPressAPIKey(profile *models.Profile) error {
	if profile == nil || !profile.UsesWordPress() {
		return nil
	}
	if strings.TrimSpace(profile.WPKey) != "" {
		return nil
	}
	key, err := GenerateWPAPIKey()
	if err != nil {
		return err
	}
	profile.WPKey = key
	return nil
}

func GenerateWPAPIKey() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
