package git

import (
	"regexp"
	"testing"
)

func TestGenerateBranchPrefixFormat(t *testing.T) {
	pattern := regexp.MustCompile(`^[a-z]+-[a-z]+$`)
	for i := 0; i < 20; i++ {
		result := GenerateBranchPrefix()
		if !pattern.MatchString(result) {
			t.Errorf("GenerateBranchPrefix() = %q, want pattern ^[a-z]+-[a-z]+$", result)
		}
	}
}

func TestGenerateBranchPrefixVariety(t *testing.T) {
	results := make(map[string]bool)
	for i := 0; i < 10; i++ {
		results[GenerateBranchPrefix()] = true
	}
	if len(results) < 2 {
		t.Errorf("GenerateBranchPrefix() produced only %d unique values in 10 calls, want ≥2", len(results))
	}
}

func TestGenerateBranchPrefixValidWords(t *testing.T) {
	adjSet := make(map[string]bool)
	for _, adj := range adjectives {
		adjSet[adj] = true
	}
	nounSet := make(map[string]bool)
	for _, noun := range nouns {
		nounSet[noun] = true
	}

	for i := 0; i < 50; i++ {
		result := GenerateBranchPrefix()
		parts := regexp.MustCompile(`^([a-z]+)-([a-z]+)$`).FindStringSubmatch(result)
		if len(parts) != 3 {
			t.Errorf("GenerateBranchPrefix() = %q, could not parse", result)
			continue
		}
		adj, noun := parts[1], parts[2]
		if !adjSet[adj] {
			t.Errorf("GenerateBranchPrefix() returned unknown adjective %q", adj)
		}
		if !nounSet[noun] {
			t.Errorf("GenerateBranchPrefix() returned unknown noun %q", noun)
		}
	}
}
