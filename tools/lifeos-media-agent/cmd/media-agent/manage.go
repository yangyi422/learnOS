package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"lifeos-media-agent/internal/source"
)

func secretService(name string) string {
	services := map[string]string{
		"calibre-username": "lifeos-media-agent-calibre-username",
		"calibre-password": "lifeos-media-agent-calibre-password",
		"jellyfin-api-key": "lifeos-media-agent-jellyfin-api-key",
		"lifeos-api-token": "lifeos-media-agent-lifeos-token",
	}
	return services[name]
}
func deleteSecret(ctx context.Context, name string) error {
	service := secretService(name)
	if service == "" {
		return fmt.Errorf("unknown credential name")
	}
	cmd := exec.CommandContext(ctx, "/usr/bin/security", "delete-generic-password", "-s", service)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("could not delete Keychain credential: %w", err)
	}
	return nil
}
func setSecret(ctx context.Context, name string) error {
	service := secretService(name)
	if service == "" {
		return fmt.Errorf("unknown credential name")
	}
	cmd := exec.CommandContext(ctx, "/usr/bin/security", "add-generic-password", "-U", "-a", "lifeos-media-agent", "-s", service, "-w")
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("could not update Keychain: %w", err)
	}
	return nil
}

func status(ctx context.Context) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	path := filepath.Join(home, "Library", "Application Support", "LifeOSMediaAgent", "state.db")
	if _, err = os.Stat(path); err != nil {
		return err
	}
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return err
	}
	defer db.Close()
	var count, pending int
	if err = db.QueryRowContext(ctx, `SELECT count(*),count(CASE WHEN acknowledged_metadata IS NOT metadata_hash OR acknowledged_cover IS NOT cover_hash THEN 1 END) FROM source_items`).Scan(&count, &pending); err != nil {
		return err
	}
	fmt.Printf("items=%d pending=%d\n", count, pending)
	return nil
}

func doctor(ctx context.Context) error {
	fmt.Printf("runtime=%s %s/%s\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	var result error
	report := func(label string, err error) {
		if err == nil {
			fmt.Printf("%s: ok\n", label)
			return
		}
		fmt.Printf("%s: %s\n", label, err)
		result = errors.Join(result, fmt.Errorf("%s: %w", label, err))
	}
	db, err := sql.Open("sqlite3", ":memory:")
	if err == nil {
		err = db.PingContext(ctx)
		db.Close()
	}
	report("sqlite", err)
	if os.Getenv("LIFEOS_URL") == "" {
		report("configuration", fmt.Errorf("LIFEOS_URL is missing"))
	} else {
		report("configuration", nil)
	}
	token, err := keychain(ctx, "lifeos-media-agent-lifeos-token")
	report("LifeOS token in Keychain", err)
	if err == nil {
		report("LifeOS API", probeLifeOS(ctx, token))
	}
	_, err = keychain(ctx, "lifeos-media-agent-jellyfin-api-key")
	report("Jellyfin token in Keychain", err)
	for _, name := range []string{"calibre", "jellyfin"} {
		report(name, scan(ctx, name, func(source.Item) error { return nil }))
	}
	return result
}

func probeLifeOS(ctx context.Context, token string) error {
	base, err := url.Parse(os.Getenv("LIFEOS_URL"))
	if err != nil || base.User != nil || base.Host == "" || (base.Scheme != "https" && !(base.Scheme == "http" && (base.Hostname() == "127.0.0.1" || base.Hostname() == "::1"))) {
		return fmt.Errorf("LifeOS URL must use HTTPS or loopback HTTP")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base.JoinPath("api/v1/library/media-agent/status").String(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	client := &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return nil
}
