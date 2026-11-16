package main

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const agentLabel = "com.lifeos.media-agent"

const launchTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>Label</key><string>com.lifeos.media-agent</string>
<key>ProgramArguments</key><array>
<string>/bin/sh</string><string>-c</string>
<string>for delay in 0 30 120; do /bin/sleep "$delay"; "$1" sync all &amp;&amp; exit 0; done; exit 1</string>
<string>media-agent</string><string>%s</string>
</array>
<key>EnvironmentVariables</key><dict>%s</dict>
<key>StartCalendarInterval</key><dict><key>Hour</key><integer>3</integer><key>Minute</key><integer>0</integer></dict>
<key>StandardOutPath</key><string>%s</string>
<key>StandardErrorPath</key><string>%s</string>
</dict></plist>
`

func xmlText(value string) string {
	var out strings.Builder
	_ = xml.EscapeText(&out, []byte(value))
	return out.String()
}
func launchPlist(binary, log string) string {
	var env strings.Builder
	for _, key := range []string{"LIFEOS_URL", "CALIBRE_OPDS_URL", "CALIBRE_ALL_BOOKS_HREF", "JELLYFIN_URL", "JELLYFIN_INCLUDE_R18"} {
		if value := os.Getenv(key); value != "" {
			fmt.Fprintf(&env, "<key>%s</key><string>%s</string>", key, xmlText(value))
		}
	}
	return fmt.Sprintf(launchTemplate, xmlText(binary), env.String(), xmlText(log), xmlText(log))
}
func serviceCommand(ctx context.Context, action string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	plist := filepath.Join(home, "Library", "LaunchAgents", agentLabel+".plist")
	domain := fmt.Sprintf("gui/%d", os.Getuid())
	target := domain + "/" + agentLabel
	logDir := filepath.Join(home, "Library", "Logs", "LifeOSMediaAgent")
	log := filepath.Join(logDir, "sync.log")
	ctl := func(args ...string) error {
		command := exec.CommandContext(ctx, "/bin/launchctl", args...)
		command.Stdout, command.Stderr = os.Stdout, os.Stderr
		return command.Run()
	}
	switch action {
	case "install":
		if _, err := lifeOSURL(); err != nil {
			return err
		}
		binary, err := os.Executable()
		if err != nil {
			return err
		}
		if err = os.MkdirAll(filepath.Dir(plist), 0700); err != nil {
			return err
		}
		if err = os.MkdirAll(logDir, 0700); err != nil {
			return err
		}
		logFile, err := os.OpenFile(log, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			return err
		}
		if err = logFile.Close(); err != nil {
			return err
		}
		file, err := os.CreateTemp(filepath.Dir(plist), ".lifeos-media-agent-*")
		if err != nil {
			return err
		}
		defer os.Remove(file.Name())
		_, err = io.WriteString(file, launchPlist(binary, log))
		closeErr := file.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
		if err = exec.CommandContext(ctx, "/usr/bin/plutil", "-lint", file.Name()).Run(); err != nil {
			return err
		}
		if exec.CommandContext(ctx, "/bin/launchctl", "print", target).Run() == nil {
			if err = ctl("bootout", target); err != nil {
				return err
			}
		}
		if err = os.Rename(file.Name(), plist); err != nil {
			return err
		}
		return ctl("bootstrap", domain, plist)
	case "status":
		return ctl("print", target)
	case "start":
		return ctl("kickstart", target)
	case "stop":
		return ctl("kill", "SIGTERM", target)
	case "uninstall":
		if exec.CommandContext(ctx, "/bin/launchctl", "print", target).Run() == nil {
			if err = ctl("bootout", target); err != nil {
				return err
			}
		}
		if err = os.Remove(plist); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	case "logs":
		command := exec.CommandContext(ctx, "/usr/bin/tail", "-n", "100", log)
		command.Stdout, command.Stderr = os.Stdout, os.Stderr
		return command.Run()
	default:
		return fmt.Errorf("unknown service command")
	}
}
