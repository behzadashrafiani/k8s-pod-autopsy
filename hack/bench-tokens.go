//go:build tools

// Command bench-tokens measures how many tokens the autopsy digest saves versus
// a raw kubectl describe+logs+events dump for the same crashed pod. It reads the
// committed cl100k_base vocab from TIKTOKEN_CACHE_DIR (default hack/vocab) so it
// runs offline. Regenerate the cache with hack/warm-cache.go if it is missing.
//
//	go run -tags tools ./hack/bench-tokens.go
package main

import (
	"fmt"
	"os"

	"github.com/pkoukk/tiktoken-go"
)

func count(enc *tiktoken.Tiktoken, s string) int { return len(enc.Encode(s, nil, nil)) }

func main() {
	// Offline: read the committed vocab dir.
	if os.Getenv("TIKTOKEN_CACHE_DIR") == "" {
		os.Setenv("TIKTOKEN_CACHE_DIR", "hack/vocab")
	}
	enc, err := tiktoken.GetEncoding("cl100k_base")
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to load cl100k_base (run hack/warm-cache.go first):", err)
		os.Exit(1)
	}
	raw, err := os.ReadFile("hack/fixtures/raw-kubectl.txt") // describe+logs+events dump
	if err != nil {
		fmt.Fprintln(os.Stderr, "reading raw fixture:", err)
		os.Exit(1)
	}
	digest, err := os.ReadFile("hack/fixtures/autopsy.md") // our rendered digest
	if err != nil {
		fmt.Fprintln(os.Stderr, "reading digest fixture:", err)
		os.Exit(1)
	}
	rawN, digN := count(enc, string(raw)), count(enc, string(digest))
	fmt.Printf("raw kubectl:  %d tokens\n", rawN)
	fmt.Printf("autopsy:      %d tokens\n", digN)
	fmt.Printf("saved:        %.0f%% (cl100k BPE; accurate for GPT, close proxy for Claude)\n",
		100*(1-float64(digN)/float64(rawN)))
}
