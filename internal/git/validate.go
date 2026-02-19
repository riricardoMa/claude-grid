package git

import (
	"fmt"
	"regexp"
)

// ValidateBranchPrefix validates a git branch name prefix against git ref-name rules.
// Returns nil if valid, or a descriptive error if invalid.
//
// Rules enforced:
// - Non-empty
// - No spaces
// - No special characters: ~, ^, :, \, ?, *, [
// - No double dots (..)
// - No leading or trailing dots (.)
// - No leading or trailing hyphens (-)
// - No leading forward slash (/)
// - No consecutive forward slashes (//)
// - ASCII printable characters only
func ValidateBranchPrefix(prefix string) error {
	if prefix == "" {
		return fmt.Errorf("branch prefix cannot be empty")
	}

	if regexp.MustCompile(`\s`).MatchString(prefix) {
		return fmt.Errorf("branch prefix cannot contain spaces")
	}

	if regexp.MustCompile(`[~^:\\?\*\[]`).MatchString(prefix) {
		return fmt.Errorf("branch prefix contains forbidden characters (~, ^, :, \\, ?, *, [)")
	}

	if regexp.MustCompile(`\.\.`).MatchString(prefix) {
		return fmt.Errorf("branch prefix cannot contain double dots (..)")
	}

	if regexp.MustCompile(`//`).MatchString(prefix) {
		return fmt.Errorf("branch prefix cannot contain consecutive forward slashes (//)")
	}

	if prefix[0] == '.' {
		return fmt.Errorf("branch prefix cannot start with a dot (.)")
	}

	if prefix[len(prefix)-1] == '.' {
		return fmt.Errorf("branch prefix cannot end with a dot (.)")
	}

	if prefix[0] == '-' {
		return fmt.Errorf("branch prefix cannot start with a hyphen (-)")
	}

	if prefix[len(prefix)-1] == '-' {
		return fmt.Errorf("branch prefix cannot end with a hyphen (-)")
	}

	if prefix[0] == '/' {
		return fmt.Errorf("branch prefix cannot start with a forward slash (/)")
	}

	for _, ch := range prefix {
		if ch < 32 || ch > 126 {
			return fmt.Errorf("branch prefix contains non-ASCII printable character: %q", ch)
		}
	}

	return nil
}
