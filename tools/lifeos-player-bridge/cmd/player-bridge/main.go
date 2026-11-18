// Command player-bridge turns a lifeos-player:// URL into a PotPlayer launch for
// a file on the mapped NAS drive. It starts, does one job and exits: no HTTP
// service, no listening port, no long-running process.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"lifeos/player-bridge/internal/config"
	"lifeos/player-bridge/internal/jellyfin"
	"lifeos/player-bridge/internal/logger"
	"lifeos/player-bridge/internal/pathmap"
	"lifeos/player-bridge/internal/player"
	"lifeos/player-bridge/internal/protocol"
	"lifeos/player-bridge/internal/security"
)

const version = "0.1.0"

const (
	exitOK         = 0
	exitUsage      = 2
	exitConfig     = 3
	exitCredential = 4
	exitJellyfin   = 5
	exitLibrary    = 6
	exitPathMap    = 7
	exitFile       = 8
	exitPlayer     = 9
)

// Bounds that reject obviously wrong media entries before touching the disk.
const (
	minMediaBytes int64 = 1 << 10
	maxMediaBytes int64 = 400 << 30
)

func main() {
	// Built as a GUI binary so a protocol launch shows no console window; attach to the
	// parent console when one exists so manual runs still print.
	attachParentConsole()
	configPath := flag.String("config", "", "path to config.yaml (default: %LOCALAPPDATA%\\LifeOS\\PlayerBridge\\config.yaml)")
	verify := flag.Bool("verify", false, "check config, credential, Jellyfin, mappings and PotPlayer, then exit")
	showVersion := flag.Bool("version", false, "print the bridge version")
	probe := flag.String("probe", "", "map one Jellyfin container path through the configured mappings and report it")
	resolve := flag.String("resolve", "", "resolve one Jellyfin item id end to end (library, path, file) without launching the player")
	play := flag.String("play", "", "play one Jellyfin item id exactly as the protocol would, but from the console")
	flag.Parse()

	if *showVersion {
		fmt.Println("lifeos-player-bridge", version)
		os.Exit(exitOK)
	}
	if *configPath == "" {
		*configPath = config.DefaultPath()
	}
	log, closeLog, err := logger.New(filepath.Join(filepath.Dir(*configPath), "logs"))
	if err != nil {
		log = nil
	}
	if closeLog != nil {
		defer closeLog()
	}
	// Console output is produced here, not by the file logger: a protocol launch has no
	// console, but a manual invocation must always show why it stopped.
	info := func(step, message string, secrets ...string) {
		fmt.Fprintf(os.Stdout, "info  %s — %s\n", step, logger.Redact(message, secrets...))
		if log != nil {
			log.Info(step, message, secrets...)
		}
	}
	fail := func(step, message string, secrets ...string) {
		fmt.Fprintf(os.Stderr, "FAIL  %s — %s\n", step, logger.Redact(message, secrets...))
		if log != nil {
			log.Fail(step, message, secrets...)
		}
	}

	if *verify {
		os.Exit(runVerify(*configPath, info, fail))
	}
	if *probe != "" {
		os.Exit(runProbe(*configPath, *probe))
	}
	if *play != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		os.Exit(run(ctx, *configPath, "lifeos-player://play?itemId="+*play, info, fail))
	}
	if *resolve != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		os.Exit(runResolve(ctx, *configPath, *resolve))
	}
	if flag.NArg() != 1 {
		fail("usage", "expected exactly one lifeos-player:// URL")
		os.Exit(exitUsage)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	os.Exit(run(ctx, *configPath, flag.Arg(0), info, fail))
}

type logFunc func(step, message string, secrets ...string)

func run(ctx context.Context, configPath, rawURL string, info, fail logFunc) int {
	request, err := protocol.Parse(rawURL)
	if err != nil {
		fail("protocol", err.Error())
		return exitUsage
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		fail("config", err.Error())
		return exitConfig
	}
	credential, err := security.ReadGeneric(cfg.Jellyfin.CredentialTarget)
	if err != nil {
		fail("credential", credentialHint(err))
		return exitCredential
	}
	client := jellyfin.New(cfg.Jellyfin.BaseURL, strings.TrimSpace(credential.Secret), time.Duration(cfg.Jellyfin.TimeoutSeconds)*time.Second)

	// Jellyfin answers slowly when it is indexing (a 3s /System/Info is normal on a busy
	// NAS), so both lookups run together; the library gate still decides what can launch.
	ancestors, playback, ancestorsErr, playbackErr := fetchItem(ctx, client, request.ItemID, cfg.Jellyfin.UserID)
	if ancestorsErr != nil {
		fail("jellyfin", errorHint(ancestorsErr), credential.Secret)
		return exitJellyfin
	}
	if !jellyfin.LibraryAllowed(ancestors, cfg.Jellyfin.AllowedLibraries) {
		fail("library", "item is outside the allowed libraries")
		return exitLibrary
	}
	if playbackErr != nil {
		fail("jellyfin", errorHint(playbackErr), credential.Secret)
		return exitJellyfin
	}
	source := playback.MediaSources[0]
	if strings.TrimSpace(source.Path) == "" {
		fail("jellyfin", "playback info has no server-side path")
		return exitJellyfin
	}
	if source.Size > 0 && (source.Size < minMediaBytes || source.Size > maxMediaBytes) {
		fail("jellyfin", fmt.Sprintf("implausible media size %d bytes", source.Size))
		return exitJellyfin
	}

	mapper := pathmap.New(mappingList(cfg), cfg.Extensions(), cfg.Playback.CheckFileExists)
	mediaPath, err := mapper.Map(source.Path)
	if err != nil {
		hint := "check path_mappings in config.yaml"
		if errors.Is(err, pathmap.ErrDriveUnavailable) {
			hint = "check the SMB mapping for the NAS drive (net use)"
		}
		fail("path", err.Error()+" — "+hint)
		if errors.Is(err, pathmap.ErrFileMissing) || errors.Is(err, pathmap.ErrDriveUnavailable) {
			return exitFile
		}
		return exitPathMap
	}

	executable, err := player.Detect(cfg.Player.Executable)
	if err != nil {
		fail("player", "PotPlayer not found — install it or set player.executable")
		return exitPlayer
	}
	if err := player.Launch(executable, mediaPath); err != nil {
		fail("player", err.Error())
		return exitPlayer
	}
	info("play", "launched PotPlayer for "+mediaPath)
	return exitOK
}

func mappingList(cfg *config.Config) []pathmap.Mapping {
	mappings := make([]pathmap.Mapping, len(cfg.PathMappings))
	for i, mapping := range cfg.PathMappings {
		mappings[i] = pathmap.Mapping{Source: mapping.Source, Target: mapping.Target}
	}
	return mappings
}

func credentialHint(err error) string {
	if errors.Is(err, security.ErrCredentialNotFound) {
		return "no stored Jellyfin token — run: cmdkey /generic:lifeos-player-bridge/jellyfin /user:<user-id> /pass:<api-token>"
	}
	if errors.Is(err, security.ErrUnsupported) {
		return "credential store is only available on Windows"
	}
	return err.Error()
}

func errorHint(err error) string {
	switch {
	case errors.Is(err, jellyfin.ErrUnauthorized):
		return "Jellyfin rejected the stored token — check the dedicated user and its library permission"
	case errors.Is(err, jellyfin.ErrNotFound):
		return "Jellyfin has no such item (deleted, or not visible to the dedicated user)"
	case errors.Is(err, jellyfin.ErrUnavailable):
		if strings.Contains(err.Error(), "Timeout exceeded") || strings.Contains(err.Error(), "deadline exceeded") {
			return "Jellyfin did not answer in time — the server may be busy or indexing, or the NAS is slow (" + err.Error() + "); raise jellyfin.timeout_seconds if this repeats"
		}
		// Keep the transport cause: "unreachable" alone hides DNS, TLS and proxy errors.
		return "Jellyfin is unreachable — check the home network and NAS (" + err.Error() + ")"
	}
	return err.Error()
}

// runVerify reports every prerequisite in one pass. It never stops at the first
// failure: fixing one link at a time is exactly the loop this check exists to
// avoid. The exit code is the first failing step's code.
func runVerify(configPath string, info, fail logFunc) int {
	code := exitOK
	report := func(ok bool, step, message string, failureCode int) {
		if ok {
			fmt.Println("OK    " + step + messageSuffix(message))
			return
		}
		if code == exitOK {
			code = failureCode
		}
		fmt.Println("FAIL  " + step + messageSuffix(message))
	}

	cfg, configErr := config.Load(configPath)
	if configErr != nil {
		report(false, "config", configErr.Error(), exitConfig)
	} else {
		report(true, "config", configPath, exitOK)
	}

	var token string
	if cfg != nil {
		credential, err := security.ReadGeneric(cfg.Jellyfin.CredentialTarget)
		if err != nil {
			report(false, "credential", credentialHint(err), exitCredential)
		} else {
			token = strings.TrimSpace(credential.Secret)
			report(true, "credential", cfg.Jellyfin.CredentialTarget, exitOK)
		}

		if err := pingJellyfin(cfg.Jellyfin.BaseURL, token, time.Duration(cfg.Jellyfin.TimeoutSeconds)*time.Second); err != nil {
			report(false, "jellyfin", errorHint(err), exitJellyfin)
		} else {
			report(true, "jellyfin", cfg.Jellyfin.BaseURL, exitOK)
		}

		if executable, err := player.Detect(cfg.Player.Executable); err != nil {
			report(false, "player", "PotPlayer not found — install it or set player.executable to PotPlayerMini64.exe", exitPlayer)
		} else {
			report(true, "player", executable, exitOK)
		}

		for _, mapping := range cfg.PathMappings {
			if err := pathmap.CheckTargetDirectory(mapping.Target); err != nil {
				report(false, "mapping", mapping.Target+" is not an accessible directory — check the SMB share or drive mapping and permissions", exitPathMap)
				continue
			}
			report(true, "mapping", mapping.Target, exitOK)
		}
		return code
	}

	// Without a parsable config there are still two useful checks to report.
	if executable, err := player.Detect("auto"); err != nil {
		report(false, "player", "PotPlayer not found — install it or set player.executable to PotPlayerMini64.exe", exitPlayer)
	} else {
		report(true, "player", executable, exitOK)
	}
	report(false, "mapping", "skipped: fill in config.yaml first", exitConfig)
	return code
}

// fetchItem asks Jellyfin for the library ancestry and the playback sources in
// parallel; on a busy NAS the two round trips dominate the run time.
func fetchItem(ctx context.Context, client *jellyfin.Client, itemID, userID string) ([]jellyfin.Ancestor, *jellyfin.PlaybackInfo, error, error) {
	type ancestorsResult struct {
		value []jellyfin.Ancestor
		err   error
	}
	type playbackResult struct {
		value *jellyfin.PlaybackInfo
		err   error
	}
	ancestorsCh := make(chan ancestorsResult, 1)
	playbackCh := make(chan playbackResult, 1)
	go func() {
		value, err := client.Ancestors(ctx, itemID, userID)
		ancestorsCh <- ancestorsResult{value: value, err: err}
	}()
	go func() {
		value, err := client.PlaybackInfo(ctx, itemID, userID)
		playbackCh <- playbackResult{value: value, err: err}
	}()
	ancestorsOutcome := <-ancestorsCh
	playbackOutcome := <-playbackCh
	return ancestorsOutcome.value, playbackOutcome.value, ancestorsOutcome.err, playbackOutcome.err
}

// runResolve walks the same steps as a real playback but stops before launching the
// player, so a failing item can be diagnosed without touching PotPlayer.
func runResolve(ctx context.Context, configPath, itemID string) int {
	request, err := protocol.Parse("lifeos-player://play?itemId=" + itemID)
	if err != nil {
		fmt.Println("FAIL  item id — " + err.Error())
		return exitUsage
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Println("FAIL  config — " + err.Error())
		return exitConfig
	}
	credential, err := security.ReadGeneric(cfg.Jellyfin.CredentialTarget)
	if err != nil {
		fmt.Println("FAIL  credential — " + credentialHint(err))
		return exitCredential
	}
	client := jellyfin.New(cfg.Jellyfin.BaseURL, strings.TrimSpace(credential.Secret), time.Duration(cfg.Jellyfin.TimeoutSeconds)*time.Second)
	ancestors, playback, ancestorsErr, playbackErr := fetchItem(ctx, client, request.ItemID, cfg.Jellyfin.UserID)
	if ancestorsErr != nil {
		fmt.Println("FAIL  jellyfin — " + errorHint(ancestorsErr))
		return exitJellyfin
	}
	names := make([]string, 0, len(ancestors))
	for _, ancestor := range ancestors {
		names = append(names, ancestor.Name+"["+ancestor.CollectionType+"]")
	}
	fmt.Println("libs  " + strings.Join(names, " > "))
	if !jellyfin.LibraryAllowed(ancestors, cfg.Jellyfin.AllowedLibraries) {
		fmt.Println("FAIL  library — item belongs to " + describeLibraries(ancestors) + ", allowed_libraries is " + strings.Join(cfg.Jellyfin.AllowedLibraries, ", "))
		return exitLibrary
	}
	fmt.Println("OK    library")
	if playbackErr != nil {
		fmt.Println("FAIL  jellyfin — " + errorHint(playbackErr))
		return exitJellyfin
	}
	source := playback.MediaSources[0]
	fmt.Printf("src   %s (container=%s size=%d)\n", source.Path, source.Container, source.Size)
	mapper := pathmap.New(mappingList(cfg), cfg.Extensions(), cfg.Playback.CheckFileExists)
	mapped, mapErr := mapper.Map(source.Path)
	if mapErr != nil {
		fmt.Println("FAIL  path — " + mapErr.Error())
		for _, mapping := range cfg.PathMappings {
			fmt.Printf("      rule %s -> %s\n", mapping.Source, mapping.Target)
		}
		if errors.Is(mapErr, pathmap.ErrFileMissing) || errors.Is(mapErr, pathmap.ErrDriveUnavailable) {
			return exitFile
		}
		return exitPathMap
	}
	info, statErr := os.Stat(mapped)
	if statErr != nil {
		fmt.Println("FAIL  file — " + statErr.Error())
		return exitFile
	}
	fmt.Printf("OK    path %s (%d bytes)\n", mapped, info.Size())
	executable, err := player.Detect(cfg.Player.Executable)
	if err != nil {
		fmt.Println("FAIL  player — PotPlayer not found")
		return exitPlayer
	}
	fmt.Println("OK    player " + executable)
	fmt.Println("ready — run the same item id from LifeOS to actually play it")
	return exitOK
}

// runProbe answers the one question setup gets wrong: does this container path
// really resolve to a file on this machine? It never launches a player.
func runProbe(configPath, containerPath string) int {
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Println("FAIL  config — " + err.Error())
		return exitConfig
	}
	for _, mapping := range cfg.PathMappings {
		fmt.Printf("rule  %s -> %s\n", mapping.Source, mapping.Target)
	}
	mapper := pathmap.New(mappingList(cfg), cfg.Extensions(), cfg.Playback.CheckFileExists)
	mapped, mapErr := mapper.Map(containerPath)
	if mapErr != nil {
		fmt.Println("FAIL  path — " + mapErr.Error())
		if errors.Is(mapErr, pathmap.ErrFileMissing) || errors.Is(mapErr, pathmap.ErrDriveUnavailable) {
			return exitFile
		}
		return exitPathMap
	}
	info, statErr := os.Stat(mapped)
	if statErr != nil {
		fmt.Println("FAIL  file — " + statErr.Error())
		return exitFile
	}
	fmt.Printf("OK    %s (%d bytes)\n", mapped, info.Size())
	return exitOK
}

// describeLibraries renders the collection types an item sits in, e.g.
// "电视剧[tvshows] > Media Folders".
func describeLibraries(ancestors []jellyfin.Ancestor) string {
	parts := make([]string, 0, len(ancestors))
	for _, ancestor := range ancestors {
		if ancestor.CollectionType != "" {
			parts = append(parts, ancestor.Name+"["+ancestor.CollectionType+"]")
		}
	}
	if len(parts) == 0 {
		return "an unknown library"
	}
	return strings.Join(parts, " > ")
}

func messageSuffix(message string) string {
	if strings.TrimSpace(message) == "" {
		return ""
	}
	return " — " + message
}

func pingJellyfin(baseURL, token string, timeout time.Duration) error {
	if timeout <= 0 {
		timeout = 8 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	endpoint := strings.TrimRight(baseURL, "/") + "/System/Info"
	if strings.TrimSpace(token) == "" {
		endpoint = strings.TrimRight(baseURL, "/") + "/System/Info/Public"
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("%w: %v", jellyfin.ErrUnavailable, err)
	}
	if strings.TrimSpace(token) != "" {
		request.Header.Set("X-Emby-Token", token)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return fmt.Errorf("%w: %v", jellyfin.ErrUnavailable, err)
	}
	defer response.Body.Close()
	switch response.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusUnauthorized, http.StatusForbidden:
		return jellyfin.ErrUnauthorized
	default:
		return fmt.Errorf("%w: HTTP %d", jellyfin.ErrUnavailable, response.StatusCode)
	}
}
