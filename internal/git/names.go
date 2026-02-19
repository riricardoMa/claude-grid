package git

import (
	"crypto/rand"
	"fmt"
)

var adjectives = []string{
	"brave", "swift", "calm", "bold", "keen", "warm", "cool", "neat", "fair", "wise",
	"glad", "soft", "pure", "wild", "free", "true", "deep", "high", "rich", "slim",
	"rare", "fast", "safe", "firm", "mild", "dark", "pale", "loud", "sly", "dry",
}

var nouns = []string{
	"fox", "elk", "owl", "lynx", "wolf", "bear", "hawk", "deer", "crow", "dove",
	"hare", "seal", "wren", "mink", "frog", "moth", "newt", "pike", "colt", "swan",
	"lark", "crab", "mole", "toad", "wasp", "goat", "lamb", "puma", "orca", "ibis",
}

// GenerateBranchPrefix returns a random adjective-noun pair like "brave-fox".
// The returned string follows the pattern ^[a-z]+-[a-z]+$.
func GenerateBranchPrefix() string {
	adjIdx := randomIndex(len(adjectives))
	nounIdx := randomIndex(len(nouns))
	return fmt.Sprintf("%s-%s", adjectives[adjIdx], nouns[nounIdx])
}

// randomIndex returns a random index in the range [0, max).
func randomIndex(max int) int {
	b := make([]byte, 1)
	rand.Read(b)
	return int(b[0]) % max
}
