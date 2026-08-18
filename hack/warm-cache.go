//go:build tools

// Command warm-cache downloads the cl100k_base tiktoken vocabulary once into
// TIKTOKEN_CACHE_DIR (default hack/vocab) so the token benchmark — and CI — can
// run fully offline afterward. Run once with network access, then commit the
// resulting file under hack/vocab/:
//
//	TIKTOKEN_CACHE_DIR=hack/vocab go run -tags tools ./hack/warm-cache.go
package main

import (
	"fmt"
	"os"

	"github.com/pkoukk/tiktoken-go"
)

func main() {
	if os.Getenv("TIKTOKEN_CACHE_DIR") == "" {
		os.Setenv("TIKTOKEN_CACHE_DIR", "hack/vocab")
	}
	if _, err := tiktoken.GetEncoding("cl100k_base"); err != nil {
		fmt.Fprintln(os.Stderr, "failed to warm cl100k_base cache:", err)
		os.Exit(1)
	}
	fmt.Printf("cl100k_base vocab cached in %s\n", os.Getenv("TIKTOKEN_CACHE_DIR"))
}
