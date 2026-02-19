package pathutil

import (
	"fmt"
	"os"
	"strings"
)

// ExpandTilde replaces a leading ~ with the user's home directory.
// Returns error for ~user syntax (unsupported).
// Passes through non-tilde paths unchanged.
func ExpandTilde(path string) (string, error) {
	if path == "" {
		return "", nil
	}

	if !strings.HasPrefix(path, "~") {
		return path, nil
	}

	if len(path) > 1 && path[1] != '/' {
		return "", fmt.Errorf("unsupported ~user syntax in path: %s", path)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	if path == "~" {
		return homeDir, nil
	}

	return homeDir + path[1:], nil
}

// ExpandTildeAll applies ExpandTilde to each element, returning on first error.
func ExpandTildeAll(paths []string) ([]string, error) {
	result := make([]string, len(paths))
	for i, p := range paths {
		expanded, err := ExpandTilde(p)
		if err != nil {
			return nil, err
		}
		result[i] = expanded
	}
	return result, nil
}
