package main

import (
	"embed"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"lifeos/desktop/internal/bridge"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// The Vue build produced by `npm run build:desktop`
// (web/vite.desktop.config.ts → desktop/frontend/dist).
//
//go:embed all:frontend/dist
var assets embed.FS

// The default API endpoint compiled into the release. An external
// desktop.config.json next to the .exe, or LIFEOS_API_BASE, still wins, so other
// machines can point at a different server without a rebuild. A fresh clone
// without the (locally maintained) config file embeds the dev default instead.
//
//go:embed desktop.config.json
var embeddedConfig []byte

func main() {
	check := flag.Bool("check", false, "start the local bridge, verify it, print the result and exit (use the console build)")
	minimal := flag.Bool("minimal", false, "open a bare Wails window with inline HTML to test WebView2 only")
	flag.Parse()

	logFile := setupLogging()

	if *minimal {
		runMinimal(logFile)
		return
	}

	dist, err := fs.Sub(assets, "frontend/dist")
	if err != nil {
		fail(logFile, "前端资源缺失", fmt.Sprintf("embedded frontend missing: %v\n请先执行: npm --prefix web run build:desktop", err))
	}

	cfg, cfgErr := bridge.LoadConfig(embeddedConfig)
	if cfgErr != nil {
		log.Printf("configuration error: %v", cfgErr)
	}

	if *check {
		runSelfCheck(cfg, cfgErr, dist, logFile)
		return
	}

	if cfgErr != nil {
		// Show a readable page inside the window instead of exiting silently.
		shell, err := bridge.StartError(cfgErr)
		if err != nil {
			fail(logFile, "无法启动本地服务", err.Error())
		}
		defer shell.Close()
		runWindow(shell.URL(), logFile)
		return
	}

	shell, err := bridge.Start(cfg, dist)
	if err != nil {
		fail(logFile, "无法启动本地服务", err.Error())
	}
	defer shell.Close()
	runWindow(shell.URL(), logFile)
}

func runWindow(url, logFile string) {
	if version := webview2RuntimeVersion(); version == "" {
		fail(logFile, "缺少 WebView2 运行时",
			"这台机器没有安装 Microsoft Edge WebView2 Runtime，桌面窗口无法创建。\n\n"+
				"安装后重新运行即可：\nhttps://developer.microsoft.com/microsoft-edge/webview2/")
	} else {
		log.Printf("WebView2 runtime: %s", version)
	}
	log.Printf("opening window at %s", url)

	windowsOptions := resolveWindowsOptions()

	app := application.New(application.Options{
		Name:        "LifeOS",
		Description: "LifeOS 个人工作台桌面客户端",
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
		Windows: windowsOptions,
	})

	// StartStateFullscreen opens without the native title bar. It is the default
	// (a full-screen workbench is the point of the desktop shell); set
	// LIFEOS_WINDOWED=1 to open as a normal 1440×900 window instead.
	startState := application.WindowStateFullscreen
	if os.Getenv("LIFEOS_WINDOWED") == "1" {
		startState = application.WindowStateNormal
	}

	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "LifeOS",
		Width:            1440,
		Height:           900,
		MinWidth:         1024,
		MinHeight:        700,
		StartState:       startState,
		BackgroundColour: application.NewRGB(238, 244, 252),
		// Marks the shell so the SPA keeps external tools inside this window
		// instead of handing them to the system browser.
		URL: url + "/?shell=desktop",
	})

	window.RegisterKeyBinding("F11", func(w application.Window) { w.ToggleFullscreen() })
	log.Printf("window: fullscreen-on-start=%t (F11 toggles; LIFEOS_WINDOWED=1 opens windowed)", startState == application.WindowStateFullscreen)

	if err := app.Run(); err != nil {
		fail(logFile, "LifeOS 启动失败", err.Error())
	}
}

// resolveWindowsOptions gives WebView2 a user data folder it can actually write
// to. An unusable folder makes CreateCoreWebView2Controller fail with
// 0x8000FFFF "Catastrophic failure" before any page loads — which is exactly
// what happened here, because %LOCALAPPDATA% denied `mkdir LifeOS`.
//
// Candidates are probed in order and the first writable one wins; nothing is
// hardcoded to a machine-specific path.
//
// Environment overrides:
//
//	LIFEOS_WEBVIEW_DATA=<dir>   force a specific data folder
//	LIFEOS_VISUAL_HOSTING=1     use visual hosting (Wails' RDP workaround)
//	LIFEOS_DISABLE_GPU=1        add --disable-gpu
//	LIFEOS_BROWSER_ARGS="..."   extra WebView2 flags
func resolveWindowsOptions() application.WindowsOptions {
	options := application.WindowsOptions{
		UseVisualHosting: os.Getenv("LIFEOS_VISUAL_HOSTING") == "1",
	}

	if dir, err := resolveWebviewDataDir(); err != nil {
		log.Printf("webview: %v", err)
	} else {
		options.WebviewUserDataPath = dir
	}

	if os.Getenv("LIFEOS_DISABLE_GPU") == "1" {
		options.AdditionalBrowserArgs = append(options.AdditionalBrowserArgs, "--disable-gpu")
	}
	if extra := strings.Fields(os.Getenv("LIFEOS_BROWSER_ARGS")); len(extra) > 0 {
		options.AdditionalBrowserArgs = append(options.AdditionalBrowserArgs, extra...)
	}

	log.Printf("webview: user-data=%q visual-hosting=%t browser-args=%v",
		options.WebviewUserDataPath, options.UseVisualHosting, options.AdditionalBrowserArgs)
	return options
}

// resolveWebviewDataDir returns the first candidate directory that can be
// created and written. WebView2 cannot start its browser process otherwise.
func resolveWebviewDataDir() (string, error) {
	candidates := make([]string, 0, 5)
	if explicit := strings.TrimSpace(os.Getenv("LIFEOS_WEBVIEW_DATA")); explicit != "" {
		candidates = append(candidates, explicit)
	}
	if base, err := os.UserCacheDir(); err == nil {
		candidates = append(candidates, filepath.Join(base, "LifeOS", "WebView2"))
	}
	if base, err := os.UserConfigDir(); err == nil {
		candidates = append(candidates, filepath.Join(base, "LifeOS", "WebView2"))
	}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "webview-data"))
	}
	candidates = append(candidates, filepath.Join(os.TempDir(), "LifeOS-WebView2"))

	for _, dir := range candidates {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			log.Printf("webview data dir %q unavailable: %v", dir, err)
			continue
		}
		probe := filepath.Join(dir, ".lifeos-write-probe")
		if err := os.WriteFile(probe, []byte("ok"), 0o600); err != nil {
			log.Printf("webview data dir %q is not writable: %v", dir, err)
			continue
		}
		_ = os.Remove(probe)
		return dir, nil
	}
	return "", errors.New("no writable WebView2 data directory found; set LIFEOS_WEBVIEW_DATA to a writable folder")
}

// runMinimal proves whether WebView2 can open a window at all, with no bridge,
// no embedded assets and no network: `LifeOS-debug.exe -minimal`.
func runMinimal(logFile string) {
	app := application.New(application.Options{
		Name:    "LifeOS minimal",
		Windows: resolveWindowsOptions(),
	})
	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  "LifeOS minimal",
		Width:  640,
		Height: 400,
		HTML:   `<html><body style="font:16px sans-serif;padding:24px">Wails + WebView2 OK</body></html>`,
	})
	if err := app.Run(); err != nil {
		fail(logFile, "最小窗口启动失败", err.Error())
	}
	fmt.Println("minimal window closed normally")
}

// runSelfCheck exercises the same bridge the window uses, then prints a report.
// It is meant for the console build: `LifeOS-debug.exe -check`.
func runSelfCheck(cfg bridge.Config, cfgErr error, dist fs.FS, logFile string) {
	if cfgErr != nil {
		fmt.Printf("CONFIG ERROR: %v\n", cfgErr)
		os.Exit(2)
	}
	fmt.Printf("apiBase: %s (from %s)\n", cfg.APIBase, cfg.Source)

	shell, err := bridge.Start(cfg, dist)
	if err != nil {
		fmt.Printf("BRIDGE ERROR: %v\n", err)
		os.Exit(1)
	}
	defer shell.Close()

	client := &http.Client{Timeout: 10 * time.Second}
	for _, path := range []string{"/healthz", "/", "/api/v1/auth/me"} {
		started := time.Now()
		res, err := client.Get(shell.URL()[:len(shell.URL())-1] + path)
		if err != nil {
			fmt.Printf("%-16s request failed: %v\n", path, err)
			continue
		}
		body, _ := io.ReadAll(io.LimitReader(res.Body, 200))
		_ = res.Body.Close()
		fmt.Printf("%-16s %d (%s) %s\n", path, res.StatusCode, time.Since(started).Round(time.Millisecond), truncate(string(body), 80))
	}
	fmt.Printf("bridge url: %s\n", shell.URL())
	fmt.Printf("log file:   %s\n", logFile)
}

func setupLogging() string {
	dir, err := os.Executable()
	if err == nil {
		dir = filepath.Dir(dir)
	} else {
		dir = os.TempDir()
	}
	path := filepath.Join(dir, "lifeos-desktop.log")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		log.SetOutput(os.Stderr)
		return "(log file unavailable: " + err.Error() + ")"
	}
	log.SetOutput(io.MultiWriter(os.Stderr, file))
	log.SetFlags(log.LstdFlags)
	log.SetPrefix("[lifeos-desktop] ")
	log.Printf("starting %s", dir)
	return path
}

func fail(logFile, title, message string) {
	log.Printf("%s: %s", title, message)
	fatalDialog("LifeOS — "+title, message+"\n\n日志: "+logFile)
	os.Exit(1)
}

func truncate(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[:limit] + "…"
}
