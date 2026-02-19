package git

import (
	"testing"
)

func TestValidateBranchPrefix(t *testing.T) {
	tests := []struct {
		name    string
		prefix  string
		wantErr bool
		errMsg  string
	}{
		// Valid cases
		{
			name:    "simple alphanumeric",
			prefix:  "abc123",
			wantErr: false,
		},
		{
			name:    "single character",
			prefix:  "a",
			wantErr: false,
		},
		{
			name:    "hyphen separated",
			prefix:  "sprint-42",
			wantErr: false,
		},
		{
			name:    "slash separated",
			prefix:  "feature/auth",
			wantErr: false,
		},
		{
			name:    "user name format",
			prefix:  "user-name",
			wantErr: false,
		},
		{
			name:    "complex valid name",
			prefix:  "feature/user-auth-v2",
			wantErr: false,
		},
		{
			name:    "numbers and underscores",
			prefix:  "release_1_0_0",
			wantErr: false,
		},
		{
			name:    "mixed case",
			prefix:  "Feature-Branch",
			wantErr: false,
		},

		// Invalid cases - spaces
		{
			name:    "space in middle",
			prefix:  "bad name",
			wantErr: true,
			errMsg:  "cannot contain spaces",
		},
		{
			name:    "leading space",
			prefix:  " badname",
			wantErr: true,
			errMsg:  "cannot contain spaces",
		},
		{
			name:    "trailing space",
			prefix:  "badname ",
			wantErr: true,
			errMsg:  "cannot contain spaces",
		},

		// Invalid cases - forbidden characters
		{
			name:    "tilde character",
			prefix:  "bad~1",
			wantErr: true,
			errMsg:  "forbidden characters",
		},
		{
			name:    "caret character",
			prefix:  "bad^1",
			wantErr: true,
			errMsg:  "forbidden characters",
		},
		{
			name:    "colon character",
			prefix:  "bad:1",
			wantErr: true,
			errMsg:  "forbidden characters",
		},
		{
			name:    "backslash character",
			prefix:  "bad\\1",
			wantErr: true,
			errMsg:  "forbidden characters",
		},
		{
			name:    "question mark",
			prefix:  "bad?1",
			wantErr: true,
			errMsg:  "forbidden characters",
		},
		{
			name:    "asterisk character",
			prefix:  "bad*1",
			wantErr: true,
			errMsg:  "forbidden characters",
		},
		{
			name:    "bracket character",
			prefix:  "bad[1]",
			wantErr: true,
			errMsg:  "forbidden characters",
		},

		// Invalid cases - double dots
		{
			name:    "double dots",
			prefix:  "bad..name",
			wantErr: true,
			errMsg:  "double dots",
		},
		{
			name:    "double dots at start",
			prefix:  "..badname",
			wantErr: true,
			errMsg:  "double dots",
		},
		{
			name:    "double dots at end",
			prefix:  "badname..",
			wantErr: true,
			errMsg:  "double dots",
		},

		// Invalid cases - leading/trailing dots
		{
			name:    "leading dot",
			prefix:  ".hidden",
			wantErr: true,
			errMsg:  "cannot start with a dot",
		},
		{
			name:    "trailing dot",
			prefix:  "hidden.",
			wantErr: true,
			errMsg:  "cannot end with a dot",
		},

		// Invalid cases - leading/trailing hyphens
		{
			name:    "leading hyphen",
			prefix:  "-badname",
			wantErr: true,
			errMsg:  "cannot start with a hyphen",
		},
		{
			name:    "trailing hyphen",
			prefix:  "badname-",
			wantErr: true,
			errMsg:  "cannot end with a hyphen",
		},

		// Invalid cases - leading slash
		{
			name:    "leading slash",
			prefix:  "/badname",
			wantErr: true,
			errMsg:  "cannot start with a forward slash",
		},

		// Invalid cases - consecutive slashes
		{
			name:    "consecutive slashes",
			prefix:  "bad//name",
			wantErr: true,
			errMsg:  "consecutive forward slashes",
		},
		{
			name:    "double slash at start",
			prefix:  "//badname",
			wantErr: true,
			errMsg:  "consecutive forward slashes",
		},

		// Invalid cases - empty
		{
			name:    "empty string",
			prefix:  "",
			wantErr: true,
			errMsg:  "cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateBranchPrefix(tt.prefix)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateBranchPrefix(%q) error = %v, wantErr %v", tt.prefix, err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateBranchPrefix(%q) error = %q, want error containing %q", tt.prefix, err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
