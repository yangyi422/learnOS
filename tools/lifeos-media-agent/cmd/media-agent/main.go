package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"

	"lifeos-media-agent/internal/source"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := run(ctx, os.Args[1:]); err != nil {
		slog.Error("scan failed", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	if len(args) == 2 && args[0] == "service" {
		return serviceCommand(ctx, args[1])
	}
	if len(args) == 1 && args[0] == "logs" {
		return serviceCommand(ctx, "logs")
	}
	if len(args) == 3 && args[0] == "secrets" && args[1] == "delete" {
		return deleteSecret(ctx, args[2])
	}
	if len(args) == 1 && args[0] == "doctor" {
		return doctor(ctx)
	}
	if len(args) == 1 && args[0] == "status" {
		return status(ctx)
	}
	if len(args) == 3 && args[0] == "secrets" && args[1] == "set" {
		return setSecret(ctx, args[2])
	}
	if len(args) == 4 && args[0] == "sync" && args[2] == "--limit" {
		limit, err := strconv.Atoi(args[3])
		if err != nil || limit < 1 {
			return fmt.Errorf("--limit must be positive")
		}
		return synchronize(ctx, args[1], limit)
	}
	if len(args) == 2 && args[0] == "sync" {
		return synchronize(ctx, args[1], 0)
	}
	if len(args) != 3 || args[0] != "sync" || args[2] != "--dry-run" {
		return fmt.Errorf("usage: media-agent sync {calibre|jellyfin} --dry-run")
	}
	encoder := json.NewEncoder(os.Stdout)
	return scanAll(ctx, args[1], func(item source.Item) error { return encoder.Encode(item) })
}

func scan(ctx context.Context, name string, emit func(source.Item) error) error {
	switch name {
	case "calibre":
		endpoint := os.Getenv("CALIBRE_OPDS_URL")
		if endpoint == "" {
			endpoint = "http://192.168.31.122:18081/opds"
		}
		client, err := source.New(endpoint, "")
		if err != nil {
			return err
		}
		username := os.Getenv("CALIBRE_USERNAME")
		if username == "" {
			if stored, err := keychain(ctx, "lifeos-media-agent-calibre-username"); err == nil {
				username = stored
			}
		}
		if username != "" {
			password, err := keychain(ctx, "lifeos-media-agent-calibre-password")
			if err != nil {
				return err
			}
			client.BasicUsername = username
			client.BasicPassword = password
		}
		return client.Calibre(ctx, "按最新", os.Getenv("CALIBRE_ALL_BOOKS_HREF"), emit)
	case "jellyfin":
		endpoint := os.Getenv("JELLYFIN_URL")
		if endpoint == "" {
			endpoint = "http://192.168.31.122:8096"
		}
		secret, err := keychain(ctx, "lifeos-media-agent-jellyfin-api-key")
		if err != nil {
			return err
		}
		client, err := source.New(endpoint, secret)
		if err != nil {
			return err
		}
		return client.Jellyfin(ctx, []string{"电影", "电视剧"}, emit)
	default:
		return fmt.Errorf("unknown source")
	}
}

var keychain = readKeychain

func readKeychain(ctx context.Context, service string) (string, error) {
	secret, err := exec.CommandContext(ctx, "/usr/bin/security", "find-generic-password", "-s", service, "-w").Output()
	value := strings.TrimSuffix(string(secret), "\n")
	if err != nil {
		var commandError *exec.ExitError
		if errors.As(err, &commandError) && commandError.ExitCode() == 44 {
			return "", fmt.Errorf("Keychain entry %s was not found", service)
		}
		return "", fmt.Errorf("cannot read Keychain entry %s: %w", service, err)
	}
	if strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("Keychain entry %s is empty", service)
	}
	return value, nil
}
