package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type posthogSink struct {
	apiKey         string
	endpoint       string
	release        string
	installationID string
	client         *http.Client
}

func newPosthogSink(config Config, client *http.Client) (*posthogSink, error) {
	host := strings.TrimSpace(config.PostHogHost)
	if host == "" {
		host = "https://us.i.posthog.com"
	}
	parsed, err := url.Parse(host)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("parse POSTHOG_HOST %q", host)
	}
	installationIDPath := config.InstallationIDPath
	if installationIDPath == "" {
		configDir, err := os.UserConfigDir()
		if err != nil {
			return nil, fmt.Errorf("resolve config directory for analytics ID: %w", err)
		}
		installationIDPath = filepath.Join(configDir, "all-cli", "telemetry", "installation-id")
	}
	installationID, err := loadOrCreateInstallationID(installationIDPath)
	if err != nil {
		return nil, err
	}
	return &posthogSink{
		apiKey:         config.PostHogKey,
		endpoint:       strings.TrimSuffix(host, "/") + "/capture/",
		release:        config.Release,
		installationID: installationID,
		client:         client,
	}, nil
}

func (p *posthogSink) capture(ctx context.Context, command, result string) {
	payload := map[string]any{
		"api_key": p.apiKey,
		"event":   "all_cli command",
		"properties": map[string]string{
			"distinct_id": p.installationID,
			"command":     command,
			"result":      result,
			"release":     p.release,
			"$lib":        "all-cli",
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return
	}
	requestContext := ctx
	if requestContext == nil {
		requestContext = context.Background()
	}
	requestContext, cancel := context.WithTimeout(requestContext, 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(requestContext, http.MethodPost, p.endpoint, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	response, err := p.client.Do(req)
	if err == nil {
		response.Body.Close()
	}
}

func loadOrCreateInstallationID(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err == nil {
		if id := strings.TrimSpace(string(content)); id != "" {
			return id, nil
		}
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("read analytics installation ID: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", fmt.Errorf("create analytics installation ID directory: %w", err)
	}
	id := randomHex(16)
	temp, err := os.CreateTemp(filepath.Dir(path), ".installation-id-*")
	if err != nil {
		return "", fmt.Errorf("create analytics installation ID: %w", err)
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if err := temp.Chmod(0o600); err != nil {
		temp.Close()
		return "", fmt.Errorf("secure analytics installation ID: %w", err)
	}
	if _, err := temp.WriteString(id + "\n"); err != nil {
		temp.Close()
		return "", fmt.Errorf("write analytics installation ID: %w", err)
	}
	if err := temp.Close(); err != nil {
		return "", fmt.Errorf("close analytics installation ID: %w", err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		return "", fmt.Errorf("persist analytics installation ID: %w", err)
	}
	return id, nil
}
