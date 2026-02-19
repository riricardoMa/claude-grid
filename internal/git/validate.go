package git

import (
	"fmt"
	"regexp"
	"unicode"
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
	// Check if empty
	if prefix == "" {
		return fmt.Errorf("branch prefix cannot be empty")
	}

	// Check for spaces
	if regexp.MustCompile(`\s`).MatchString(prefix) {
		return fmt.Errorf("branch prefix cannot contain spaces")
	}

	// Check for forbidden characters: ~, ^, :, \, ?, *, [
	if regexp.MustCompile(`[~^:\\?\*\[]`).MatchString(prefix) {
		return fmt.Errorf("branch prefix contains forbidden characters (~, ^, :, \\, ?, *, [)")
	}

	// Check for double dots (..)
	if regexp.MustCompile(`\.\.`).MatchString(prefix) {
		return fmt.Errorf("branch prefix cannot contain double dots (..)")
	}

	// Check for consecutive forward slashes (before leading slash check)
	if regexp.MustCompile(`//`).MatchString(prefix) {
		return fmt.Errorf("branch prefix cannot contain consecutive forward slashes (//)")
	}

	// Check for leading dot
	if prefix[0] == '.' {
		return fmt.Errorf("branch prefix cannot start with a dot (.)")
	}

	// Check for trailing dot
	if prefix[len(prefix)-1] == '.' {
		return fmt.Errorf("branch prefix cannot end with a dot (.)")
	}

	// Check for leading hyphen
	if prefix[0] == '-' {
		return fmt.Errorf("branch prefix cannot start with a hyphen (-)")
	}

	// Check for trailing hyphen
	if prefix[len(prefix)-1] == '-' {
		return fmt.Errorf("branch prefix cannot end with a hyphen (-)")
	}

	// Check for leading forward slash
	if prefix[0] == '/' {
		return fmt.Errorf("branch prefix cannot start with a forward slash (/)")
	}

	// Check for ASCII printable characters only
	for _, ch := range prefix {
		if ch < 32 || ch > 126 {
			return fmt.Errorf("branch prefix contains non-ASCII printable character: %q", ch)
		}
		// Additional check: ensure it's a valid printable character
		if !unicode.IsPrint(rune(ch)) && ch != ' ' {
			return fmt.Errorf("branch prefix contains non-printable character: %q", ch)
		}
	}

	return nil
}
