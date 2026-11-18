package pathmap

import (
	"fmt"
	"io"
	"os"
	"path"
	"strings"
)

// NormalizeSource accepts only absolute POSIX container paths. Reject traversal
// before cleaning: path.Clean would otherwise erase the evidence.
func NormalizeSource(source string) (string, error) {
	source = strings.TrimSpace(source)
	if !strings.HasPrefix(source, "/") || strings.ContainsAny(source, "\\\x00") || hasTraversal(source, "/") {
		return "", fmt.Errorf("source must be an absolute container path without backslashes or .. segments")
	}
	return strings.TrimRight(path.Clean(source), "/") + "/", nil
}

// NormalizeTarget preserves the UNC volume (server + share). Do not run a
// Windows path through path.Clean, which collapses its leading double slash.
func NormalizeTarget(target string) (string, error) {
	target = strings.ReplaceAll(strings.TrimSpace(target), "/", "\\")
	var root, rest string
	switch {
	case len(target) >= 3 && ((target[0] >= 'A' && target[0] <= 'Z') || (target[0] >= 'a' && target[0] <= 'z')) && target[1:3] == `:\`:
		root, rest = target[:3], target[3:]
	case strings.HasPrefix(target, `\\`):
		parts := strings.Split(target[2:], `\`)
		if len(parts) < 2 || !safeWindowsSegment(parts[0]) || !safeWindowsSegment(parts[1]) {
			return "", fmt.Errorf("UNC target requires a server and share")
		}
		root = `\\` + strings.Join(parts[:2], `\`) + `\`
		rest = strings.Join(parts[2:], `\`)
	default:
		return "", fmt.Errorf("target must be a Windows drive path like Z:\\Movies\\ or an SMB UNC path like \\\\server\\share\\Movies\\")
	}
	parts := make([]string, 0)
	for _, segment := range strings.Split(rest, `\`) {
		if segment == "" {
			continue
		}
		if !safeWindowsSegment(segment) {
			return "", fmt.Errorf("target contains an unsafe Windows path segment")
		}
		parts = append(parts, segment)
	}
	if len(parts) == 0 {
		return root, nil
	}
	return root + strings.Join(parts, `\`) + `\`, nil
}

func hasTraversal(value, separator string) bool {
	for _, segment := range strings.Split(value, separator) {
		if segment == ".." {
			return true
		}
	}
	return false
}

// Windows trims trailing dots/spaces and treats ':' as an alternate stream.
// Refuse such aliases as well as separators, device prefixes and control bytes.
func safeWindowsSegment(segment string) bool {
	if segment == "" || strings.HasSuffix(segment, ".") || strings.HasSuffix(segment, " ") || strings.ContainsAny(segment, `<>:"/\|?*`) {
		return false
	}
	for _, r := range segment {
		if r < 32 {
			return false
		}
	}
	return true
}

// CheckTargetDirectory checks both existence and directory read permission,
// including for SMB shares. An empty readable directory is valid.
func CheckTargetDirectory(target string) error {
	dir, err := os.Open(target)
	if err != nil {
		return err
	}
	defer dir.Close()
	info, err := dir.Stat()
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("mapping target is not a directory")
	}
	_, err = dir.ReadDir(1)
	if err == io.EOF {
		return nil
	}
	return err
}
